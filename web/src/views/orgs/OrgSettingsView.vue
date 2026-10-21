<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { orgApi } from '@/api/org'
import { useAuthStore } from '@/stores/auth'
import type { Organization, OrgMember } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, User, Delete, ArrowRight, Setting, UserFilled, WarningFilled } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const orgName = computed(() => (route.params.org as string) || '')

const org = ref<Organization | null>(null)
const members = ref<OrgMember[]>([])
const loading = ref(true)
const error = ref('')
const displayName = ref('')
const description = ref('')
const saving = ref(false)
const activeSection = ref('general')

const menuItems = [
  { key: 'general', label: '通用设置', icon: Setting },
  { key: 'members', label: '成员', icon: UserFilled },
  { key: 'danger', label: '危险区域', icon: WarningFilled }
]

const showAddMember = ref(false)
const newMember = ref({ username: '', role: 'member' })
const addingMember = ref(false)

const editId = ref<number | null>(null)
const editRole = ref('member')

const leaveLoading = ref(false)
const deleteLoading = ref(false)

const currentMember = computed(() => members.value.find(m => m.username === auth.user?.username))
const isOwner = computed(() => currentMember.value?.role === 'owner')
const isAdminOrOwner = computed(() => isOwner.value || currentMember.value?.role === 'admin')

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [o, m] = await Promise.all([
      orgApi.get(orgName.value),
      orgApi.listMembers(orgName.value)
    ])
    org.value = o
    displayName.value = o.display_name || ''
    description.value = o.description || ''
    members.value = m.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  if (!isAdminOrOwner.value) {
    ElMessage.error('没有权限修改组织设置')
    return
  }
  saving.value = true
  try {
    await orgApi.update(org.value!.name, {
      display_name: displayName.value,
      description: description.value
    })
    ElMessage.success('设置已保存')
  } catch (err: any) {
    ElMessage.error(err.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function addMember() {
  if (!newMember.value.username) return
  addingMember.value = true
  try {
    await orgApi.addMember(org.value!.name, newMember.value.username, newMember.value.role)
    ElMessage.success('成员已添加')
    showAddMember.value = false
    newMember.value = { username: '', role: 'member' }
    await loadAll()
  } catch (err: any) {
    ElMessage.error(err.message || '添加失败')
  } finally {
    addingMember.value = false
  }
}

async function startEditRole(m: OrgMember) {
  editId.value = m.id
  editRole.value = m.role
}

async function saveRoleEdit(m: OrgMember) {
  try {
    await orgApi.updateMember(org.value!.name, m.user_id, editRole.value)
    ElMessage.success('已更新')
    editId.value = null
    await loadAll()
  } catch (err: any) {
    ElMessage.error(err.message || '更新失败')
  }
}

async function removeMember(userId: number, username: string) {
  try {
    await ElMessageBox.confirm(`确定要移除成员 ${username} 吗？`, '移除成员', { type: 'warning' })
    await orgApi.removeMember(org.value!.name, userId)
    ElMessage.success('已移除')
    await loadAll()
  } catch { /* cancelled */ }
}

async function leaveOrg() {
  try {
    await ElMessageBox.confirm('确定要退出这个组织吗？', '退出组织', { type: 'warning' })
    leaveLoading.value = true
    await orgApi.leave(org.value!.name)
    ElMessage.success('已退出组织')
    router.push('/orgs')
  } catch (err: any) {
    if (err.message) ElMessage.error(err.message)
  } finally {
    leaveLoading.value = false
  }
}

async function deleteOrg() {
  try {
    await ElMessageBox.confirm(
      `确定要删除组织 "${org.value!.name}" 吗？此操作不可恢复，组织下的所有仓库、团队、成员数据将被永久删除。`,
      '删除组织',
      { type: 'error', confirmButtonText: '删除', confirmButtonClass: 'el-button--danger' }
    )
    deleteLoading.value = true
    await orgApi.delete(org.value!.name)
    ElMessage.success('组织已删除')
    router.push('/orgs')
  } catch (err: any) {
    if (err.message) ElMessage.error(err.message)
  } finally {
    deleteLoading.value = false
  }
}

onMounted(loadAll)

watch(orgName, loadAll, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadAll" />

  <div v-else-if="org" class="settings-layout">
    <aside class="settings-sidebar">
      <div class="org-info">
        <div class="org-avatar">{{ org.name.charAt(0).toUpperCase() }}</div>
        <div>
          <div class="org-name">{{ org.display_name || org.name }}</div>
          <div class="org-handle muted">{{ org.name }}</div>
        </div>
      </div>
      <nav class="settings-nav">
        <button
          v-for="item in menuItems"
          :key="item.key"
          class="nav-item"
          :class="{ active: activeSection === item.key }"
          @click="activeSection = item.key"
        >
          <el-icon :size="18"><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <main class="settings-content">
      <section v-if="activeSection === 'general'" class="card section">
        <h3>通用设置</h3>
        <p class="muted section-desc">管理组织的基本信息</p>
        <el-form label-position="top" class="settings-form" style="margin-top: 16px; max-width: 560px" @submit.prevent="saveSettings">
          <el-form-item label="显示名称">
            <el-input v-model="displayName" placeholder="组织显示名称" />
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="description" type="textarea" :rows="3" placeholder="组织描述" />
          </el-form-item>
          <el-button type="primary" native-type="submit" :loading="saving">保存</el-button>
        </el-form>
      </section>

      <section v-if="activeSection === 'members'" class="card section">
        <div class="spread">
          <div>
            <h3>成员</h3>
            <p class="muted section-desc">管理组织成员及其权限</p>
          </div>
          <el-button v-if="isAdminOrOwner" type="primary" size="small" @click="showAddMember = true">
            <el-icon><Plus /></el-icon> 邀请成员
          </el-button>
        </div>

        <div v-if="members.length === 0" class="muted" style="margin-top: 16px; text-align: center">暂无成员</div>

        <div v-else class="file-list" style="margin-top: 12px">
          <div v-for="m in members" :key="m.id" class="file-row" style="flex-wrap: wrap; gap: 8px">
            <el-icon :size="18"><User /></el-icon>
            <div class="flex-1">
              <div>{{ m.username }}</div>
              <div class="muted" style="font-size: 12px">{{ m.email }}</div>
            </div>
            <span class="badge badge-default">{{ m.role }}</span>

            <template v-if="editId === m.id">
              <el-select v-model="editRole" size="small" style="width: 100px">
                <el-option label="Owner" value="owner" />
                <el-option label="Admin" value="admin" />
                <el-option label="Member" value="member" />
              </el-select>
              <el-button type="primary" size="small" text @click="saveRoleEdit(m)">保存</el-button>
              <el-button size="small" text @click="editId = null">取消</el-button>
            </template>
            <template v-else-if="isAdminOrOwner">
              <el-button size="small" text @click="startEditRole(m)">改角色</el-button>
              <el-button v-if="m.role !== 'owner'" type="danger" size="small" text @click="removeMember(m.user_id, m.username)">
                <el-icon><Delete /></el-icon> 移除
              </el-button>
              <span v-else class="muted" style="font-size: 12px">所有者</span>
            </template>
          </div>
        </div>
      </section>

      <section v-if="activeSection === 'danger'" class="card section danger-section">
        <h3>危险区域</h3>
        <p class="muted section-desc">这些操作不可逆，请谨慎操作</p>

        <div class="danger-list">
          <div v-if="!isOwner" class="danger-item">
            <div class="danger-info">
              <div class="danger-title">退出组织</div>
              <p class="muted">退出后将无法访问该组织的仓库、团队和其他资源。</p>
            </div>
            <el-button type="danger" :loading="leaveLoading" @click="leaveOrg">
              <el-icon><ArrowRight /></el-icon> 退出组织
            </el-button>
          </div>

          <div v-if="isOwner" class="danger-item">
            <div class="danger-info">
              <div class="danger-title">删除组织</div>
              <p class="muted">删除组织将永久移除所有仓库、团队、成员数据，不可恢复。</p>
            </div>
            <el-button type="danger" :loading="deleteLoading" @click="deleteOrg">
              <el-icon><Delete /></el-icon> 删除组织
            </el-button>
          </div>
        </div>
      </section>
    </main>

    <el-dialog v-model="showAddMember" title="邀请成员" width="400px">
      <el-form label-position="top" @submit.prevent="addMember">
        <el-form-item label="用户名" required>
          <el-input v-model="newMember.username" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="newMember.role" style="width: 100%">
            <el-option label="Member" value="member" />
            <el-option label="Admin" value="admin" />
            <el-option label="Owner" value="owner" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <div style="display:flex; justify-content: flex-end; gap: 8px; width: 100%">
            <el-button @click="showAddMember = false">取消</el-button>
            <el-button type="primary" native-type="submit" :loading="addingMember">邀请</el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.settings-layout {
  display: flex;
  gap: 24px;
  align-items: flex-start;
}

.settings-sidebar {
  width: 260px;
  flex-shrink: 0;
  position: sticky;
  top: 16px;
}

.org-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  margin-bottom: 16px;
}

.org-avatar {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: #10b981;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 600;
}

.org-name {
  font-weight: 600;
  color: #303133;
}

.org-handle {
  font-size: 12px;
  margin-top: 2px;
}

.settings-nav {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: #606266;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}

.nav-item:hover {
  background: #f0fdf4;
  color: #10b981;
}

.nav-item.active {
  background: #ecfdf5;
  color: #10b981;
  font-weight: 500;
  border-left: 3px solid #10b981;
}

.settings-content {
  flex: 1;
  min-width: 0;
}

.section {
  padding: 24px;
  margin-bottom: 16px;
}

.section-desc {
  margin-top: 4px;
  font-size: 13px;
}

.spread {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.flex-1 {
  flex: 1;
  min-width: 0;
}

.badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.badge-default {
  background: #e5e7eb;
  color: #374151;
}

.muted {
  color: #909399;
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.file-row {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  background: #fafafa;
  border-radius: 6px;
  gap: 12px;
}

.danger-section {
  border-color: #fde2e2;
}

.danger-list {
  margin-top: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.danger-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border: 1px solid #fde2e2;
  border-radius: 8px;
  background: #fef0f0;
}

.danger-info {
  flex: 1;
}

.danger-title {
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.loading {
  display: flex;
  justify-content: center;
  padding: 40px;
}

@media (max-width: 768px) {
  .settings-layout {
    flex-direction: column;
  }

  .settings-sidebar {
    width: 100%;
    position: static;
  }

  .danger-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .section {
    padding: 16px;
  }

  .settings-form {
    max-width: 100% !important;
  }
}
</style>