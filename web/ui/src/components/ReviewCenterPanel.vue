<script setup lang="ts">
import { ref, toRef } from 'vue'
import { useReviewWorkspace } from '../composables/useReviewWorkspace'
import { useAutoResizeTextarea } from '../composables/useAutoResizeTextarea'

const props = withDefaults(
  defineProps<{
    runId: string
    briefPath?: string
    storyboardPath?: string
    expandPath?: string
    briefRunesMin?: number
  }>(),
  { briefRunesMin: 2000 },
)

const emit = defineEmits<{
  saved: []
  resumed: [runId: string]
  cancel: []
}>()

const {
  briefText,
  storyBackground,
  mood,
  tone,
  shots,
  loading,
  saving,
  resuming,
  error,
  activeTab,
  briefRuneCount,
  briefBelowMin,
  briefRunesMin,
  saveDraft,
  confirmAndProduce,
  updateShotField,
  updateExpandMeta,
  updateActionBeat,
  shotTransition,
} = useReviewWorkspace(
  toRef(props, 'runId'),
  toRef(props, 'briefPath'),
  toRef(props, 'storyboardPath'),
  toRef(props, 'expandPath'),
  toRef(props, 'briefRunesMin'),
)

const rootRef = ref<HTMLElement | null>(null)
const { onInput: onAutoResize } = useAutoResizeTextarea(
  [briefText, storyBackground, mood, tone, shots],
  rootRef,
)

async function onSaveDraft() {
  if (await saveDraft()) emit('saved')
}

async function onConfirm() {
  if (await confirmAndProduce()) emit('resumed', props.runId)
}
</script>

