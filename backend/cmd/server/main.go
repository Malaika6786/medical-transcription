// Package main is the entry point for the Corti Medical Transcription Backend
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"corti-backend/internal/ai"
	"corti-backend/internal/auth"
	"corti-backend/internal/corti"
	"corti-backend/internal/cryptofield"
	"corti-backend/internal/embedding"
	"corti-backend/internal/handlers"
	"corti-backend/internal/middleware"
	"corti-backend/internal/nhs"
	"corti-backend/internal/pgstore"
	"corti-backend/internal/utils"
)

func main() {
	// Load configuration
	config := utils.LoadConfig()

	// Refuse to boot with a data-flow configuration that sends patient
	// audio/transcript/extraction content outside the UK/EU by default —
	// see SYSTMONE_INTEGRATION_REPORT.md, "Default AI data flows are not
	// UK-hosted". Set DATA_RESIDENCY_UK_ONLY=false only for local/demo use
	// with synthetic data.
	if err := config.ValidateDataResidency(); err != nil {
		log.Fatalf("%v", err)
	}

	// Initialize Fiber app with custom error handler
	app := fiber.New(fiber.Config{
		ErrorHandler:          middleware.ErrorHandler,
		DisableStartupMessage: false,
		BodyLimit:             100 * 1024 * 1024, // 100MB for audio uploads
		ReadBufferSize:        16384,
		WriteBufferSize:       16384,
		StreamRequestBody:     true,
	})

	// Apply global middleware
	app.Use(middleware.RecoverConfig())
	app.Use(middleware.RequestLogger())
	app.Use(middleware.CORSConfig(config))

	// Connect to PostgreSQL — the system of record for users, roles, sessions,
	// templates, and embeddings (docs/adr/0001). The legacy JSON files are only
	// read by cmd/migrate-json now.
	ctx, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()

	store, err := pgstore.Connect(ctx, config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v\n"+
			"  Is PostgreSQL running and the database created?\n"+
			"    brew services start postgresql@17 && createdb meditrans\n"+
			"  Then import legacy JSON data with: go run ./cmd/migrate-json", err)
	}
	defer store.Close()

	if err := store.ApplySchema(ctx); err != nil {
		log.Fatalf("Failed to apply database schema: %v", err)
	}
	if err := store.EnsureSeed(ctx); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}

	// Field-level encryption for patient PII (nhs_number/name/date_of_birth
	// — see internal/cryptofield and SYSTMONE_INTEGRATION_REPORT.md, "No
	// encryption at application/database field level"). Patient/NHS routes
	// are only registered when a key is configured — see setupRoutes.
	var fieldCipher *cryptofield.Cipher
	if config.FieldEncryptionKey != "" {
		fieldCipher, err = cryptofield.NewFromHexKey(config.FieldEncryptionKey)
		if err != nil {
			log.Fatalf("Invalid FIELD_ENCRYPTION_KEY: %v", err)
		}
		store.SetFieldCipher(fieldCipher)
		log.Println("Field encryption enabled for patient PII (FIELD_ENCRYPTION_KEY configured)")
	} else {
		log.Println("WARNING: FIELD_ENCRYPTION_KEY not set — patient/NHS integration routes are disabled. " +
			"Generate one with `openssl rand -hex 32` before using patient records or GP Connect: Send Document.")
	}

	// All three storage contracts are served by the Postgres store.
	var (
		userStore    auth.UserStorage    = store
		roleStore    auth.RoleStorage    = store
		sessionStore auth.SessionStorage = store
	)

	// JWT secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "xstek-demo-secret-key-2026-change-in-production"
		log.Println("WARNING: Using default JWT secret. Set JWT_SECRET env var in production!")
	}
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)

	// Embedding pipeline: bge-small-en-v1.5 via an OpenAI-compatible endpoint
	// (docs/adr/0002). Session saves only notify the worker — they never wait
	// on the model, and NULL embedding columns are backfilled at startup.
	embedder := ai.NewEmbedder(
		config.EmbedBaseURL,
		config.EmbedModel,
		config.EmbedAPIKey,
		time.Duration(config.EmbedTimeoutSeconds)*time.Second,
	)
	embedWorker := embedding.NewWorker(store, embedder)
	go embedWorker.Run(ctx)
	log.Printf(
		"Embedding pipeline: endpoint=%s model=%s dims=%d",
		config.EmbedBaseURL,
		config.EmbedModel,
		ai.EmbeddingDimensions,
	)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userStore, roleStore, jwtManager)
	rolesHandler := handlers.NewRolesHandler(roleStore, userStore)
	sessionsHandler := handlers.NewSessionsHandler(sessionStore, embedWorker)
	searchHandler := handlers.NewSearchHandler(store, embedder)
	adminHandler := handlers.NewAdminHandler(userStore, roleStore, sessionStore, store)

	// Initialize Corti clients
	tokenManager := corti.NewTokenManager(config)
	asyncClient := corti.NewAsyncClient(config, tokenManager)
	ambientProxy := corti.NewAmbientProxy(config, tokenManager, asyncClient)
	dictationProxy := corti.NewDictationProxy(config, tokenManager)
	embeddedCache := corti.NewEmbeddedTokenCache(config)

	asyncHandler := handlers.NewAsyncHandler(asyncClient, store)
	ambientHandler := handlers.NewAmbientHandler(ambientProxy)
	dictationHandler := handlers.NewDictationHandler(dictationProxy)
	embeddedHandler := handlers.NewEmbeddedHandler(embeddedCache)

	// Initialize AI Assistance module (independent of Corti — the transcript
	// text is the only handoff between the two).
	aiTimeout := time.Duration(config.AITimeoutSeconds) * time.Second
	llmClient := ai.NewOpenAICompatClient(
		config.AIBaseURL,
		config.AIModel,
		config.AIAPIKey,
		config.AITemperature,
		aiTimeout,
		config.AIMaxCompletionTokens,
	)
	aiHandler := handlers.NewAIHandler(llmClient, aiTimeout)
	log.Printf(
		"AI Assistance module: endpoint=%s model=%s temperature=%g max_completion_tokens=%d",
		config.AIBaseURL,
		config.AIModel,
		config.AITemperature,
		config.AIMaxCompletionTokens,
	)

	// NHS/SystmOne integration handlers (internal/nhs) — patients/audit
	// only registered once field encryption is configured (see above);
	// PDS/MESH clients are always built (they simply return a clear error
	// if their credentials aren't configured — see nhs.PDSClient/MESHClient)
	// so the routes exist and respond sensibly even before real NHS
	// credentials are wired in.
	var patientHandler *handlers.PatientHandler
	var nhsHandler *handlers.NHSHandler
	var auditHandler *handlers.AuditHandler
	if fieldCipher != nil {
		pdsClient := nhs.NewPDSClient(config.PDSBaseURL, config.PDSAPIKey)
		meshClient := nhs.NewMESHClient(nhs.MESHConfig{
			BaseURL:         config.MeshBaseURL,
			MailboxID:       config.MeshMailboxID,
			MailboxPassword: config.MeshMailboxPassword,
			SharedKey:       config.MeshSharedKey,
		})
		patientHandler = handlers.NewPatientHandler(store)
		nhsHandler = handlers.NewNHSHandler(sessionStore, store, pdsClient, meshClient, config.NHSOrgODSCode, "XStek AI Medical Transcription")
		auditHandler = handlers.NewAuditHandler(store)
		log.Printf("NHS integration module: PDS=%s MESH=%s (see docs/nhs/README.md for what's live vs. needs NHS-issued credentials)", config.PDSBaseURL, config.MeshBaseURL)
	}

	// Setup routes
	setupRoutes(app, authHandler, rolesHandler, asyncHandler, ambientHandler, dictationHandler, sessionsHandler, embeddedHandler, aiHandler, searchHandler, adminHandler, patientHandler, nhsHandler, auditHandler, store)

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "corti-backend",
			"version": "2.0.0",
		})
	})

	// Start server
	addr := fmt.Sprintf("%s:%s", config.ServerHost, config.ServerPort)
	log.Printf("Starting Corti Backend Server on %s", addr)

	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server shutdown complete")
}

