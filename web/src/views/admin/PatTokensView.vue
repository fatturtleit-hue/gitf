<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import { patApi, type PatToken } from '@/api/pat'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import { ElMessage } from 'element-plus'
import { formatDate } from '@/utils/format'

const tokens = ref<PatToken[]>([])
const loading = ref(true)
const error = ref('')

const showCreate = ref(false)
const newToken = reactive({ name: '', expires_at: '' })
const creating = ref(false)

const showTokenResult = ref(false)
const createdToken = ref('')
const createdTokenName = ref('')
const copied = ref(false)

const showDeleteConfirm = ref(false)
const pendingDelete = ref<PatToken | null>(null)
const deleting = ref(false)

const sortedTokens = computed(() =>
  [...tokens.value].sort((a, b) => (a.created_at < b.created_at ? 1 : -1))
)

function isExpired(expiresAt?: string) {
  if (!expiresAt) return false
  return new Date(expiresAt).getTime() < Date.now()
}

function daysUntilExpiry(expiresAt?: string) {
  if (!expiresAt) return null
  const ms = new Date(expiresAt).getTime() - Date.now()
  return Math.ceil(ms / 86400000)
}

async function loadTokens() {
  loading.value = true
  error.value = ''
  try {
    const res = await patApi.list()
    tokens.value = res.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function createToken() {
  if (!newToken.name) return
  creating.value = true
  try {
    const res = await patApi.create({
      name: newToken.name,
      expires_at: newToken.expires_at || undefined
    })
    createdToken.value = res.token
    createdTokenName.value = newToken.name
    showTokenResult.value = true
    showCreate.value = false
    newToken.name = ''
    newToken.expires_at = ''
    ElMessage.success('令牌已创建')
    await loadTokens()
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    creating.value = false
  }
}

function askDelete(token: PatToken) {
  pendingDelete.value = token
  showDeleteConfirm.value = true
}

async function confirmDelete() {
  if (!pendingDelete.value) return
  deleting.value = true
  try {
    await patApi.del(pendingDelete.value.id)
    ElMessage.success('令牌已删除')
    showDeleteConfirm.value = false
    pendingDelete.value = null
    await loadTokens()
  } catch (err: any) {
    ElMessage.error(err.message || '删除失败')
  } finally {
    deleting.value = false
  }
}

async function copyToken() {
  try {
    await navigator.clipboard.writeText(createdToken.value)
    copied.value = true
    ElMessage.success('令牌已复制到剪贴板')
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}

onMounted(loadTokens)
</script>

<template>
  <div class="container page">
    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
    </div>

    <ErrorState v-else-if="error" :message="error" @retry="loadTokens" />

    <template v-else>
      <header class="page-header">
        <div class="header-text">
          <h1>个人访问令牌 (PAT)</h1>
          <p class="subtle">创建用于 API 调用或 Git 操作的访问令牌</p>
        </div>
        <el-button type="primary" size="default" @click="showCreate = true">
          <el-icon><Plus /></el-icon>
          <span>新建令牌</span>
        </el-button>
      </header>

      <div class="info-card">
        <div class="info-icon">
          <el-icon :size="20"><InfoFilled /></el-icon>
        </div>
        <div class="info-body">
          <div class="info-title">关于个人访问令牌</div>
          <div class="info-desc">
            令牌可用于 <code class="mono">git clone</code>、<code class="mono">git push</code> 以及所有 API 调用。
            令牌<strong>仅在创建时显示一次</strong>，请妥善保管。
          </div>
        </div>
      </div>

      <EmptyState
        v-if="sortedTokens.length === 0"
        icon="Key"
        title="还没有令牌"
        message="创建你的第一个 Personal Access Token"
      >
        <template #actions>
          <el-button type="primary" @click="showCreate = true">
            <el-icon><Plus /></el-icon>
            <span>新建令牌</span>
          </el-button>
        </template>
      </EmptyState>

      <div v-else class="token-grid">
        <div
          v-for="t in sortedTokens"
          :key="t.id"
          class="token-card"
          :class="{ expired: isExpired(t.expires_at) }"
        >
          <div class="token-header">
            <div class="token-icon">
              <el-icon :size="20"><Key /></el-icon>
            </div>
            <div class="token-meta">
              <div class="token-name">
                <span>{{ t.name }}</span>
                <span
                  v-if="isExpired(t.expires_at)"
                  class="status-pill status-expired"
                >
                  已过期
                </span>
                <span
                  v-else-if="t.expires_at"
                  class="status-pill status-expiring"
                >
                  {{ daysUntilExpiry(t.expires_at) }} 天后到期
                </span>
                <span v-else class="status-pill status-active">活跃</span>
              </div>
              <code class="token-prefix">{{ t.token_prefix }}••••••••</code>
            </div>
            <el-dropdown trigger="click">
              <button class="more-btn" aria-label="更多操作">
                <el-icon><More /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="askDelete(t)">
                    <el-icon><Delete /></el-icon> 删除令牌
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>

          <div class="token-meta-grid">
            <div class="meta-item">
              <el-icon :size="13"><Calendar /></el-icon>
              <span class="meta-label">创建</span>
              <span class="meta-value">{{ formatDate(t.created_at) }}</span>
            </div>
            <div v-if="t.last_used_at" class="meta-item">
              <el-icon :size="13"><Clock /></el-icon>
              <span class="meta-label">使用</span>
              <span class="meta-value">{{ formatDate(t.last_used_at) }}</span>
            </div>
            <div v-else class="meta-item">
              <el-icon :size="13"><Clock /></el-icon>
              <span class="meta-label">使用</span>
              <span class="meta-value meta-muted">从未使用</span>
            </div>
            <div class="meta-item">
              <el-icon :size="13"><Timer /></el-icon>
              <span class="meta-label">到期</span>
              <span v-if="t.expires_at" class="meta-value">
                {{ formatDate(t.expires_at) }}
              </span>
              <span v-else class="meta-value meta-muted">永不过期</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 创建对话框 -->
    <el-dialog v-model="showCreate" title="新建个人访问令牌" width="480px" align-center>
      <el-form label-position="top" @submit.prevent="createToken" class="dialog-form">
        <el-form-item label="令牌名称" required>
          <el-input
            v-model="newToken.name"
            size="large"
            placeholder="例如：CI 部署密钥"
            maxlength="50"
            show-word-limit
            autofocus
          />
        </el-form-item>
        <el-form-item label="有效期">
          <el-input
            v-model="newToken.expires_at"
            type="date"
            size="large"
            placeholder="选择到期日期（可选）"
            :min="new Date(Date.now() + 86400000).toISOString().slice(0, 10)"
          />
          <div class="field-hint">留空表示永不过期</div>
        </el-form-item>
        <div class="dialog-actions">
          <el-button @click="showCreate = false">取消</el-button>
          <el-button type="primary" native-type="submit" :loading="creating">
            <span v-if="!creating">创建令牌</span>
            <span v-else>创建中…</span>
          </el-button>
        </div>
      </el-form>
    </el-dialog>

    <!-- 一次性显示令牌 -->
    <el-dialog
      v-model="showTokenResult"
      title="令牌已创建"
      width="560px"
      align-center
      :close-on-click-modal="false"
      :show-close="false"
    >
      <div class="success-icon">
        <el-icon :size="32"><CircleCheckFilled /></el-icon>
      </div>
      <h3 class="success-title">{{ createdTokenName }}</h3>

      <el-alert
        title="请立即保存此令牌，关闭后将无法再次查看。"
        type="warning"
        :closable="false"
        show-icon
        class="warning-alert"
      />

      <div class="token-display">
        <code class="token-code mono">{{ createdToken }}</code>
        <button class="copy-btn" :class="{ active: copied }" @click="copyToken">
          <el-icon :size="16">
            <component :is="copied ? 'Check' : 'CopyDocument'" />
          </el-icon>
          <span>{{ copied ? '已复制' : '复制' }}</span>
        </button>
      </div>

      <el-alert
        title="使用示例"
        type="info"
        :closable="false"
        show-icon
        class="usage-alert"
      >
        <pre class="usage-code mono"># 替换 git 操作中的密码字段
git clone https://&lt;your-token&gt;@your-gitf-server.com/owner/repo.git

# 或作为 Bearer Token 调用 API
curl -H "Authorization: Bearer &lt;your-token&gt;" http://your-gitf-server.com/api/v1/user</pre>
      </el-alert>

      <template #footer>
        <el-button type="primary" @click="showTokenResult = false">
          我已保存，关闭
        </el-button>
      </template>
    </el-dialog>

    <!-- 删除确认 -->
    <el-dialog
      v-model="showDeleteConfirm"
      title="删除令牌"
      width="420px"
      align-center
      :close-on-click-modal="!deleting"
    >
      <div v-if="pendingDelete" class="delete-body">
        <div class="delete-icon">
          <el-icon :size="36"><WarningFilled /></el-icon>
        </div>
        <h3 class="delete-title">确定要删除这个令牌吗？</h3>
        <p class="delete-target">
          <span class="delete-target-name">{{ pendingDelete.name }}</span>
          <code class="delete-target-prefix mono">{{ pendingDelete.token_prefix }}••••••••</code>
        </p>
        <p class="delete-warn">
          使用此令牌的所有应用将<strong>立即失去访问权限</strong>，且操作<strong>不可撤销</strong>。
        </p>
      </div>
      <template #footer>
        <el-button :disabled="deleting" @click="showDeleteConfirm = false">取消</el-button>
        <el-button type="danger" :loading="deleting" @click="confirmDelete">
          <span v-if="!deleting">确 定 删 除</span>
          <span v-else>删除中…</span>
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.page {
  padding-top: 24px;
  padding-bottom: 48px;
  max-width: 920px;
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.header-text {
  min-width: 0;
}

.page-header h1 {
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.01em;
  margin: 0 0 4px;
}

.page-header .subtle {
  font-size: 13px;
  margin: 0;
  color: var(--text-secondary);
}

.page-header .el-button {
  gap: 4px;
}

// ── Info card ────────────────────────────────────────────────
.info-card {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px 18px;
  background: linear-gradient(135deg, var(--accent-light) 0%, rgba(99, 102, 241, 0.06) 100%);
  border: 1px solid rgba(99, 102, 241, 0.18);
  border-radius: 10px;
  margin-bottom: 24px;
}

.info-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--surface);
  color: var(--accent);
  flex-shrink: 0;
}

.info-body {
  flex: 1;
  min-width: 0;
}

.info-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 4px;
}

.info-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;

  code {
    background: rgba(99, 102, 241, 0.08);
    color: var(--accent);
    padding: 1px 5px;
    border-radius: 3px;
    font-size: 12px;
    margin: 0 2px;
  }

  strong {
    color: var(--text);
    font-weight: 600;
  }
}

