<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { repoApi } from '@/api/repo'
import type { Repository } from '@/api/types'
import PageHeader from '@/components/PageHeader.vue'
import EmptyState from '@/components/EmptyState.vue'
import ErrorState from '@/components/ErrorState.vue'
import { formatDate } from '@/utils/format'

const repos = ref<Repository[]>([])
const pagination = ref<{
  page: number
  per_page: number
  total: number
  total_pages: number
} | null>(null)
const searchQuery = ref('')
const loading = ref(true)
const error = ref('')

const filteredRepos = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return repos.value
  return repos.value.filter(
    (r: Repository) =>
      r.name.toLowerCase().includes(q) ||
      (r.description ?? '').toLowerCase().includes(q) ||
      r.owner_username.toLowerCase().includes(q)
  )
})

async function loadRepos(page = 1) {
  loading.value = true
  error.value = ''
  try {
    const res = await repoApi.list(undefined, page, 12)
    repos.value = res.items
    pagination.value = res.pagination
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function onSearch(val: string) {
  searchQuery.value = val
}

function onPageChange(page: number) {
  loadRepos(page)
}

async function confirmDelete(repo: Repository) {
  try {
    await ElMessageBox.confirm(
      `确定要删除仓库 "${repo.owner_username}/${repo.name}" 吗？此操作不可撤销。`,
      '删除仓库',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await repoApi.del(repo.owner_username, repo.name)
    ElMessage.success('仓库已删除')
    loadRepos(pagination.value?.page ?? 1)
  } catch (err: any) {
    if (err !== 'cancel') {
      error.value = err.message || '删除失败'
    }
  }
}

onMounted(() => loadRepos())
</script>

<template>
  <section class="container page">
    <PageHeader title="仓库">
      <template #actions>
        <router-link to="/repos/new" class="new-btn">
          <el-icon><Plus /></el-icon>
          <span>新建仓库</span>
        </router-link>
      </template>
    </PageHeader>

    <div class="search-wrap">
      <el-input
        v-model="searchQuery"
        placeholder="按名称、描述或所有者搜索…"
        clearable
        size="default"
        @input="onSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="28"><Loading /></el-icon>
    </div>
    <ErrorState
      v-else-if="error"
      :message="error"
      @retry="loadRepos(1)"
    />
    <EmptyState
      v-else-if="filteredRepos.length === 0"
      icon="FolderOpened"
      :title="searchQuery ? '没有匹配的结果' : '还没有仓库'"
      :message="searchQuery ? '尝试其他搜索词' : '创建你的第一个仓库'"
    >
      <template #actions>
        <router-link to="/repos/new" class="el-button el-button--primary">创建仓库</router-link>
      </template>
    </EmptyState>
    <template v-else>
      <div class="repo-grid">
        <div v-for="repo in filteredRepos" :key="repo.id" class="repo-card card">
          <div class="repo-card-header">
            <div class="repo-icon">
              <el-icon :size="18"><Folder /></el-icon>
            </div>
            <div class="repo-meta">
              <router-link :to="`/${repo.owner_username}/${repo.name}`" class="repo-link text-ellipsis">
                <span class="subtle">{{ repo.owner_username }} /</span>{{ repo.name }}
              </router-link>
              <p class="repo-desc text-clamp-2">{{ repo.description || '暂无描述' }}</p>
            </div>
          </div>

          <div class="repo-card-meta">
            <span class="meta-item">
              <el-icon :size="12"><Collection /></el-icon> {{ repo.default_branch }}
            </span>
            <span class="meta-item">
              <el-icon :size="12"><Clock /></el-icon> {{ formatDate(repo.updated_at) }}
            </span>
          </div>

          <div class="card-actions">
            <span :class="['visibility-badge', repo.is_private ? 'private' : 'public']">
              <el-icon :size="12"><component :is="repo.is_private ? 'Lock' : 'Unlock'" /></el-icon>
              {{ repo.is_private ? 'Private' : 'Public' }}
            </span>
            <span class="spacer"></span>
            <el-dropdown trigger="click">
              <button class="icon-btn" aria-label="更多操作">
                <el-icon><More /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="confirmDelete(repo)">
                    <el-icon><Delete /></el-icon> 删除仓库
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>

      <div v-if="pagination && pagination.total_pages > 1" class="pagination-wrap">
        <el-pagination
          :current-page="pagination.page"
          :page-size="pagination.per_page"
          :total="pagination.total"
          :page-sizes="[12, 24, 48]"
          layout="total, sizes, prev, pager, next"
          background
          small
          @current-change="onPageChange"
        />
      </div>
    </template>
  </section>
</template>

<style scoped lang="scss">
.page {
  padding-top: 24px;
  padding-bottom: 48px;
}

.new-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.search-wrap {
  margin-bottom: 18px;
  max-width: 480px;
}

.repo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 14px;
}

.repo-card {
  display: flex;
  flex-direction: column;
  padding: 18px;
  transition: box-shadow 0.15s, border-color 0.15s, transform 0.15s;
}

.repo-card:hover {
  border-color: var(--gray-300);
  box-shadow: var(--shadow);
  transform: translateY(-1px);
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

.repo-link {
  display: block;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
  transition: color 0.15s;
}

.repo-link:hover {
  color: var(--accent);
}

.repo-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 4px 0 0;
  min-height: 36px;
  line-height: 1.5;
}

.repo-card-meta {
  display: flex;
  gap: 16px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--border-light);
  font-size: 12px;
  color: var(--text-muted);
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.card-actions {
  display: flex;
  align-items: center;
  margin-top: 12px;
  gap: 6px;
}

.visibility-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  border-radius: var(--radius-pill);
  font-size: 11px;
  font-weight: 500;
  border: 1px solid var(--border);

  &.public {
    background: var(--accent-light);
    color: var(--accent);
    border-color: transparent;
  }

  &.private {
    background: #fef3c7;
    color: #92400e;
    border-color: transparent;
  }
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s;
}

.icon-btn:hover {
  background: var(--gray-100);
  color: var(--text);
}

.spacer {
  flex: 1;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

@media (max-width: 640px) {
  .pagination-wrap {
    justify-content: center;
  }

  .search-wrap {
    max-width: 100%;
  }
}
</style>
