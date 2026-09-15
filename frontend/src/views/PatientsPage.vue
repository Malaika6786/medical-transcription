<template>
  <div class="patients-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex flex-column flex-sm-row align-sm-center justify-sm-space-between ga-3 mb-6">
          <div>
            <h1 class="text-h4 font-weight-bold mb-2">
              <v-icon icon="mdi-badge-account-horizontal-outline" class="mr-2" color="primary" />
              NHS Patients
            </h1>
            <p class="text-body-1 text-medium-emphasis">
              Structured, NHS-number-based patient records — required before a document can be
              sent to a GP practice via GP Connect. See
              <code>SYSTMONE_INTEGRATION_REPORT.md</code> for background.
            </p>
          </div>
          <v-btn color="primary" size="large" @click="openCreateDialog">
            <v-icon icon="mdi-account-plus" class="mr-2" />
            Add Patient
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <v-row v-if="loadError">
      <v-col cols="12">
        <v-alert type="warning" variant="tonal">
          {{ loadError }}
          <div class="text-caption mt-1" v-if="notConfigured">
            The backend only registers these routes once <code>FIELD_ENCRYPTION_KEY</code> is
            configured — see <code>docs/nhs/README.md</code>.
          </div>
        </v-alert>
      </v-col>
    </v-row>

    <v-row>
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-text>
            <div v-if="isLoading" class="text-center py-8">
              <v-progress-circular indeterminate color="primary" size="48" />
              <p class="text-body-2 text-medium-emphasis mt-4">Loading patients...</p>
            </div>
            <div v-else-if="patients.length === 0 && !loadError" class="text-center py-8">
              <v-icon icon="mdi-account-off-outline" size="48" color="medium-emphasis" class="mb-2" />
              <p class="text-body-1 text-medium-emphasis">No patient records yet.</p>
            </div>
            <v-data-table v-else-if="patients.length > 0" :headers="headers" :items="patients" :items-per-page="10">
              <template v-slot:item.nhsNumber="{ item }">
                <span v-if="item.nhsNumber" class="text-mono">{{ formatNHS(item.nhsNumber) }}</span>
                <v-chip v-else size="x-small" variant="tonal" color="warning">Not on file</v-chip>
              </template>
              <template v-slot:item.pdsVerifiedAt="{ item }">
                <v-chip v-if="item.pdsVerifiedAt" size="small" color="success" variant="tonal">
                  <v-icon icon="mdi-check-decagram" size="14" class="mr-1" />
                  Verified {{ formatDate(item.pdsVerifiedAt) }}
                </v-chip>
                <v-chip v-else size="small" color="warning" variant="tonal">Not PDS-verified</v-chip>
              </template>
              <template v-slot:item.actions="{ item }">
                <div class="d-flex ga-1">
                  <v-btn icon variant="text" size="small" @click="editPatient(item)">
                    <v-icon icon="mdi-pencil" size="18" />
                    <v-tooltip activator="parent" location="top">Edit</v-tooltip>
                  </v-btn>
                  <v-btn
                    icon
                    variant="text"
                    size="small"
                    color="success"
                    :disabled="!item.nhsNumber"
                    :loading="verifyingId === item.id"
                    @click="verify(item)"
                  >
                    <v-icon icon="mdi-shield-check-outline" size="18" />
                    <v-tooltip activator="parent" location="top">Verify against PDS</v-tooltip>
                  </v-btn>
                  <v-btn icon variant="text" size="small" color="error" @click="confirmDelete(item)">
                    <v-icon icon="mdi-delete-outline" size="18" />
                    <v-tooltip activator="parent" location="top">Delete</v-tooltip>
                  </v-btn>
                </div>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Create/Edit Dialog -->
    <v-dialog v-model="showFormDialog" max-width="500">
      <v-card class="glass-card">
        <v-card-title>
          <v-icon :icon="editingPatient ? 'mdi-pencil' : 'mdi-account-plus'" class="mr-2" />
          {{ editingPatient ? 'Edit Patient' : 'Add Patient' }}
        </v-card-title>
        <v-card-text>
          <v-text-field
            v-model="form.name"
            label="Full Name *"
            prepend-inner-icon="mdi-account"
            density="compact"
            variant="outlined"
            class="mb-2"
          />
          <v-text-field
            v-model="form.nhsNumber"
            label="NHS Number"
            placeholder="e.g. 943 476 5919"
            prepend-inner-icon="mdi-card-account-details-outline"
            density="compact"
            variant="outlined"
            class="mb-2"
            hint="10 digits — validated against the modulus-11 check digit server-side"
            persistent-hint
          />
          <v-text-field
            v-model="form.dateOfBirth"
            label="Date of Birth"
            type="date"
            prepend-inner-icon="mdi-calendar"
            density="compact"
            variant="outlined"
            class="mb-2"
          />
          <v-select
            v-model="form.sex"
            :items="['unknown', 'male', 'female', 'other']"
            label="Sex"
            density="compact"
            variant="outlined"
          />
          <v-alert v-if="formError" type="error" variant="tonal" density="compact" class="mt-2">
            {{ formError }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="showFormDialog = false">Cancel</v-btn>
          <v-btn color="primary" :loading="saving" @click="save">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation — patient records are clinically significant
         (NHS-number-linked), so this requires typing the patient's name
         rather than a single confirm click. -->
    <v-dialog v-model="deleteDialog" max-width="420">
      <v-card class="glass-card">
        <v-card-title class="text-h6">
          <v-icon icon="mdi-alert" color="error" class="mr-2" />
          Confirm Delete
        </v-card-title>
        <v-card-text>
          <p class="mb-3">
            This permanently deletes the patient record for
            <strong>{{ patientToDelete?.name }}</strong>, including its NHS number and PDS
            verification status. This cannot be undone.
          </p>
          <v-text-field
            v-model="deleteConfirmText"
            :label="`Type &quot;${patientToDelete?.name}&quot; to confirm`"
            density="compact"
            variant="outlined"
            autofocus
            @keyup.enter="handleDelete"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" :disabled="deleteConfirmText !== patientToDelete?.name" @click="handleDelete">
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listPatients, createPatient, updatePatient, deletePatient, verifyPatientPDS,
  type Patient, type PatientPayload
} from '@/services/nhs'
import { useToast } from '@/composables/useToast'

