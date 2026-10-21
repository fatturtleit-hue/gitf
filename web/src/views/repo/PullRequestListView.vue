<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { prApi, type PullRequest } from '@/api/pr'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import { formatDate } from '@/utils/format'

const route = useRoute()
const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const branch = computed(() => (route.query.branch as string) || '')

const prs = ref<PullRequest[]>([])
const loading = ref(true)
const error = ref('')

async function loadPRs() {
  loading.value = true
  error.value = ''
  try {
    const res = await prApi.list(owner.value, repoName.value)
    prs.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadPRs)
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadPRs" />

  <EmptyState v-else-if="prs.length === 0" icon="Connection" title="暂无 Pull Request" message="还没有创建 PR">
    <template #actions>
      <router-link :to="{ name: 'repo-pull-new', query: { branch } }" class="el-button el-button--primary">新建 PR</router-link>
    </template>
  </EmptyState>

  <div v-else class="file-list">
    <router-link
      v-for="pr in prs"
      :key="pr.id"
      :to="{ name: 'repo-pull-detail', params: { number: pr.number }, query: { branch } }"
      class="file-row"
      style="flex-direction: column; align-items: flex-start; gap: 4px"
    >
      <div class="row gap-2">
        <span :class="['badge', pr.status === 'open' ? 'badge-success' : pr.status === 'merged' ? 'badge-accent' : 'badge-default']">{{ pr.status }}</span>
        <strong>{{ pr.title }}</strong>
        <span class="muted">#{{ pr.number }}</span>
      </div>
      <div class="muted" style="font-size: 12px">
        {{ pr.head_branch }} → {{ pr.base_branch }} · {{ pr.author_username }} · {{ formatDate(pr.created_at) }}
      </div>
    </router-link>
  </div>
</template>
