<template>
  <div class="audit-log-page">
    <v-row>
      <v-col cols="12">
        <div class="mb-6">
          <h1 class="text-h4 font-weight-bold mb-2">
            <v-icon icon="mdi-shield-search-outline" class="mr-2" color="primary" />
            Clinical Audit Trail
          </h1>
          <p class="text-body-1 text-medium-emphasis">
            Who viewed, edited, or sent which patient's data, and when — append-only, read-only
            here by design (<code>internal/pgstore/audit_store.go</code>).
          </p>
        </div>
      </v-col>
    </v-row>

    <v-row v-if="loadError">
      <v-col cols="12">
        <v-alert type="warning" variant="tonal">{{ loadError }}</v-alert>
      </v-col>
    </v-row>

    <v-row>
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-text>
            <div v-if="isLoading" class="text-center py-8">
              <v-progress-circular indeterminate color="primary" size="48" />
            </div>
            <div v-else-if="events.length === 0 && !loadError" class="text-center py-8">
              <v-icon icon="mdi-shield-off-outline" size="48" color="medium-emphasis" class="mb-2" />
              <p class="text-body-1 text-medium-emphasis">No audit events recorded yet.</p>
            </div>
            <v-data-table v-else-if="events.length > 0" :headers="headers" :items="events" :items-per-page="25">
              <template v-slot:item.occurredAt="{ item }">{{ formatDate(item.occurredAt) }}</template>
              <template v-slot:item.action="{ item }">
                <v-chip size="small" variant="tonal" :color="actionColor(item.action)">{{ item.action }}</v-chip>
              </template>
              <template v-slot:item.actorName="{ item }">{{ item.actorName || item.actorId }}</template>
              <template v-slot:item.resource="{ item }">{{ item.resourceType }} / {{ item.resourceId }}</template>
              <template v-slot:item.patientId="{ item }">
                <span v-if="item.patientId" class="text-mono text-caption">{{ item.patientId }}</span>
                <span v-else class="text-medium-emphasis">—</span>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listAuditEvents, type AuditEvent } from '@/services/nhs'

const events = ref<AuditEvent[]>([])
const isLoading = ref(true)
const loadError = ref('')

const headers = [
  { title: 'When', key: 'occurredAt', sortable: true },
  { title: 'Actor', key: 'actorName', sortable: true },
  { title: 'Action', key: 'action', sortable: true },
  { title: 'Resource', key: 'resource', sortable: false },
  { title: 'Patient', key: 'patientId', sortable: false },
]

const formatDate = (d: string) =>
  new Date(d).toLocaleString('en-GB', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })

const actionColor = (action: string) => {
  if (action.startsWith('nhs.send_to_gp')) return 'success'
  if (action.startsWith('nhs.')) return 'info'
  if (action.includes('delete')) return 'error'
  if (action.includes('update') || action.includes('create')) return 'warning'
  return 'default'
}

const fetchEvents = async () => {
  isLoading.value = true
  loadError.value = ''
  try {
    events.value = await listAuditEvents({ limit: 200 })
  } catch (err: any) {
    loadError.value = err.response?.status === 404
      ? 'NHS integration is not enabled on this deployment.'
      : (err.response?.data?.error || 'Failed to load the audit trail')
  } finally {
    isLoading.value = false
  }
}

onMounted(fetchEvents)
</script>

<style scoped>
.text-mono {
  font-family: monospace;
}
</style>
