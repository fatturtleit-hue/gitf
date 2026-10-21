<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { repoApi } from '@/api/repo'
import { userApi } from '@/api/user'
import type { User } from '@/api/types'
import { useAuthStore } from '@/stores/auth'
import PageHeader from '@/components/PageHeader.vue'
import ErrorState from '@/components/ErrorState.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatSize } from '@/utils/format'

const auth = useAuthStore()

const stats = ref<{ disk_total: number; disk_used: number; disk_free: number; repo_count: number; user_count: number } | null>(null)
const users = ref<User[]>([])
const loading = ref(true)
const error = ref('')

const showCreate = ref(false)
const newUser = ref({ username: '', email: '', password: '', is_admin: false })
const creating = ref(false)

const editingUser = ref<User | null>(null)
const editingEmail = ref('')
const editingIsAdmin = ref(false)
const editing = ref(false)

const showEditDialog = computed({
  get: () => !!editingUser.value,
  set: (v) => { if (!v) editingUser.value = null }
})

const diskPercent = computed(() => {
  if (!stats.value || !stats.value.disk_total) return 0
  return Math.round((stats.value.disk_used / stats.value.disk_total) * 100)
})

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const s = await repoApi.adminStats()
    stats.value = s
    if (auth.isAdmin) {
      try { const u = await userApi.list(); users.value = u.items } catch { users.value = [] }
    }
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function createUser() {
  creating.value = true
  try {
    await userApi.create(newUser.value)
    ElMessage.success('用户创建成功')
    showCreate.value = false
    newUser.value = { username: '', email: '', password: '', is_admin: false }
    await loadData()
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    creating.value = false
  }
}

function startEdit(u: User) {
  editingUser.value = u
  editingEmail.value = u.email
  editingIsAdmin.value = !!u.is_site_admin
}

async function saveEdit() {
  if (!editingUser.value) return
  editing.value = true
  try {
    const body: Record<string, any> = { is_admin: editingIsAdmin.value }
    const email = editingEmail.value.trim()
    if (email && email !== editingUser.value.email) {
      body.email = email
    }
    await userApi.update(editingUser.value.id, body)
    ElMessage.success('已更新')
    editingUser.value = null
    await loadData()
  } catch (err: any) {
    ElMessage.error(err.message || '更新失败')
  } finally {
    editing.value = false
  }
}

async function deleteUser(user: User) {
  try {
    await ElMessageBox.confirm(`确定删除用户 ${user.username} 吗？`, '删除用户', { type: 'warning' })
    await userApi.del(user.id)
    ElMessage.success('用户已删除')
    await loadData()
  } catch { /* cancelled */ }
}

onMounted(loadData)
</script>