// ── Token grid ──────────────────────────────────────────────
.token-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 14px;
}

.token-card {
  padding: 18px 20px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  transition: border-color 0.15s, box-shadow 0.15s;

  &:hover {
    border-color: var(--gray-300);
    box-shadow: var(--shadow);
  }

  &.expired {
    opacity: 0.7;
  }
}

.token-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}

.token-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--accent-light);
  color: var(--accent);
  flex-shrink: 0;
}

.token-meta {
  flex: 1;
  min-width: 0;
}

.token-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
  flex-wrap: wrap;
}

.token-prefix {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
  background: var(--gray-50);
  padding: 2px 6px;
  border-radius: 4px;
}

.status-pill {
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: var(--radius-pill);
  border: 1px solid transparent;

  &.status-active {
    background: var(--accent-light);
    color: var(--accent);
  }

  &.status-expiring {
    background: #fef3c7;
    color: #92400e;
  }

  &.status-expired {
    background: #fee2e2;
    color: #b91c1c;
  }
}

.more-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: var(--radius);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;

  &:hover {
    background: var(--gray-100);
    color: var(--text);
  }
}

.token-meta-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 16px;
  padding-top: 12px;
  border-top: 1px solid var(--border-light);
  font-size: 12px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--text-muted);
  min-width: 0;

  .el-icon {
    color: var(--gray-400);
    flex-shrink: 0;
  }
}

