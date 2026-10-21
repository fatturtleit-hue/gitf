<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { repoApi } from '@/api/repo'
import { prApi } from '@/api/pr'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()
const owner = computed(() => (route.params.owner as string) || '')
const repoName = computed(() => (route.params.name as string) || '')
const branch = computed(() => (route.query.branch as string) || 'main')

const branches = ref<string[]>([])
const form = reactive({ title: '', body: '', head_branch: '', base_branch: '' })
const loading = ref(false)

onMounted(async () => {
  form.base_branch = branch.value
  form.head_branch = branch.value
  try {
    const res = await repoApi.branches(owner.value, repoName.value)
    branches.value = res.branches
    if (res.branches.length > 0) {
      form.head_branch = res.branches[0]
    }
  } catch { /* ignore */ }
})

async function onSubmit() {
  if (!form.title || !form.head_branch || !form.base_branch) return
  loading.value = true
  try {
    const pr = await prApi.create(owner.value, repoName.value, form)
    ElMessage.success('PR 创建成功')
    router.push({ name: 'repo-pull-detail', params: { number: pr.number }, query: { branch: branch.value } })
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="card" style="padding: 20px">
    <h3 style="margin-bottom: 16px">新建 Pull Request</h3>
    <el-form :model="form" label-position="top" @submit.prevent="onSubmit">
      <el-form-item label="标题" required>
        <el-input v-model="form.title" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.body" type="textarea" :rows="4" />
      </el-form-item>
      <div class="branch-row">
        <el-form-item label="来源分支" style="flex: 1" required>
          <el-select v-model="form.head_branch" style="width: 100%">
            <el-option v-for="b in branches" :key="b" :label="b" :value="b" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标分支" style="flex: 1" required>
          <el-select v-model="form.base_branch" style="width: 100%">
            <el-option v-for="b in branches" :key="b" :label="b" :value="b" />
          </el-select>
        </el-form-item>
      </div>
      <el-button type="primary" native-type="submit" :loading="loading" :disabled="!form.title">创建 PR</el-button>
    </el-form>
  </div>
</template>

<style scoped lang="scss">
.branch-row {
  display: flex;
  gap: 16px;
}

@media (max-width: 640px) {
  .branch-row {
    flex-direction: column;
    gap: 0;
  }
}
</style>
