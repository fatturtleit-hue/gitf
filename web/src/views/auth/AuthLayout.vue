<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <div class="auth-layout">
    <div class="bg-decoration">
      <div class="bg-blob bg-blob-1" />
      <div class="bg-blob bg-blob-2" />
    </div>

    <div class="auth-form">
      <div class="auth-form-inner">
        <router-link to="/" class="brand">
          <svg class="brand-logo" viewBox="0 0 512 512" width="36" height="36" aria-hidden="true">
            <g transform="rotate(-45, 256, 256)">
              <path d="M 60,170 C 125,108 195,108 256,170 C 317,232 387,232 452,170"
                    stroke="#4ADE80" stroke-width="28" fill="none" stroke-linecap="round"/>
              <path d="M 50,256 C 115,194 185,194 256,256 C 327,318 397,318 462,256"
                    stroke="#22C55E" stroke-width="28" fill="none" stroke-linecap="round"/>
              <path d="M 40,342 C 105,280 175,280 256,342 C 337,404 407,404 472,342"
                    stroke="#16A34A" stroke-width="28" fill="none" stroke-linecap="round"/>
            </g>
          </svg>
          <span class="brand-mark">gitf</span>
        </router-link>
        <router-view />
      </div>

      <div class="auth-footer">
        <span>© {{ new Date().getFullYear() }} gitf</span>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.auth-layout {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--bg);
  overflow: hidden;
  padding: 24px;
}

.bg-decoration {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
}

.bg-blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.18;
}

.bg-blob-1 {
  width: 420px;
  height: 420px;
  background: var(--accent);
  top: -120px;
  left: -100px;
}

.bg-blob-2 {
  width: 360px;
  height: 360px;
  background: #8b5cf6;
  bottom: -120px;
  right: -80px;
}

.auth-form {
  position: relative;
  width: 100%;
  max-width: 420px;
  z-index: 1;
}

.auth-form-inner {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 36px 32px;
  box-shadow:
    0 1px 2px rgba(0, 0, 0, 0.04),
    0 4px 16px rgba(0, 0, 0, 0.04);
}

.brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin-bottom: 8px;
  gap: 8px;
}

.brand-logo {
  display: block;
}

.brand-mark {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--accent);
  transition: opacity 0.15s;
}

.brand:hover .brand-mark {
  opacity: 0.8;
}

.auth-footer {
  margin-top: 20px;
  text-align: center;
  font-size: 12px;
  color: var(--text-muted);
}

@media (max-width: 480px) {
  .auth-form-inner {
    padding: 28px 20px;
  }
  .bg-blob-1 {
    width: 280px;
    height: 280px;
  }
  .bg-blob-2 {
    width: 240px;
    height: 240px;
  }
}
</style>
