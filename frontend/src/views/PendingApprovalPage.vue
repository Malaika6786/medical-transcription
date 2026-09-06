<template>
  <v-row justify="center" class="mt-12">
    <v-col cols="12" sm="8" md="6" lg="5">
      <v-card class="glass-card pa-2">
        <v-card-text class="text-center pa-6">
          <v-icon
            :icon="isRejected ? 'mdi-close-circle-outline' : 'mdi-clock-outline'"
            :color="isRejected ? 'error' : 'warning'"
            size="56"
            class="mb-4"
          />
          <h2 class="text-h5 font-weight-bold mb-2">
            {{ isRejected ? 'Signup Declined' : 'Awaiting Approval' }}
          </h2>
          <p class="text-body-1 text-medium-emphasis mb-1">
            {{ statusMessage }}
          </p>
          <p v-if="!isRejected" class="text-body-2 text-medium-emphasis mb-6">
            This app is by invitation only, so every new account is reviewed by a superuser before it's activated.
          </p>
          <p v-else class="text-body-2 text-medium-emphasis mb-6">
            Contact your administrator if you think this was a mistake.
          </p>

          <v-btn
            color="primary"
            variant="tonal"
            :loading="checking"
            class="mb-3"
            block
            @click="checkStatus"
          >
            <v-icon icon="mdi-refresh" class="mr-2" />
            Check Status
          </v-btn>
          <v-btn variant="text" block @click="handleLogout">
            Log Out
          </v-btn>
        </v-card-text>
      </v-card>
    </v-col>
  </v-row>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { currentUser, refreshUser, isApproved, logout, defaultLandingPath } from '@/stores/auth'

const router = useRouter()
const checking = ref(false)

const isRejected = computed(() => currentUser.value?.status === 'rejected')
const statusMessage = computed(() => {
  const role = currentUser.value?.requestedRole || 'account'
  return isRejected.value
    ? `Your request for ${role} access was not approved.`
    : `Your ${role} account is waiting for a superuser to approve it.`
})

const checkStatus = async () => {
  checking.value = true
  try {
    await refreshUser()
    if (isApproved.value) {
      router.push(defaultLandingPath())
    }
  } finally {
    checking.value = false
  }
}

const handleLogout = () => logout()
</script>
