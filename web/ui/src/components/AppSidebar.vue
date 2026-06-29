<script setup lang="ts">
import CreationOptionsPanel from './CreationOptionsPanel.vue'

defineProps<{
  envBadge: { text: string; className: string }
  authEnabled?: boolean
  outputDirLocked?: boolean
}>()

defineEmits<{ logout: [] }>()
</script>

<template>
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-mark">FA</div>
      <div>
        <div class="brand-title">FlowAgent</div>
        <div class="brand-sub">微电影 Agent Studio</div>
      </div>
    </div>
    <div class="studio-label muted">创作台</div>
    <CreationOptionsPanel variant="sidebar" :output-dir-readonly="outputDirLocked" />
    <div class="sidebar-foot">
      <div :class="envBadge.className">{{ envBadge.text }}</div>
      <button v-if="authEnabled" class="ghost logout-btn" @click="$emit('logout')">退出登录</button>
    </div>
  </aside>
</template>

<style scoped>
.studio-label {
  font-size: 12px;
  font-weight: 600;
  margin: -8px 0 0;
  padding: 6px 10px;
  border-radius: 8px;
  background: var(--panel-2);
  border: 1px solid var(--border);
}
.logout-btn {
  margin-top: 0.5rem;
  width: 100%;
  font-size: 0.8rem;
}
</style>
