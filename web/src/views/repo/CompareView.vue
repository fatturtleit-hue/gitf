<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { repoApi } from '@/api/repo'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import type { FileDiff } from '@/api/types'

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const base = computed(() => (route.params.base as string) || '')
const head = computed(() => (route.params.head as string) || '')

const diffs = ref<FileDiff[]>([])
const loading = ref(true)
const error = ref('')

async function loadCompare() {
  loading.value = true
  error.value = ''
  try {
    const res = await repoApi.compare(owner.value, repoName.value, base.value, head.value)
    diffs.value = res.diffs
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

watch(() => [base.value, head.value], loadCompare, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadCompare" />

  <EmptyState v-else-if="diffs.length === 0" icon="Document" title="没有差异" message="两个引用之间没有可比较的差异" />

  <div v-else>
    <div v-for="diff in diffs" :key="diff.path" class="card diff-card">
      <div class="diff-header mono">{{ diff.status }}: {{ diff.path }}</div>
      <pre class="diff-body"><code>{{ diff.patch }}</code></pre>
    </div>
  </div>
</template>

<style scoped lang="scss">
.diff-card {
  margin-bottom: 16px;
  overflow: hidden;
}

.diff-header {
  padding: 10px 16px;
  background: var(--gray-50);
  border-bottom: 1px solid var(--border);
  font-size: 13px;
  font-weight: 500;
}

.diff-body {
  margin: 0;
  padding: 12px 16px;
  overflow-x: auto;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.6;
  background: var(--surface);
}
</style>
