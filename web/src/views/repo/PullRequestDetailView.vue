<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { prApi, type PullRequest, type PRReview } from '@/api/pr'
import ErrorState from '@/components/ErrorState.vue'
import { ElMessage } from 'element-plus'
import { formatDate } from '@/utils/format'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const route = useRoute()
const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const number = computed(() => Number(route.params.number))

const pr = ref<PullRequest | null>(null)
const reviews = ref<PRReview[]>([])
const diffs = ref<{ path: string; patch: string; status: string }[]>([])
const loading = ref(true)
const error = ref('')
const newReview = ref({ state: 'commented', body: '' })
const submitting = ref(false)

const bodyHtml = computed(() => {
  if (!pr.value?.body) return ''
  return DOMPurify.sanitize(marked.parse(pr.value.body) as string)
})

function renderDiffPatch(patch: string): string {
  const escaped = patch
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
  return escaped
    .replace(/^\+/gm, '<span class="add-line">+')
    .replace(/^-/gm, '<span class="del-line">-')
    .replace(/^@@/gm, '<span class="hdr-line">@@')
}

async function loadPR() {
  loading.value = true
  error.value = ''
  try {
    const [p, r, d] = await Promise.all([
      prApi.get(owner.value, repoName.value, number.value),
      prApi.listReviews(owner.value, repoName.value, number.value),
      prApi.getDiff(owner.value, repoName.value, number.value)
    ])
    pr.value = p
    reviews.value = r.items
    diffs.value = d.diffs || []
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function mergePR() {
  try {
    await prApi.merge(owner.value, repoName.value, number.value)
    ElMessage.success('PR 已合并')
    await loadPR()
  } catch (err: any) {
    ElMessage.error(err.message || '合并失败')
  }
}

async function closePR() {
  try {
    await prApi.close(owner.value, repoName.value, number.value)
    ElMessage.success('PR 已关闭')
    await loadPR()
  } catch (err: any) {
    ElMessage.error(err.message || '关闭失败')
  }
}

async function submitReview() {
  if (!newReview.value.body) return
  submitting.value = true
  try {
    const r = await prApi.createReview(owner.value, repoName.value, number.value, newReview.value)
    reviews.value.push(r)
    newReview.value = { state: 'commented', body: '' }
    ElMessage.success('审核已提交')
  } catch (err: any) {
    ElMessage.error(err.message || '提交失败')
  } finally {
    submitting.value = false
  }
}

watch(number, loadPR, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadPR" />

  <div v-else-if="pr">
    <div class="card" style="padding: 20px; margin-bottom: 16px">
      <h2>{{ pr.title }}<span class="muted" style="font-size: 16px; margin-left: 6px"> #{{ pr.number }}</span></h2>
      <div class="row gap-3 pr-meta-row" style="margin-top: 8px">
        <span :class="['badge', pr.status === 'open' ? 'badge-success' : pr.status === 'merged' ? 'badge-accent' : 'badge-default']">{{ pr.status }}</span>
        <span class="muted">{{ pr.head_branch }} → {{ pr.base_branch }} · {{ pr.author_username }} · {{ formatDate(pr.created_at) }}</span>
        <el-button v-if="pr.status === 'open'" type="success" size="small" @click="mergePR">合并</el-button>
        <el-button v-if="pr.status === 'open'" type="warning" size="small" @click="closePR">关闭</el-button>
      </div>
      <div v-if="bodyHtml" class="markdown" style="margin-top: 16px" v-html="bodyHtml"></div>
    </div>

    <div v-if="diffs.length" class="diffs">
      <h3 style="margin-bottom: 12px">变更 ({{ diffs.length }} 个文件)</h3>
      <div v-for="d in diffs" :key="d.path" class="card diff-card">
        <div class="diff-header mono">{{ d.status }}: {{ d.path }}</div>
        <pre class="diff-body"><code v-html="renderDiffPatch(d.patch)"></code></pre>
      </div>
    </div>

    <div class="reviews" style="margin-top: 16px">
      <h3 style="margin-bottom: 12px">审核 ({{ reviews.length }})</h3>
      <div v-for="r in reviews" :key="r.id" class="card" style="padding: 12px; margin-bottom: 8px">
        <div class="muted" style="font-size: 12px; margin-bottom: 4px">
          <strong>{{ r.reviewer_username }}</strong> · {{ r.state }} · {{ formatDate(r.created_at) }}
        </div>
        <div v-if="r.body">{{ r.body }}</div>
      </div>
    </div>

    <div v-if="pr.status === 'open'" class="card" style="padding: 16px; margin-top: 16px">
      <el-select v-model="newReview.state" size="small" style="width: 150px; margin-bottom: 8px">
        <el-option label="评论" value="commented" />
        <el-option label="批准" value="approved" />
        <el-option label="需要修改" value="changes_requested" />
      </el-select>
      <el-input v-model="newReview.body" type="textarea" :rows="3" placeholder="审核评论…" />
      <el-button type="primary" style="margin-top: 8px" :loading="submitting" @click="submitReview">提交审核</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.diff-card {
  margin-bottom: 12px;
  overflow: hidden;
}

.diff-header {
  padding: 8px 16px;
  background: var(--gray-50);
  border-bottom: 1px solid var(--border);
  font-size: 13px;
  font-weight: 500;
}

.diff-body {
  margin: 0;
  padding: 12px 16px;
  overflow-x: auto;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.6;
  background: var(--surface);
}

:deep(.add-line) { color: var(--success); }
:deep(.del-line) { color: var(--danger); }
:deep(.hdr-line) { color: var(--accent); }

@media (max-width: 640px) {
  .pr-meta-row {
    flex-wrap: wrap;
    gap: 6px;
  }
}
</style>
