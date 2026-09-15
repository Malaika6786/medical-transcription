<template>
  <v-app>
    <!-- Mobile Navigation Drawer -->
    <v-navigation-drawer
      v-if="isAuthenticated"
      v-model="drawer"
      temporary
      class="glass-card"
    >
      <v-list density="compact" nav>
        <v-list-item
          prepend-icon="mdi-microphone-message"
          title="xstek Medical"
          class="mb-2"
        />
        <v-divider class="mb-2" />
        
        <v-list-item
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          :prepend-icon="item.icon"
          :title="item.title"
          :active="$route.path === item.path"
          color="primary"
          @click="drawer = false"
        />

        <v-divider class="my-2" />

        <!-- Admin Menu — shown per-item based on permissions -->
        <template v-if="hasAnyAdminItem">
          <v-list-subheader>Admin</v-list-subheader>
          <v-list-item
            v-for="item in adminItems"
            :key="item.path"
            :to="item.path"
            :prepend-icon="item.icon"
            :title="item.title"
            :active="$route.path === item.path"
            color="secondary"
            @click="drawer = false"
          />
          <v-divider class="my-2" />
        </template>

        <v-list-item
          prepend-icon="mdi-theme-light-dark"
          :title="isDark ? 'Light Mode' : 'Dark Mode'"
          @click="toggleTheme"
        />

        <v-list-item
          prepend-icon="mdi-logout"
          title="Sign Out"
          class="text-error"
          @click="handleLogout"
        />
      </v-list>

      <template v-slot:append>
        <div class="pa-4">
          <div class="d-flex align-center">
            <v-icon :icon="getRoleIcon(primaryRole)" class="mr-2" size="small" />
            <div>
              <div class="text-body-2 font-weight-medium">{{ currentUser?.name }}</div>
              <div class="text-caption text-medium-emphasis">{{ formatRole(primaryRole) }}</div>
            </div>
          </div>
        </div>
      </template>
    </v-navigation-drawer>

    <!-- App Bar - Only show when authenticated. Single compact row: fixed
         logo, a horizontally-scrollable nav strip that never wraps, and
         fixed theme/profile controls on the right. Shrinks and gains
         opacity once the page scrolls, for a floating "app shell" feel. -->
    <v-app-bar
      v-if="isAuthenticated"
      elevation="0"
      :height="scrolled ? 58 : 68"
      class="navbar-glass"
      :class="{ 'navbar-glass--scrolled': scrolled }"
    >
      <!-- Mobile hamburger menu -->
      <v-app-bar-nav-icon
        v-if="isMobile"
        aria-label="Open navigation menu"
        @click="drawer = !drawer"
      />

      <router-link to="/home" class="nav-logo">
        <span class="nav-logo-icon-wrap">
          <v-icon icon="mdi-microphone-message" color="primary" :size="isMobile ? 18 : 22" class="nav-logo-icon" />
          <span class="nav-logo-pulse-ring"></span>
        </span>
        <span v-if="!isMobile" class="nav-logo-text">XStek AI</span>
      </router-link>

      <!-- Scrollable nav strip (desktop only — mobile uses the drawer). -->
      <div v-if="!isMobile" class="nav-scroll-wrap">
        <div class="nav-scroll-fade nav-scroll-fade--left" :class="{ visible: canScrollLeft }"></div>
        <div
          ref="navScrollEl"
          class="nav-scroll"
          @wheel="onNavWheel"
          @pointerdown="onNavPointerDown"
          @pointermove="onNavPointerMove"
          @pointerup="onNavPointerUp"
          @pointerleave="onNavPointerUp"
          @click.capture="onNavClickCapture"
        >
          <div class="nav-active-indicator" :style="indicatorStyle"></div>
          <router-link
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            class="nav-pill"
            :class="{ 'nav-pill--active': $route.path === item.path }"
            :data-tour="item.tourId"
            :ref="(el: any) => setNavItemRef(item.path, el)"
          >
            <v-icon :icon="item.icon" class="nav-pill-icon" size="18" />
            <span class="nav-pill-label">{{ item.title }}</span>
          </router-link>
        </div>
        <div class="nav-scroll-fade nav-scroll-fade--right" :class="{ visible: canScrollRight }"></div>
      </div>

      <v-spacer v-if="isMobile" />

      <div class="nav-right">
        <v-btn
          icon
          variant="text"
          class="theme-btn"
          :aria-label="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          @click="toggleTheme"
          data-tour="theme-toggle"
        >
          <v-icon :icon="isDark ? 'mdi-white-balance-sunny' : 'mdi-moon-waning-crescent'" />
        </v-btn>

        <!-- User Menu (Desktop only) -->
        <v-menu v-if="!isMobile" transition="slide-y-transition">
          <template v-slot:activator="{ props }">
            <v-btn
              v-bind="props"
              variant="tonal"
              color="primary"
              class="ml-2"
            >
              <v-icon icon="mdi-account-circle" class="mr-2" />
              {{ currentUser?.name || 'User' }}
              <v-icon icon="mdi-chevron-down" size="18" class="ml-1" />
            </v-btn>
          </template>
          <v-list class="glass-card" density="compact">
            <v-list-item>
              <template v-slot:prepend>
                <v-icon icon="mdi-email" size="small" />
              </template>
              <v-list-item-title class="text-caption">{{ currentUser?.email }}</v-list-item-title>
            </v-list-item>
            <v-list-item>
              <template v-slot:prepend>
                <v-icon :icon="getRoleIcon(primaryRole)" size="small" />
              </template>
              <v-list-item-title class="text-caption">{{ formatRole(primaryRole) }}</v-list-item-title>
            </v-list-item>

            <!-- Admin menu items — filtered by permissions -->
            <template v-if="hasAnyAdminItem">
              <v-divider class="my-1" />
              <v-list-item
                v-for="item in adminItems"
                :key="item.path"
                :to="item.path"
                :active="$route.path === item.path"
              >
                <template v-slot:prepend>
                  <v-icon :icon="item.icon" size="small" />
                </template>
                <v-list-item-title>{{ item.title }}</v-list-item-title>
              </v-list-item>
            </template>

            <v-divider class="my-1" />
            <v-list-item @click="handleLogout" class="text-error">
              <template v-slot:prepend>
                <v-icon icon="mdi-logout" size="small" color="error" />
              </template>
              <v-list-item-title>Sign Out</v-list-item-title>
            </v-list-item>
          </v-list>
        </v-menu>
      </div>
    </v-app-bar>

    <!-- Main Content -->
    <v-main>
      <v-container v-if="isAuthenticated" fluid class="pa-2 pa-sm-4 pa-md-6 main-container">
        <router-view v-slot="{ Component }">
          <transition name="page" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </v-container>
      <!-- Login page takes full screen -->
      <router-view v-else />
    </v-main>

    <ToastHost />
  </v-app>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useTheme } from 'vuetify'
