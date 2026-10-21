<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string
    message?: string
    retryable?: boolean
  }>(),
  {
    title: '出错了',
    message: '加载失败，请稍后重试',
    retryable: true
  }
)

const emit = defineEmits<{
  (e: 'retry'): void
}>()
</script>

<template>
  <div class="error-card">
    <el-icon :size="40" class="error-icon"><Warning /></el-icon>
    <h3>{{ title }}</h3>
    <p class="error-msg">{{ message }}</p>
    <el-button v-if="retryable" @click="emit('retry')">
      <el-icon><Refresh /></el-icon> 重试
    </el-button>
  </div>
</template>

<style scoped lang="scss">
.error-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48px 24px;
  text-align: center;
  background: var(--gray-50);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.error-icon {
  color: var(--danger);
  margin-bottom: 8px;
}

h3 {
  font-size: 15px;
  margin: 0 0 4px;
}

.error-msg {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 12px;
}
</style>