.meta-label {
  color: var(--text-muted);
  flex-shrink: 0;
}

.meta-value {
  color: var(--text-secondary);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-muted {
  color: var(--text-muted);
  font-style: italic;
  font-weight: 400;
}

// ── Create dialog ────────────────────────────────────────────
.dialog-form {
  margin: 0;

  :deep(.el-form-item) {
    margin-bottom: 18px;
  }

  :deep(.el-form-item__label) {
    font-size: 13px;
    font-weight: 500;
  }
}

.field-hint {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}

// ── Success dialog ──────────────────────────────────────────
.success-icon {
  display: flex;
  justify-content: center;
  color: var(--success);
  margin: 8px 0 12px;
}

.success-title {
  text-align: center;
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 16px;
  word-break: break-all;
}

.warning-alert {
  margin-bottom: 12px;
  border-radius: var(--radius);
}

.token-display {
  position: relative;
  display: flex;
  align-items: stretch;
  background: #1e293b;
  border-radius: 8px;
  overflow: hidden;
  margin-bottom: 12px;
}

.token-code {
  flex: 1;
  padding: 14px 16px;
  font-size: 13px;
  line-height: 1.5;
  color: #f1f5f9;
  background: transparent;
  word-break: break-all;
  max-height: 120px;
  overflow-y: auto;
}

.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 84px;
  padding: 0 16px;
  border: none;
  border-left: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.06);
  color: #cbd5e1;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    background: rgba(255, 255, 255, 0.12);
    color: #f1f5f9;
  }

  &.active {
    background: rgba(34, 197, 94, 0.18);
    color: #4ade80;
  }
}

.usage-alert {
  border-radius: var(--radius);
}

.usage-code {
  margin: 8px 0 0;
  padding: 10px 12px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-secondary);
  overflow-x: auto;
  white-space: pre;
}

// ── Delete confirm dialog ────────────────────────────────
.delete-body {
  text-align: center;
  padding: 4px 0 8px;
}

.delete-icon {
  display: flex;
  justify-content: center;
  color: var(--danger);
  margin-bottom: 14px;
}

.delete-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 14px;
  color: var(--text);
}

.delete-target {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px 16px;
  background: var(--bg);
  border: 1px solid var(--border-light);
  border-radius: 8px;
  margin-bottom: 14px;
}

.delete-target-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
}

.delete-target-prefix {
  font-size: 12px;
  color: var(--text-muted);
}

.delete-warn {
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-secondary);
  margin: 0;

  strong {
    color: var(--danger);
    font-weight: 600;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .token-grid {
    grid-template-columns: 1fr;
  }

  .page-header h1 {
    font-size: 20px;
  }
}
</style>
