<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { repoApi } from '@/api/repo'
import { protectionApi } from '@/api/protection'
import type { Repository, Collaborator, BranchProtection, CreateProtectionRequest } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')

const repo = ref<Repository | null>(null)
const collaborators = ref<Collaborator[]>([])
const loading = ref(true)
const error = ref('')
const description = ref('')
const isPrivate = ref(false)
const saving = ref(false)

const showAddCollab = ref(false)
const newCollab = ref({ username: '', permission: 'read', quota_bytes: 0 })
const addingCollab = ref(false)

const editId = ref<number | null>(null)
const editPerm = ref('read')
const editQuota = ref(0)

// ── 分支保护 ──
const protections = ref<BranchProtection[]>([])
const protectionsVisible = ref(true)
const showAddProt = ref(false)
const addingProt = ref(false)
const newProt = ref<CreateProtectionRequest>({
  pattern: '',
  require_approvals: 0,
  allow_force_push: false,
  allow_delete: false
})

async function loadProtections() {
  try {
    const p = await protectionApi.list(owner.value, repoName.value)
    protections.value = p.items
    protectionsVisible.value = true
  } catch {
    // Non-admin callers cannot list protections — hide the section quietly.
    protectionsVisible.value = false
  }
}

async function addProtection() {
  if (!newProt.value.pattern) return
  addingProt.value = true
  try {
    await protectionApi.create(owner.value, repoName.value, newProt.value)
    ElMessage.success('保护规则已添加')
    showAddProt.value = false
    newProt.value = { pattern: '', require_approvals: 0, allow_force_push: false, allow_delete: false }
    await loadProtections()
  } catch (err: any) {
    ElMessage.error(err.message || '添加失败')
  } finally {
    addingProt.value = false
  }
}

