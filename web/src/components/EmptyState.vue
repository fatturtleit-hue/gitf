<script setup lang="ts">
withDefaults(
  defineProps<{
    icon?: string
    title: string
    message?: string
  }>(),
  {
    icon: 'Box',
    message: ''
  }
)
</script>

<template>
  <div class="empty-card">
    <el-icon :size="32" class="empty-icon">
      <component :is="icon" />
    </el-icon>
    <h3>{{ title }}</h3>
    <p v-if="message" class="empty-msg">{{ message }}</p>
    <div class="empty-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.empty-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48px 24px;
  text-align: center;
  background: var(--gray-50);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.empty-icon {
  color: var(--gray-300);
  margin-bottom: 8px;
}

h3 {
  font-size: 15px;
  margin: 0 0 4px;
}

.empty-msg {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 12px;
}

.empty-actions {
  display: flex;
  gap: 8px;
}
</style>
