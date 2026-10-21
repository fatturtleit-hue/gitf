<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { orgApi } from '@/api/org'
import type { OrgMember } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'

const route = useRoute()
const orgName = computed(() => (route.params.org as string) || '')

const members = ref<OrgMember[]>([])
const loading = ref(true)
const error = ref('')

async function loadMembers() {
  loading.value = true
  error.value = ''
  try {
    const res = await orgApi.listMembers(orgName.value)
    members.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

watch(orgName, loadMembers, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadMembers" />

  <EmptyState v-else-if="members.length === 0" icon="User" title="暂无成员" message="这个组织还没有任何成员" />

  <div v-else class="file-list">
    <div v-for="member in members" :key="member.id" class="file-row">
      <el-icon :size="18"><User /></el-icon>
      <span class="flex-1">{{ member.username }}</span>
      <span class="badge badge-default">{{ member.role }}</span>
    </div>
  </div>
</template>
