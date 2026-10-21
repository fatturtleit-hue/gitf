<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { webhookApi, type Webhook } from '@/api/webhook'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'

const route = useRoute()
const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')

const hooks = ref<Webhook[]>([])
const loading = ref(true)
const error = ref('')
const showAdd = ref(false)
const newHook = reactive({ url: '', secret: '' })
const adding = ref(false)

async function loadHooks() {
  loading.value = true
  error.value = ''
  try {
    const res = await webhookApi.list(owner.value, repoName.value)
    hooks.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function addHook() {
  if (!newHook.url) return
  adding.value = true
  try {
    await webhookApi.create(owner.value, repoName.value, {
      url: newHook.url,
      events: ['*'],
      secret: newHook.secret || undefined
    })
    ElMessage.success('Webhook 已创建')
    showAdd.value = false
    newHook.url = ''
    newHook.secret = ''
    await loadHooks()
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    adding.value = false
  }
}

async function deleteHook(hookId: number) {
  try {
    await ElMessageBox.confirm('确定删除此 Webhook 吗？', '删除', { type: 'warning' })
    await webhookApi.del(owner.value, repoName.value, hookId)
    ElMessage.success('已删除')
    await loadHooks()
  } catch { /* cancelled */ }
}

onMounted(loadHooks)
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadHooks" />

  <EmptyState v-else-if="hooks.length === 0" icon="Link" title="暂无 Webhook" message="创建 Webhook 触发外部服务">
    <template #actions>
      <el-button type="primary" @click="showAdd = true">新建 Webhook</el-button>
    </template>
  </EmptyState>

  <div v-else>
    <div class="spread" style="margin-bottom: 12px">
      <span style="font-weight: 500">{{ hooks.length }} 个 Webhook</span>
      <el-button type="primary" size="small" @click="showAdd = true">新建</el-button>
    </div>
    <div class="file-list">
      <div v-for="h in hooks" :key="h.id" class="file-row" style="flex-direction: column; align-items: flex-start; gap: 4px">
        <div class="row gap-2">
          <span :class="['badge', h.active ? 'badge-success' : 'badge-default']">{{ h.active ? 'Active' : 'Inactive' }}</span>
          <strong class="mono text-ellipsis" style="max-width: 400px">{{ h.url }}</strong>
        </div>
        <div class="muted" style="font-size: 12px">{{ h.content_type }} · {{ formatDate(h.created_at) }}</div>
        <el-button type="danger" size="small" text @click="deleteHook(h.id)">删除</el-button>
      </div>
    </div>
  </div>

  <el-dialog v-model="showAdd" title="新建 Webhook" width="480px">
    <el-form label-position="top" @submit.prevent="addHook">
      <el-form-item label="URL" required>
        <el-input v-model="newHook.url" placeholder="https://example.com/webhook" />
      </el-form-item>
      <el-form-item label="Secret (可选)">
        <el-input v-model="newHook.secret" />
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
