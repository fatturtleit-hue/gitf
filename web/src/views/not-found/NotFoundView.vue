<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="not-found">
    <div class="content">
      <div class="num">404</div>
      <div class="text">
        <h1>页面不存在</h1>
        <p class="subtle">您访问的页面已被移动、删除或从未存在</p>
      </div>
      <div class="actions">
        <el-button type="primary" @click="router.push('/')">返回首页</el-button>
        <el-button @click="router.back()">返回上页</el-button>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.not-found {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - var(--header-h));
  padding: 24px;
}

.content {
  text-align: center;
  max-width: 420px;
}

.num {
  font-size: 96px;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.04em;
  background: linear-gradient(135deg, var(--accent) 0%, #8b5cf6 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 12px;
}

h1 {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 8px;
}

.subtle {
  font-size: 14px;
  margin: 0 0 28px;
}

.actions {
  display: flex;
  gap: 8px;
  justify-content: center;
}
</style>