<template>
  <div ref="rootRef" class="review-center">
    <header class="review-center-header">
      <div>
        <h2 class="review-center-title">审阅扩写与分镜</h2>
        <p class="muted review-center-sub">在中间大区域编辑正文或分镜，确认后再生成视频。</p>
      </div>
      <nav class="review-tabs" role="tablist">
        <button
          type="button"
          role="tab"
          class="review-tab"
          :class="{ active: activeTab === 'brief' }"
          :aria-selected="activeTab === 'brief'"
          @click="activeTab = 'brief'"
        >
          扩写正文
          <span class="tab-count" :class="{ warn: briefBelowMin }">{{ briefRuneCount }}/{{ briefRunesMin }}</span>
        </button>
        <button
          type="button"
          role="tab"
          class="review-tab"
          :class="{ active: activeTab === 'storyboard' }"
          :aria-selected="activeTab === 'storyboard'"
          @click="activeTab = 'storyboard'"
        >
          分镜（{{ shots.length }} 镜）
        </button>
      </nav>
    </header>

    <div v-if="loading" class="review-center-body muted">加载产物…</div>
    <div v-else class="review-center-body">
      <div v-show="activeTab === 'brief'" class="brief-panel">
        <p v-if="briefBelowMin" class="brief-warn">
          正文不足 {{ briefRunesMin }} 字，建议补充镜头语言与场景过渡后再生成视频。
        </p>
        <div class="brief-meta-grid">
          <label class="field-label">故事背景</label>
          <textarea
            :value="storyBackground"
            class="shot-field brief-meta-input"
            rows="1"
            placeholder="时代、地点、核心矛盾"
            @input="
              updateExpandMeta('story_background', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <label class="field-label">情绪 (mood)</label>
          <textarea
            :value="mood"
            class="shot-field brief-meta-input"
            rows="1"
            placeholder="neutral / tense / epic …"
            @input="
              updateExpandMeta('mood', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <label class="field-label">基调 (tone)</label>
          <textarea
            :value="tone"
            class="shot-field brief-meta-input"
            rows="1"
            placeholder="史诗 / 悬疑 / 温馨 …"
            @input="
              updateExpandMeta('tone', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
        </div>
        <textarea
          v-model="briefText"
          class="brief-editor-main"
          placeholder="扩写正文（镜头语言、运镜、过渡切换）…"
          spellcheck="false"
          @input="onAutoResize"
        />
      </div>
      <div v-show="activeTab === 'storyboard'" class="shot-list-panel">
        <div v-for="(shot, i) in shots" :key="shot.id || i" class="shot-card">
          <div class="shot-meta">
            <div>
              <strong>{{ shot.id || `s${String(i + 1).padStart(2, '0')}` }}</strong>
              <p v-if="shot.narrative_beat" class="shot-beat">本镜剧情：{{ shot.narrative_beat }}</p>
            </div>
            <div class="shot-meta-controls">
              <select
                class="shot-select"
                :value="shot.shot_size || ''"
                @change="updateShotField(i, 'shot_size', ($event.target as HTMLSelectElement).value)"
              >
                <option value="">景别</option>
                <option value="wide">wide</option>
                <option value="medium">medium</option>
                <option value="close">close</option>
              </select>
              <input
                type="number"
                class="shot-duration"
                min="3"
                max="15"
                step="0.5"
                :value="shot.duration_sec ?? 10"
                title="时长（秒）"
                @input="updateShotField(i, 'duration_sec', Number(($event.target as HTMLInputElement).value))"
              />
            </div>
          </div>
          <label class="field-label">剧情功能 (narrative_beat)</label>
          <textarea
            :value="shot.narrative_beat || ''"
            class="shot-field"
            rows="1"
            placeholder="铺垫 / 发展 / 揭示 / 高潮 / 收束"
            @input="
              updateShotField(i, 'narrative_beat', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <label class="field-label">原文摘录 (brief_excerpt)</label>
          <textarea
            :value="shot.brief_excerpt || ''"
            class="shot-field"
            rows="1"
            placeholder="扩写原文对应 1-2 句"
            @input="
              updateShotField(i, 'brief_excerpt', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <label class="field-label">镜头语言 / 运镜</label>
          <textarea
            :value="shot.camera_angle || ''"
            class="shot-field"
            rows="1"
            placeholder="景别、角度、运镜"
            @input="
              updateShotField(i, 'camera_angle', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <label class="field-label">场景背景</label>
          <textarea
            :value="shot.scene_background || ''"
            class="shot-field"
            rows="1"
            placeholder="画面背景与环境"
            @input="
              updateShotField(i, 'scene_background', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <label class="field-label">画面描述（visual_prompt）</label>
          <textarea
            :value="shot.visual_prompt || ''"
            class="shot-field shot-field-lg"
            rows="3"
            placeholder="AI 出图/视频描述"
            @input="
              updateShotField(i, 'visual_prompt', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <label class="field-label">动作节拍（3 条）</label>
          <textarea
            v-for="(_, bi) in 3"
            :key="bi"
            :value="(shot.action_beats || [])[bi] || ''"
            class="shot-field"
            rows="1"
            :placeholder="`节拍 ${bi + 1}`"
            @input="
              updateActionBeat(i, bi, ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <label class="field-label">旁白</label>
          <textarea
            :value="shot.narration || ''"
            class="shot-field"
            rows="1"
            placeholder="旁白"
            @input="
              updateShotField(i, 'narration', ($event.target as HTMLTextAreaElement).value);
              onAutoResize($event)
            "
          />
          <p v-if="shot.physics_cues" class="muted shot-cues">物理：{{ shot.physics_cues }}</p>
          <p v-if="shotTransition(i)" class="shot-transition">{{ shotTransition(i) }}</p>
        </div>
        <p v-if="shots.length === 0" class="muted">暂无分镜数据</p>
      </div>
    </div>

    <footer class="review-center-footer">
      <p v-if="error" class="review-error">{{ error }}</p>
      <div class="review-actions">
        <button type="button" class="ghost" :disabled="saving || resuming" @click="emit('cancel')">
          返回
        </button>
        <button type="button" class="ghost" :disabled="saving || resuming || loading" @click="onSaveDraft">
          {{ saving ? '保存中…' : '保存草稿' }}
        </button>
        <button type="button" class="primary" :disabled="saving || resuming || loading" @click="onConfirm">
          {{ resuming ? '启动中…' : '确认并生成视频' }}
        </button>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.review-center {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  height: 100%;
}
.review-center-header {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: flex-end;
  gap: 12px;
  padding: 18px 18px 12px;
  border-bottom: 1px solid var(--border);
}
.review-center-title {
  margin: 0 0 4px;
  font-size: 18px;
}
.review-center-sub {
  margin: 0;
  font-size: 13px;
}
.review-tabs {
  display: flex;
  gap: 6px;
}
.review-tab {
  border: 1px solid var(--border);
  background: #0b101a;
  color: var(--muted);
  border-radius: 999px;
  padding: 8px 16px;
  font-size: 13px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.review-tab.active {
  color: var(--accent);
  border-color: rgba(110, 168, 254, 0.55);
  background: rgba(110, 168, 254, 0.1);
}
.tab-count {
  font-size: 11px;
  opacity: 0.85;
}
.tab-count.warn {
  color: var(--danger);
}
.review-center-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 14px 18px;
  overflow: hidden;
}
.brief-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  gap: 8px;
}
.brief-warn {
  margin: 0;
  font-size: 13px;
  color: var(--danger);
}
.brief-editor-main {
  flex: 1;
  width: 100%;
  min-height: calc(100vh - 280px);
  border: 1px solid var(--border);
  border-radius: 12px;
  background: #0b101a;
  color: var(--text);
  padding: 20px;
  font: inherit;
  font-size: 15px;
  line-height: 1.7;
  resize: none;
  box-sizing: border-box;
}
.shot-list-panel {
  flex: 1;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: calc(100vh - 260px);
}
.shot-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px 14px;
  background: var(--panel-2);
}
.brief-meta-grid {
  display: grid;
  grid-template-columns: 100px 1fr;
  gap: 6px 10px;
  align-items: start;
  margin-bottom: 8px;
}
.brief-meta-input {
  min-height: 44px;
  resize: none;
  overflow: hidden;
}
.shot-beat {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--accent);
}
.shot-meta-controls {
  display: flex;
  gap: 8px;
  align-items: center;
}
.shot-select,
.shot-duration {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #0b101a;
  color: var(--text);
  padding: 6px 8px;
  font: inherit;
  font-size: 12px;
}
.shot-duration {
  width: 64px;
}
.shot-meta {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 10px;
}
.field-label {
  display: block;
  font-size: 12px;
  color: var(--muted);
  margin: 8px 0 4px;
}
.shot-field {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: #0b101a;
  color: var(--text);
  padding: 10px 12px;
  font: inherit;
  line-height: 1.55;
  resize: none;
  overflow: hidden;
  box-sizing: border-box;
  min-height: 44px;
}
.shot-field-lg {
  min-height: 120px;
}
.shot-cues {
  font-size: 12px;
  margin: 8px 0 0;
}
.shot-transition {
  font-size: 12px;
  margin: 10px 0 0;
  padding: 8px 10px;
  border-radius: 8px;
  background: rgba(110, 168, 254, 0.08);
  color: var(--accent);
}
.review-center-footer {
  border-top: 1px solid var(--border);
  padding: 12px 18px 16px;
}
.review-error {
  color: var(--danger);
  margin: 0 0 8px;
  font-size: 13px;
}
.review-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  flex-wrap: wrap;
}
</style>
