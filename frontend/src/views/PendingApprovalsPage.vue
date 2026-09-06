<template>
  <div class="pending-approvals-page">
    <v-row>
      <v-col cols="12">
        <div class="mb-6">
          <h1 class="text-h4 font-weight-bold mb-2">
            <v-icon icon="mdi-account-clock" class="mr-2" color="primary" />
            Pending Approvals
          </h1>
          <p class="text-body-1 text-medium-emphasis">
            New signups are held here until a superuser approves them — this app is by invitation only.
          </p>
        </div>
      </v-col>
    </v-row>

    <v-row>
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-text>
            <div v-if="isLoading" class="text-center py-8">
              <v-progress-circular indeterminate color="primary" size="48" />
              <p class="text-body-2 text-medium-emphasis mt-4">Loading pending requests...</p>
            </div>
            <div v-else-if="pending.length === 0" class="text-center py-8">
              <v-icon icon="mdi-check-circle-outline" size="48" color="success" class="mb-2" />
              <p class="text-body-1 text-medium-emphasis">No pending requests right now.</p>
            </div>
            <v-data-table v-else :headers="headers" :items="pending" :items-per-page="10">
              <template v-slot:item.requestedRole="{ item }">
                <v-chip :color="getRoleColor(item.requestedRole)" size="small" variant="tonal">
                  <v-icon :icon="getRoleIcon(item.requestedRole)" size="14" class="mr-1" />
                  {{ formatRole(item.requestedRole) }}
                </v-chip>
              </template>
              <template v-slot:item.createdAt="{ item }">{{ formatDate(item.createdAt) }}</template>
              <template v-slot:item.actions="{ item }">
                <div class="d-flex ga-1">
                  <v-btn
                    color="success"
                    variant="tonal"
                    size="small"
                    :loading="actingId === item.id"
                    @click="approve(item)"
                  >
                    <v-icon icon="mdi-check" class="mr-1" size="16" />
                    Approve
                  </v-btn>
                  <v-btn
                    color="error"
                    variant="tonal"
                    size="small"
                    :loading="actingId === item.id"
                    @click="reject(item)"
                  >
                    <v-icon icon="mdi-close" class="mr-1" size="16" />
                    Reject
                  </v-btn>
                </div>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-snackbar v-model="showSuccess" color="success" timeout="3000">{{ successMessage }}</v-snackbar>
    <v-snackbar v-model="showError" color="error" timeout="4000">{{ errorMessage }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/services/api'

interface PendingUser {
  id: string
  username: string
  email: string
  name: string
  requestedRole: string
  createdAt: string
}

const pending = ref<PendingUser[]>([])
const isLoading = ref(true)
const actingId = ref<string | null>(null)

const showSuccess = ref(false)
const showError = ref(false)
const successMessage = ref('')
const errorMessage = ref('')

const headers = [
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Username', key: 'username', sortable: true },
  { title: 'Email', key: 'email', sortable: true },
  { title: 'Requested Role', key: 'requestedRole', sortable: true },
  { title: 'Requested', key: 'createdAt', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false },
]

const getRoleColor = (role: string) => {
  switch (role) {
    case 'admin':  return 'error'
    case 'doctor': return 'primary'
    default:       return 'info'
  }
}

const getRoleIcon = (role: string) => {
  switch (role) {
    case 'admin':  return 'mdi-shield-account'
    case 'doctor': return 'mdi-stethoscope'
    default:       return 'mdi-account'
  }
}

const formatRole = (role: string) => role.charAt(0).toUpperCase() + role.slice(1)

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit'
  })
}

const fetchPending = async () => {
  isLoading.value = true
  try {
    const response = await api.get('/users/pending')
    pending.value = response.data.users || []
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'Failed to load pending requests'
    showError.value = true
  } finally {
    isLoading.value = false
  }
}

const approve = async (item: PendingUser) => {
  actingId.value = item.id
  try {
    await api.post(`/users/${item.id}/approve`)
    pending.value = pending.value.filter(u => u.id !== item.id)
    successMessage.value = `${item.name} approved as ${formatRole(item.requestedRole)}`
    showSuccess.value = true
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'Failed to approve user'
    showError.value = true
  } finally {
    actingId.value = null
  }
}

const reject = async (item: PendingUser) => {
  actingId.value = item.id
  try {
    await api.post(`/users/${item.id}/reject`)
    pending.value = pending.value.filter(u => u.id !== item.id)
    successMessage.value = `${item.name}'s request was rejected`
    showSuccess.value = true
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'Failed to reject user'
    showError.value = true
  } finally {
    actingId.value = null
  }
}

onMounted(fetchPending)
</script>