const toast = useToast()

const patients = ref<Patient[]>([])
const isLoading = ref(true)
const loadError = ref('')
const notConfigured = ref(false)

const headers = [
  { title: 'Name', key: 'name', sortable: true },
  { title: 'NHS Number', key: 'nhsNumber', sortable: false },
  { title: 'Date of Birth', key: 'dateOfBirth', sortable: true },
  { title: 'Sex', key: 'sex', sortable: true },
  { title: 'PDS Status', key: 'pdsVerifiedAt', sortable: false },
  { title: 'Actions', key: 'actions', sortable: false },
]

const fetchPatients = async () => {
  isLoading.value = true
  loadError.value = ''
  notConfigured.value = false
  try {
    patients.value = await listPatients()
  } catch (err: any) {
    if (err.response?.status === 404) {
      notConfigured.value = true
      loadError.value = 'NHS integration is not enabled on this deployment.'
    } else {
      loadError.value = err.response?.data?.error || 'Failed to load patients'
    }
  } finally {
    isLoading.value = false
  }
}

const formatNHS = (n: string) => (n.length === 10 ? `${n.slice(0, 3)} ${n.slice(3, 6)} ${n.slice(6)}` : n)
const formatDate = (d: string) => new Date(d).toLocaleDateString('en-GB', { year: 'numeric', month: 'short', day: 'numeric' })

// --- Create/Edit ---
const showFormDialog = ref(false)
const editingPatient = ref<Patient | null>(null)
const saving = ref(false)
const formError = ref('')
const form = ref<PatientPayload>({ name: '', nhsNumber: '', dateOfBirth: '', sex: 'unknown' })

const openCreateDialog = () => {
  editingPatient.value = null
  form.value = { name: '', nhsNumber: '', dateOfBirth: '', sex: 'unknown' }
  formError.value = ''
  showFormDialog.value = true
}

const editPatient = (p: Patient) => {
  editingPatient.value = p
  form.value = { name: p.name, nhsNumber: p.nhsNumber || '', dateOfBirth: p.dateOfBirth || '', sex: p.sex }
  formError.value = ''
  showFormDialog.value = true
}

const save = async () => {
  if (!form.value.name.trim()) {
    formError.value = 'Name is required'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    if (editingPatient.value) {
      await updatePatient(editingPatient.value.id, form.value)
      toast.success('Patient updated')
    } else {
      await createPatient(form.value)
      toast.success('Patient created')
    }
    showFormDialog.value = false
    await fetchPatients()
  } catch (err: any) {
    formError.value = err.response?.data?.error || 'Failed to save patient'
  } finally {
    saving.value = false
  }
}

// --- Verify against PDS ---
const verifyingId = ref<string | null>(null)
const verify = async (p: Patient) => {
  verifyingId.value = p.id
  try {
    await verifyPatientPDS(p.id)
    toast.success(`${p.name} verified against PDS`)
    await fetchPatients()
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'PDS verification failed')
  } finally {
    verifyingId.value = null
  }
}

// --- Delete ---
const deleteDialog = ref(false)
const patientToDelete = ref<Patient | null>(null)
const deleteConfirmText = ref('')
const confirmDelete = (p: Patient) => {
  patientToDelete.value = p
  deleteConfirmText.value = ''
  deleteDialog.value = true
}
const handleDelete = async () => {
  if (!patientToDelete.value || deleteConfirmText.value !== patientToDelete.value.name) return
  try {
    await deletePatient(patientToDelete.value.id)
    patients.value = patients.value.filter(p => p.id !== patientToDelete.value!.id)
    toast.success('Patient deleted')
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to delete patient')
  } finally {
    deleteDialog.value = false
  }
}

onMounted(fetchPatients)
</script>

<style scoped>
.text-mono {
  font-family: monospace;
}
</style>
