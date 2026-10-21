<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { milestoneApi } from '@/api/milestone'
import type { Milestone } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'

const route = useRoute()
const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')

const milestones = ref<Milestone[]>([])
const loading = ref(true)
const error = ref('')
const showAdd = ref(false)
const newMilestone = reactive({ title: '', description: '', due_date: '' })
const adding = ref(false)

async function loadMilestones() {
  loading.value = true
  error.value = ''
  try {
    const res = await milestoneApi.list(owner.value, repoName.value)
    milestones.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function addMilestone() {
  if (!newMilestone.title) return
  adding.value = true
  try {
    await milestoneApi.create(owner.value, repoName.value, {
      title: newMilestone.title,
      description: newMilestone.description || undefined,
      due_date: newMilestone.due_date || undefined
    })
    ElMessage.success('里程碑已创建')
    showAdd.value = false
    newMilestone.title = ''
    newMilestone.description = ''
    newMilestone.due_date = ''
    await loadMilestones()
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    adding.value = false
  }
}

async function closeMilestone(title: string) {
  try {
    await ElMessageBox.confirm(`确定关闭里程碑 "${title}" 吗？`, '关闭', { type: 'warning' })
    await milestoneApi.close(owner.value, repoName.value, title)
    ElMessage.success('已关闭')
    await loadMilestones()
  } catch { /* cancelled */ }
}

onMounted(loadMilestones)
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadMilestones" />

  <EmptyState v-else-if="milestones.length === 0" icon="Flag" title="暂无里程碑" message="创建里程碑来规划版本">
    <template #actions>
      <el-button type="primary" @click="showAdd = true">新建里程碑</el-button>
    </template>
  </EmptyState>

  <div v-else>
    <div class="spread" style="margin-bottom: 12px">
      <span style="font-weight: 500">{{ milestones.length }} 个里程碑</span>
      <el-button type="primary" size="small" @click="showAdd = true">新建</el-button>
    </div>
    <div class="file-list">
      <div v-for="m in milestones" :key="m.id" class="file-row" style="flex-direction: column; align-items: flex-start; gap: 4px">
        <div class="row gap-2">
          <span :class="['badge', m.status === 'open' ? 'badge-success' : 'badge-default']">{{ m.status }}</span>
          <strong>{{ m.title }}</strong>
        </div>
        <div v-if="m.description" class="muted" style="font-size: 12px">{{ m.description }}</div>
        <div class="muted" style="font-size: 12px">
          {{ formatDate(m.created_at) }}
          <span v-if="m.due_date"> · 截止 {{ formatDate(m.due_date) }}</span>
        </div>
        <el-button v-if="m.status === 'open'" type="warning" size="small" text @click="closeMilestone(m.title)">关闭</el-button>
      </div>
    </div>
  </div>

  <el-dialog v-model="showAdd" title="新建里程碑" width="480px">
    <el-form label-position="top" @submit.prevent="addMilestone">
      <el-form-item label="标题" required>
        <el-input v-model="newMilestone.title" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="newMilestone.description" type="textarea" :rows="2" />
      </el-form-item>
      <el-form-item>
        <div style="display:flex; justify-content: flex-end; gap: 8px; width: 100%">
          <el-button @click="showAdd = false">取消</el-button>
          <el-button type="primary" native-type="submit" :loading="adding">创建</el-button>
        </div>
      </el-form-item>
    </el-form>
  </el-dialog>
</template>
