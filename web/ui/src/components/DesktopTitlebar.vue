<script setup lang="ts">
import { flowDesktop, onDesktopDragMouseDown } from '../composables/useDesktopWindow'
</script>

<template>
  <header class="desktop-titlebar" @mousedown="onDesktopDragMouseDown">
    <div class="desktop-titlebar-drag" aria-hidden="true" />
    <div class="desktop-titlebar-controls">
      <button type="button" class="titlebar-btn" title="最小化" @click="flowDesktop('minimize')">—</button>
      <button type="button" class="titlebar-btn" title="全屏" @click="flowDesktop('maximize')">⛶</button>
      <button type="button" class="titlebar-btn titlebar-btn-close" title="关闭" @click="flowDesktop('close')">×</button>
    </div>
  </header>
</template>

<style scoped>
.desktop-titlebar {
  display: flex;
  align-items: stretch;
  flex-shrink: 0;
  height: 36px;
  min-height: 36px;
  background: #0c0f14;
  z-index: 200;
  user-select: none;
}
.desktop-titlebar-drag {
  flex: 1;
  min-width: 0;
}
.desktop-titlebar-controls {
  display: flex;
  flex-shrink: 0;
  height: 100%;
}
.titlebar-btn {
  width: 46px;
  height: 100%;
  border: none;
  border-radius: 0;
  background: transparent;
  color: var(--muted);
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  padding: 0;
  transition: background 0.12s, color 0.12s;
}
.titlebar-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text);
}
.titlebar-btn-close:hover {
  background: #e81123;
  color: #fff;
}
</style>