import { useRouter, useRoute } from 'vue-router'
import { useDisplay } from 'vuetify'
import {
  isAuthenticated,
  currentUser,
  logout,
  hasPermission
} from '@/stores/auth'
import ToastHost from '@/components/ToastHost.vue'

const theme = useTheme()
const router = useRouter()
const route = useRoute()
const display = useDisplay()
const isDark = computed(() => theme.global.current.value.dark)
const drawer = ref(false)

const isMobile = computed(() => display.smAndDown.value)

// --- navbar compresses slightly once the page has scrolled, for a
// "floating app shell" feel. ---
const scrolled = ref(false)
function onWindowScroll() { scrolled.value = window.scrollY > 8 }
onMounted(() => window.addEventListener('scroll', onWindowScroll, { passive: true }))
onUnmounted(() => window.removeEventListener('scroll', onWindowScroll))

// --- horizontally-scrollable nav strip: wheel-to-scroll, drag-to-scroll,
// and edge fade indicators when there's more content off-screen. ---
const navScrollEl = ref<HTMLElement | null>(null)
const navItemEls = new Map<string, HTMLElement>()
function setNavItemRef(path: string, el: unknown) {
  // router-link is a component, so the function ref receives its component
  // instance (exposing $el), not the underlying <a> directly — unwrap it.
  const node = el && typeof el === 'object' && '$el' in (el as Record<string, unknown>)
    ? (el as { $el: unknown }).$el
    : el
  if (node instanceof HTMLElement) navItemEls.set(path, node)
  else navItemEls.delete(path)
}
const canScrollLeft = ref(false)
const canScrollRight = ref(false)
function updateScrollFades() {
  const el = navScrollEl.value
  if (!el) return
  canScrollLeft.value = el.scrollLeft > 4
  canScrollRight.value = el.scrollLeft < el.scrollWidth - el.clientWidth - 4
}
function onNavWheel(e: WheelEvent) {
  const el = navScrollEl.value
  if (!el) return
  if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) {
    el.scrollLeft += e.deltaY
    e.preventDefault()
  }
}
// Drag-to-scroll, but only once the pointer has actually moved past a
// small threshold — capturing the pointer (and suppressing the resulting
// click) immediately on pointerdown broke plain clicks on the nav links.
let dragging = false
let dragMoved = false
let dragStartX = 0
let dragStartScroll = 0
let capturedPointerId: number | null = null
function onNavPointerDown(e: PointerEvent) {
  const el = navScrollEl.value
  if (!el) return
  dragging = true
  dragMoved = false
  dragStartX = e.clientX
  dragStartScroll = el.scrollLeft
  capturedPointerId = e.pointerId
}
function onNavPointerMove(e: PointerEvent) {
  if (!dragging) return
  const el = navScrollEl.value
  if (!el) return
  const dx = e.clientX - dragStartX
  if (!dragMoved && Math.abs(dx) > 4) {
    dragMoved = true
    if (capturedPointerId !== null) el.setPointerCapture(capturedPointerId)
  }
  if (dragMoved) el.scrollLeft = dragStartScroll - dx
}
function onNavPointerUp() {
  const el = navScrollEl.value
  if (dragMoved && el && capturedPointerId !== null) {
    try { el.releasePointerCapture(capturedPointerId) } catch { /* already released */ }
  }
  dragging = false
  capturedPointerId = null
  // Swallow the click that follows a real drag so it doesn't also navigate.
  if (dragMoved) {
    suppressNextClick = true
    setTimeout(() => { suppressNextClick = false }, 0)
  }
  dragMoved = false
}
let suppressNextClick = false
function onNavClickCapture(e: MouseEvent) {
  if (suppressNextClick) { e.preventDefault(); e.stopPropagation() }
}

