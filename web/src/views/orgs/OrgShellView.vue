<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { orgApi } from '@/api/org'
import type { Organization } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'

const route = useRoute()
const router = useRouter()

const orgName = computed(() => (route.params.org as string) || '')

const org = ref<Organization | null>(null)
const loading = ref(true)
const error = ref('')

const tabs = [
  { name: 'org-members', label: '成员' },
  { name: 'org-teams', label: '团队' },
  { name: 'org-repos', label: '仓库' },
  { name: 'org-settings', label: '设置' }
]

async function loadOrg() {
  loading.value = true
  error.value = ''
  try {
    org.value = await orgApi.get(orgName.value)
  } catch (err: any) {
    error.value = err.message || '组织不存在'
  } finally {
    loading.value = false
  }
}

watch(orgName, loadOrg, { immediate: true })
</script>

<template>
  <div v-if="loading" class="container loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadOrg" />

  <div v-else-if="org" class="container org-page">
    <header class="org-header">
      <h1>{{ org.display_name || org.name }}</h1>
      <p v-if="org.description" class="subtle">{{ org.description }}</p>
    </header>

    <el-tabs :model-value="route.name" @tab-click="(tab: any) => router.push({ name: tab.paneName as string })">
      <el-tab-pane v-for="tab in tabs" :key="tab.name" :label="tab.label" :name="tab.name" />
    </el-tabs>

    <router-view v-slot="{ Component }">
      <transition name="fade-slide" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </div>
</template>

<style scoped lang="scss">
.org-page {
  padding-top: 24px;
  padding-bottom: 48px;
  position: relative;
}

:deep(.el-tabs__active-bar),
:deep(.el-tabs__item) {
  transition: none !important;
}

.org-page > router-view,
.org-page > router-view > * {
  min-width: 0;
}

.org-header {
  margin-bottom: 16px;
}

.org-header p {
  margin-top: 6px;
}
</style>
