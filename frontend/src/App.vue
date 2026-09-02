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

    <!-- App Bar - Only show when authenticated -->
    <v-app-bar v-if="isAuthenticated" elevation="0" class="glass-card">
      <!-- Mobile hamburger menu -->
      <v-app-bar-nav-icon
        v-if="isMobile"
        @click="drawer = !drawer"
      />

      <v-app-bar-title>
        <router-link to="/home" class="d-flex align-center text-decoration-none">
          <v-icon v-if="!isMobile" icon="mdi-microphone-message" color="primary" size="32" class="mr-3" />
          <v-icon v-else icon="mdi-microphone-message" color="primary" size="24" class="mr-1" />
          <span v-if="!isMobile" class="gradient-text font-weight-bold text-h5">XStek AI Medical Transcription</span>
          <span v-else class="gradient-text font-weight-bold mobile-title">XStek AI Medical Transcription</span>
        </router-link>
      </v-app-bar-title>

      <template v-slot:append>
        <!-- Desktop Navigation -->
        <template v-if="!isMobile">
          <v-btn
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            variant="text"
            class="mx-1"
            :color="$route.path === item.path ? 'primary' : undefined"
            :data-tour="'nav-' + item.path.replace('/', '').replace('-', '-')"
          >
            <v-icon :icon="item.icon" class="mr-2" />
            {{ item.title }}
          </v-btn>

        </template>

        <v-btn
          icon
          variant="text"
          @click="toggleTheme"
          data-tour="theme-toggle"
        >
          <v-icon :icon="isDark ? 'mdi-white-balance-sunny' : 'mdi-moon-waning-crescent'" />
        </v-btn>

        <!-- User Menu (Desktop only) -->
        <v-menu v-if="!isMobile">
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
      </template>
    </v-app-bar>

    <!-- Main Content -->
    <v-main>
      <v-container v-if="isAuthenticated" fluid class="pa-2 pa-sm-4 pa-md-6 main-container">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </v-container>
      <!-- Login page takes full screen -->
      <router-view v-else />
    </v-main>
  </v-app>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useTheme } from 'vuetify'
import { useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import {
  isAuthenticated,
  currentUser,
  logout,
  hasPermission
} from '@/stores/auth'

const theme = useTheme()
const router = useRouter()
const display = useDisplay()
const isDark = computed(() => theme.global.current.value.dark)
const drawer = ref(false)

const isMobile = computed(() => display.smAndDown.value)

// Nav items declare the permission string required to see them (null = always visible).
const allNavItems = [
  { title: 'Home',               path: '/home',               icon: 'mdi-home',                       permission: null },
  { title: 'File Transcription', path: '/async-transcription', icon: 'mdi-file-upload',               permission: 'file_transcription.access' },
  { title: 'Ambient AI',         path: '/ambient-session',    icon: 'mdi-broadcast',                  permission: 'ambient.access' },
  { title: 'Dictation',          path: '/dictation',          icon: 'mdi-microphone-message',         permission: 'dictation.access' },
  { title: 'Embedded Assistant', path: '/embedded-assistant', icon: 'mdi-robot-outline',              permission: 'embedded_assistant.access' },
  { title: 'Saved Sessions',     path: '/saved-sessions',     icon: 'mdi-content-save-all',           permission: null },
  { title: 'Semantic Search',    path: '/search',             icon: 'mdi-text-search',                permission: null },
  { title: 'Docs',               path: '/docs',               icon: 'mdi-book-open-page-variant',     permission: 'documentation.view' },
]

const adminMenuItems = [
  { title: 'User Management', path: '/users',            icon: 'mdi-account-group',               permission: 'users.manage' },
  { title: 'Role Management', path: '/roles',            icon: 'mdi-shield-key',                  permission: 'users.manage' },
  { title: 'Template Manager', path: '/template-manager', icon: 'mdi-file-cog',                  permission: 'templates.manage' },
  { title: 'Template Test',   path: '/transcript-test',  icon: 'mdi-test-tube',                   permission: 'users.manage' },
  { title: 'Corti Sections',  path: '/corti-sections',   icon: 'mdi-format-list-bulleted-type',   permission: 'corti_sections.view' },
  { title: 'Demo Guide',      path: '/demo-guide',       icon: 'mdi-presentation',                permission: null },
]

const navItems   = computed(() => allNavItems.filter(i => !i.permission || hasPermission(i.permission)))
const adminItems = computed(() => adminMenuItems.filter(i => !i.permission || hasPermission(i.permission)))
const hasAnyAdminItem = computed(() => adminItems.value.length > 0)

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
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Ensure no horizontal scrollbars */
.main-container {
  max-width: 100%;
  overflow-x: hidden;
}

/* Mobile responsive adjustments */
@media (max-width: 600px) {
  .v-app-bar-title {
    flex: 1 1 auto;
  }
}

/* Mobile title styling */
.mobile-title {
  font-size: 0.85rem;
  line-height: 1.2;
  white-space: nowrap;
}

/* Extra small screens */
@media (max-width: 360px) {
  .mobile-title {
    font-size: 0.75rem;
  }
}
</style>
