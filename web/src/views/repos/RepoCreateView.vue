<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { repoApi } from '@/api/repo'
import { useAuthStore } from '@/stores/auth'
import PageHeader from '@/components/PageHeader.vue'

const auth = useAuthStore()
const router = useRouter()

const form = reactive({
  name: '',
  description: '',
  defaultBranch: 'main',
  isPrivate: false,
  initReadme: true
})

const loading = ref(false)
const error = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

const ownerPrefix = computed(() => auth.user?.username ?? 'you')
const apiOrigin = computed(() => (typeof window !== 'undefined' ? window.location.origin : ''))
const nameValid = computed(() => /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,49}$/.test(form.name))
const canSubmit = computed(() => form.name.length > 0 && nameValid.value && form.defaultBranch.length > 0)
const pushedUrl = computed(() => form.name ? `${apiOrigin.value}/${ownerPrefix.value}/${form.name}` : `${apiOrigin.value}/${ownerPrefix.value}/…`)
const branchName = computed(() => form.defaultBranch || 'main')
const commands = computed(() => [
  { label: '已有仓库？', code: `cd existing-project\ngit remote add origin ${pushedUrl.value}\ngit push -u origin ${branchName.value}` },
  { label: '新项目？', code: `mkdir ${form.name || 'new-project'} && cd $_\necho "# ${form.name || 'new-project'}" > README.md\ngit init\ngit remote add origin ${pushedUrl.value}\ngit add .\ngit commit -m "Initial commit"\ngit push -u origin ${branchName.value}` }
])

async function onSubmit() {
  if (!canSubmit.value) return
  loading.value = true
  error.value = ''

  try {
    const repo = await repoApi.create({
      name: form.name,
      description: form.description || undefined,
      default_branch: form.defaultBranch || 'main',
      is_private: form.isPrivate
    })
    ElMessage.success('仓库创建成功')
    router.push(`/${repo.owner_username}/${repo.name}`)
  } catch (err: any) {
    error.value = err.message || '创建失败'
  } finally {
    loading.value = false
  }
}

function setVisibility(v: boolean) {
  form.isPrivate = v
}

onMounted(() => {
  inputRef.value?.focus()
})

const copied = ref<string | null>(null)
async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = text
    ElMessage.success('已复制')
    setTimeout(() => { copied.value = null }, 1600)
  } catch {
    ElMessage.warning('复制失败')
  }
}
</script>

