<template>
  <div class="user-management-page">
    <v-row>
      <v-col cols="12">
        <div class="d-flex flex-column flex-sm-row align-sm-center justify-sm-space-between ga-3 mb-6">
          <div>
            <h1 class="text-h4 font-weight-bold mb-2">
              <v-icon icon="mdi-account-group" class="mr-2" color="primary" />
              User Management
            </h1>
            <p class="text-body-1 text-medium-emphasis">Manage users and their permissions</p>
          </div>
          <div class="d-flex flex-wrap ga-2">
            <v-btn color="warning" variant="tonal" @click="showForceLogoutDialog = true">
              <v-icon icon="mdi-logout-variant" class="mr-2" />
              Force Logout All
            </v-btn>
            <v-btn color="primary" size="large" @click="openCreateDialog">
              <v-icon icon="mdi-account-plus" class="mr-2" />
              Add User
            </v-btn>
          </div>
        </div>
      </v-col>
    </v-row>

    <!-- Users Table -->
    <v-row>
      <v-col cols="12">
        <v-card class="glass-card">
          <v-card-text>
            <div v-if="isLoading" class="text-center py-8">
              <v-progress-circular indeterminate color="primary" size="48" />
              <p class="text-body-2 text-medium-emphasis mt-4">Loading users...</p>
            </div>
            <template v-else>
              <v-text-field
                v-model="search"
                label="Search by name, email, or username"
                prepend-inner-icon="mdi-magnify"
                variant="outlined"
                density="compact"
                clearable
                single-line
                hide-details
                class="mb-4"
                style="max-width: 420px"
              />
              <v-data-table
                :headers="headers"
                :items="users"
                :search="search"
                :items-per-page="10"
                class="users-table"
              >
              <template v-slot:item.roles="{ item }">
                <div class="d-flex flex-wrap ga-1">
                  <v-chip
                    v-for="role in item.roles"
                    :key="role"
                    :color="getRoleColor(role)"
                    size="x-small"
                    variant="tonal"
                  >
                    <v-icon :icon="getRoleIcon(role)" size="12" class="mr-1" />
                    {{ formatRole(role) }}
                  </v-chip>
                </div>
              </template>
              <template v-slot:item.isActive="{ item }">
                <v-chip :color="item.isActive ? 'success' : 'error'" size="small" variant="tonal">
                  {{ item.isActive ? 'Active' : 'Inactive' }}
                </v-chip>
              </template>
              <template v-slot:item.status="{ item }">
                <v-chip
                  :color="item.status === 'approved' ? 'success' : item.status === 'rejected' ? 'error' : 'warning'"
                  size="small"
                  variant="tonal"
                >
                  {{ item.status === 'approved' ? 'Approved' : item.status === 'rejected' ? 'Rejected' : 'Pending' }}
                </v-chip>
              </template>
              <template v-slot:item.createdAt="{ item }">{{ formatDate(item.createdAt) }}</template>
              <template v-slot:item.lastLogin="{ item }">{{ item.lastLogin ? formatDate(item.lastLogin) : 'Never' }}</template>
              <template v-slot:item.actions="{ item }">
                <div class="d-flex ga-1">
                  <v-btn icon variant="text" size="small" @click="editUser(item)" :disabled="item.id === currentUser?.id">
                    <v-icon icon="mdi-pencil" size="18" />
                    <v-tooltip activator="parent" location="top">Edit</v-tooltip>
                  </v-btn>
                  <v-btn icon variant="text" size="small" @click="router.push(`/user-dashboard/${item.id}`)">
                    <v-icon icon="mdi-view-dashboard-outline" size="18" />
                    <v-tooltip activator="parent" location="top">View Dashboard</v-tooltip>
                  </v-btn>
                  <v-btn icon variant="text" size="small" @click="openSessionsDialog(item)">
                    <v-icon icon="mdi-folder-eye-outline" size="18" />
                    <v-tooltip activator="parent" location="top">View Sessions</v-tooltip>
                  </v-btn>
                  <v-btn icon variant="text" size="small" @click="openPermissionsDialog(item)" :disabled="item.id === currentUser?.id">
                    <v-icon icon="mdi-shield-key" size="18" />
                    <v-tooltip activator="parent" location="top">Permission Overrides</v-tooltip>
                  </v-btn>
                  <v-btn icon variant="text" size="small" color="warning" @click="openResetPasswordDialog(item)" :disabled="item.id === currentUser?.id">
                    <v-icon icon="mdi-lock-reset" size="18" />
                    <v-tooltip activator="parent" location="top">Reset Password</v-tooltip>
                  </v-btn>
                  <v-btn icon variant="text" size="small" color="error" @click="confirmDelete(item)" :disabled="item.id === currentUser?.id">
                    <v-icon icon="mdi-delete" size="18" />
                    <v-tooltip activator="parent" location="top">Delete</v-tooltip>
                  </v-btn>
                </div>
              </template>
              </v-data-table>
            </template>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Create User Dialog -->
    <v-dialog v-model="showCreateDialog" max-width="560">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-account-plus" class="mr-2" color="primary" />
          Add New User
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-form ref="createFormRef" @submit.prevent="createUser">
            <v-text-field
              v-model="newUser.name" label="Full Name" prepend-inner-icon="mdi-account"
              variant="outlined" :rules="[(v: string) => !!v || 'Name is required']" class="mb-3"
            />
            <v-text-field
              v-model="newUser.email" label="Email" type="email" prepend-inner-icon="mdi-email"
              variant="outlined"
              :rules="[(v: string) => !!v || 'Email is required', (v: string) => /.+@.+\..+/.test(v) || 'Invalid email']"
              class="mb-3"
            />
            <v-text-field
              v-model="newUser.password" label="Password" type="password" prepend-inner-icon="mdi-lock"
              variant="outlined"
              :rules="[(v: string) => !!v || 'Password is required', (v: string) => v.length >= 6 || 'Min 6 characters']"
              class="mb-3"
            />
            <div class="mb-1 text-body-2 font-weight-medium">Roles</div>
            <v-card variant="outlined" class="mb-2 pa-2">
              <v-checkbox
                v-for="role in availableRoles"
                :key="role.id"
                v-model="newUser.roles"
                :value="role.id"
                :label="role.name + (role.description ? ' — ' + role.description : '')"
                hide-details
                density="compact"
              />
            </v-card>
          </v-form>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showCreateDialog = false">Cancel</v-btn>
          <v-btn color="primary" :loading="isSaving" @click="createUser">
            <v-icon icon="mdi-check" class="mr-1" />Create User
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Edit User Dialog -->
    <v-dialog v-model="showEditDialog" max-width="560">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-account-edit" class="mr-2" color="secondary" />
          Edit User
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-form ref="editFormRef" @submit.prevent="updateUser">
            <v-text-field
              v-model="editingUser.name" label="Full Name" prepend-inner-icon="mdi-account"
              variant="outlined" :rules="[(v: string) => !!v || 'Name is required']" class="mb-3"
            />
            <v-text-field
              :model-value="editingUser.email" label="Email" prepend-inner-icon="mdi-email"
              variant="outlined" disabled class="mb-3"
            />
            <div class="mb-1 text-body-2 font-weight-medium">Roles</div>
            <v-card variant="outlined" class="mb-3 pa-2">
              <v-checkbox
                v-for="role in availableRoles"
                :key="role.id"
                v-model="editingUser.roles"
                :value="role.id"
                :label="role.name + (role.description ? ' — ' + role.description : '')"
                hide-details
                density="compact"
              />
            </v-card>
            <v-switch v-model="editingUser.isActive" label="Active" color="success" hide-details />
          </v-form>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showEditDialog = false">Cancel</v-btn>
          <v-btn color="secondary" :loading="isSaving" @click="updateUser">
            <v-icon icon="mdi-check" class="mr-1" />Save Changes
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Permission Overrides Dialog -->
    <v-dialog v-model="showPermDialog" max-width="600">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-shield-key" class="mr-2" color="primary" />
          Permission Overrides — {{ permUser?.name }}
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <!-- Effective Permissions Preview -->
          <div class="mb-4">
            <div class="text-body-2 font-weight-medium mb-2">Effective Permissions</div>
            <div v-if="permBreakdown" class="d-flex flex-wrap ga-1 mb-1">
              <v-chip
                v-for="p in permBreakdown.effective"
                :key="p"
                size="x-small"
                color="success"
                variant="tonal"
              >{{ p }}</v-chip>
              <span v-if="!permBreakdown.effective?.length" class="text-caption text-medium-emphasis">None</span>
            </div>
            <div v-if="permBreakdown?.granted?.length" class="text-caption text-success mb-1">
              + Granted: {{ permBreakdown.granted.join(', ') }}
            </div>
            <div v-if="permBreakdown?.denied?.length" class="text-caption text-error">
              - Denied: {{ permBreakdown.denied.join(', ') }}
            </div>
          </div>

          <v-divider class="mb-4" />

          <!-- Grant a permission -->
          <div class="mb-4">
            <div class="text-body-2 font-weight-medium mb-2 text-success">Grant Extra Permission</div>
            <div class="d-flex ga-2">
              <v-select
                v-model="grantPerm"
                :items="availablePermissions"
                label="Select permission to grant"
                variant="outlined"
                density="compact"
                hide-details
                clearable
                class="flex-grow-1"
              />
              <v-btn color="success" :loading="isPermSaving" @click="grantPermission" :disabled="!grantPerm">
                <v-icon icon="mdi-plus" />
              </v-btn>
            </div>
          </div>

          <!-- Deny a permission -->
          <div>
            <div class="text-body-2 font-weight-medium mb-2 text-error">Deny Permission</div>
            <div class="d-flex ga-2">
              <v-select
                v-model="denyPerm"
                :items="availablePermissions"
                label="Select permission to deny"
                variant="outlined"
                density="compact"
                hide-details
                clearable
                class="flex-grow-1"
              />
              <v-btn color="error" :loading="isPermSaving" @click="denyPermission" :disabled="!denyPerm">
                <v-icon icon="mdi-minus" />
              </v-btn>
            </div>
          </div>

          <!-- Current overrides -->
          <div v-if="(permBreakdown?.granted?.length || permBreakdown?.denied?.length)" class="mt-4">
            <v-divider class="mb-3" />
            <div class="text-body-2 font-weight-medium mb-2">Remove Overrides</div>
            <div class="d-flex flex-wrap ga-1">
              <v-chip
                v-for="p in permBreakdown?.granted"
                :key="'g-' + p"
                size="small"
                color="success"
                variant="tonal"
                closable
                @click:close="removeOverride(p, 'grant')"
              >+ {{ p }}</v-chip>
              <v-chip
                v-for="p in permBreakdown?.denied"
                :key="'d-' + p"
                size="small"
                color="error"
                variant="tonal"
                closable
                @click:close="removeOverride(p, 'deny')"
              >- {{ p }}</v-chip>
            </div>
          </div>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showPermDialog = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="showDeleteDialog" max-width="400">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-alert-circle" class="mr-2" color="error" />
          Confirm Delete
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <p>Are you sure you want to delete <strong>{{ deletingUser?.name }}</strong>?</p>
          <p class="text-medium-emphasis text-body-2 mt-2">This action cannot be undone.</p>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showDeleteDialog = false">Cancel</v-btn>
          <v-btn color="error" :loading="isDeleting" @click="deleteUser">
            <v-icon icon="mdi-delete" class="mr-1" />Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Reset Password Dialog -->
    <v-dialog v-model="showResetPasswordDialog" max-width="440">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-lock-reset" class="mr-2" color="warning" />
          Reset Password — {{ resetPasswordUser?.name }}
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-text-field
            v-model="newPassword"
            label="New Password"
            :type="showNewPassword ? 'text' : 'password'"
            prepend-inner-icon="mdi-lock"
            :append-inner-icon="showNewPassword ? 'mdi-eye-off' : 'mdi-eye'"
            @click:append-inner="showNewPassword = !showNewPassword"
            variant="outlined"
            :rules="[(v: string) => v.length >= 6 || 'Min 6 characters']"
            hint="Min 6 characters"
            persistent-hint
          />
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showResetPasswordDialog = false">Cancel</v-btn>
          <v-btn color="warning" :loading="isResettingPassword" @click="resetPassword" :disabled="newPassword.length < 6">
            <v-icon icon="mdi-lock-reset" class="mr-1" />Reset Password
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Force Logout Dialog -->
    <v-dialog v-model="showForceLogoutDialog" max-width="500">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-logout-variant" class="mr-2" color="warning" />
          Force Logout All Users
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-alert type="warning" variant="tonal" class="mb-4">
            <strong>Warning:</strong> This action will immediately log out ALL users, including yourself.
          </v-alert>
          <p>All users will need to log in again after this action.</p>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showForceLogoutDialog = false">Cancel</v-btn>
          <v-btn color="warning" :loading="isForceLoggingOut" @click="forceLogoutAll">
            <v-icon icon="mdi-logout-variant" class="mr-1" />Force Logout All
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- View Sessions Dialog (superuser: keep an eye on any user's activity) -->
    <v-dialog v-model="showSessionsDialog" max-width="700">
      <v-card class="glass-card">
        <v-card-title class="d-flex align-center pa-4">
          <v-icon icon="mdi-folder-eye-outline" class="mr-2" color="primary" />
          Sessions — {{ sessionsUser?.name }}
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4" style="max-height: 60vh; overflow-y: auto;">
          <div v-if="sessionsLoading" class="text-center py-8">
            <v-progress-circular indeterminate color="primary" />
          </div>
          <div v-else-if="sessionsList.length === 0" class="text-center py-8 text-medium-emphasis">
            No saved sessions for this user.
          </div>
          <v-expansion-panels v-else variant="accordion">
            <v-expansion-panel v-for="s in sessionsList" :key="s.id">
              <v-expansion-panel-title>
                <div>
                  <div class="font-weight-medium">{{ s.title }}</div>
                  <div class="text-caption text-medium-emphasis">{{ s.type }} · {{ formatDate(s.updatedAt) }}</div>
                </div>
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <p class="text-body-2" style="white-space: pre-wrap;">{{ s.transcript }}</p>
              </v-expansion-panel-text>
            </v-expansion-panel>
          </v-expansion-panels>
        </v-card-text>
        <v-divider />
        <v-card-actions class="pa-4">
          <v-spacer />
          <v-btn variant="text" @click="showSessionsDialog = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/services/api'
import { currentUser, logout } from '@/stores/auth'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const toast = useToast()

interface UserRecord {
  id: string
  email: string
  name: string
  roles: string[]
  permissions: string[]
  granted_permissions: string[]
  denied_permissions: string[]
  isActive: boolean
  status: string
  requestedRole?: string
  createdAt: string
  lastLogin?: string
}

interface Role {
  id: string
  name: string
  description: string
  is_system: boolean
  permissions: string[]
}

interface PermissionBreakdown {
  effective: string[]
  from_roles: string[]
  granted: string[]
  denied: string[]
}

const users            = ref<UserRecord[]>([])
const search           = ref('')
const availableRoles   = ref<Role[]>([])
const availablePermissions = ref<string[]>([])

const isLoading           = ref(true)
const isSaving            = ref(false)
const isDeleting          = ref(false)
const isForceLoggingOut   = ref(false)
const isPermSaving        = ref(false)
const isResettingPassword = ref(false)

const showCreateDialog       = ref(false)
const showEditDialog         = ref(false)
const showDeleteDialog       = ref(false)
const showForceLogoutDialog  = ref(false)
const showPermDialog         = ref(false)
const showResetPasswordDialog = ref(false)


const createFormRef = ref()
const editFormRef   = ref()

const newUser = ref({ name: '', email: '', password: '', roles: ['doctor'] as string[] })

const editingUser = ref<UserRecord>({
  id: '', email: '', name: '', roles: [], permissions: [],
  granted_permissions: [], denied_permissions: [],
  isActive: true, status: 'approved', createdAt: ''
})

const deletingUser        = ref<UserRecord | null>(null)
const resetPasswordUser   = ref<UserRecord | null>(null)
const newPassword         = ref('')
const showNewPassword     = ref(false)
const permUser       = ref<UserRecord | null>(null)
const permBreakdown  = ref<PermissionBreakdown | null>(null)
const grantPerm      = ref<string | null>(null)
const denyPerm       = ref<string | null>(null)

interface SessionSummary {
  id: string
  title: string
  type: string
  transcript: string
  updatedAt: string
}
const showSessionsDialog = ref(false)
const sessionsUser  = ref<UserRecord | null>(null)
const sessionsList  = ref<SessionSummary[]>([])
const sessionsLoading = ref(false)

const headers = [
  { title: 'Name',       key: 'name',      sortable: true },
  { title: 'Email',      key: 'email',     sortable: true },
  { title: 'Roles',      key: 'roles',     sortable: false },
  { title: 'Active',     key: 'isActive',  sortable: true },
  { title: 'Approval',   key: 'status',    sortable: true },
  { title: 'Created',    key: 'createdAt', sortable: true },
  { title: 'Last Login', key: 'lastLogin', sortable: true },
  { title: 'Actions',    key: 'actions',   sortable: false, align: 'center' as const },
]

const getRoleColor = (role: string) => {
  switch (role) {
    case 'superuser': return 'error'
    case 'doctor':    return 'primary'
    default:          return 'info'
  }
}

const getRoleIcon = (role: string) => {
  switch (role) {
    case 'superuser': return 'mdi-shield-crown'
    case 'doctor':    return 'mdi-stethoscope'
    default:          return 'mdi-account'
  }
}

const formatRole = (role: string) => {
  const found = availableRoles.value.find(r => r.id === role)
  return found?.name ?? (role.charAt(0).toUpperCase() + role.slice(1))
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit'
  })
}

