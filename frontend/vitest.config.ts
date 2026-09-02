import { defineConfig } from 'vitest/config'
import path from 'path'

// Deliberately separate from vite.config.ts: the report tests exercise pure
// TypeScript, so they need neither the Vue nor the Vuetify plugin.
export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.spec.ts'],
  },
})
