<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { repoApi } from '@/api/repo'
import type { Repository } from '@/api/types'
import EmptyState from '@/components/EmptyState.vue'
import ErrorState from '@/components/ErrorState.vue'
import { formatDate } from '@/utils/format'

const auth = useAuthStore()
const repos = ref<Repository[]>([])
const loading = ref(true)
const error = ref('')

async function loadRepos() {
  loading.value = true
  error.value = ''
  try {
    const res = await repoApi.list(undefined, 1, 6)
    repos.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (auth.isAuthenticated) {
    loadRepos()
  } else {
    loading.value = false
  }
})
</script>

<template>
  <div v-if="auth.isAuthenticated" class="container page">
    <header class="page-header">
      <div class="greeting">
        <h1>你好，{{ auth.user?.username }}</h1>
        <p class="subtle">欢迎回到 gitf</p>
      </div>
      <router-link to="/repos/new" class="el-button el-button--primary">
        <el-icon><Plus /></el-icon>
        <span>新建仓库</span>
      </router-link>
    </header>

    <div class="quick-nav">
      <router-link to="/repos" class="el-button">
        <el-icon><Files /></el-icon>
        <span>全部仓库</span>
      </router-link>
      <router-link to="/orgs" class="el-button">
        <el-icon><OfficeBuilding /></el-icon>
        <span>组织</span>
      </router-link>
      <router-link to="/admin" class="el-button">
        <el-icon><Setting /></el-icon>
        <span>管理</span>
      </router-link>
    </div>

    <section class="section">
      <div v-if="loading" class="loading">
        <el-icon class="is-loading" :size="28"><Loading /></el-icon>
      </div>
      <ErrorState v-else-if="error" :message="error" @retry="loadRepos" />
      <EmptyState
        v-else-if="repos.length === 0"
        icon="FolderOpened"
        title="还没有仓库"
        message="创建你的第一个仓库"
      >
        <template #actions>
          <router-link to="/repos/new" class="el-button el-button--primary">创建仓库</router-link>
        </template>
      </EmptyState>
      <template v-else>
        <div class="section-head">
          <h3>最近仓库</h3>
          <router-link to="/repos" class="more-link">
            查看全部
            <el-icon :size="14"><ArrowRight /></el-icon>
          </router-link>
        </div>
        <div class="repo-grid">
          <router-link
            v-for="repo in repos"
            :key="repo.id"
            :to="`/${repo.owner_username}/${repo.name}`"
            class="repo-card card"
          >
            <div class="repo-card-header">
              <div class="repo-icon">
                <el-icon :size="18"><Folder /></el-icon>
              </div>
              <div class="repo-meta">
                <div class="repo-title text-ellipsis">
                  <span class="subtle">{{ repo.owner_username }} /</span> {{ repo.name }}
                </div>
                <div class="repo-desc text-clamp-2">
                  {{ repo.description || '暂无描述' }}
                </div>
              </div>
            </div>
            <div class="repo-card-footer">
              <span class="meta-item">
                <el-icon :size="12"><Collection /></el-icon> {{ repo.default_branch }}
              </span>
              <span class="meta-item">
                <el-icon :size="12"><Clock /></el-icon> {{ formatDate(repo.updated_at) }}
              </span>
            </div>
          </router-link>
        </div>
      </template>
    </section>
  </div>

  <div v-else class="hero">
    <div class="hero-content">
      <div class="hero-mark">
        <svg class="hero-logo" viewBox="0 0 512 512" width="48" height="48" aria-hidden="true">
          <g transform="rotate(-45, 256, 256)">
            <path d="M 60,170 C 125,108 195,108 256,170 C 317,232 387,232 452,170"
                  stroke="#4ADE80" stroke-width="28" fill="none" stroke-linecap="round"/>
            <path d="M 50,256 C 115,194 185,194 256,256 C 327,318 397,318 462,256"
                  stroke="#22C55E" stroke-width="28" fill="none" stroke-linecap="round"/>
            <path d="M 40,342 C 105,280 175,280 256,342 C 337,404 407,404 472,342"
                  stroke="#16A34A" stroke-width="28" fill="none" stroke-linecap="round"/>
          </g>
        </svg>
        gitf
      </div>
      <h1>轻量级自托管 Git 服务</h1>
      <p class="hero-desc">
        单二进制部署，无需依赖云端。托管仓库、跟踪 issue、运行 CI — 一切尽在你的掌控。
      </p>
      <div class="hero-actions">
        <router-link v-if="auth.registrationEnabled" to="/register" class="el-button el-button--primary hero-btn">
          开始使用
        </router-link>
        <router-link to="/login" class="el-button hero-btn">
          登录
        </router-link>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
// ── Logged-in view ─────────────────────────────────────────
.page {
  padding-top: 24px;
  padding-bottom: 48px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 28px;
  gap: 16px;
}

.greeting {
  min-width: 0;

  h1 {
    font-size: 24px;
    font-weight: 600;
    letter-spacing: -0.01em;
    margin: 0 0 4px;
  }

  .subtle {
    font-size: 13px;
    margin: 0;
  }
}

.quick-nav {
  display: flex;
  gap: 8px;
  margin-bottom: 28px;
  flex-wrap: wrap;

  .el-button {
    gap: 6px;
  }
}

.section {
  margin-top: 4px;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;

  h3 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
  }
}

.more-link {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  font-size: 13px;
  color: var(--text-secondary);
  transition: color 0.15s;

  &:hover {
    color: var(--accent);
  }
}

.repo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 14px;
}

.repo-card {
  display: flex;
  flex-direction: column;
  padding: 16px 18px;
  transition: box-shadow 0.15s, border-color 0.15s, transform 0.15s;
  text-decoration: none;
  color: var(--text);

  &:hover {
    border-color: var(--gray-300);
    box-shadow: var(--shadow);
    transform: translateY(-1px);
  }
}

.repo-card-header {
  display: flex;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.repo-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--accent-light);
  color: var(--accent);
  flex-shrink: 0;
}

.repo-meta {
  flex: 1;
  min-width: 0;
}

.repo-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}

.repo-desc {
  font-size: 13px;
  color: var(--text-secondary);
  min-height: 38px;
  line-height: 1.5;
}

.repo-card-footer {
  display: flex;
  gap: 16px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--border-light);
  font-size: 12px;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--text-muted);
}

// ── Hero (logged-out) ───────────────────────────────────────
.hero {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - var(--header-h));
  padding: 48px 24px;
  text-align: center;
}

.hero-content {
  max-width: 520px;
}

.hero-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--accent);
  margin-bottom: 16px;
}

.hero-logo {
  display: block;
}

.hero h1 {
  font-size: 36px;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin: 0 0 16px;
}

.hero-desc {
  font-size: 16px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin: 0 0 32px;
}

.hero-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.hero-btn {
  min-width: 120px;
  height: 40px;
  font-size: 14px;
  font-weight: 500;
}

@media (max-width: 640px) {
  .hero {
    padding: 32px 16px;
  }

  .hero h1 {
    font-size: 26px;
  }

  .hero-desc {
    font-size: 14px;
  }

  .hero-mark {
    font-size: 22px;
    gap: 6px;
  }
}

@media (max-width: 480px) {
  .hero h1 {
    font-size: 22px;
  }

  .hero-desc {
    font-size: 13px;
  }

  .hero-actions {
    flex-direction: column;
    align-items: center;
  }

  .hero-btn {
    width: 100%;
  }
}
</style>
