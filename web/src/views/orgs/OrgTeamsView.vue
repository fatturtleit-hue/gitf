<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { orgApi } from '@/api/org'
import type { Team } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'

const route = useRoute()
const orgName = computed(() => (route.params.org as string) || '')

const teams = ref<Team[]>([])
const loading = ref(true)
const error = ref('')

async function loadTeams() {
  loading.value = true
  error.value = ''
  try {
    const res = await orgApi.listTeams(orgName.value)
    teams.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

watch(orgName, loadTeams, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadTeams" />

  <EmptyState v-else-if="teams.length === 0" icon="UserFilled" title="暂无团队" message="这个组织还没有任何团队" />

  <div v-else class="file-list">
    <div v-for="team in teams" :key="team.id" class="file-row">
      <el-icon :size="18"><UserFilled /></el-icon>
      <div class="flex-1">
        <div>{{ team.name }}</div>
        <div v-if="team.description" class="muted" style="font-size: 12px">{{ team.description }}</div>
      </div>
    </div>
  </div>
</template>