<template>
  <section class="container" style="padding-top: 24px; padding-bottom: 48px">
    <PageHeader :title="auth.isAdmin ? '管理面板' : '控制面板'">
      <template v-if="auth.isAdmin" #actions>
        <el-button type="primary" @click="showCreate = true">
          <el-icon><Plus /></el-icon> 新建用户
        </el-button>
      </template>
    </PageHeader>

    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
    </div>

    <ErrorState v-else-if="error" :message="error" @retry="loadData" />

    <template v-else>
      <div class="stats-grid">
        <div class="stat-card card">
          <div class="stat-value">{{ stats?.repo_count ?? 0 }}</div>
          <div class="stat-label">仓库</div>
        </div>
        <div class="stat-card card">
          <div class="stat-value">{{ stats?.user_count ?? 0 }}</div>
          <div class="stat-label">用户</div>
        </div>
        <template v-if="auth.isAdmin">
          <div class="stat-card card">
            <div class="stat-value">{{ formatSize(stats?.disk_total ?? 0) }}</div>
            <div class="stat-label">磁盘总量</div>
          </div>
          <div class="stat-card card">
            <div class="stat-value">{{ formatSize(stats?.disk_used ?? 0) }}</div>
            <div class="stat-label">已用磁盘</div>
          </div>
          <div class="stat-card card">
            <div class="stat-value">{{ formatSize(stats?.disk_free ?? 0) }}</div>
            <div class="stat-label">剩余磁盘</div>
          </div>
        </template>
      </div>

      <div v-if="auth.isAdmin && stats && stats.disk_total" class="disk-bar card" style="margin-top: 16px; padding: 16px">
        <div class="spread" style="margin-bottom: 8px">
          <span style="font-size: 13px; font-weight: 500">磁盘使用率</span>
          <span class="mono muted" style="font-size: 12px">{{ diskPercent }}% ({{ formatSize(stats.disk_used) }} / {{ formatSize(stats.disk_total) }})</span>
        </div>
        <div class="bar-track">
          <div class="bar-fill" :style="{ width: diskPercent + '%' }"></div>
        </div>
      </div>

      <template v-if="auth.isAdmin">
        <h3 style="margin: 24px 0 12px">用户管理</h3>
        <div class="file-list">
          <div v-for="user in users" :key="user.id" class="file-row" style="flex-wrap: wrap; gap: 8px">
            <el-icon :size="18"><User /></el-icon>
            <div class="flex-1">
              <div>
                {{ user.username }}
                <span v-if="user.is_site_admin" class="badge badge-accent" style="margin-left: 8px">Admin</span>
              </div>
              <div class="muted" style="font-size: 12px">{{ user.email }}</div>
            </div>
            <el-button size="small" text @click="startEdit(user)">编辑</el-button>
            <el-button type="danger" size="small" text @click="deleteUser(user)">删除</el-button>
          </div>
        </div>
      </template>

      <div v-else class="card" style="padding: 32px; text-align: center; margin-top: 24px">
        <el-icon :size="40" color="var(--gray-300)"><Lock /></el-icon>
        <p style="margin: 12px 0 4px; color: var(--text-secondary)">仅管理员可管理用户和查看磁盘信息</p>
        <p class="muted" style="font-size: 12px">当前账号：{{ auth.user?.username }}</p>
      </div>
    </template>

    <el-dialog v-model="showCreate" title="新建用户" width="480px">
      <el-form label-position="top" @submit.prevent="createUser">
        <el-form-item label="用户名"><el-input v-model="newUser.username" /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="newUser.email" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="newUser.password" type="password" /></el-form-item>
        <el-form-item><el-checkbox v-model="newUser.is_admin">站点管理员</el-checkbox></el-form-item>
        <el-form-item>
          <div style="display:flex; justify-content: flex-end; gap: 8px; width: 100%">
            <el-button @click="showCreate = false">取消</el-button>
            <el-button type="primary" native-type="submit" :loading="creating">创建</el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-dialog>

    <el-dialog v-model="showEditDialog" title="编辑用户" width="480px">
      <el-form v-if="editingUser" label-position="top" @submit.prevent="saveEdit">
        <div class="muted" style="margin-bottom: 16px">用户名: <strong>{{ editingUser.username }}</strong></div>
        <el-form-item label="邮箱">
          <el-input v-model="editingEmail" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="editingIsAdmin">站点管理员</el-checkbox>
        </el-form-item>
        <el-form-item>
          <div style="display:flex; justify-content: flex-end; gap: 8px; width: 100%">
            <el-button @click="editingUser = null">取消</el-button>
            <el-button type="primary" native-type="submit" :loading="editing">保存</el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-dialog>
  </section>
</template>

<style scoped lang="scss">
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
}
.stat-card { padding: 20px; text-align: center; }
.stat-value { font-size: 28px; font-weight: 700; color: var(--accent); }
.stat-label { font-size: 13px; color: var(--text-secondary); margin-top: 4px; }
.bar-track { height: 8px; background: var(--gray-100); border-radius: 4px; overflow: hidden; }
.bar-fill { height: 100%; background: var(--accent); border-radius: 4px; transition: width 0.3s; }

@media (max-width: 480px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
  }

  .stat-card {
    padding: 14px;
  }

  .stat-value {
    font-size: 22px;
  }
}
</style>
