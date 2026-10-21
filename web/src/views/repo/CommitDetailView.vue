<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { repoApi } from '@/api/repo'
import type { CommitInfo } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import { formatDate } from '@/utils/format'

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const hash = computed(() => (route.params.hash as string) || '')

const commit = ref<CommitInfo | null>(null)
const loading = ref(true)
const error = ref('')

async function loadCommit() {
  loading.value = true
  error.value = ''
  try {
    commit.value = await repoApi.getCommit(owner.value, repoName.value, hash.value)
  } catch (err: any) {
    error.value = err.message || '加载提交详情失败'
  } finally {
    loading.value = false
  }
}

watch(hash, loadCommit, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadCommit" />

  <div v-else-if="commit" class="card commit-detail">
    <div class="commit-header">
      <h3>{{ commit.message }}</h3>
      <div class="commit-meta">
        <span><strong>{{ commit.author }}</strong> {{ formatDate(commit.date) }}</span>
        <span class="mono">{{ commit.hash }}</span>
      </div>
    </div>

    <div v-if="commit.parents.length" class="commit-parents">
      <strong>父提交：</strong>
      <router-link
        v-for="parent in commit.parents"
        :key="parent"
        :to="{ name: 'repo-commit', params: { hash: parent } }"
        class="mono"
      >
        {{ parent.substring(0, 8) }}
      </router-link>
    </div>
  </div>
</template>

<style scoped lang="scss">
.commit-detail {
  padding: 20px;
}

.commit-header {
  border-bottom: 1px solid var(--border);
  padding-bottom: 16px;
  margin-bottom: 16px;
}

.commit-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 13px;
  color: var(--text-secondary);
}

.commit-parents {
  font-size: 13px;
  display: flex;
  gap: 8px;
  align-items: center;
}

.commit-parents a {
  color: var(--accent);
}
</style>
