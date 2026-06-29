<script setup lang="ts">
import { ref } from 'vue'
import { openDirectory, pickOutputFolder, validateOutputDir } from '../api'

const model = defineModel<string>({ default: '' })

const props = defineProps<{
  desktopMode?: boolean
  disabled?: boolean
  readonly?: boolean
}>()

const validating = ref(false)
const opening = ref(false)
const error = ref('')

async function browse() {
  error.value = ''
  if (props.desktopMode) {
    try {
      const picked = await pickOutputFolder()
      if (picked) {
        model.value = picked
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    }
    return
  }
  await validatePath()
}

async function validatePath() {
  const path = model.value.trim()
  if (!path) {
    error.value = '请输入工作区目录路径'
    return
  }
  validating.value = true
  error.value = ''
  try {
    model.value = await validateOutputDir(path, 'workspace')
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    validating.value = false
  }
}

async function openDir() {
  const path = model.value.trim()
  if (!path) {
    error.value = '请先选择或输入工作区目录'
    return
  }
  opening.value = true
  error.value = ''
  try {
    await openDirectory(path, 'workspace')
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    opening.value = false
  }
}
</script>

<template>
  <div class="output-dir-picker">
    <label class="field-label">工作区目录</label>
    <div class="dir-row">
      <input
        v-model="model"
        type="text"
        class="dir-input"
        placeholder="选择或输入保存项目的文件夹…"
        :disabled="disabled"
        :readonly="readonly"
      />
      <button type="button" class="ghost browse-btn" :disabled="disabled || validating || readonly" @click="browse">
        {{ desktopMode ? '浏览…' : '校验' }}
      </button>
      <button
        v-if="desktopMode"
        type="button"
        class="ghost browse-btn"
        :disabled="disabled || opening || readonly || !model.trim()"
        title="在资源管理器中打开"
        @click="openDir"
      >
        打开目录
      </button>
    </div>
    <p v-if="error" class="dir-error">{{ error }}</p>
    <p v-else-if="readonly" class="muted dir-hint">当前项目已绑定此目录，新建项目可更换工作区</p>
    <p v-else class="muted dir-hint">新建项目将自动保存到此工作区下的子文件夹</p>
  </div>
</template>

<style scoped>
.field-label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--muted);
}
.dir-row {
  display: flex;
  gap: 8px;
}
.dir-input {
  flex: 1;
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: #0b101a;
  color: var(--text);
  padding: 8px 10px;
  font-size: 12px;
}
.browse-btn {
  flex-shrink: 0;
  font-size: 12px;
  white-space: nowrap;
}
.dir-error {
  margin: 6px 0 0;
  font-size: 11px;
  color: var(--danger);
}
.dir-hint {
  margin: 6px 0 0;
  font-size: 11px;
}
</style>
