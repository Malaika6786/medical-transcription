<template>
  <v-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)" max-width="520" persistent>
    <v-card class="glass-card">
      <v-card-title class="d-flex align-center">
        <v-icon icon="mdi-hospital-box-outline" color="primary" class="mr-2" />
        Send to GP via SystmOne
      </v-card-title>
      <v-card-text>
        <v-alert type="info" variant="tonal" density="compact" class="mb-4">
          Sends this session's generated document to a patient's registered GP practice over
          MESH (GP Connect: Send Document). Requires a PDS-verified NHS number.
        </v-alert>

        <v-select
          v-model="selectedPatientId"
          :items="patients"
          item-title="name"
          item-value="id"
          label="Patient *"
          density="compact"
          variant="outlined"
          class="mb-2"
          :loading="loadingPatients"
          :disabled="sending"
        >
          <template #item="{ props, item }">
            <v-list-item v-bind="props" :subtitle="item.raw.nhsNumber ? formatNHS(item.raw.nhsNumber) : 'No NHS number on file'" />
          </template>
        </v-select>
        <div class="text-caption text-medium-emphasis mb-4">
          Don't see the patient?
          <router-link to="/patients" target="_blank">Add one on the Patients page</router-link>,
          then reopen this dialog.
        </div>

        <template v-if="selectedPatient">
          <v-alert
            :type="selectedPatient.pdsVerifiedAt ? 'success' : 'warning'"
            variant="tonal"
            density="compact"
            class="mb-4"
          >
            <template v-if="selectedPatient.pdsVerifiedAt">
              PDS-verified {{ formatDate(selectedPatient.pdsVerifiedAt) }}
            </template>
            <template v-else-if="!selectedPatient.nhsNumber">
              This patient has no NHS number on file — add one on the Patients page first.
            </template>
            <template v-else>
              Not yet PDS-verified.
              <v-btn size="small" variant="tonal" color="warning" class="ml-2" :loading="verifying" @click="verify">
                Verify Now
              </v-btn>
            </template>
          </v-alert>
        </template>

        <v-text-field
          v-model="recipientMailboxId"
          label="Recipient MESH Mailbox ID *"
          hint="The destination practice's MESH mailbox ID — obtained during NHS onboarding"
          persistent-hint
          density="compact"
          variant="outlined"
          :disabled="sending"
        />

        <v-alert v-if="errorMessage" type="error" variant="tonal" density="compact" class="mt-4">
          {{ errorMessage }}
        </v-alert>
        <v-alert v-if="successMessage" type="success" variant="tonal" density="compact" class="mt-4">
          {{ successMessage }}
        </v-alert>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" :disabled="sending" @click="close">Close</v-btn>
        <v-btn
          color="primary"
          :loading="sending"
          :disabled="!canSend"
          @click="send"
        >
          <v-icon icon="mdi-send" class="mr-1" />
          Send Document
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  listPatients, verifyPatientPDS, linkSessionToPatient, sendSessionToGP,
  type Patient
} from '@/services/nhs'

const props = defineProps<{ modelValue: boolean; sessionId: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', value: boolean): void; (e: 'sent'): void }>()

const patients = ref<Patient[]>([])
const loadingPatients = ref(false)
const selectedPatientId = ref<string | null>(null)
const recipientMailboxId = ref('')
const verifying = ref(false)
const sending = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

const selectedPatient = computed(() => patients.value.find(p => p.id === selectedPatientId.value) || null)
const canSend = computed(() =>
  !!selectedPatient.value?.pdsVerifiedAt && recipientMailboxId.value.trim().length > 0 && !sending.value
)

const formatNHS = (n: string) => (n.length === 10 ? `${n.slice(0, 3)} ${n.slice(3, 6)} ${n.slice(6)}` : n)
const formatDate = (d: string) => new Date(d).toLocaleDateString('en-GB', { year: 'numeric', month: 'short', day: 'numeric' })

async function loadPatients() {
  loadingPatients.value = true
  try {
    patients.value = await listPatients()
  } catch {
    errorMessage.value = 'Could not load patients — is NHS integration enabled on this deployment?'
  } finally {
    loadingPatients.value = false
  }
}

async function verify() {
  if (!selectedPatient.value) return
  verifying.value = true
  errorMessage.value = ''
  try {
    const updated = await verifyPatientPDS(selectedPatient.value.id)
    const idx = patients.value.findIndex(p => p.id === updated.id)
    if (idx !== -1) patients.value[idx] = updated
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'PDS verification failed'
  } finally {
    verifying.value = false
  }
}

async function send() {
  if (!selectedPatientId.value) return
  sending.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await linkSessionToPatient(props.sessionId, selectedPatientId.value)
    const result = await sendSessionToGP(props.sessionId, recipientMailboxId.value.trim())
    successMessage.value = `Sent — MESH message ID ${result.meshMessageId || '(pending)'}`
    emit('sent')
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'Failed to send document'
  } finally {
    sending.value = false
  }
}

function close() {
  emit('update:modelValue', false)
}

watch(() => props.modelValue, (open) => {
  if (open) {
    errorMessage.value = ''
    successMessage.value = ''
    recipientMailboxId.value = ''
    selectedPatientId.value = null
    loadPatients()
  }
})
</script>
