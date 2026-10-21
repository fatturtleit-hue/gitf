<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { issueApi } from '@/api/issue'
import type { Issue, Comment } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import { formatDate } from '@/utils/format'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { ElMessage } from 'element-plus'

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const number = computed(() => Number(route.params.number))

const issue = ref<Issue | null>(null)
const comments = ref<Comment[]>([])
const loading = ref(true)
const error = ref('')
const newComment = ref('')
const submitting = ref(false)

const bodyHtml = computed(() => {
  if (!issue.value?.body) return ''
  return DOMPurify.sanitize(marked.parse(issue.value.body) as string)
})

async function loadIssue() {
  loading.value = true
  error.value = ''
  try {
    const [i, c] = await Promise.all([
      issueApi.get(owner.value, repoName.value, number.value),
      issueApi.listComments(owner.value, repoName.value, number.value)
    ])
    issue.value = i
    comments.value = c.items
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function addComment() {
  if (!newComment.value.trim()) return
  submitting.value = true
  try {
    const c = await issueApi.addComment(owner.value, repoName.value, number.value, newComment.value)
    comments.value.push(c)
    newComment.value = ''
  } catch (err: any) {
    error.value = err.message || '评论失败'
  } finally {
    submitting.value = false
  }
}

async function toggleStatus() {
  if (!issue.value) return
  const isOpen = issue.value.status === 'open'
  try {
    if (isOpen) {
      await issueApi.close(owner.value, repoName.value, number.value)
      ElMessage.success('Issue 已关闭')
    } else {
      await issueApi.reopen(owner.value, repoName.value, number.value)
      ElMessage.success('Issue 已重新打开')
    }
    await loadIssue()
  } catch (err: any) {
    ElMessage.error(err.message || '操作失败')
  }
}

watch(number, loadIssue, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadIssue" />

  <div v-else-if="issue">
    <div class="card issue-header">
      <h2>{{ issue.title }}<span class="muted issue-number"> #{{ issue.number }}</span></h2>
      <div class="issue-meta">
        <span :class="['badge', issue.status === 'open' ? 'badge-success' : 'badge-default']">{{ issue.status }}</span>
        <span class="muted">{{ issue.author_username }} 创建于 {{ formatDate(issue.created_at) }}</span>
        <el-button size="small" :type="issue.status === 'open' ? 'warning' : 'success'" @click="toggleStatus">
          {{ issue.status === 'open' ? '关闭' : '重开' }}
        </el-button>
      </div>
      <div v-if="bodyHtml" class="markdown" style="margin-top: 16px" v-html="bodyHtml"></div>
    </div>

    <div class="comments">
      <div v-for="comment in comments" :key="comment.id" class="card comment-card">
        <div class="comment-meta muted">{{ comment.author_username }} 评论于 {{ formatDate(comment.created_at) }}</div>
        <div class="markdown" v-html="DOMPurify.sanitize(marked.parse(comment.body) as string)"></div>
      </div>
    </div>

    <div class="card comment-form">
      <el-input v-model="newComment" type="textarea" :rows="4" placeholder="写下你的评论…" />
      <el-button type="primary" style="margin-top: 12px" :loading="submitting" @click="addComment">发表评论</el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.issue-header {
  padding: 20px;
  margin-bottom: 16px;
}

.issue-number {
  font-weight: 400;
  margin-left: 6px;
}

.issue-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 8px;
  flex-wrap: wrap;
}

.comment-card {
  padding: 16px;
  margin-bottom: 12px;
}

.comment-meta {
  font-size: 12px;
  margin-bottom: 8px;
}

.comment-form {
  padding: 16px;
  margin-top: 16px;
}
</style>