async function removeProtection(p: BranchProtection) {
  try {
    await ElMessageBox.confirm(`确定要删除对 “${p.pattern}” 的保护规则吗？`, '删除保护规则', { type: 'warning' })
    await protectionApi.del(owner.value, repoName.value, p.id)
    ElMessage.success('已删除')
    await loadProtections()
  } catch { /* cancelled */ }
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [r, c] = await Promise.all([
      repoApi.get(owner.value, repoName.value),
      repoApi.listCollaborators(owner.value, repoName.value)
    ])
    repo.value = r
    description.value = r.description || ''
    isPrivate.value = r.is_private
    collaborators.value = c.items
    await loadProtections()
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  try {
    await repoApi.update(owner.value, repoName.value, {
      description: description.value,
      is_private: isPrivate.value
    })
    ElMessage.success('设置已保存')
  } catch (err: any) {
    ElMessage.error(err.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function addCollaborator() {
  if (!newCollab.value.username) return
  addingCollab.value = true
  try {
    await repoApi.addCollaborator(
      owner.value, repoName.value,
      newCollab.value.username,
      newCollab.value.permission,
      newCollab.value.quota_bytes
    )
    ElMessage.success('协作者已添加')
    showAddCollab.value = false
    newCollab.value = { username: '', permission: 'read', quota_bytes: 0 }
    await loadAll()
  } catch (err: any) {
    ElMessage.error(err.message || '添加失败')
  } finally {
    addingCollab.value = false
  }
}

async function updateCollab(c: Collaborator) {
  editId.value = c.id
  editPerm.value = c.permission
  editQuota.value = c.quota_bytes
}

async function saveCollabEdit(c: Collaborator) {
  try {
    await repoApi.updateCollaborator(owner.value, repoName.value, c.id, editPerm.value, editQuota.value)
    ElMessage.success('已更新')
    editId.value = null
    await loadAll()
  } catch (err: any) {
    ElMessage.error(err.message || '更新失败')
  }
}

async function removeCollab(userId: number, username: string) {
  try {
    await ElMessageBox.confirm(`确定要移除协作者 ${username} 吗？`, '移除协作者', { type: 'warning' })
    await repoApi.removeCollaborator(owner.value, repoName.value, userId)
    ElMessage.success('已移除')
    await loadAll()
  } catch { /* cancelled */ }
}

function formatSize(bytes: number): string {
  if (!bytes) return '无限制'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${(bytes / 1073741824).toFixed(1)} GB`
}

onMounted(loadAll)
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadAll" />

  <div v-else-if="repo" class="settings">
    <div class="card section">
      <h3>仓库设置</h3>
      <el-form label-position="top" style="margin-top: 16px" @submit.prevent="saveSettings">
        <el-form-item label="描述">
          <el-input v-model="description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="可见性">
          <el-switch v-model="isPrivate" active-text="Private" inactive-text="Public" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="saving">保存</el-button>
      </el-form>
    </div>

    <div v-if="protectionsVisible" class="card section">
      <div class="spread">
        <h3>分支保护</h3>
        <el-button type="primary" size="small" @click="showAddProt = true">
          <el-icon><Plus /></el-icon> 添加规则
        </el-button>
      </div>
      <p class="muted" style="font-size: 12px; margin: 8px 0 0">
        被保护的分支禁止删除和强制推送（除非规则允许）；“需要批准”会让对应的 Pull Request 必须先拿到足够数量的批准评审才能合并。
        规则需服务器配置 HOOK_SECRET 后才会在 push 时强制执行。
      </p>

      <div v-if="protections.length === 0" class="muted" style="margin-top: 16px; text-align: center">暂无保护规则</div>

      <div v-else class="file-list" style="margin-top: 12px">
        <div v-for="p in protections" :key="p.id" class="file-row" style="flex-wrap: wrap; gap: 8px">
          <el-icon :size="18"><Lock /></el-icon>
          <div class="flex-1">
            <div><code>{{ p.pattern }}</code></div>
            <div class="muted" style="font-size: 12px">
              <span v-if="p.require_approvals > 0">合并需 {{ p.require_approvals }} 个批准 · </span>
              <span v-if="p.allow_force_push">允许强推 · </span>
              <span v-else>禁止强推 · </span>
              <span v-if="p.allow_delete">允许删除</span>
              <span v-else>禁止删除</span>
            </div>
          </div>
          <el-button type="danger" size="small" text @click="removeProtection(p)">删除</el-button>
        </div>
      </div>
    </div>

    <div class="card section">
      <div class="spread">
        <h3>协作者</h3>
        <el-button type="primary" size="small" @click="showAddCollab = true">
          <el-icon><Plus /></el-icon> 添加
        </el-button>
      </div>

      <div v-if="collaborators.length === 0" class="muted" style="margin-top: 16px; text-align: center">暂无协作者</div>

      <div v-else class="file-list" style="margin-top: 12px">
        <div v-for="c in collaborators" :key="c.id" class="file-row" style="flex-wrap: wrap; gap: 8px">
          <el-icon :size="18"><User /></el-icon>
          <div class="flex-1">
            <div>{{ c.username }}</div>
            <div class="muted" style="font-size: 12px">{{ c.email }}</div>
          </div>

          <template v-if="editId === c.id">
            <el-select v-model="editPerm" size="small" style="width: 100px">
              <el-option label="Read" value="read" />
              <el-option label="Write" value="write" />
              <el-option label="Admin" value="admin" />
            </el-select>
            <el-input-number v-model="editQuota" size="small" :min="0" placeholder="配额" style="width: 120px" />
            <el-button type="primary" size="small" text @click="saveCollabEdit(c)">保存</el-button>
            <el-button size="small" text @click="editId = null">取消</el-button>
          </template>
          <template v-else>
            <span class="badge badge-default">{{ c.permission }}</span>
            <span class="muted" style="font-size: 12px">{{ formatSize(c.quota_bytes) }}</span>
            <el-button size="small" text @click="updateCollab(c)">编辑</el-button>
            <el-button type="danger" size="small" text @click="removeCollab(c.user_id, c.username)">移除</el-button>
          </template>
        </div>
      </div>
    </div>

    <el-dialog v-model="showAddCollab" title="添加协作者" width="400px">
      <el-form label-position="top" @submit.prevent="addCollaborator">
        <el-form-item label="用户名" required>
          <el-input v-model="newCollab.username" />
        </el-form-item>
        <el-form-item label="权限">
          <el-select v-model="newCollab.permission" style="width: 100%">
            <el-option label="Read" value="read" />
            <el-option label="Write" value="write" />
            <el-option label="Admin" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="配额 (0 = 无限制)">
          <el-input-number v-model="newCollab.quota_bytes" :min="0" style="width: 100%" />
        </el-form-item>
        <el-form-item>
          <div style="display:flex; justify-content: flex-end; gap: 8px; width: 100%">
            <el-button @click="showAddCollab = false">取消</el-button>
            <el-button type="primary" native-type="submit" :loading="addingCollab">添加</el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-dialog>

    <el-dialog v-model="showAddProt" title="添加分支保护规则" width="460px">
      <el-form label-position="top" @submit.prevent="addProtection">
        <el-form-item label="分支名或匹配模式" required>
          <el-input v-model="newProt.pattern" placeholder="例如 main 或 *" />
        </el-form-item>
        <el-form-item label="合并所需批准数（0 = 不要求）">
          <el-input-number v-model="newProt.require_approvals" :min="0" :max="10" style="width: 100%" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="newProt.allow_force_push">允许强制推送</el-checkbox>
          <el-checkbox v-model="newProt.allow_delete">允许删除分支</el-checkbox>
        </el-form-item>
        <el-form-item>
          <div style="display:flex; justify-content: flex-end; gap: 8px; width: 100%">
            <el-button @click="showAddProt = false">取消</el-button>
            <el-button type="primary" native-type="submit" :loading="addingProt">添加规则</el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.section {
  padding: 20px;
  margin-bottom: 16px;
}
</style>