// setupRoutes configures all API routes.
func setupRoutes(
	app *fiber.App,
	authHandler *handlers.AuthHandler,
	rolesHandler *handlers.RolesHandler,
	asyncHandler *handlers.AsyncHandler,
	ambientHandler *handlers.AmbientHandler,
	dictationHandler *handlers.DictationHandler,
	sessionsHandler *handlers.SessionsHandler,
	embeddedHandler *handlers.EmbeddedHandler,
	aiHandler *handlers.AIHandler,
	searchHandler *handlers.SearchHandler,
	adminHandler *handlers.AdminHandler,
	patientHandler *handlers.PatientHandler, // nil if FIELD_ENCRYPTION_KEY is not configured
	nhsHandler *handlers.NHSHandler, // nil if FIELD_ENCRYPTION_KEY is not configured
	auditHandler *handlers.AuditHandler, // nil if FIELD_ENCRYPTION_KEY is not configured
	store *pgstore.Store,
) {
	api := app.Group("/api")

	// Convenience alias for the permission middleware factory.
	requirePerm := middleware.RequirePermission
	// Convenience alias for the demo-trial-limit middleware factory —
	// applied per-route below, only on the specific action that "uses" a
	// feature (not e.g. polling/read routes in the same group).
	requireDemoAllowance := func(feature string) fiber.Handler {
		return middleware.RequireDemoAllowance(store, feature)
	}

	// Auth middleware used on every protected route.
	authMW := authHandler.AuthMiddleware()
	// Dictation's WS has no prior REST call to gate it the way ambient's
	// interactionId does (see below) — it needs real authentication, but
	// browsers can't send an Authorization header during a WS handshake,
	// so it accepts the token via ?token= instead. See the doc comment on
	// AuthMiddlewareAllowQueryToken.
	wsAuthMW := authHandler.AuthMiddlewareAllowQueryToken()

	// ============================================
	// WebSocket Routes (registered before their group's other routes)
	// ============================================
	// Ambient: PUBLIC upgrade — security relies on the interactionId being
	// obtained via a protected REST call (POST /ambient/start) first;
	// that's where ambient's own auth/permission/demo-limit checks live.
	api.Use("/ambient/ws/:interactionId", ambientHandler.HandleWebSocketUpgrade())
	api.Get("/ambient/ws/:interactionId", ambientHandler.HandleWebSocket())
	// Dictation: unlike ambient, there's no prior REST call — the WS
	// connection *is* the start of the session, so auth/permission/demo-
	// limit all have to be checked right here.
	api.Use("/dictation/ws", dictationHandler.HandleWebSocketUpgrade())
	api.Get("/dictation/ws", wsAuthMW, requirePerm(auth.PermDictation), requireDemoAllowance("dictation"), dictationHandler.HandleWebSocket())

	// ============================================
	// Authentication (Public + protected)
	// ============================================
	authGroup := api.Group("/auth")
	authGroup.Post("/login", authHandler.HandleLogin)
	authGroup.Post("/signup", authHandler.HandleSignup)
	authGroup.Get("/me", authMW, authHandler.HandleGetCurrentUser)
	authGroup.Post("/force-logout-all", authMW, requirePerm(auth.PermUsersManage), authHandler.HandleForceLogoutAll)

	// A demo account's own trial-usage counts — any authenticated user (a
	// non-demo account just gets empty counts back).
	api.Get("/users/me/demo-usage", authMW, adminHandler.HandleGetMyDemoUsage)

	// Superuser: view any user's saved sessions ("keep an eye on all the
	// activity happening in the app").
	api.Get("/admin/sessions/:userId", authMW, requirePerm(auth.PermUsersManage), adminHandler.HandleGetUserSessions)

	// Command Center analytics (requires users.manage).
	api.Get("/admin/analytics/overview", authMW, requirePerm(auth.PermUsersManage), adminHandler.HandleGetAnalyticsOverview)
	api.Get("/admin/analytics/usage-timeseries", authMW, requirePerm(auth.PermUsersManage), adminHandler.HandleGetUsageTimeseries)
	api.Get("/admin/analytics/demo-funnel", authMW, requirePerm(auth.PermUsersManage), adminHandler.HandleGetDemoFunnel)
	api.Get("/admin/analytics/recent-sessions", authMW, requirePerm(auth.PermUsersManage), adminHandler.HandleGetRecentSessionsAllUsers)

	// ============================================
	// User Management  (requires users.manage)
	// ============================================
	users := api.Group("/users", authMW, requirePerm(auth.PermUsersManage))
	users.Get("/", authHandler.HandleListUsers)
	users.Post("/", authHandler.HandleCreateUser)
	users.Put("/:id", authHandler.HandleUpdateUser)
	users.Delete("/:id", authHandler.HandleDeleteUser)

	// Per-user permission management (requires users.manage)
	users.Get("/:id/permissions", authHandler.HandleGetUserPermissions)
	users.Put("/:id/roles", authHandler.HandleUpdateUserRoles)
	users.Post("/:id/permissions/grant", authHandler.HandleGrantPermission)
	users.Post("/:id/permissions/deny", authHandler.HandleDenyPermission)
	users.Delete("/:id/permissions/:perm", authHandler.HandleRemovePermissionOverride)
	users.Put("/:id/password", authHandler.HandleResetPassword)

	// Signup-approval workflow (requires users.manage).
	users.Get("/pending", adminHandler.HandleListPendingUsers)
	users.Post("/:id/approve", adminHandler.HandleApproveUser)
	users.Post("/:id/reject", adminHandler.HandleRejectUser)

	// ============================================
	// Role Management  (requires users.manage)
	// ============================================
	roles := api.Group("/roles", authMW, requirePerm(auth.PermUsersManage))
	roles.Get("/", rolesHandler.HandleListRoles)
	roles.Post("/", rolesHandler.HandleCreateRole)
	roles.Get("/:id", rolesHandler.HandleGetRole)
	roles.Put("/:id", rolesHandler.HandleUpdateRole)
	roles.Delete("/:id", rolesHandler.HandleDeleteRole)

	// Permission constants listing  (requires users.manage)
	api.Get("/permissions", authMW, requirePerm(auth.PermUsersManage), rolesHandler.HandleListPermissions)

	// ============================================
	// Async Transcription  (requires file_transcription.access)
	// ============================================
	transcribe := api.Group("/transcribe", authMW, requirePerm(auth.PermFileTranscription))
	transcribe.Post("/upload", requireDemoAllowance("file_transcription"), asyncHandler.HandleUpload)
	transcribe.Get("/:interactionId", asyncHandler.HandleGetTranscript)
	transcribe.Get("/:interactionId/poll", asyncHandler.HandlePollTranscript)
	transcribe.Post("/:interactionId/document", requireDemoAllowance("document_generation"), asyncHandler.HandleGenerateDocument)
	transcribe.Get("/:interactionId/document/:documentId", asyncHandler.HandleGetDocument)
	transcribe.Get("/:interactionId/documents", asyncHandler.HandleListDocuments)
	transcribe.Post("/generate-from-transcript", asyncHandler.GenerateFromTranscript)

	// ============================================
	// Template Management
	// ============================================
	// Listing templates — any authenticated user with file_transcription can use them.
	api.Get("/templates", authMW, requirePerm(auth.PermFileTranscription), asyncHandler.GetAvailableTemplates)

	// Corti sections + custom template CRUD — templates.manage only.
	customTemplates := api.Group("/templates/custom", authMW, requirePerm(auth.PermTemplatesManage))
	customTemplates.Get("/", asyncHandler.GetCustomTemplates)
	customTemplates.Get("/sections", asyncHandler.GetAvailableSections)
	customTemplates.Get("/:key", asyncHandler.GetCustomTemplate)
	customTemplates.Post("/", asyncHandler.CreateCustomTemplate)
	customTemplates.Put("/:key", asyncHandler.UpdateCustomTemplate)
	customTemplates.Delete("/:key", asyncHandler.DeleteCustomTemplate)

	api.Get("/templates/corti-sections", authMW, requirePerm(auth.PermCortiSectionsView), asyncHandler.GetCortiTemplateSections)

	// ============================================
	// Ambient Streaming  (requires ambient.access)
	// ============================================
	ambient := api.Group("/ambient", authMW, requirePerm(auth.PermAmbientAccess))
	ambient.Post("/start", requireDemoAllowance("ambient"), ambientHandler.HandleStartSession)
	ambient.Get("/session/:interactionId", ambientHandler.HandleGetSessionStatus)
	ambient.Get("/stats", ambientHandler.HandleGetStats)

	// ============================================
	// Dictation  (requires dictation.access)
	// ============================================
	dictation := api.Group("/dictation", authMW, requirePerm(auth.PermDictation))
	dictation.Get("/stats", dictationHandler.HandleGetStats)

	// ============================================
	// Embedded Assistant  (requires embedded_assistant.access)
	// ============================================
	api.Get("/embedded/token", authMW, requirePerm(auth.PermEmbeddedAssistant), embeddedHandler.HandleGetToken)

	// ============================================
	// AI Assistance  (gated by ambient.access for now — swap to a dedicated
	// ai_assistant.access permission when the RBAC granularity is needed)
	// ============================================
	aiGroup := api.Group("/ai", authMW, requirePerm(auth.PermAmbientAccess))
	aiGroup.Post("/summarize", requireDemoAllowance("ai_summary"), aiHandler.HandleSummarize)
	aiGroup.Post("/chat", aiHandler.HandleChat)
	aiGroup.Get("/status", aiHandler.HandleStatus)

	// ============================================
	// Saved Sessions  (any authenticated user)
	// ============================================
	sessions := api.Group("/sessions", authMW)
	sessions.Get("/", sessionsHandler.HandleListSessions)
	sessions.Get("/:id", sessionsHandler.HandleGetSession)
	sessions.Post("/", sessionsHandler.HandleSaveSession)
	sessions.Put("/:id/document", sessionsHandler.HandleUpdateSessionDocument)
	sessions.Delete("/:id", sessionsHandler.HandleDeleteSession)

	// ============================================
	// Semantic Search  (any authenticated user — results are scoped to the
	// caller's own sessions inside the SQL)
	// ============================================
	api.Post("/search", authMW, requireDemoAllowance("search"), searchHandler.HandleSearch)

	// ============================================
	// NHS/SystmOne Integration  (requires patients.manage / nhs.integration
	// / audit.view — see SYSTMONE_INTEGRATION_REPORT.md). Only registered
	// once FIELD_ENCRYPTION_KEY is configured; otherwise these paths 404
	// rather than silently handling patient data unencrypted.
	// ============================================
	if patientHandler != nil && nhsHandler != nil && auditHandler != nil {
		patients := api.Group("/patients", authMW, requirePerm(auth.PermPatientsManage))
		patients.Post("/", patientHandler.HandleCreatePatient)
		patients.Get("/", patientHandler.HandleListPatients)
		patients.Get("/:id", patientHandler.HandleGetPatient)
		patients.Put("/:id", patientHandler.HandleUpdatePatient)
		patients.Delete("/:id", patientHandler.HandleDeletePatient)
		patients.Post("/:id/verify-pds", requirePerm(auth.PermNHSIntegration), nhsHandler.HandleVerifyPatientAgainstPDS)

		sessions.Put("/:id/patient", requirePerm(auth.PermPatientsManage), patientHandler.HandleLinkSession)

		nhsGroup := api.Group("/nhs", authMW, requirePerm(auth.PermNHSIntegration))
		nhsGroup.Post("/pds/trace", nhsHandler.HandleTraceByNHSNumber)
		nhsGroup.Post("/pds/search", nhsHandler.HandleSearchByDemographics)
		nhsGroup.Post("/sessions/:id/send-to-gp", nhsHandler.HandleSendToGP)

		api.Get("/audit", authMW, requirePerm(auth.PermAuditView), auditHandler.HandleListAuditEvents)
	}

	log.Println("Routes configured (RBAC v2 — permission-based)")
}
