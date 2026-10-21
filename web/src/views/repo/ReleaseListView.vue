<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { releaseApi } from '@/api/release'
import type { Release } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import { formatDate } from '@/utils/format'

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')

const releases = ref<Release[]>([])
const loading = ref(true)
const error = ref('')

async function loadReleases() {
  loading.value = true
  error.value = ''
  try {
    const res = await releaseApi.list(owner.value, repoName.value)
    releases.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

watch(() => [owner.value, repoName.value], loadReleases, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadReleases" />

  <EmptyState
    v-else-if="releases.length === 0"
    icon="Present"
    title="暂无发布"
    message="这个仓库还没有任何 Release"
  />

  <div v-else class="file-list">
    <div v-for="release in releases" :key="release.id" class="file-row" style="flex-direction: column; align-items: flex-start; gap: 6px">
      <div class="row gap-2">
        <strong>{{ release.name || release.tag_name }}</strong>
        <span v-if="release.is_draft" class="badge badge-warn">Draft</span>
        <span v-if="release.is_prerelease" class="badge badge-warn">Pre-release</span>
      </div>
      <div class="muted" style="font-size: 12px">{{ release.tag_name }} · {{ release.author_username }} · {{ formatDate(release.created_at) }}</div>
    </div>
  </div>
</template>