<template>
  <section class="page">
    <div class="container">
      <PageHeader title="新建仓库" subtitle="创建一个新的 Git 仓库来存放你的代码">
        <template #actions>
          <router-link to="/repos" class="el-button cancel-btn">
            <el-icon><Close /></el-icon><span>取消</span>
          </router-link>
        </template>
      </PageHeader>

      <div class="layout">
        <div class="main-col">
          <el-form :model="form" @submit.prevent="onSubmit" class="create-form" label-position="top">
            <div class="form-section">
              <div class="form-section-label">基本信息</div>

              <el-form-item label="仓库名称" required>
                <div class="slug-input">
                  <span class="slug-prefix mono">{{ ownerPrefix }}</span>
                  <span class="slug-sep">/</span>
                  <input
                    ref="inputRef"
                    v-model="form.name"
                    class="slug-field mono"
                    placeholder="my-awesome-project"
                    spellcheck="false"
                    autocapitalize="off"
                    autocomplete="off"
                    maxlength="50"
                  />
                </div>
                <transition name="hint" mode="out-in">
                  <div v-if="form.name && !nameValid" key="err" class="hint hint-err">
                    <el-icon><Warning /></el-icon>
                    仅允许字母、数字、下划线和连字符
                  </div>
                  <div v-else key="ok" class="hint">
                    <el-icon><InfoFilled /></el-icon>
                    仓库地址：<code class="mono">{{ ownerPrefix }}/{{ form.name || '...' }}</code>
                  </div>
                </transition>
              </el-form-item>

              <el-form-item label="描述（可选）">
                <el-input
                  v-model="form.description"
                  type="textarea"
                  :rows="2"
                  :maxlength="200"
                  show-word-limit
                  placeholder="一句话描述这个仓库..."
                />
              </el-form-item>
            </div>

            <div class="form-section">
              <div class="form-section-label">配置</div>

              <div class="form-row">
                <el-form-item label="默认分支" class="flex-1">
                  <el-input v-model="form.defaultBranch" class="mono-input" maxlength="40" />
                </el-form-item>
              </div>

              <div class="form-row">
                <el-form-item label="可见性" class="full">
                  <div class="visibility-toggle">
                    <button
                      type="button"
                      class="vis-option"
                      :class="{ 'vis-option--active': !form.isPrivate }"
                      @click="setVisibility(false)"
                    >
                      <div class="vis-icon vis-icon--public">
                        <el-icon :size="20"><Unlock /></el-icon>
                      </div>
                      <div class="vis-text">
                        <strong>Public</strong>
                        <span>任何人都可以看到此仓库</span>
                      </div>
                      <div class="vis-check">
                        <el-icon v-if="!form.isPrivate"><Check /></el-icon>
                      </div>
                    </button>
                    <button
                      type="button"
                      class="vis-option"
                      :class="{ 'vis-option--active': form.isPrivate }"
                      @click="setVisibility(true)"
                    >
                      <div class="vis-icon vis-icon--private">
                        <el-icon :size="20"><Lock /></el-icon>
                      </div>
                      <div class="vis-text">
                        <strong>Private</strong>
                        <span>仅受邀的协作者可以访问</span>
                      </div>
                      <div class="vis-check">
                        <el-icon v-if="form.isPrivate"><Check /></el-icon>
                      </div>
                    </button>
                  </div>
                </el-form-item>
              </div>

              <el-form-item label="初始化选项" class="full">
                <label class="check-card">
                  <el-checkbox v-model="form.initReadme" />
                  <div>
                    <div class="check-label">添加 README 文件</div>
                    <div class="check-desc">首次推送后会在仓库中自动创建一个 README.md</div>
                  </div>
                </label>
              </el-form-item>
            </div>

            <transition name="error">
              <el-alert
                v-if="error"
                :title="error"
                type="error"
                :closable="false"
                show-icon
                class="error-alert"
              />
            </transition>

            <div class="form-actions">
              <el-button type="primary" native-type="submit" size="large" :loading="loading" :disabled="!canSubmit" class="submit-btn">
                <span>{{ loading ? '创建中...' : '创建仓库' }}</span>
                <el-icon v-if="!loading"><Right /></el-icon>
              </el-button>
            </div>
          </el-form>
        </div>

        <aside class="side-col">
          <div class="tip-card card">
            <div class="tip-header">
              <div class="tip-icon">
                <el-icon :size="20"><Promotion /></el-icon>
              </div>
              <div>
                <div class="tip-title">创建后</div>
                <div class="tip-sub">推送代码到新仓库</div>
              </div>
            </div>
            <div class="tip-cmds">
              <div v-for="(cmd, i) in commands" :key="i" class="cmd-block">
                <div class="cmd-label">{{ cmd.label }}</div>
                <div class="cmd-code-wrap">
                  <pre class="cmd-code mono"><code>{{ cmd.code }}</code></pre>
                  <button class="copy-btn" :class="{ active: copied === cmd.code }" @click="copy(cmd.code)">
                    <el-icon><CopyDocument v-if="copied !== cmd.code" /><Check v-else /></el-icon>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div class="info-card card">
            <div class="info-title">
              <el-icon><InfoFilled /></el-icon> 小贴士
            </div>
            <ul class="info-list">
              <li>Public 仓库对所有人开放可读</li>
              <li>Private 仓库仅你邀请的协作者可见</li>
              <li>创建后可随时修改可见性和设置</li>
            </ul>
          </div>
        </aside>
      </div>
    </div>
  </section>
</template>

<style scoped lang="scss">
.page {
  min-height: calc(100vh - var(--header-h));
  background: var(--bg);
  padding: 24px 0 64px;
}

.layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 360px);
  gap: 24px;
  align-items: start;
  max-width: 1080px;
  width: 100%;
  padding: 0 16px;
  margin: 0 auto;
  box-sizing: border-box;
}

.main-col {
  min-width: 0;
  max-width: 100%;
}

.side-col {
  min-width: 0;
  max-width: 100%;
}

.create-form {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 28px 32px;
  box-shadow: var(--shadow-sm);
}

.form-section {
  margin-bottom: 24px;

  &:last-of-type {
    margin-bottom: 0;
  }
}

.form-section-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 12px;
}

.form-row {
  display: flex;
  gap: 16px;

  .flex-1 {
    flex: 1;
  }

  .full {
    width: 100%;
  }
}

// ── Slug input ──────────────────────────────────────────────────────
.slug-input {
  display: flex;
  align-items: center;
  border: 1.5px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  overflow: hidden;
  transition: border-color 0.15s, box-shadow 0.15s;

  &:hover {
    border-color: var(--gray-300);
  }

  &:focus-within {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-ring);
  }
}

.slug-prefix {
  padding: 0 12px;
  font-size: 14px;
  color: var(--text-secondary);
  background: var(--gray-50);
  height: 38px;
  display: flex;
  align-items: center;
  border-right: 1px solid var(--border);
}

.slug-sep {
  padding: 0 6px;
  color: var(--text-muted);
  font-size: 14px;
}

.slug-field {
  flex: 1;
  height: 38px;
  border: none;
  outline: none;
  background: transparent;
  font-size: 14px;
  color: var(--text);
  padding: 0 8px;
  min-width: 0;
}

.hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 6px;
  line-height: 1.5;

  code {
    color: var(--text-secondary);
    background: var(--gray-50);
    padding: 1px 5px;
    border-radius: 3px;
  }
}

