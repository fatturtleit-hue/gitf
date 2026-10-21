<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { issueApi } from '@/api/issue'
import type { Issue } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const branch = computed(() => (route.query.branch as string) || 'main')

const issues = ref<Issue[]>([])
const loading = ref(true)
const error = ref('')

async function loadIssues() {
  loading.value = true
  error.value = ''
  try {
    const res = await issueApi.list(owner.value, repoName.value, { status: 'open' })
    issues.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

watch(() => [owner.value, repoName.value], loadIssues, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadIssues" />

  <EmptyState
    v-else-if="issues.length === 0"
    icon="Tickets"
    title="暂无 Issues"
    message="还没有人创建 issue"
  >
    <template #actions>
      <router-link
        :to="{ name: 'repo-issue-new', query: { branch } }"
        class="el-button el-button--primary"
      >
        新建 Issue
      </router-link>
    </template>
  </EmptyState>

  <div v-else class="file-list">
    <router-link
      v-for="issue in issues"
      :key="issue.id"
      class="file-row"
      :to="{ name: 'repo-issue-detail', params: { number: issue.number }, query: { branch } }"
    >
      <span :class="['status-dot', issue.status === 'open' ? 'status-open' : 'status-closed']"></span>
      <span class="flex-1 text-ellipsis">{{ issue.title }}</span>
      <span class="muted" style="font-size: 12px">#{{ issue.number }} by {{ issue.author_username }}</span>
    </router-link>
  </div>
</template>