const openSessionsDialog = async (item: UserRecord) => {
  sessionsUser.value = item
  showSessionsDialog.value = true
  sessionsLoading.value = true
  sessionsList.value = []
  try {
    const response = await api.get(`/admin/sessions/${item.id}`)
    sessionsList.value = response.data.sessions || []
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to load sessions')
    showSessionsDialog.value = false
  } finally {
    sessionsLoading.value = false
  }
}

const fetchUsers = async () => {
  isLoading.value = true
  try {
    const response = await api.get('/users')
    if (response.data.success) {
      users.value = response.data.users
    }
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to load users')
  } finally {
    isLoading.value = false
  }
}

const fetchRoles = async () => {
  try {
    const response = await api.get('/roles')
    if (response.data.success) {
      availableRoles.value = response.data.roles
    }
  } catch { /* non-fatal */ }
}

const fetchPermissions = async () => {
  try {
    const response = await api.get('/permissions')
    if (response.data.success) {
      availablePermissions.value = response.data.permissions
    }
  } catch { /* non-fatal */ }
}

const openCreateDialog = () => {
  newUser.value = { name: '', email: '', password: '', roles: ['doctor'] }
  showCreateDialog.value = true
}

const createUser = async () => {
  isSaving.value = true
  try {
    const response = await api.post('/users', newUser.value)
    if (response.data.success) {
      users.value.push(response.data.user)
      showCreateDialog.value = false
      toast.success('User created successfully!')
    }
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to create user')
  } finally {
    isSaving.value = false
  }
}

