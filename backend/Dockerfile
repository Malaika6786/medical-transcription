# Production image for the Go/Fiber backend — added per
# SYSTMONE_INTEGRATION_REPORT.md, "No production IaC/container setup
# found". Multi-stage: build in a full Go image, ship only the compiled
# binary in a minimal runtime image (smaller attack surface, faster
# deploys, easier to reason about for a DTAC Technical Security review
# than "a binary someone built and copied by hand").

FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO disabled — the pgx driver is pure Go, and a static binary is simpler
# to run in a minimal base image below.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/corti-backend ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS run
WORKDIR /app
COPY --from=build /out/corti-backend ./corti-backend
# cmd/server/templates/custom/*.json are one-time seed data consumed by
# `go run ./cmd/migrate-json`, not read by the running server — not copied
# into this image; run that migration once against the target database
# instead (see CLAUDE.md).
# No .env is copied into the image on purpose — all configuration is
# expected via real environment variables at deploy time (see
# docker-compose.prod.yml and backend/.env.example), never baked into the
# image itself.
USER nonroot:nonroot
EXPOSE 8090
ENTRYPOINT ["./corti-backend"]
