<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { repoApi } from '@/api/repo'
import type { Repository, RepoStats } from '@/api/types'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

const repo = ref<Repository | null>(null)
const branches = ref<string[]>([])
const branch = ref('main')
const stats = ref<RepoStats | null>(null)
const loading = ref(true)
const error = ref('')
const copied = ref(false)

const owner = computed(() => (route.params.owner as string) || '')
const name = computed(() => (route.params.name as string) || '')
const cloneUrl = computed(() => `${window.location.origin}/${owner.value}/${name.value}.git`)

const tabs = [
  { name: 'repo-tree', label: '代码' },
  { name: 'repo-commits', label: '提交' },
  { name: 'repo-pulls', label: 'PR' },
  { name: 'repo-issues', label: 'Issues' },
  { name: 'repo-labels', label: '标签' },
  { name: 'repo-milestones', label: '里程碑' },
  { name: 'repo-tags', label: 'Tags' },
  { name: 'repo-releases', label: '发布' },
  { name: 'repo-hooks', label: 'Hooks' },
  { name: 'repo-settings', label: '设置' }
]

async function loadRepo() {
  loading.value = true
  error.value = ''
  try {
    const r = await repoApi.get(owner.value, name.value)
    repo.value = r
    branch.value = r.default_branch
    loadStats(r.default_branch)
  } catch (err: any) {
    error.value = err.message || '仓库不存在'
  } finally {
    loading.value = false
  }
}

async function loadBranches() {
  try {
    const res = await repoApi.branches(owner.value, name.value)
    branches.value = res.branches
  } catch {
    branches.value = []
  }
}

async function loadStats(b: string) {
  try {
    stats.value = await repoApi.stats(owner.value, name.value, b)
  } catch {
    stats.value = null
  }
}

function onBranchChange(val: string) {
  branch.value = val
  loadStats(val)
  router.replace({ query: { ...route.query, branch: val } })
}

function copyCloneUrl() {
  navigator.clipboard.writeText(cloneUrl.value).then(() => {
    copied.value = true
    ElMessage.success('克隆地址已复制')
    setTimeout(() => (copied.value = false), 1600)
  })
}

watch(() => [route.params.owner, route.params.name], loadRepo, { immediate: true })

onMounted(() => {
  loadBranches()
})
</script>

<template>
  <div v-if="loading" class="container loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <section v-else-if="error" class="container" style="padding: 64px 24px; text-align: center">
    <el-icon :size="40" color="var(--gray-300)"><Warning /></el-icon>
    <h2 style="margin: 12px 0 4px">仓库不存在</h2>
    <p class="subtle">{{ error }}</p>
    <router-link to="/repos" class="el-button" style="margin-top: 8px">← 返回仓库列表</router-link>
  </section>

  <div v-else-if="repo" class="container repo-page">
    <nav class="breadcrumb-row" aria-label="面包屑">
      <router-link to="/repos" class="breadcrumb-link">
        <el-icon :size="14"><Files /></el-icon>
        <span>仓库</span>
      </router-link>
      <span class="sep">/</span>
      <span class="crumb-text">{{ repo.owner_username }}</span>
      <span class="sep">/</span>
      <span class="crumb-text current">{{ repo.name }}</span>
    </nav>

    <header class="repo-header">
      <div class="repo-header-left">
        <h1 class="repo-title">
          <span class="title-owner">{{ repo.owner_username }}</span>
          <span class="title-sep">/</span>
          <span class="title-name">{{ repo.name }}</span>
        </h1>
        <span :class="['visibility-badge', repo.is_private ? 'private' : 'public']">
          <el-icon :size="11"><component :is="repo.is_private ? 'Lock' : 'Unlock'" /></el-icon>
          {{ repo.is_private ? 'Private' : 'Public' }}
        </span>
      </div>

      <div class="repo-header-actions">
        <div v-if="stats" class="stats">
          <span class="stat-item" title="分支数">
            <el-icon :size="14"><Operation /></el-icon> {{ stats.branch_count }}
          </span>
          <span class="stat-item" title="提交数">
            <el-icon :size="14"><Clock /></el-icon> {{ stats.commit_count }}
          </span>
        </div>
        <el-select v-model="branch" size="default" class="branch-select" @change="onBranchChange">
          <el-option v-for="b in branches" :key="b" :label="b" :value="b">
            <span class="mono">{{ b }}</span>
          </el-option>
        </el-select>

        <el-dropdown trigger="click">
          <button class="el-button el-button--default">
            <el-icon><Link /></el-icon>
            <span>Code</span>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="copyCloneUrl">
                <div class="clone-item">
                  <code class="mono text-ellipsis">{{ cloneUrl }}</code>
                  <el-icon :size="14" :color="copied ? 'var(--success)' : ''">
                    <component :is="copied ? 'Check' : 'DocumentCopy'" />
                  </el-icon>
                </div>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </header>

    <p v-if="repo.description" class="repo-desc">{{ repo.description }}</p>

    <el-tabs :model-value="route.name" @tab-click="(tab: any) => $router.push({ name: tab.paneName as string, query: route.query })" class="repo-tabs">
      <el-tab-pane v-for="tab in tabs" :key="tab.name" :label="tab.label" :name="tab.name" />
    </el-tabs>

    <div class="repo-content">
      <router-view v-slot="{ Component }">
        <transition name="fade-slide" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>
  </div>