const editUser = (user: UserRecord) => {
  editingUser.value = { ...user, roles: [...(user.roles ?? [])] }
  showEditDialog.value = true
}

const updateUser = async () => {
  isSaving.value = true
  try {
    // Update basic info
    await api.put(`/users/${editingUser.value.id}`, {
      name:     editingUser.value.name,
      isActive: editingUser.value.isActive
    })
    // Update roles
    const rolesResponse = await api.put(`/users/${editingUser.value.id}/roles`, {
      roles: editingUser.value.roles
    })
    if (rolesResponse.data.success) {
      const index = users.value.findIndex(u => u.id === editingUser.value.id)
      if (index !== -1) users.value[index] = rolesResponse.data.user
    }
    showEditDialog.value = false
    toast.success('User updated successfully!')
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to update user')
  } finally {
    isSaving.value = false
  }
}

const confirmDelete = (user: UserRecord) => {
  deletingUser.value = user
  showDeleteDialog.value = true
}

const deleteUser = async () => {
  if (!deletingUser.value) return
  isDeleting.value = true
  try {
    const response = await api.delete(`/users/${deletingUser.value.id}`)
    if (response.data.success) {
      users.value = users.value.filter(u => u.id !== deletingUser.value?.id)
      showDeleteDialog.value = false
      toast.success('User deleted successfully!')
    }
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to delete user')
  } finally {
    isDeleting.value = false
  }
}

