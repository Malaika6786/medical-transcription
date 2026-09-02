<template>
  <div class="role-management-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex align-center justify-space-between mb-6">
          <div>
            <h1 class="text-h4 font-weight-bold mb-2">
              <v-icon icon="mdi-shield-key" class="mr-2" color="primary" />
              Role Management
            </h1>
            <p class="text-body-1 text-medium-emphasis">
              Define roles and the permissions each role grants
            </p>
          </div>
          <v-btn color="primary" size="large" @click="openCreateDialog">
            <v-icon icon="mdi-plus" class="mr-2" />
            Add Role
          </v-btn>
        </div>
      </v-col>
    </v-row>

    <!-- Roles Grid -->
    <v-row>
      <v-col
        v-if="isLoading"
        cols="12"
        class="text-center py-8"
      >
        <v-progress-circular indeterminate color="primary" size="48" />
        <p class="text-body-2 text-medium-emphasis mt-4">Loading roles...</p>
      </v-col>

      <v-col
        v-else
        v-for="role in roles"
        :key="role.id"
        cols="12"
        md="6"
        lg="4"
      >
        <v-card class="glass-card h-100">
          <v-card-title class="d-flex align-center pa-4">
            <v-icon icon="mdi-shield-account" class="mr-2" :color="role.is_system ? 'warning' : 'primary'" />
            <span>{{ role.name }}</span>
            <v-spacer />
            <v-chip
              v-if="role.is_system"
              size="x-small"
              color="warning"
              variant="tonal"
              class="ml-2"
            >System</v-chip>
          </v-card-title>

          <v-card-subtitle class="px-4 pb-2" v-if="role.description">
            {{ role.description }}
          </v-card-subtitle>

          <v-card-text class="pa-4 pt-0">
            <div class="text-caption text-medium-emphasis mb-2">
              {{ role.permissions.length }} permission{{ role.permissions.length !== 1 ? 's' : '' }}
            </div>
            <div class="d-flex flex-wrap ga-1">
              <v-chip
                v-for="perm in role.permissions"
                :key="perm"
                size="x-small"
                color="primary"
                variant="tonal"
              >{{ perm }}</v-chip>
              <span v-if="!role.permissions.length" class="text-caption text-medium-emphasis">No permissions</span>
            </div>
          </v-card-text>

          <v-divider />

          <v-card-actions class="pa-3">
            <v-spacer />
            <v-btn
              icon
              variant="text"
              size="small"
              @click="editRole(role)"
            >
              <v-icon icon="mdi-pencil" size="18" />
              <v-tooltip activator="parent" location="top">Edit</v-tooltip>
            </v-btn>
            <v-btn
              icon
              variant="text"
              size="small"
              color="error"
              :disabled="role.is_system"
              @click="confirmDeleteRole(role)"
            >
              <v-icon icon="mdi-delete" size="18" />
              <v-tooltip activator="parent" location="top">
                {{ role.is_system ? 'System roles cannot be deleted' : 'Delete' }}
              </v-tooltip>
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>

    <!-- Create Role Dialog -->
    <v-dialog v-model="showCreateDialog" max-width="600">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-shield-plus" class="mr-2" color="primary" />
          Create Role
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-text-field
            v-model="formRole.name"
            label="Role Name"
            prepend-inner-icon="mdi-shield-account"
            variant="outlined"
            class="mb-3"
          />
          <v-text-field
            v-model="formRole.description"
            label="Description (optional)"
            prepend-inner-icon="mdi-text"
            variant="outlined"
            class="mb-3"
          />
          <div class="text-body-2 font-weight-medium mb-2">Permissions</div>
          <v-card variant="outlined" class="pa-2">
            <v-row dense>
              <v-col
                v-for="perm in allPermissions"
                :key="perm"
                cols="12"
                sm="6"
              >
                <v-checkbox
                  v-model="formRole.permissions"
                  :value="perm"
                  :label="perm"
                  hide-details
                  density="compact"
                />
              </v-col>
            </v-row>
          </v-card>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showCreateDialog = false">Cancel</v-btn>
          <v-btn color="primary" :loading="isSaving" @click="createRole" :disabled="!formRole.name">
            <v-icon icon="mdi-check" class="mr-1" />Create Role
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Edit Role Dialog -->
    <v-dialog v-model="showEditDialog" max-width="600">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-shield-edit" class="mr-2" color="secondary" />
          Edit Role — {{ editingRole?.name }}
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-text-field
            v-model="formRole.name"
            label="Role Name"
            prepend-inner-icon="mdi-shield-account"
            variant="outlined"
            class="mb-3"
          />
          <v-text-field
            v-model="formRole.description"
            label="Description (optional)"
            prepend-inner-icon="mdi-text"
            variant="outlined"
            class="mb-3"
          />
          <div class="text-body-2 font-weight-medium mb-2">Permissions</div>
          <v-card variant="outlined" class="pa-2">
            <v-row dense>
              <v-col
                v-for="perm in allPermissions"
                :key="perm"
                cols="12"
                sm="6"
              >
                <v-checkbox
                  v-model="formRole.permissions"
                  :value="perm"
                  :label="perm"
                  hide-details
                  density="compact"
                />
              </v-col>
            </v-row>
          </v-card>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showEditDialog = false">Cancel</v-btn>
          <v-btn color="secondary" :loading="isSaving" @click="updateRole" :disabled="!formRole.name">
            <v-icon icon="mdi-check" class="mr-1" />Save Changes
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation -->
    <v-dialog v-model="showDeleteDialog" max-width="420">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-alert-circle" class="mr-2" color="error" />
          Delete Role
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <p>Delete role <strong>{{ deletingRole?.name }}</strong>?</p>
          <p class="text-medium-emphasis text-body-2 mt-2">
            This cannot be undone. Users assigned this role will lose its permissions.
          </p>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showDeleteDialog = false">Cancel</v-btn>
          <v-btn color="error" :loading="isDeleting" @click="deleteRole">
            <v-icon icon="mdi-delete" class="mr-1" />Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="showSuccess" color="success" timeout="3000">{{ successMessage }}</v-snackbar>
    <v-snackbar v-model="showError" color="error" timeout="5000">
      {{ errorMessage }}
      <template v-slot:actions>
        <v-btn variant="text" @click="showError = false">Close</v-btn>
      </template>
    </v-snackbar>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/services/api'

