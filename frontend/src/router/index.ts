import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated, isApproved, hasPermission, defaultLandingPath } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginPage.vue'),
      meta: { requiresAuth: false, isLoginPage: true }
    },
    {
      path: '/signup',
      name: 'signup',
      component: () => import('@/views/SignupPage.vue'),
      meta: { requiresAuth: false, isLoginPage: true }
    },
    {
      path: '/pending-approval',
      name: 'pending-approval',
      component: () => import('@/views/PendingApprovalPage.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/pending-users',
      name: 'pending-users',
      component: () => import('@/views/PendingApprovalsPage.vue'),
      meta: { requiresAuth: true, permission: 'users.manage' }
    },
    {
      path: '/',
      name: 'root',
      redirect: () => defaultLandingPath()
    },
    {
      path: '/home',
      name: 'home',
      component: () => import('@/views/HomePage.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/my-dashboard',
      name: 'my-dashboard',
      component: () => import('@/views/MyDashboardPage.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/command-center',
      name: 'command-center',
      component: () => import('@/views/CommandCenterPage.vue'),
      meta: { requiresAuth: true, permission: 'users.manage' }
    },
    {
      path: '/user-dashboard/:userId',
      name: 'user-dashboard',
      component: () => import('@/views/MyDashboardPage.vue'),
      meta: { requiresAuth: true, permission: 'users.manage' }
    },
    {
      path: '/async-transcription',
      name: 'async-transcription',
      component: () => import('@/views/AsyncTranscriptionPage.vue'),
      meta: { requiresAuth: true, permission: 'file_transcription.access' }
    },
    {
      path: '/ambient-session',
      name: 'ambient-session',
      component: () => import('@/views/AmbientSessionPage.vue'),
      meta: { requiresAuth: true, permission: 'ambient.access' }
    },
    {
      path: '/dictation',
      name: 'dictation',
      component: () => import('@/views/DictationPage.vue'),
      meta: { requiresAuth: true, permission: 'dictation.access' }
    },
    {
      path: '/saved-sessions',
      name: 'saved-sessions',
      component: () => import('@/views/SavedSessionsPage.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/search',
      name: 'semantic-search',
      component: () => import('@/views/SemanticSearchPage.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/demo-guide',
      name: 'demo-guide',
      component: () => import('@/views/DemoGuidePage.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/docs',
      name: 'documentation',
      component: () => import('@/views/DocumentationPage.vue'),
      meta: { requiresAuth: true, permission: 'documentation.view' }
    },
    {
      path: '/users',
      name: 'user-management',
      component: () => import('@/views/UserManagementPage.vue'),
      meta: { requiresAuth: true, permission: 'users.manage' }
    },
    {
      path: '/roles',
      name: 'role-management',
      component: () => import('@/views/RoleManagementPage.vue'),
      meta: { requiresAuth: true, permission: 'users.manage' }
    },
    {
      path: '/template-manager',
      name: 'template-manager',
      component: () => import('@/views/TemplateManagerPage.vue'),
      meta: { requiresAuth: true, permission: 'templates.manage' }
    },
    {
      path: '/transcript-test',
      name: 'transcript-test',
      component: () => import('@/views/TranscriptTestPage.vue'),
      meta: { requiresAuth: true, permission: 'users.manage' }
    },
    {
      path: '/corti-sections',
      name: 'corti-sections',
      component: () => import('@/views/CortiSectionsPage.vue'),
      meta: { requiresAuth: true, permission: 'corti_sections.view' }
    },
    {
      path: '/embedded-assistant',
      name: 'embedded-assistant',
      component: () => import('@/views/EmbeddedAssistantPage.vue'),
      meta: { requiresAuth: true, permission: 'embedded_assistant.access' }
    },
    {
      path: '/patients',
      name: 'patients',
      component: () => import('@/views/PatientsPage.vue'),
      meta: { requiresAuth: true, permission: 'patients.manage' }
    },
    {
      path: '/audit-log',
      name: 'audit-log',
      component: () => import('@/views/AuditLogPage.vue'),
      meta: { requiresAuth: true, permission: 'audit.view' }
    }
  ]
})

router.beforeEach((to, _from, next) => {
  const requiresAuth  = to.meta.requiresAuth !== false
  const isLoginPage   = to.meta.isLoginPage === true
  const isPendingPage = to.name === 'pending-approval'
  const permission    = to.meta.permission as string | undefined

  // Unauthenticated user trying to reach protected route → login.
  if (requiresAuth && !isAuthenticated.value) {
    return next({ name: 'login', query: { redirect: to.fullPath } })
  }

  // Authenticated but not yet approved (pending/rejected) → the status
  // page, wherever they were headed, so a retroactively-pending account
  // never lands on a broken-looking empty-nav screen with no explanation.
  if (isAuthenticated.value && !isApproved.value && !isPendingPage) {
    return next({ name: 'pending-approval' })
  }

  // Approved user trying to reach login/signup, or the pending page after
  // having since been approved → send into the app, landing on their
  // role-appropriate dashboard (same landing logic as the mobile app's
  // HomeShell).
  if (isAuthenticated.value && isApproved.value && (isLoginPage || isPendingPage)) {
    return next({ path: defaultLandingPath() })
  }

  // Permission check — single call, no switch needed.
  if (permission && !hasPermission(permission)) {
    return next({ name: 'home' })
  }

  next()
})

export default router