// --- sliding active-pill indicator, positioned from the matching nav
// item's real offset/width so it glides smoothly between pages. ---
const indicatorStyle = ref<{ left: string; width: string; opacity: number }>({ left: '0px', width: '0px', opacity: 0 })
function updateIndicator() {
  const el = navItemEls.get(route.path)
  if (el) {
    indicatorStyle.value = { left: el.offsetLeft + 'px', width: el.offsetWidth + 'px', opacity: 1 }
    el.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
  } else {
    indicatorStyle.value = { ...indicatorStyle.value, opacity: 0 }
  }
}
function onNavScroll() { updateScrollFades() }

let resizeObs: ResizeObserver | null = null
onMounted(async () => {
  await nextTick()
  updateIndicator()
  updateScrollFades()
  if (navScrollEl.value) {
    navScrollEl.value.addEventListener('scroll', onNavScroll, { passive: true })
    resizeObs = new ResizeObserver(() => { updateIndicator(); updateScrollFades() })
    resizeObs.observe(navScrollEl.value)
  }
})
onUnmounted(() => {
  navScrollEl.value?.removeEventListener('scroll', onNavScroll)
  resizeObs?.disconnect()
})
watch(() => route.path, () => nextTick(() => { updateIndicator(); updateScrollFades() }))

