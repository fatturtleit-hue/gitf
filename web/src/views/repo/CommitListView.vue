<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { repoApi } from '@/api/repo'
import type { CommitInfo } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import { formatDate } from '@/utils/format'

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const branch = computed(() => (route.query.branch as string) || 'main')

const commits = ref<CommitInfo[]>([])
const loading = ref(true)
const error = ref('')
const copied = ref('')

async function loadCommits() {
  loading.value = true
  error.value = ''
  try {
    const res = await repoApi.commits(owner.value, repoName.value, branch.value, 100)
    commits.value = res.commits
  } catch (err: any) {
    error.value = err.message || '加载提交记录失败'
  } finally {
    loading.value = false
  }
}

function copyHash(hash: string) {
  navigator.clipboard.writeText(hash).then(() => {
    copied.value = hash
    setTimeout(() => {
      if (copied.value === hash) copied.value = ''
    }, 1200)
  })
}

watch(() => branch.value, loadCommits, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadCommits" />

  <EmptyState
    v-else-if="commits.length === 0"
    icon="Clock"
    title="暂无提交"
    message="这个仓库还没有任何提交记录"
  />

  <div v-else class="file-list" style="margin-top: 12px">
    <div
      v-for="commit in commits"
      :key="commit.hash"
      class="file-row"
      style="flex-direction: column; align-items: flex-start; gap: 6px"
    >
      <router-link
        :to="{ name: 'repo-commit', params: { hash: commit.hash }, query: { branch } }"
        style="font-weight: 500; color: var(--text); text-decoration: none"
        class="commit-msg text-ellipsis"
      >
        {{ commit.message }}
      </router-link>

      <div class="row gap-3" style="font-size: 12px; color: var(--gray-400)">
        <code class="mono muted" style="cursor: pointer" @click="copyHash(commit.hash)">
          {{ commit.short_hash }} {{ copied === commit.hash ? '✓' : '' }}
        </code>
        <span>{{ commit.author }}</span>
        <span>{{ formatDate(commit.date) }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.commit-msg {
  width: 100%;
}
</style>
