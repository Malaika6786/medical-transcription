<template>
  <!-- One snackbar per queued message, stacked. In practice at most one or
       two are ever visible at once (they self-dismiss on their own
       timeout), but stacking avoids a fast double-action silently
       swallowing the first message's toast. -->
  <v-snackbar
    v-for="(msg, i) in queue"
    :key="msg.id"
    :model-value="true"
    :color="msg.color"
    :timeout="msg.timeout"
    :style="{ marginBottom: `${i * 56}px` }"
    location="bottom"
    @update:model-value="(v: boolean) => !v && dismiss(msg.id)"
  >
    {{ msg.text }}
    <template v-slot:actions>
      <v-btn variant="text" size="small" @click="dismiss(msg.id)">Close</v-btn>
    </template>
  </v-snackbar>
</template>

<script setup lang="ts">
import { useToastQueue } from '@/composables/useToast'

const { queue, dismiss } = useToastQueue()
</script>