// A superuser's job is running the app, not doing clinical work — so their
// primary nav is the org-wide/management surfaces (Command Center up
// front, approvals/users/roles one click away, not buried in a menu). The
// doctor-facing tools every other role lives in (Ambient AI, Saved
// Sessions, etc.) fold behind "Clinician Tools" → their own My Dashboard,
// the exact page every non-superuser lands on — one click, not hidden,
// just not competing with their actual job for the primary nav slot.
interface NavItem { title: string; path: string; icon: string; permission: string | null; tourId?: string }

const superUserNavItems: NavItem[] = [
  { title: 'Command Center',    path: '/command-center', icon: 'mdi-view-dashboard-variant', permission: null },
  { title: 'Pending Approvals', path: '/pending-users',  icon: 'mdi-account-clock',          permission: null },
  { title: 'User Management',   path: '/users',          icon: 'mdi-account-group',          permission: null },
  { title: 'Role Management',   path: '/roles',          icon: 'mdi-shield-key',              permission: null },
  { title: 'NHS Patients',      path: '/patients',       icon: 'mdi-badge-account-horizontal-outline', permission: null },
  { title: 'Audit Log',         path: '/audit-log',      icon: 'mdi-shield-search-outline',  permission: null },
  { title: 'Clinician Tools',   path: '/my-dashboard',   icon: 'mdi-stethoscope',             permission: null },
]

// Nav items declare the permission string required to see them (null = always
// visible). tourId feeds data-tour for useProductTour's welcome tour — fixed
// values here instead of deriving them from the path (the old derivation
// produced "nav-ambient-session"/"nav-async-transcription", which didn't
// match the tour's actual selectors "nav-ambient"/"nav-file-transcription",
// so those two welcome-tour steps silently never highlighted anything).
const standardNavItems: NavItem[] = [
  { title: 'Home',               path: '/home',               icon: 'mdi-home',                       permission: null },
  { title: 'My Dashboard',       path: '/my-dashboard',       icon: 'mdi-view-dashboard',              permission: null },
  { title: 'File Transcription', path: '/async-transcription', icon: 'mdi-file-upload',               permission: 'file_transcription.access', tourId: 'nav-file-transcription' },
  { title: 'Ambient AI',         path: '/ambient-session',    icon: 'mdi-broadcast',                  permission: 'ambient.access', tourId: 'nav-ambient' },
  { title: 'Dictation',          path: '/dictation',          icon: 'mdi-microphone-message',         permission: 'dictation.access' },
  { title: 'Embedded Assistant', path: '/embedded-assistant', icon: 'mdi-robot-outline',              permission: 'embedded_assistant.access' },
  { title: 'NHS Patients',       path: '/patients',           icon: 'mdi-badge-account-horizontal-outline', permission: 'patients.manage' },
  { title: 'Saved Sessions',     path: '/saved-sessions',     icon: 'mdi-content-save-all',           permission: null, tourId: 'nav-saved-sessions' },
  { title: 'Semantic Search',    path: '/search',             icon: 'mdi-text-search',                permission: null },
  { title: 'Docs',               path: '/docs',               icon: 'mdi-book-open-page-variant',     permission: 'documentation.view' },
]

// Secondary/occasional tools — kept in the account menu rather than the
// main bar regardless of role. Command Center/Pending Approvals/User &
// Role Management moved out of here into superUserNavItems above, so a
// superuser doesn't see them listed twice.
const adminMenuItems = [
  { title: 'Template Manager', path: '/template-manager', icon: 'mdi-file-cog',                  permission: 'templates.manage' },
  { title: 'Template Test',   path: '/transcript-test',  icon: 'mdi-test-tube',                   permission: 'users.manage' },
  { title: 'Corti Sections',  path: '/corti-sections',   icon: 'mdi-format-list-bulleted-type',   permission: 'corti_sections.view' },
  { title: 'Demo Guide',      path: '/demo-guide',       icon: 'mdi-presentation',                permission: null },
]

