<template>
  <v-alert
    v-if="isDemoAccount"
    :type="remaining === 0 ? 'warning' : 'info'"
    variant="tonal"
    density="compact"
    class="mb-4"
  >
    <div class="d-flex align-center justify-space-between flex-wrap ga-2">
      <span>
        <strong>{{ label }}</strong> — {{ description }}
        <span v-if="!loading">
          {{ ' ' }}({{ remaining }}/{{ limit }} free {{ remaining === 1 ? 'try' : 'tries' }} left)
        </span>
      </span>
    </div>
    <p v-if="remaining === 0" class="text-body-2 mb-0 mt-1">
      You've used all {{ limit }} free trials for this feature. Contact us to upgrade your account for full access.
    </p>
  </v-alert>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import api from '@/services/api'
import { currentUser } from '@/stores/auth'

const props = defineProps<{
  feature: string
  label: string
  description: string
}>()

const emit = defineEmits<{ (e: 'update:limitReached', reached: boolean): void }>()

const limit = 3
const used = ref(0)
const loading = ref(true)

const isDemoAccount = computed(() => currentUser.value?.roles?.includes('user') ?? false)
const remaining = computed(() => Math.max(0, limit - used.value))

watch(remaining, (r) => emit('update:limitReached', r === 0), { immediate: true })

const load = async () => {
  if (!isDemoAccount.value) {
    loading.value = false
    return
  }
  loading.value = true
  try {
    const response = await api.get('/users/me/demo-usage')
    used.value = response.data.usage?.[props.feature] ?? 0
  } catch {
    // Best-effort — the server still enforces the real limit even if this
    // proactive display fails to load; don't block the feature over it.
    used.value = 0
  } finally {
    loading.value = false
  }
}

onMounted(load)
defineExpose({ refresh: load })
</script>
