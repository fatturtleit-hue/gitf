<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { repoApi } from '@/api/repo'
import type { TreeEntry } from '@/api/types'
import ErrorState from '@/components/ErrorState.vue'
import EmptyState from '@/components/EmptyState.vue'
import { formatSize } from '@/utils/format'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const route = useRoute()

const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const branch = computed(() => (route.query.branch as string) || 'main')
const path = computed(() => (route.query.path as string) || '')

const entries = ref<TreeEntry[]>([])
const loading = ref(true)
const error = ref('')
const readmeHtml = ref('')

const breadcrumbs = computed(() => {
  if (!path.value) return []
  const parts = path.value.split('/')
  const result: { name: string; path: string }[] = []
  let acc = ''
  for (const part of parts) {
    acc = acc ? `${acc}/${part}` : part
    result.push({ name: part, path: acc })
  }
  return result
})

const isRoot = computed(() => path.value === '')
const hasReadme = computed(() =>
  entries.value.some((e: TreeEntry) => e.type === 'blob' && e.name.toLowerCase() === 'readme.md')
)

function fileIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  const map: Record<string, string> = {
    ts: 'Document',
    js: 'Document',
    py: 'Document',
    go: 'Document',
    rs: 'Document',
    html: 'Document',
    css: 'Document',
    scss: 'Document',
    json: 'Document',
    md: 'Document',
    txt: 'Document',
    yaml: 'Document',
    yml: 'Document',
    toml: 'Document',
    lock: 'Lock',
    png: 'Picture',
    jpg: 'Picture',
    jpeg: 'Picture',
    svg: 'Picture',
    gif: 'Picture',
    ico: 'Picture',
    pdf: 'Document',
    zip: 'Folder'
  }
  return map[ext] || 'Document'
}

async function loadEntries() {
  loading.value = true
  error.value = ''
  try {
    const res = await repoApi.tree(owner.value, repoName.value, branch.value, path.value)
    entries.value = res.entries
    await loadReadme()
  } catch (err: any) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
  }
}

async function loadReadme() {
  if (!isRoot.value || !hasReadme.value) {
    readmeHtml.value = ''
    return
  }
  const readme = entries.value.find((e: TreeEntry) => e.type === 'blob' && e.name.toLowerCase() === 'readme.md')
  if (!readme) return
  try {
    const content = await repoApi.raw(owner.value, repoName.value, branch.value, readme.path)
    const html = await marked.parse(content)
    readmeHtml.value = DOMPurify.sanitize(html)
  } catch {
    readmeHtml.value = ''
  }
}

watch(() => [branch.value, path.value], loadEntries, { immediate: true })
</script>

<template>
  <div v-if="loading" class="loading">
    <el-icon class="is-loading" :size="32"><Loading /></el-icon>
  </div>

  <ErrorState v-else-if="error" :message="error" @retry="loadEntries" />

  <EmptyState
    v-else-if="entries.length === 0"
    icon="FolderOpened"
    title="此目录为空"
    message="该目录下没有任何文件"
  />

  <template v-else>
    <div class="breadcrumb" style="margin-bottom: 12px; font-size: 13px">
      <router-link :to="{ query: { branch: branch, path: '' } }" class="subtle">{{ repoName }}</router-link>
      <span v-for="part in breadcrumbs" :key="part.path">
        <span class="muted"> / </span>
        <router-link :to="{ query: { branch: branch, path: part.path } }">{{ part.name }}</router-link>
      </span>
    </div>

    <div class="file-list">
      <router-link
        v-for="entry in entries"
        :key="entry.path"
        class="file-row"
        :to="{
          name: entry.type === 'tree' ? 'repo-tree' : 'repo-blob',
          query: { branch: branch, path: entry.path }
        }"
      >
        <el-icon :size="18" color="var(--gray-400)" style="margin-right: 10px">
          <component :is="entry.type === 'tree' ? 'Folder' : fileIcon(entry.name)" />
        </el-icon>
        <span class="flex-1 text-ellipsis">{{ entry.name }}</span>
        <span class="muted" style="font-size: 12px">{{ entry.type === 'blob' ? formatSize(entry.size) : '' }}</span>
      </router-link>
    </div>

    <el-card v-if="isRoot && hasReadme" style="margin-top: 24px">
      <div class="markdown" v-html="readmeHtml"></div>
    </el-card>
  </template>
</template>

<style scoped lang="scss">
.breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}

.breadcrumb > * {
  display: inline-flex;
  align-items: center;
}

@media (max-width: 480px) {
  .file-row {
    padding: 8px 12px;
  }
}
</style>
