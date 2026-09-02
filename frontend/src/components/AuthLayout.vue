<template>
  <div class="login-page">
    <div class="login-container">
      <!-- Logo & Branding -->
      <div class="branding">
        <div class="logo-container">
          <v-icon icon="mdi-stethoscope" size="64" color="primary" />
        </div>
        <h1 class="app-title">XStek AI Medical Transcription</h1>
        <p class="app-subtitle">Powered by xstek AI</p>
      </div>

      <slot />

      <!-- Footer -->
      <p class="footer-text text-caption text-medium-emphasis mt-6 text-center">
        © 2026 xstek. All rights reserved.
      </p>
    </div>

    <!-- Background Elements -->
    <div class="bg-element bg-element-1"></div>
    <div class="bg-element bg-element-2"></div>
    <div class="bg-element bg-element-3"></div>
  </div>
</template>

<script setup lang="ts">
// Shared chrome for LoginPage.vue and SignupPage.vue — branding, footer,
// and background decoration were previously copy-pasted identically in
// both. Each page supplies its own card (form) via the default slot.
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, #0a0a0f 0%, #1a1a2e 50%, #0f0f1a 100%);
}

.login-container {
  width: 100%;
  max-width: 420px;
  z-index: 1;
}

.branding {
  text-align: center;
  margin-bottom: 32px;
}

.logo-container {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100px;
  height: 100px;
  border-radius: 24px;
  background: linear-gradient(135deg, rgba(29, 233, 182, 0.2) 0%, rgba(124, 92, 255, 0.2) 100%);
  border: 1px solid rgba(29, 233, 182, 0.3);
  margin-bottom: 16px;
  box-shadow: 0 8px 32px rgba(29, 233, 182, 0.15);
}

.app-title {
  font-size: 1.75rem;
  font-weight: 700;
  background: linear-gradient(135deg, #1DE9B6 0%, #7C5CFF 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 4px;
}

.app-subtitle {
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.875rem;
}

:deep(.login-card) {
  background: rgba(30, 30, 46, 0.8) !important;
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px !important;
}

.footer-text {
  opacity: 0.6;
}

:deep(.signup-link) {
  color: #1DE9B6;
  font-weight: 600;
  text-decoration: none;
}

:deep(.signup-link:hover) {
  text-decoration: underline;
}

/* Background Elements */
.bg-element {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  opacity: 0.3;
  pointer-events: none;
}

.bg-element-1 {
  width: 400px;
  height: 400px;
  background: linear-gradient(135deg, #1DE9B6 0%, #00BFA5 100%);
  top: -100px;
  right: -100px;
}

.bg-element-2 {
  width: 300px;
  height: 300px;
  background: linear-gradient(135deg, #7C5CFF 0%, #536DFE 100%);
  bottom: -50px;
  left: -50px;
}

.bg-element-3 {
  width: 200px;
  height: 200px;
  background: linear-gradient(135deg, #FF6B6B 0%, #FF8E53 100%);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  opacity: 0.15;
}
</style>