interface Role {
  id: string
  name: string
  description: string
  is_system: boolean
  permissions: string[]
}

interface RoleForm {
  name: string
  description: string
  permissions: string[]
}

const roles          = ref<Role[]>([])
const allPermissions = ref<string[]>([])

const isLoading = ref(true)
const isSaving  = ref(false)
const isDeleting = ref(false)

const showCreateDialog = ref(false)
const showEditDialog   = ref(false)
const showDeleteDialog = ref(false)

const showSuccess    = ref(false)
const showError      = ref(false)
const successMessage = ref('')
const errorMessage   = ref('')

const editingRole = ref<Role | null>(null)
const deletingRole = ref<Role | null>(null)

const formRole = ref<RoleForm>({ name: '', description: '', permissions: [] })

const fetchRoles = async () => {
  isLoading.value = true
  try {
    const response = await api.get('/roles')
    if (response.data.success) {
      roles.value = response.data.roles
    }
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'Failed to load roles'
    showError.value = true
  } finally {
    isLoading.value = false
  }
}

const fetchPermissions = async () => {
  try {
    const response = await api.get('/permissions')
    if (response.data.success) {
      allPermissions.value = response.data.permissions
    }
  } catch { /* non-fatal */ }
}

const openCreateDialog = () => {
  formRole.value = { name: '', description: '', permissions: [] }
  showCreateDialog.value = true
}

const createRole = async () => {
  isSaving.value = true
  try {
    const response = await api.post('/roles', formRole.value)
    if (response.data.success) {
      roles.value.push(response.data.role)
      showCreateDialog.value = false
      successMessage.value = 'Role created!'
      showSuccess.value = true
    }
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'Failed to create role'
    showError.value = true
  } finally {
    isSaving.value = false
  }
}

const editRole = (role: Role) => {
  editingRole.value = role
  formRole.value = {
    name:        role.name,
    description: role.description,
    permissions: [...role.permissions]
  }
  showEditDialog.value = true
}

const updateRole = async () => {
  if (!editingRole.value) return
  isSaving.value = true
  try {
    const response = await api.put(`/roles/${editingRole.value.id}`, formRole.value)
    if (response.data.success) {
      const index = roles.value.findIndex(r => r.id === editingRole.value?.id)
      if (index !== -1) roles.value[index] = response.data.role
      showEditDialog.value = false
      successMessage.value = 'Role updated!'
      showSuccess.value = true
    }
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'Failed to update role'
    showError.value = true
  } finally {
    isSaving.value = false
  }
}

const confirmDeleteRole = (role: Role) => {
  deletingRole.value = role
  showDeleteDialog.value = true
}

const deleteRole = async () => {
  if (!deletingRole.value) return
  isDeleting.value = true
  try {
    const response = await api.delete(`/roles/${deletingRole.value.id}`)
    if (response.data.success) {
      roles.value = roles.value.filter(r => r.id !== deletingRole.value?.id)
      showDeleteDialog.value = false
      successMessage.value = 'Role deleted!'
      showSuccess.value = true
    }
  } catch (err: any) {
    errorMessage.value = err.response?.data?.error || 'Failed to delete role'
    showError.value = true
  } finally {
    isDeleting.value = false
  }
}

onMounted(() => {
  fetchRoles()
  fetchPermissions()
})
</script>

<style scoped>
.role-management-page { min-height: calc(100vh - 120px); }
</style>