.hint-err {
  color: var(--danger);

  code {
    color: var(--danger);
  }
}

.hint-enter-active,
.hint-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}
.hint-enter-from,
.hint-leave-to {
  opacity: 0;
  transform: translateY(-2px);
}

// ── Visibility toggle ──────────────────────────────────────────────
.visibility-toggle {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-top: 4px;
}

.vis-option {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border: 1.5px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;

  &:hover {
    border-color: var(--gray-300);
    transform: translateY(-1px);
  }

  &--active {
    border-color: var(--accent);
    background: var(--accent-light);
    box-shadow: 0 0 0 1px var(--accent);

    .vis-icon--public {
      background: #dbeafe;
      color: #2563eb;
    }
    .vis-icon--private {
      background: #fee2e2;
      color: #dc2626;
    }
    .vis-check {
      color: var(--accent);
      opacity: 1;
    }
  }
}

.vis-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--gray-100);
  color: var(--text-secondary);
  flex-shrink: 0;
  transition: all 0.15s;

  &--public,
  &--private {
    // active state handled by parent .vis-option--active
  }
}

.vis-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;

  strong {
    font-size: 14px;
    font-weight: 600;
    color: var(--text);
  }

  span {
    font-size: 12px;
    color: var(--text-secondary);
  }
}

.vis-check {
  color: var(--accent);
  opacity: 0;
  transition: opacity 0.15s;
  flex-shrink: 0;
}

// ── Checkbox card ────────────────────────────────────────────────
.check-card {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 14px;
  border: 1.5px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  cursor: pointer;
  transition: all 0.15s;
  margin-top: 4px;

  &:hover {
    border-color: var(--gray-300);
  }

  :deep(.el-checkbox) {
    margin-top: 1px;
  }
}

.check-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  margin-bottom: 2px;
}

.check-desc {
  font-size: 12px;
  color: var(--text-secondary);
}

// ── Form actions ───────────────────────────────────────────────
.form-actions {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--border-light);
  display: flex;
  justify-content: flex-end;
}

.submit-btn {
  min-width: 160px;
  height: 40px;
  font-weight: 500;
}

// ── Error alert ───────────────────────────────────────────────
.error-alert {
  margin-top: 8px;
}

.error-enter-active,
.error-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}
.error-enter-from,
.error-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

// ── Side cards ───────────────────────────────────────────────
.side-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
  position: sticky;
  top: calc(var(--header-h) + 24px);
}

.tip-card {
  padding: 20px;
  background: linear-gradient(135deg, var(--surface) 0%, var(--gray-50) 100%);
  border: 1px solid var(--border);
}

.tip-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px dashed var(--border);
}

.tip-icon {
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

.tip-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  line-height: 1.3;
}

.tip-sub {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
}

.tip-cmds {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cmd-block {
  // block wrapper
}

.cmd-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 6px;
}

.cmd-code-wrap {
  position: relative;
  background: #1e293b;
  border-radius: var(--radius);
  overflow: hidden;
}

.cmd-code {
  margin: 0;
  padding: 12px 12px 12px 14px;
  font-size: 12px;
  line-height: 1.6;
  color: #cbd5e1;
  font-family: ui-monospace, 'SF Mono', Consolas, monospace;
  overflow-x: auto;
  white-space: pre;
  font-weight: 400;
}

.copy-btn {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 5px;
  background: rgba(255, 255, 255, 0.08);
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    background: rgba(255, 255, 255, 0.16);
    color: #e2e8f0;
  }

  &.active {
    background: rgba(34, 197, 94, 0.18);
    color: #4ade80;
  }
}

.info-card {
  padding: 16px 20px;
  background: var(--gray-50);
  border: 1px solid var(--border);
}

.info-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 8px;

  .el-icon {
    color: var(--accent);
  }
}

.info-list {
  margin: 0;
  padding-left: 20px;
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.7;

  li {
    margin-bottom: 2px;
  }
}

// ── Override el-input styling for consistency ─────────────────────
:deep(.el-textarea__inner) {
  font-family: var(--font-ui);
}

.mono-input :deep(.el-input__wrapper) {
  background: var(--gray-50);
}

// ── Cancel button ─────────────────────────────────────────────
.cancel-btn {
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  gap: 4px;

  span {
    display: inline-block;
    line-height: 1;
  }
}

// ── Responsive ───────────────────────────────────────────────
@media (max-width: 1080px) {
  .layout {
    grid-template-columns: 1fr;
    max-width: 720px;
  }
  .side-col {
    position: static;
  }
}

@media (max-width: 480px) {
  .visibility-toggle {
    grid-template-columns: 1fr;
  }
  .create-form {
    padding: 20px 16px;
  }
  .form-actions {
    justify-content: stretch;
  }
  .submit-btn {
    flex: 1;
  }
}
</style>
