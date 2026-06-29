<script setup lang="ts">
import SettingsView from './SettingsView.vue'
import type { ConfigStatus } from '../constants'

defineProps<{
  config: ConfigStatus | null
}>()

const emit = defineEmits<{
  settingsSaved: []
}>()

const open = defineModel<boolean>('open', { default: false })

function close() {
  open.value = false
}

function onSettingsSaved() {
  emit('settingsSaved')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="settings-drawer-root">
      <div class="settings-drawer-backdrop" @click="close" />
      <aside class="settings-drawer-panel" role="dialog" aria-label="环境设置">
        <header class="settings-drawer-head">
          <h2>环境设置</h2>
          <button type="button" class="ghost close-btn" aria-label="关闭" @click="close">✕</button>
        </header>
        <div class="settings-drawer-body">
          <SettingsView :config="config" embedded @settings-saved="onSettingsSaved" />
        </div>
      </aside>
    </div>
  </Teleport>
</template>

<style scoped>
.settings-drawer-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
}
.settings-drawer-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
}
.settings-drawer-panel {
  position: relative;
  width: min(560px, 92vw);
  height: 100%;
  background: var(--bg);
  border-left: 1px solid var(--border);
  box-shadow: var(--shadow);
  display: flex;
  flex-direction: column;
  animation: slideIn 0.22s ease;
}
@keyframes slideIn {
  from {
    transform: translateX(100%);
  }
  to {
    transform: translateX(0);
  }
}
.settings-drawer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.settings-drawer-head h2 {
  margin: 0;
  font-size: 18px;
}
.close-btn {
  padding: 6px 10px;
  font-size: 14px;
}
.settings-drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 0 20px 24px;
}
</style>
