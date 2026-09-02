import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated, hasPermission } from '@/stores/auth'

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
      path: '/',
      name: 'root',
      redirect: '/ambient-session'
    },
    {
      path: '/home',
      name: 'home',
      component: () => import('@/views/HomePage.vue'),
      meta: { requiresAuth: true }
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
    }
  ]
})

router.beforeEach((to, _from, next) => {
  const requiresAuth = to.meta.requiresAuth !== false
  const isLoginPage  = to.meta.isLoginPage === true
  const permission   = to.meta.permission as string | undefined

  // Authenticated user trying to reach login → send to app.
  if (isLoginPage && isAuthenticated.value) {
    return next({ path: '/ambient-session' })
  }

  // Unauthenticated user trying to reach protected route → login.
  if (requiresAuth && !isAuthenticated.value) {
    return next({ name: 'login', query: { redirect: to.fullPath } })
  }

  // Permission check — single call, no switch needed.
  if (permission && !hasPermission(permission)) {
    return next({ name: 'home' })
  }

  next()
})

export default router
