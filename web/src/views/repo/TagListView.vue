<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { repoApi } from '@/api/repo'
import type { TagInfo } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')

const tags = ref<TagInfo[]>([])
const loading = ref(true)
const error = ref('')

  async function loadTags() {
    loading.value = true
    error.value = ''
    try {
      const res = await repoApi.tags(owner.value, repoName.value)
      tags.value = res.tags || []
    } catch (err: any) {
      error.value = err.message || '加载失败'
    } finally {
      loading.value = false
    }
  }

watch(() => [owner.value, repoName.value], loadTags, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadTags" />

  <EmptyState v-else-if="tags.length === 0" icon="CollectionTag" title="暂无标签" message="这个仓库还没有任何标签" />

  <div v-else class="file-list">
    <div v-for="tag in tags" :key="tag.name" class="file-row">
      <el-icon :size="16" color="var(--accent)"><CollectionTag /></el-icon>
      <span class="flex-1">{{ tag.name }}</span>
      <code class="mono muted">{{ tag.hash.substring(0, 8) }}</code>
    </div>
  </div>
</template>
