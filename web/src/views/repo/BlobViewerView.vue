<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { repoApi } from '@/api/repo'
import type { BlobInfo } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import { formatSize } from '@/utils/format'
import hljs from 'highlight.js/lib/core'
import typescript from 'highlight.js/lib/languages/typescript'
import javascript from 'highlight.js/lib/languages/javascript'
import python from 'highlight.js/lib/languages/python'
import go from 'highlight.js/lib/languages/go'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import scss from 'highlight.js/lib/languages/scss'
import json from 'highlight.js/lib/languages/json'
import markdown from 'highlight.js/lib/languages/markdown'
import bash from 'highlight.js/lib/languages/bash'
import yaml from 'highlight.js/lib/languages/yaml'
import ini from 'highlight.js/lib/languages/ini'
import rust from 'highlight.js/lib/languages/rust'
import java from 'highlight.js/lib/languages/java'
import cpp from 'highlight.js/lib/languages/cpp'
import c from 'highlight.js/lib/languages/c'
import 'highlight.js/styles/github.css'

hljs.registerLanguage('ts', typescript)
hljs.registerLanguage('js', javascript)
hljs.registerLanguage('py', python)
hljs.registerLanguage('go', go)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('css', css)
hljs.registerLanguage('scss', scss)
hljs.registerLanguage('json', json)
hljs.registerLanguage('md', markdown)
hljs.registerLanguage('sh', bash)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('yml', yaml)
hljs.registerLanguage('toml', ini)
hljs.registerLanguage('rs', rust)
hljs.registerLanguage('java', java)
hljs.registerLanguage('cpp', cpp)
hljs.registerLanguage('c', c)

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const branch = computed(() => (route.query.branch as string) || 'main')
const filePath = computed(() => (route.query.path as string) || '')

const blob = ref<BlobInfo | null>(null)
const loading = ref(true)
const error = ref('')
const highlightedContent = ref('')

const parentPath = computed(() => {
  const p = blob.value?.path || ''
  const i = p.lastIndexOf('/')
  return i > 0 ? p.substring(0, i) : ''
})

const rawUrl = computed(() => {
  if (!blob.value) return ''
  return `/api/v1/repos/${encodeURIComponent(owner.value)}/${encodeURIComponent(repoName.value)}/raw?branch=${encodeURIComponent(
    branch.value
  )}&path=${encodeURIComponent(blob.value.path)}`
})

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

function highlight(content: string, path: string): string {
  const ext = path.split('.').pop()?.toLowerCase() || ''
  if (hljs.getLanguage(ext)) {
    return hljs.highlight(content, { language: ext }).value
  }
  return escapeHtml(content)
}

async function loadBlob() {
  if (!filePath.value) {
    error.value = '未指定文件路径'
    loading.value = false
    return
  }
  loading.value = true
  error.value = ''
  try {
    const b = await repoApi.blob(owner.value, repoName.value, branch.value, filePath.value)
    blob.value = b
    highlightedContent.value = highlight(b.content, b.path)
  } catch (err: any) {
    error.value = err.message || '文件不存在'
  } finally {
    loading.value = false
  }
}

watch(() => [branch.value, filePath.value], loadBlob, { immediate: true })
</script>

<template>
  <div class="blob-viewer">
    <div v-if="loading" class="loading">
      <el-icon class="is-loading" :size="40"><Loading /></el-icon>
    </div>

    <ErrorState
      v-else-if="error"
      :message="error"
      @retry="loadBlob"
    />

    <template v-else-if="blob">
      <header class="blob-header">
        <router-link
          class="back-link"
          :to="{ name: 'repo-tree', query: { branch: branch, path: parentPath } }"
        >
          <el-icon><ArrowLeft /></el-icon>
          <span class="mono">{{ blob.path }}</span>
        </router-link>

        <div class="blob-actions">
          <a :href="rawUrl" target="_blank" rel="noopener" class="el-button el-button--default el-button--small">
            <el-icon><TopRight /></el-icon> 原始文件
          </a>
        </div>
      </header>

      <div class="blob-meta">
        <span class="meta-item mono">
          <el-icon :size="14"><ScaleToOriginal /></el-icon> {{ formatSize(blob.size) }}
        </span>
        <span class="meta-item mono">
          <el-icon :size="14"><Tickets /></el-icon> {{ blob.content.split('\n').length }} 行
        </span>
        <span class="meta-item mono">
          <el-icon :size="14"><Collection /></el-icon> {{ branch }}
        </span>
      </div>

      <article class="code-card">
        <pre class="code-block"><code class="break-word" v-html="highlightedContent"></code></pre>
      </article>
    </template>
  </div>
</template>

<style scoped lang="scss">
.blob-viewer {
  min-height: 200px;
}

.blob-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: var(--text);
}

.back-link:hover {
  color: var(--accent);
}

.blob-meta {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--text-secondary);
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.code-card {
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  overflow: hidden;
}

.code-block {
  margin: 0;
  padding: 16px;
  overflow-x: auto;
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.6;
}
</style>
