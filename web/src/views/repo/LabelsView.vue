<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { labelApi } from '@/api/label'
import type { Label } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import { ElMessage } from 'element-plus'

const route = useRoute()
const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')

const labels = ref<Label[]>([])
const loading = ref(true)
const error = ref('')
const showAdd = ref(false)
const newLabel = reactive({ name: '', color: '' })
const adding = ref(false)

async function loadLabels() {
  loading.value = true
  error.value = ''
  try {
    const res = await labelApi.list(owner.value, repoName.value)
    labels.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function addLabel() {
  if (!newLabel.name) return
  adding.value = true
  try {
    await labelApi.create(owner.value, repoName.value, { name: newLabel.name, color: newLabel.color || undefined })
    ElMessage.success('标签已创建')
    showAdd.value = false
    newLabel.name = ''
    newLabel.color = ''
    await loadLabels()
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    adding.value = false
  }
}

async function deleteLabel(labelId: number) {
  try {
    await labelApi.del(owner.value, repoName.value, labelId)
    ElMessage.success('已删除')
    await loadLabels()
  } catch (err: any) {
    ElMessage.error(err.message || '删除失败')
  }
}

const presetColors = ['#6366f1', '#22c55e', '#f59e0b', '#ef4444', '#3b82f6', '#8b5cf6', '#ec4899']

onMounted(loadLabels)
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadLabels" />

  <EmptyState v-else-if="labels.length === 0" icon="PriceTag" title="暂无标签" message="创建标签来分类 issue">
    <template #actions>
      <el-button type="primary" @click="showAdd = true">新建标签</el-button>
    </template>
  </EmptyState>

  <div v-else>
    <div class="spread" style="margin-bottom: 12px">
      <span style="font-weight: 500">{{ labels.length }} 个标签</span>
      <el-button type="primary" size="small" @click="showAdd = true">新建</el-button>
    </div>
    <div class="file-list">
      <div v-for="l in labels" :key="l.id" class="file-row">
        <span class="label-dot" :style="{ background: '#' + (l.color || '6366f1') }"></span>
        <span class="flex-1">{{ l.name }}</span>
        <code class="mono muted">{{ l.color }}</code>
        <el-button type="danger" size="small" text @click="deleteLabel(l.id)">删除</el-button>
      </div>
    </div>
  </div>

  <el-dialog v-model="showAdd" title="新建标签" width="400px">
    <el-form label-position="top" @submit.prevent="addLabel">
      <el-form-item label="名称" required>
        <el-input v-model="newLabel.name" />
      </el-form-item>
      <el-form-item label="颜色">
        <el-input v-model="newLabel.color" placeholder="6366f1" style="margin-bottom: 8px" />
        <div class="row gap-2">
          <button v-for="c in presetColors" :key="c" class="color-btn" :style="{ background: c }" @click="newLabel.color = c.slice(1)" />
        </div>
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

<style scoped lang="scss">
.label-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.color-btn {
  width: 24px;
  height: 24px;
  border: 2px solid transparent;
  border-radius: 4px;
  cursor: pointer;
  padding: 0;
}

.color-btn:hover {
  border-color: var(--border);
}
</style>
