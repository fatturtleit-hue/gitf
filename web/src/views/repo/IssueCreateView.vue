<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { issueApi } from '@/api/issue'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

const owner = route.params.owner as string
const repoName = route.params.name as string
const branch = (route.query.branch as string) || 'main'

const form = reactive({
  title: '',
  body: ''
})
const loading = ref(false)

async function onSubmit() {
  if (!form.title) return
  loading.value = true
  try {
    const issue = await issueApi.create(owner, repoName, { title: form.title, body: form.body })
    ElMessage.success('Issue 创建成功')
    router.push({ name: 'repo-issue-detail', params: { number: issue.number }, query: { branch } })
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="card" style="padding: 20px">
    <h3 style="margin-bottom: 16px">新建 Issue</h3>
    <el-form :model="form" @submit.prevent="onSubmit" label-position="top">
      <el-form-item label="标题" required>
        <el-input v-model="form.title" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.body" type="textarea" :rows="6" />
      </el-form-item>
      <el-button type="primary" native-type="submit" :loading="loading" :disabled="!form.title">创建</el-button>
    </el-form>
  </div>
</template>
