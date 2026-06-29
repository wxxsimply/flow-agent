<script setup lang="ts">
import { computed } from 'vue'
import { useStudioCreativeOptions } from '../composables/useStudioCreativeOptions'
import { budgetHintForStack, stackProfileLabel } from '../constants'
import OutputDirPicker from './OutputDirPicker.vue'

const {
  style,
  orientation,
  visualTheme,
  narratorVoice,
  targetDurationSec,
  stackProfile,
  durationBounds,
  outputDir,
  desktopMode,
  activeMediaProviderLabel,
  voices,
  themes,
  clampDurationForStack,
} = useStudioCreativeOptions()

defineProps<{
  disabled?: boolean
  outputDirReadonly?: boolean
  variant?: 'sidebar' | 'panel'
}>()

const stackBadge = computed(() => {
  const tier = stackProfileLabel(stackProfile.value)
  const vendor = activeMediaProviderLabel.value || '未配置媒体供应商'
  return `${vendor} · ${tier}`
})

const budgetNote = computed(() => budgetHintForStack(stackProfile.value, targetDurationSec.value))

function onDurationInput() {
  clampDurationForStack()
}
</script>

<template>
  <aside class="creation-options" :class="{ 'sidebar-variant': variant === 'sidebar' }">
    <h2>创作参数</h2>
    <p class="stack-badge muted">{{ stackBadge }}</p>

    <OutputDirPicker
      v-model="outputDir"
      :desktop-mode="desktopMode"
      :disabled="disabled || outputDirReadonly"
      :readonly="outputDirReadonly"
    />

    <label class="field">
      <span class="field-label">2D / 3D</span>
      <select v-model="style" :disabled="disabled">
        <option value="2d">2D</option>
        <option value="3d">3D</option>
      </select>
    </label>

    <label class="field">
      <span class="field-label">横竖屏</span>
      <select v-model="orientation" :disabled="disabled">
        <option value="portrait">竖屏 9:16</option>
        <option value="landscape">横屏 16:9</option>
      </select>
    </label>

    <label class="field">
      <span class="field-label">动画风格</span>
      <select v-model="visualTheme" :disabled="disabled">
        <option v-for="t in themes" :key="t.id" :value="t.id">
          {{ t.label }}{{ t.hint ? ` · ${t.hint}` : '' }}
        </option>
      </select>
    </label>

    <label class="field">
      <span class="field-label">旁白音色</span>
      <select v-model="narratorVoice" :disabled="disabled">
        <option v-for="v in voices" :key="v.id" :value="v.id">
          {{ v.label }}{{ v.description ? ` · ${v.description}` : '' }}
        </option>
      </select>
    </label>

    <label class="field dur-field">
      <span class="field-label">目标时长</span>
      <div class="dur-row">
        <input
          v-model.number="targetDurationSec"
          type="range"
          :min="durationBounds.min"
          :max="durationBounds.max"
          :step="durationBounds.step"
          :disabled="disabled"
          @input="onDurationInput"
        />
        <span class="dur-val">{{ targetDurationSec }}s</span>
      </div>
      <span class="muted dur-note">{{ budgetNote }}</span>
    </label>
  </aside>
</template>

<style scoped>
.creation-options {
  background: rgba(18, 23, 34, 0.88);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 14px;
  box-shadow: var(--shadow);
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.creation-options.sidebar-variant {
  background: transparent;
  border: none;
  box-shadow: none;
  padding: 0;
}
.creation-options h2 {
  margin: 0;
  font-size: 15px;
}
.stack-badge {
  margin: 0;
  font-size: 12px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field-label {
  font-size: 12px;
  color: var(--muted);
}
.field select {
  background: #0b101a;
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 8px 10px;
  font: inherit;
  font-size: 13px;
}
.dur-field .dur-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.dur-field input[type='range'] {
  flex: 1;
}
.dur-val {
  font-size: 13px;
  min-width: 36px;
}
.dur-note {
  font-size: 11px;
}
</style>