const isSuperUser = computed(() => hasPermission('users.manage'))
const navItems = computed(() => {
  if (isSuperUser.value) return superUserNavItems
  return standardNavItems.filter(i => !i.permission || hasPermission(i.permission))
})
const adminItems = computed(() => adminMenuItems.filter(i => !i.permission || hasPermission(i.permission)))
const hasAnyAdminItem = computed(() => adminItems.value.length > 0)
watch(navItems, () => nextTick(() => { updateIndicator(); updateScrollFades() }))

const toggleTheme = () => {
  theme.global.name.value = isDark.value ? 'customLightTheme' : 'customDarkTheme'
}

const handleLogout = () => {
  logout()
  router.push('/login')
}

// Use the primary (first) role for display purposes.
const primaryRole = computed(() => currentUser.value?.roles?.[0])

const getRoleIcon = (role: string | undefined) => {
  switch (role) {
    case 'superuser': return 'mdi-shield-crown'
    case 'doctor':    return 'mdi-stethoscope'
    default:          return 'mdi-account'
  }
}

const formatRole = (role: string | undefined) => {
  switch (role) {
    case 'superuser': return 'Super User'
    case 'doctor':    return 'Doctor'
    default:          return role ? role.charAt(0).toUpperCase() + role.slice(1) : 'User'
  }
}
</script>

<style scoped>
/* Page transition: a quick fade + slight vertical drift instead of the
   dashboard abruptly disappearing/appearing on navigation. */
.page-enter-active,
.page-leave-active {
  transition: opacity 180ms ease, transform 180ms ease;
}
.page-enter-from { opacity: 0; transform: translateY(4px); }
.page-leave-to   { opacity: 0; transform: translateY(-4px); }

/* Ensure no horizontal scrollbars */
.main-container {
  max-width: 100%;
  overflow-x: hidden;
}

/* ============================================
   Navbar shell — glass/floating effect that compresses and gains
   opacity once the page scrolls (bound via :height + .navbar-glass--scrolled
   on the component above; this just supplies the surface treatment).
   ============================================ */
.navbar-glass {
  background: rgba(var(--v-theme-surface), 0.72) !important;
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.16);
  transition: background 260ms ease, box-shadow 260ms ease, border-color 260ms ease, height 260ms ease;
}
.navbar-glass--scrolled {
  background: rgba(var(--v-theme-surface), 0.92) !important;
  border-bottom-color: rgba(var(--v-theme-on-surface), 0.14);
  box-shadow: 0 6px 28px rgba(0, 0, 0, 0.24);
}
.navbar-glass :deep(.v-toolbar__content) {
  transition: height 260ms ease;
}

/* Logo — kept relatively static (this is a medical app, not a game): a
   static gradient wordmark plus a very infrequent, subtle glow pulse ring
   around the mic icon rather than any continuous animation. */