const openPermissionsDialog = async (user: UserRecord) => {
  permUser.value = user
  grantPerm.value = null
  denyPerm.value  = null
  showPermDialog.value = true
  await fetchPermissionBreakdown(user.id)
}

const fetchPermissionBreakdown = async (userId: string) => {
  try {
    const response = await api.get(`/users/${userId}/permissions`)
    if (response.data.success) {
      permBreakdown.value = response.data.data
    }
  } catch { /* non-fatal */ }
}

const grantPermission = async () => {
  if (!permUser.value || !grantPerm.value) return
  isPermSaving.value = true
  try {
    await api.post(`/users/${permUser.value.id}/permissions/grant`, { permission: grantPerm.value })
    grantPerm.value = null
    await fetchPermissionBreakdown(permUser.value.id)
    await fetchUsers()
    toast.success('Permission granted!')
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to grant permission')
  } finally {
    isPermSaving.value = false
  }
}

const denyPermission = async () => {
  if (!permUser.value || !denyPerm.value) return
  isPermSaving.value = true
  try {
    await api.post(`/users/${permUser.value.id}/permissions/deny`, { permission: denyPerm.value })
    denyPerm.value = null
    await fetchPermissionBreakdown(permUser.value.id)
    await fetchUsers()
    toast.success('Permission denied!')
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to deny permission')
  } finally {
    isPermSaving.value = false
  }
}

