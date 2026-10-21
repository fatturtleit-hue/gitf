<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { orgApi } from '@/api/org'
import { repoApi } from '@/api/repo'
import type { Organization, Repository } from '@/api/types'
import PageHeader from '@/components/PageHeader.vue'
import EmptyState from '@/components/EmptyState.vue'
import ErrorState from '@/components/ErrorState.vue'

const route = useRoute()
const orgName = computed(() => (route.params.org as string) || '')

const org = ref<Organization | null>(null)
const repos = ref<Repository[]>([])
const loading = ref(true)
const error = ref('')

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [o, r] = await Promise.all([
      orgApi.get(orgName.value),
      repoApi.list()
    ])
    org.value = o
    repos.value = r.items.filter(repo => repo.owner_username === orgName.value)
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

watch(orgName, loadAll, { immediate: true })
</script>

<template>
  <section class="container" style="padding-top: 24px; padding-bottom: 48px">
    <PageHeader :title="org ? (org.display_name || org.name) : '组织仓库'">
      <template #actions>
        <router-link :to="`/repos/new?org=${orgName}`" class="el-button el-button--primary">
          <el-icon><Plus /></el-icon> 新建仓库
        </router-link>
      </template>
    </PageHeader>

    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
    </div>
    <ErrorState v-else-if="error" :message="error" @retry="loadAll" />
    <EmptyState
      v-else-if="repos.length === 0"
      icon="Folder"
      title="还没有仓库"
      message="该组织下暂无仓库"
    >
      <template #actions>
        <router-link :to="`/repos/new?org=${orgName}`" class="el-button el-button--primary">创建仓库</router-link>
      </template>
    </EmptyState>
    <div v-else class="repo-grid">
      <router-link
        v-for="repo in repos"
        :key="repo.id"
        :to="`/${repo.owner_username}/${repo.name}`"
        class="repo-card card"
      >
        <div class="repo-card-header">
          <div class="repo-title">{{ repo.name }}</div>
          <div class="repo-desc">{{ repo.description || '暂无描述' }}</div>
        </div>
        <div class="repo-meta">
          <span><el-icon :size="12"><Collection /></el-icon> {{ repo.default_branch }}</span>
          <span><el-icon :size="12"><Clock /></el-icon> {{ repo.updated_at }}</span>
        </div>
      </router-link>
    </div>
  </section>
</template>

<style scoped lang="scss">
.repo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.repo-card {
  display: flex;
  flex-direction: column;
  padding: 16px;
  transition: box-shadow 0.15s;
}

.repo-card:hover {
  box-shadow: var(--shadow);
}

.repo-title {
  font-size: 15px;
  font-weight: 600;
}

.repo-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 6px;
}

.repo-meta {
  display: flex;
  gap: 12px;
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-secondary);
}
</style>
