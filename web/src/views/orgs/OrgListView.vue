<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { orgApi } from '@/api/org'
import type { Organization } from '@/api/types'
import PageHeader from '@/components/PageHeader.vue'
import EmptyState from '@/components/EmptyState.vue'
import ErrorState from '@/components/ErrorState.vue'

const orgs = ref<Organization[]>([])
const loading = ref(true)
const error = ref('')

async function loadOrgs() {
  loading.value = true
  error.value = ''
  try {
    const res = await orgApi.list()
    orgs.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadOrgs)
</script>

<template>
  <section class="container" style="padding-top: 24px; padding-bottom: 48px">
    <PageHeader title="组织">
      <template #actions>
        <router-link to="/orgs/new" class="el-button el-button--primary">
          <el-icon><Plus /></el-icon> 新建组织
        </router-link>
      </template>
    </PageHeader>

    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
    </div>
    <ErrorState v-else-if="error" :message="error" @retry="loadOrgs" />
    <EmptyState
      v-else-if="orgs.length === 0"
      icon="OfficeBuilding"
      title="还没有组织"
      message="创建一个组织来管理多个仓库和成员"
    >
      <template #actions>
        <router-link to="/orgs/new" class="el-button el-button--primary">创建组织</router-link>
      </template>
    </EmptyState>
    <div v-else class="repo-grid">
      <router-link
        v-for="org in orgs"
        :key="org.id"
        :to="`/orgs/${org.name}`"
        class="repo-card card"
      >
        <div class="repo-card-header">
          <div class="repo-title">{{ org.display_name || org.name }}</div>
          <div class="repo-desc">{{ org.description || '暂无描述' }}</div>
        </div>
      </router-link>
    </div>
  </section>
</template>

<style scoped lang="scss">
.repo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.repo-card {
  display: flex;
  flex-direction: column;
  padding: 16px;
  transition: box-shadow 0.15s;
}

.repo-card:hover {
  box-shadow: var(--shadow);
}

.repo-title {
  font-size: 15px;
  font-weight: 600;
}

.repo-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 6px;
}
</style>
