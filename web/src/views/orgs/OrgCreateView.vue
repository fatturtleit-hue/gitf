<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { orgApi } from '@/api/org'
import { ElMessage } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'

const router = useRouter()

const form = reactive({
  name: '',
  display_name: '',
  description: ''
})
const loading = ref(false)

async function onSubmit() {
  if (!form.name) return
  loading.value = true
  try {
    const org = await orgApi.create(form)
    ElMessage.success('组织创建成功')
    router.push(`/orgs/${org.name}`)
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="container container-narrow" style="padding-top: 24px; padding-bottom: 48px">
    <PageHeader title="新建组织">
      <template #actions>
        <router-link to="/orgs" class="el-button">取消</router-link>
      </template>
    </PageHeader>

    <div class="card" style="padding: 24px">
      <el-form :model="form" @submit.prevent="onSubmit" label-position="top">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="显示名称">
          <el-input v-model="form.display_name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" :disabled="!form.name || loading">创建</el-button>
      </el-form>
    </div>
  </section>
</template>