.nav-logo {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  flex-shrink: 0;
  padding-right: 10px;
  margin-right: 4px;
}
.nav-logo-icon-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.nav-logo-pulse-ring {
  position: absolute;
  inset: -7px;
  border-radius: 50%;
  pointer-events: none;
  animation: logo-pulse 6s ease-in-out infinite;
}
@keyframes logo-pulse {
  0%, 90%, 100% { box-shadow: 0 0 0 0 rgba(45, 212, 240, 0); }
  95%           { box-shadow: 0 0 0 7px rgba(45, 212, 240, 0.16); }
}
.nav-logo-text {
  font-family: inherit;
  font-size: 1.05rem;
  font-weight: 700;
  line-height: 1.2;
  white-space: nowrap;
  background: linear-gradient(100deg, #00d9c4, #8b5cf6 55%, #fb5aa0);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

/* ============================================
   Horizontally-scrollable nav strip — never wraps to a second line.
   Scrollbar hidden but wheel/drag/touch scrolling all still work; edge
   fades hint that more items exist off-screen.
   ============================================ */
.nav-scroll-wrap {
  position: relative;
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  align-items: center;
  height: 100%;
  overflow: hidden;
}
.nav-scroll {
  position: relative;
  display: flex;
  align-items: center;
  height: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-behavior: smooth;
  scrollbar-width: none;
  -ms-overflow-style: none;
  touch-action: pan-x;
  cursor: grab;
  padding: 0 4px;
}
.nav-scroll::-webkit-scrollbar { display: none; }
.nav-scroll:active { cursor: grabbing; }

.nav-scroll-fade {
  position: absolute;
  top: 0; bottom: 0;
  width: 36px;
  pointer-events: none;
  opacity: 0;
  transition: opacity 200ms ease;
  z-index: 2;
}
.nav-scroll-fade--left  { left: 0;  background: linear-gradient(90deg, rgba(var(--v-theme-surface), 0.9), transparent); }
.nav-scroll-fade--right { right: 0; background: linear-gradient(270deg, rgba(var(--v-theme-surface), 0.9), transparent); }
.nav-scroll-fade.visible { opacity: 1; }

/* Sliding active-pill indicator — glides between items on navigation
   instead of each button animating independently. */
.nav-active-indicator {
  position: absolute;
  top: 6px;
  bottom: 6px;
  left: 0;
  width: 0;
  border-radius: 10px;
  background: linear-gradient(100deg, rgba(45, 212, 240, 0.2), rgba(168, 85, 247, 0.2));
  border: 1px solid rgba(45, 212, 240, 0.35);
  box-shadow: 0 0 16px rgba(45, 212, 240, 0.2);
  opacity: 0;
  pointer-events: none;
  z-index: 0;
  transition: left 300ms cubic-bezier(.2, .8, .2, 1), width 300ms cubic-bezier(.2, .8, .2, 1), opacity 200ms ease;
}

.nav-pill {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 8px 14px;
  margin: 0 2px;
  border-radius: 10px;
  border: 1px solid transparent;
  color: rgba(var(--v-theme-on-surface), 0.62);
  text-decoration: none;
  font-size: 13.5px;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
  transition: color 200ms ease, background 200ms ease, border-color 200ms ease, box-shadow 200ms ease;
}
.nav-pill:hover {
  background: rgba(var(--v-theme-on-surface), 0.06);
  border-color: rgba(45, 212, 240, 0.25);
  color: rgb(var(--v-theme-on-surface));
  box-shadow: 0 0 12px rgba(45, 212, 240, 0.12);
}
.nav-pill-icon {
  transition: transform 200ms ease;
}
.nav-pill:hover .nav-pill-icon {
  transform: translateY(-2px);
}
.nav-pill--active {
  color: rgb(var(--v-theme-on-surface));
}
.nav-pill--active .nav-pill-icon {
  transform: translateY(-2px);
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

/* Theme toggle — a playful spin on hover only, never a continuous animation. */
.theme-btn :deep(.v-icon) {
  transition: transform 300ms ease;
}
.theme-btn:hover :deep(.v-icon) {
  transform: rotate(180deg);
}

@media (max-width: 360px) {
  .nav-logo-text { font-size: 0.92rem; }
}
</style>

<style>
/* Unscoped on purpose: crossfades the whole app's surface/text colors when
   the theme toggles, instead of every element snapping instantly. */
.v-application,
.v-application .v-card,
.v-application .v-app-bar,
.v-application .v-navigation-drawer,
.v-application .v-list,
.v-application .v-btn {
  transition: background-color 280ms ease, color 280ms ease, border-color 280ms ease;
}
</style>
