<script setup lang="ts">
defineProps<{
  title: string
}>()
</script>

<template>
  <div class="spread page-header">
    <h1>{{ title }}</h1>
    <div class="actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped lang="scss">
.page-header {
  margin-bottom: 16px;
}

.actions {
  display: flex;
  gap: 8px;
}
</style>
