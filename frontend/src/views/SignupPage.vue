<template>
  <AuthLayout>
    <v-card class="login-card glass-card">
      <v-card-text class="pa-6">
        <h2 class="text-h5 font-weight-bold mb-2 text-center">Create Account</h2>
        <p class="text-body-2 text-medium-emphasis text-center mb-6">
          Sign up to start using the transcription dashboard
        </p>

        <v-form @submit.prevent="handleSignup" ref="formRef">
          <v-text-field
            v-model="name"
            label="Full Name"
            prepend-inner-icon="mdi-account-outline"
            variant="outlined"
            :rules="[rules.required]"
            :disabled="authLoading"
            class="mb-3"
            autocomplete="name"
          />

          <v-text-field
            v-model="username"
            label="Username"
            prepend-inner-icon="mdi-at"
            variant="outlined"
            :rules="[rules.required, rules.username]"
            :disabled="authLoading"
            class="mb-3"
            autocomplete="username"
          />

          <v-text-field
            v-model="email"
            label="Email"
            type="email"
            prepend-inner-icon="mdi-email-outline"
            variant="outlined"
            :rules="[rules.required, rules.email]"
            :disabled="authLoading"
            class="mb-3"
            autocomplete="email"
          />

          <v-text-field
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            label="Password"
            prepend-inner-icon="mdi-lock-outline"
            :append-inner-icon="showPassword ? 'mdi-eye-off' : 'mdi-eye'"
            @click:append-inner="showPassword = !showPassword"
            variant="outlined"
            :rules="[rules.required, rules.minLength]"
            :disabled="authLoading"
            class="mb-3"
            autocomplete="new-password"
          />

          <v-text-field
            v-model="confirmPassword"
            :type="showPassword ? 'text' : 'password'"
            label="Confirm Password"
            prepend-inner-icon="mdi-lock-check-outline"
            variant="outlined"
            :rules="[rules.required, rules.matchesPassword]"
            :disabled="authLoading"
            class="mb-3"
            autocomplete="new-password"
          />

          <v-select
            v-model="role"
            label="Account Type"
            prepend-inner-icon="mdi-account-badge-outline"
            variant="outlined"
            :items="roleOptions"
            item-title="label"
            item-value="value"
            :disabled="authLoading"
            :hint="roleHint"
            persistent-hint
            class="mb-4"
          />

          <v-alert
            type="info"
            variant="tonal"
            density="compact"
            class="mb-4"
          >
            New accounts need approval from a superuser before they can sign in fully. You can still log in right after signing up to check your status.
          </v-alert>

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
            <v-icon icon="mdi-account-plus-outline" class="mr-2" />
            Create Account
          </v-btn>
        </v-form>

        <p class="text-body-2 text-medium-emphasis text-center mt-5 mb-0">
          Already have an account?
          <RouterLink to="/login" class="signup-link">Sign in</RouterLink>
        </p>
      </v-card-text>
    </v-card>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import AuthLayout from '@/components/AuthLayout.vue'
import { signup, authError, authLoading, clearError, currentUser, defaultLandingPath } from '@/stores/auth'

const router = useRouter()

const name = ref('')
const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const formRef = ref()

// "superuser" is deliberately not offered — that account is provisioned
// once at seed time, never through self-signup (backend rejects it too).
const role = ref('user')
const roleOptions = [
  { value: 'user', label: 'User (Demo / Trial)' },
  { value: 'doctor', label: 'Doctor' },
  { value: 'admin', label: 'Admin' },
]
const roleHint = computed(() => {
  return role.value === 'user'
    ? 'Try out the app with limited trial access to each feature.'
    : 'Full access once a superuser approves your account.'
})

const rules = {
  required: (v: string) => !!v || 'This field is required',
  email: (v: string) => /.+@.+\..+/.test(v) || 'Please enter a valid email',
  username: (v: string) => /^[a-zA-Z0-9_.-]{3,32}$/.test(v) || '3-32 characters: letters, numbers, _ . -',
  minLength: (v: string) => (v && v.length >= 6) || 'Password must be at least 6 characters',
  matchesPassword: (v: string) => v === password.value || 'Passwords do not match',
}

const isFormValid = computed(() => {
  return (
    !!name.value &&
    !!username.value &&
    !!email.value &&
    password.value.length >= 6 &&
    confirmPassword.value === password.value
  )
})

const handleSignup = async () => {
  const success = await signup({
    username: username.value,
    email: email.value,
    password: password.value,
    name: name.value,
    role: role.value,
  })
  if (success) {
    router.push(currentUser.value?.status === 'approved' ? defaultLandingPath() : '/pending-approval')
  }
}
</script>