const removeOverride = async (perm: string, kind: 'grant' | 'deny') => {
  if (!permUser.value) return
  isPermSaving.value = true
  try {
    await api.delete(`/users/${permUser.value.id}/permissions/${encodeURIComponent(perm)}?type=${kind}`)
    await fetchPermissionBreakdown(permUser.value.id)
    await fetchUsers()
    toast.success('Override removed!')
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to remove override')
  } finally {
    isPermSaving.value = false
  }
}

const openResetPasswordDialog = (user: UserRecord) => {
  resetPasswordUser.value = user
  newPassword.value = ''
  showNewPassword.value = false
  showResetPasswordDialog.value = true
}

const resetPassword = async () => {
  if (!resetPasswordUser.value || newPassword.value.length < 6) return
  isResettingPassword.value = true
  try {
    const response = await api.put(`/users/${resetPasswordUser.value.id}/password`, {
      new_password: newPassword.value
    })
    if (response.data.success) {
      showResetPasswordDialog.value = false
      toast.success(`Password reset for ${resetPasswordUser.value.name}`)
    }
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to reset password')
  } finally {
    isResettingPassword.value = false
  }
}

const forceLogoutAll = async () => {
  isForceLoggingOut.value = true
  try {
    const response = await api.post('/auth/force-logout-all')
    if (response.data.success) {
      showForceLogoutDialog.value = false
      sessionStorage.setItem('session_expired_message', 'All users have been logged out by an administrator.')
      logout()
    }
  } catch (err: any) {
    toast.error(err.response?.data?.error || 'Failed to force logout all users')
  } finally {
    isForceLoggingOut.value = false
  }
}

onMounted(() => {
  fetchUsers()
  fetchRoles()
  fetchPermissions()
})
</script>

<style scoped>
.user-management-page { min-height: calc(100vh - 120px); }
.users-table { background: transparent !important; }
:deep(.v-data-table) { background: transparent !important; }
:deep(.v-data-table-header) { background: rgba(255, 255, 255, 0.05) !important; }
:deep(.v-data-table__tr:hover) { background: rgba(255, 255, 255, 0.05) !important; }
</style>