</template>

<style scoped lang="scss">
.repo-page {
  padding-top: 20px;
  padding-bottom: 48px;
}

.breadcrumb-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 16px;
  font-size: 13px;
}

.breadcrumb-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--text-secondary);
  padding: 3px 8px;
  border-radius: var(--radius);
  transition: all 0.12s;

  &:hover {
    color: var(--accent);
    background: var(--gray-50);
  }
}

.sep {
  color: var(--gray-300);
}

.crumb-text {
  color: var(--text-secondary);

  &.current {
    color: var(--text);
    font-weight: 500;
  }
}

.repo-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.repo-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex-wrap: wrap;
}

.repo-title {
  font-size: 24px;
  font-weight: 600;
  margin: 0;
  letter-spacing: -0.01em;
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.title-owner {
  color: var(--text-secondary);
  font-weight: 500;
}

.title-sep {
  color: var(--gray-300);
  font-weight: 400;
}

.title-name {
  color: var(--text);
}

.visibility-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  border-radius: var(--radius-pill);
  font-size: 11px;
  font-weight: 500;
  border: 1px solid transparent;

  &.public {
    background: var(--accent-light);
    color: var(--accent);
  }

  &.private {
    background: #fef3c7;
    color: #92400e;
  }
}

.repo-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.stats {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-right: 4px;
  border-right: 1px solid var(--border);
  margin-right: 4px;
}

.stat-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.branch-select {
  width: 160px;
}

.repo-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 12px 0 0;
  line-height: 1.5;
  max-width: 800px;
}

.clone-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 320px;
}

.clone-item code {
  flex: 1;
  font-size: 12px;
  color: var(--text);
  background: var(--bg);
  padding: 2px 6px;
  border-radius: 4px;
}

.repo-tabs {
  margin-top: 20px;
}

.repo-tabs :deep(.el-tabs__active-bar) {
  transition: none !important;
}

.repo-tabs :deep(.el-tabs__item) {
  transition: none !important;
}

.repo-tabs :deep(.el-tabs__nav-wrap)::after {
  background-color: var(--border-light);
}

.repo-content {
  margin-top: 20px;
  position: relative;
}

.repo-content > * {
  min-width: 0;
}

@media (max-width: 768px) {
  .clone-item {
    width: 260px;
  }

  .branch-select {
    width: 130px;
  }
}

@media (max-width: 640px) {
  .repo-title {
    font-size: 18px;
  }
  .stats {
    display: none;
  }

  .clone-item {
    width: 220px;
  }

  .branch-select {
    width: 120px;
  }

  .repo-header-actions {
    gap: 4px;
  }

  .repo-desc {
    max-width: 100%;
  }
}
</style>
