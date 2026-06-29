<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteRun, fetchRunHistory } from '../api'
import { STAGE_LABELS, stackProfileLabel, type RunHistoryItem } from '../constants'
import { useStudioRunState } from '../composables/useStudioRunState'

const emit = defineEmits<{ openRun: [runId: string]; deleted: [runId: string] }>()

const { currentRunId, polling } = useStudioRunState()

const runs = ref<RunHistoryItem[]>([])
const loading = ref(true)
const error = ref('')
const menuOpen = ref(false)
const menuX = ref(0)
const menuY = ref(0)
const menuRun = ref<RunHistoryItem | null>(null)

function stageLabel(stage: string): string {
  return STAGE_LABELS[stage] || stage
}

function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString('zh-CN')
  } catch {
    return iso
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    runs.value = await fetchRunHistory()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)

function openContextMenu(event: MouseEvent, run: RunHistoryItem) {
  event.preventDefault()
  event.stopPropagation()
  menuRun.value = run
  menuX.value = event.clientX
  menuY.value = event.clientY
  menuOpen.value = true
}

function closeMenu() {
  menuOpen.value = false
  menuRun.value = null
}

async function confirmDelete() {
  const run = menuRun.value
  if (!run) return
  closeMenu()
  if (polling.value && currentRunId.value === run.run_id) {
    error.value = '任务进行中，请先等待完成或新建项目后再删除'
    return
  }
  const ok = window.confirm(
    `确定删除「${run.title || '未命名项目'}」吗？\n\n将同时删除本地项目目录及全部生成文件，此操作不可恢复。`,
  )
  if (!ok) return
  try {
    await deleteRun(run.run_id)
    runs.value = runs.value.filter((r) => r.run_id !== run.run_id)
    emit('deleted', run.run_id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}
</script>

<template>
  <section class="view active history-view">
    <header class="topbar">
      <div>
        <h1>历史创作</h1>
        <p class="muted">查看并继续之前的视频生成任务</p>
      </div>
      <button class="ghost" :disabled="loading" @click="load">刷新</button>
    </header>

    <p v-if="loading" class="muted">加载中…</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <p v-else-if="runs.length === 0" class="muted">暂无历史记录，去创作台开始第一个项目吧。</p>

    <div v-else class="history-list">
      <article
        v-for="run in runs"
        :key="run.run_id"
        class="history-card"
        @click="emit('openRun', run.run_id)"
        @contextmenu="openContextMenu($event, run)"
      >
        <div class="history-card-head">
          <h3>{{ run.title || '未命名项目' }}</h3>
          <span class="stage-badge" :class="run.stage">{{ stageLabel(run.stage) }}</span>
        </div>
        <p class="muted history-meta">
          {{ formatTime(run.started_at) }}
          <span v-if="run.stack_profile"> · {{ stackProfileLabel(run.stack_profile) }}</span>
        </p>
        <div class="history-tags">
          <span v-if="run.has_master_video" class="tag ok">有成片</span>
          <span v-if="run.awaiting_review" class="tag warn">待审阅</span>
          <span v-if="run.failed" class="tag err">失败</span>
          <span v-if="run.finished" class="tag ok">已完成</span>
        </div>
      </article>
    </div>

    <div
      v-if="menuOpen && menuRun"
      class="context-menu"
      :style="{ left: `${menuX}px`, top: `${menuY}px` }"
      @click.stop
    >
      <button type="button" class="context-menu-item danger" @click="confirmDelete">
        删除记录及本地文件
      </button>
    </div>
  </section>
</template>

<style scoped>
.history-view {
  padding: 1.5rem 2rem;
}
.history-list {
  display: grid;
  gap: 0.75rem;
  margin-top: 1rem;
}
.history-card {
  padding: 1rem 1.25rem;
  border-radius: 10px;
  background: var(--panel, #1a1d24);
  border: 1px solid #2a2f3a;
  cursor: pointer;
  transition: border-color 0.15s;
}
.history-card:hover {
  border-color: #4f8cff;
}
.history-card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}
.history-card h3 {
  margin: 0;
  font-size: 1rem;
}
.stage-badge {
  font-size: 0.75rem;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  background: #2a3140;
  white-space: nowrap;
}
.history-meta {
  margin: 0.5rem 0 0;
  font-size: 0.85rem;
}
.history-tags {
  display: flex;
  gap: 0.4rem;
  margin-top: 0.6rem;
  flex-wrap: wrap;
}
.tag {
  font-size: 0.72rem;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
  background: #333;
}
.tag.ok {
  background: #14532d;
  color: #86efac;
}
.tag.warn {
  background: #713f12;
  color: #fcd34d;
}
.tag.err {
  background: #7f1d1d;
  color: #fca5a5;
}
.error {
  color: #f87171;
}
.context-menu {
  position: fixed;
  z-index: 1000;
  min-width: 180px;
  padding: 6px;
  border-radius: 10px;
  background: var(--panel, #1a1d24);
  border: 1px solid #2a2f3a;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}
.context-menu-item {
  display: block;
  width: 100%;
  text-align: left;
  border: none;
  background: transparent;
  color: #f87171;
  font: inherit;
  font-size: 13px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
}
.context-menu-item:hover {
  background: #2a3140;
}
</style>
