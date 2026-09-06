<template>
  <AuthLayout>
    <v-card class="login-card glass-card">
      <v-card-text class="pa-6">
        <h2 class="text-h5 font-weight-bold mb-2 text-center">Welcome Back</h2>
        <p class="text-body-2 text-medium-emphasis text-center mb-6">
          Sign in to access your transcription dashboard
        </p>

        <!-- Session Expired Warning -->
        <v-alert
          v-if="sessionExpiredMessage"
          type="warning"
          variant="tonal"
          density="compact"
          class="mb-4"
          closable
          @click:close="clearSessionExpiredMessage"
        >
          <div class="d-flex align-center">
            <v-icon icon="mdi-clock-alert-outline" class="mr-2" />
            {{ sessionExpiredMessage }}
          </div>
        </v-alert>

        <v-form @submit.prevent="handleLogin" ref="formRef">
          <v-text-field
            v-model="loginId"
            label="Username or Email"
            prepend-inner-icon="mdi-account-outline"
            variant="outlined"
            :rules="[rules.required]"
            :disabled="authLoading"
            class="mb-3"
            autocomplete="username"
          />

          <v-text-field
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            label="Password"
            prepend-inner-icon="mdi-lock-outline"
            :append-inner-icon="showPassword ? 'mdi-eye-off' : 'mdi-eye'"
            @click:append-inner="showPassword = !showPassword"
            variant="outlined"
            :rules="[rules.required]"
            :disabled="authLoading"
            class="mb-4"
            autocomplete="current-password"
          />

          <v-alert
            v-if="authError"
            type="error"
            variant="tonal"
            density="compact"
            class="mb-4"
            closable
            @click:close="clearError"
          >
            {{ authError }}
          </v-alert>

          <v-btn
            type="submit"
            block
            size="large"
            color="primary"
            :loading="authLoading"
            :disabled="!isFormValid"
          >
            <v-icon icon="mdi-login" class="mr-2" />
            Sign In
          </v-btn>
        </v-form>

        <p class="text-body-2 text-medium-emphasis text-center mt-5 mb-0">
          Don't have an account?
          <RouterLink to="/signup" class="signup-link">Sign up</RouterLink>
        </p>
      </v-card-text>
    </v-card>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import AuthLayout from '@/components/AuthLayout.vue'
import { login, authError, authLoading, clearError, defaultLandingPath } from '@/stores/auth'

const router = useRouter()
const route = useRoute()

const loginId = ref('')
const password = ref('')
const showPassword = ref(false)
const formRef = ref()
const sessionExpiredMessage = ref<string | null>(null)

const rules = {
  required: (v: string) => !!v || 'This field is required',
}

const isFormValid = computed(() => {
  return !!loginId.value && !!password.value
})

// Check for session expired message on mount
onMounted(() => {
  // Check query param
  if (route.query.session_expired === 'true') {
    // Get message from sessionStorage
    const message = sessionStorage.getItem('session_expired_message')
    if (message) {
      sessionExpiredMessage.value = message
      sessionStorage.removeItem('session_expired_message')
    } else {
      sessionExpiredMessage.value = 'Your session has expired. Please log in again.'
    }
  }
})

const clearSessionExpiredMessage = () => {
  sessionExpiredMessage.value = null
}

const handleLogin = async () => {
  // Clear any session expired message on login attempt
  sessionExpiredMessage.value = null

  const success = await login(loginId.value, password.value)
  if (success) {
    // Check if there was a redirect path, otherwise land on the
    // role-appropriate dashboard.
    const redirectPath = route.query.redirect as string
    router.push(redirectPath || defaultLandingPath())
  }
}
</script>
