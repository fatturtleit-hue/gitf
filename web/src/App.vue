<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const hideToolbar = computed(() => route.path === '/login' || route.path === '/register')

function logout() {
  auth.logout()
  router.push('/login')
}

const userInitial = computed(() => (auth.user?.username?.charAt(0) ?? '?').toUpperCase())
</script>

<template>
  <header v-if="!hideToolbar" class="topbar">
    <div class="topbar-inner container">
      <router-link to="/" class="brand" aria-label="gitf home">
        <svg class="brand-logo" viewBox="0 0 512 512" width="26" height="26" aria-hidden="true">
          <g transform="rotate(-45, 256, 256)">
            <path d="M 60,170 C 125,108 195,108 256,170 C 317,232 387,232 452,170"
                  stroke="#4ADE80" stroke-width="28" fill="none" stroke-linecap="round"/>
            <path d="M 50,256 C 115,194 185,194 256,256 C 327,318 397,318 462,256"
                  stroke="#22C55E" stroke-width="28" fill="none" stroke-linecap="round"/>
            <path d="M 40,342 C 105,280 175,280 256,342 C 337,404 407,404 472,342"
                  stroke="#16A34A" stroke-width="28" fill="none" stroke-linecap="round"/>
          </g>
        </svg>
        <span class="brand-text">gitf</span>
      </router-link>

      <template v-if="auth.isAuthenticated">
        <nav class="nav-center" aria-label="主导航">
          <router-link to="/repos" class="nav-link" active-class="nav-active">仓库</router-link>
          <router-link to="/orgs" class="nav-link" active-class="nav-active">组织</router-link>
        </nav>

        <div class="spacer"></div>

        <router-link to="/repos/new" class="el-button el-button--primary el-button--small nav-new-btn">
          <el-icon><Plus /></el-icon>
          <span class="nav-new-text">新建</span>
        </router-link>

        <el-dropdown trigger="click">
          <button class="user-btn" aria-label="用户菜单">
            <span class="avatar avatar-sm">{{ userInitial }}</span>
            <span class="user-label">{{ auth.user?.username }}</span>
            <el-icon><ArrowDown /></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <div class="menu-header">
                <strong>{{ auth.user?.username }}</strong>
                <div class="muted">{{ auth.user?.email }}</div>
              </div>
              <el-dropdown-item @click="$router.push('/admin')">
                <el-icon><Grid /></el-icon> 管理
              </el-dropdown-item>
              <el-dropdown-item @click="$router.push('/tokens')">
                <el-icon><Key /></el-icon> PAT 令牌
              </el-dropdown-item>
              <el-dropdown-item divided @click="logout">
                <el-icon><SwitchButton /></el-icon> 退出
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </template>

      <template v-else>
        <div class="spacer"></div>
        <router-link to="/login" class="el-button el-button--text">登录</router-link>
        <router-link v-if="auth.registrationEnabled" to="/register" class="el-button el-button--primary el-button--small">
          注册
        </router-link>
      </template>
    </div>
  </header>

  <main :class="{ 'full-height': hideToolbar }">
    <router-view />
  </main>
</template>

<style scoped lang="scss">
.topbar {
  position: sticky;
  top: 0;
  z-index: 100;
  height: var(--header-h);
  background: var(--surface);
  border-bottom: 1px solid var(--border);
}

.topbar-inner {
  display: flex;
  align-items: center;
  height: 100%;
  gap: 8px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  font-size: 18px;
  color: var(--text);
}

.brand:hover {
  color: var(--text);
}

.brand-text {
  letter-spacing: -0.02em;
}

.nav-center {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: 24px;
}

.nav-link {
  display: flex;
  align-items: center;
  height: 32px;
  padding: 0 12px;
  border-radius: var(--radius);
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: background 0.05s;
}

.nav-link:hover {
  background: var(--ink-green-light);
  color: var(--text);
}

.nav-active {
  color: var(--text);
  background: var(--ink-green-light);
}

.spacer {
  flex: 1;
}

.nav-new-btn {
  margin-right: 8px;
}

.user-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 34px;
  padding: 0 8px 0 4px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  cursor: pointer;
  transition: border-color 0.15s;
}

.user-btn:hover {
  border-color: var(--gray-300);
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--ink-green-light);
  color: var(--ink-green);
  font-size: 12px;
  font-weight: 600;
}

.user-label {
  font-size: 13px;
  font-weight: 500;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-header {
  padding: 8px 16px;
  line-height: 1.5;
  border-bottom: 1px solid var(--border);
  margin-bottom: 4px;
}

main {
  min-height: calc(100vh - var(--header-h));
}

main.full-height {
  min-height: 100vh;
}

@media (max-width: 768px) {
  .nav-center {
    margin-left: 12px;
    gap: 0;
  }

  .nav-link {
    padding: 0 8px;
    font-size: 13px;
  }

  .nav-new-text {
    display: none;
  }

  .nav-new-btn {
    margin-right: 4px;
    padding-left: 8px;
    padding-right: 8px;
  }
}

@media (max-width: 640px) {
  .user-label {
    display: none;
  }
}

@media (max-width: 480px) {
  .brand-text {
    display: none;
  }

  .nav-link {
    padding: 0 6px;
    font-size: 12px;
  }

  .topbar-inner {
    gap: 4px;
  }
}
</style>
