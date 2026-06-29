<script setup lang="ts">

import { onMounted, onUnmounted, ref } from 'vue'

import { deleteRun, fetchRunHistory, openDirectory } from '../api'

import { STAGE_LABELS, type RunHistoryItem } from '../constants'

import { useStudioRunState } from '../composables/useStudioRunState'



const emit = defineEmits<{

  openRun: [runId: string]

  openSettings: []

  deleted: [runId: string]

}>()



const { currentRunId, polling } = useStudioRunState()



const runs = ref<RunHistoryItem[]>([])

const loading = ref(true)

const error = ref('')

const menuOpen = ref(false)

const menuX = ref(0)

const menuY = ref(0)

const menuRun = ref<RunHistoryItem | null>(null)

const deleting = ref(false)



function stageLabel(stage: string): string {

  return STAGE_LABELS[stage] || stage

}



function workspaceLabel(runDir?: string): string {

  if (!runDir) return ''

  const parts = runDir.replace(/\\/g, '/').split('/')

  if (parts.length < 2) return runDir

  return parts[parts.length - 2] || runDir

}



function formatTime(iso: string): string {

  try {

    const d = new Date(iso)

    return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`

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



function onSelect(runId: string) {

  emit('openRun', runId)

}



async function openRunDir(runDir: string, event: Event) {

  event.stopPropagation()

  if (!runDir.trim()) return

  try {

    await openDirectory(runDir, 'project')

  } catch (e) {

    error.value = e instanceof Error ? e.message : String(e)

  }

}



function openContextMenu(event: MouseEvent, run: RunHistoryItem) {

  event.preventDefault()

  menuRun.value = run

  menuX.value = event.clientX

  menuY.value = event.clientY

  menuOpen.value = true

}



function closeMenu() {

  menuOpen.value = false

  menuRun.value = null

}



function canDeleteRun(run: RunHistoryItem): boolean {

  return !(polling.value && currentRunId.value === run.run_id)

}



async function confirmDelete() {

  const run = menuRun.value

  if (!run) return

  closeMenu()

  if (!canDeleteRun(run)) {

    error.value = '任务进行中，请先等待完成或新建项目后再删除'

    return

  }

  const label = run.title || '未命名'

  const ok = window.confirm(

    `确定删除「${label}」吗？\n\n将同时删除本地项目目录及全部生成文件，此操作不可恢复。`,

  )

  if (!ok) return

  deleting.value = true

  error.value = ''

  try {

    await deleteRun(run.run_id)

    runs.value = runs.value.filter((r) => r.run_id !== run.run_id)

    emit('deleted', run.run_id)

  } catch (e) {

    error.value = e instanceof Error ? e.message : String(e)

  } finally {

    deleting.value = false

  }

}



function onDocClick() {

  if (menuOpen.value) closeMenu()

}



onMounted(() => {

  void load()

  document.addEventListener('click', onDocClick)

})



onUnmounted(() => {

  document.removeEventListener('click', onDocClick)

})



defineExpose({ refresh: load })

</script>



<template>

  <aside class="history-rail">

    <header class="history-rail-head">

      <h2 class="history-rail-title">历史</h2>

      <button type="button" class="ghost refresh-btn" :disabled="loading || deleting" title="刷新" @click="load">

        ↻

      </button>

    </header>



    <div class="history-rail-body">

      <p v-if="loading" class="muted rail-msg">加载中…</p>

      <p v-else-if="error" class="rail-error">{{ error }}</p>

      <p v-else-if="runs.length === 0" class="muted rail-msg">暂无记录</p>

      <ul v-else class="history-rail-list">

        <li

          v-for="run in runs"

          :key="run.run_id"

          class="history-rail-item"

          :class="{ active: currentRunId === run.run_id }"

          :title="run.run_dir ? `工作区: ${workspaceLabel(run.run_dir)}\n${run.run_dir}` : undefined"

          @click="onSelect(run.run_id)"

          @contextmenu="openContextMenu($event, run)"

        >

          <span class="item-title">{{ run.title || '未命名' }}</span>

          <span class="item-stage">{{ stageLabel(run.stage) }}</span>

          <span class="item-time muted">{{ formatTime(run.started_at) }}</span>

          <span v-if="run.has_master_video" class="item-dot ok" title="有成片" />

          <span v-else-if="run.awaiting_review" class="item-dot warn" title="待审阅" />

          <span v-else-if="run.failed" class="item-dot err" title="失败" />

          <button

            v-if="run.run_dir"

            type="button"

            class="item-folder"

            title="打开项目目录"

            @click="openRunDir(run.run_dir!, $event)"

          >

            📁

          </button>

        </li>

      </ul>

    </div>



    <footer class="history-rail-foot">

      <button type="button" class="ghost settings-btn" @click="emit('openSettings')">

        ⚙ 设置

      </button>

    </footer>



    <div

      v-if="menuOpen && menuRun"

      class="context-menu"

      :style="{ left: `${menuX}px`, top: `${menuY}px` }"

      @click.stop

    >

      <button

        type="button"

        class="context-menu-item danger"

        :disabled="!canDeleteRun(menuRun)"

        @click="confirmDelete"

      >

        删除记录及本地文件

      </button>

    </div>

  </aside>

</template>



<style scoped>

.history-rail {

  display: flex;

  flex-direction: column;

  height: 100%;

  min-height: 0;

  border-right: 1px solid var(--border);

  background: rgba(12, 15, 20, 0.95);

  overflow: hidden;

  position: relative;

}

.history-rail-head {

  display: flex;

  align-items: center;

  justify-content: space-between;

  padding: 16px 12px 10px;

  flex-shrink: 0;

}

.history-rail-title {

  margin: 0;

  font-size: 13px;

  font-weight: 700;

  color: var(--muted);

  text-transform: uppercase;

  letter-spacing: 0.06em;

}

.refresh-btn {

  padding: 4px 8px;

  font-size: 14px;

  line-height: 1;

}

.history-rail-body {

  flex: 1;

  min-height: 0;

  overflow-y: auto;

  padding: 0 8px;

}

.rail-msg,

.rail-error {

  font-size: 12px;

  padding: 8px;

  margin: 0;

}

.rail-error {

  color: var(--danger);

}

.history-rail-list {

  list-style: none;

  margin: 0;

  padding: 0;

  display: flex;

  flex-direction: column;

  gap: 4px;

}

.history-rail-item {

  position: relative;

  padding: 8px 10px;

  border-radius: 8px;

  border: 1px solid transparent;

  cursor: pointer;

  transition: background 0.12s, border-color 0.12s;

}

.history-rail-item:hover {

  background: var(--panel-2);

  border-color: var(--border);

}

.history-rail-item.active {

  background: rgba(110, 168, 254, 0.1);

  border-color: rgba(110, 168, 254, 0.45);

}

.item-title {

  display: block;

  font-size: 12px;

  font-weight: 600;

  white-space: nowrap;

  overflow: hidden;

  text-overflow: ellipsis;

  padding-right: 8px;

}

.item-stage {

  display: block;

  font-size: 10px;

  color: var(--muted);

  margin-top: 2px;

}

.item-time {

  display: block;

  font-size: 10px;

  margin-top: 2px;

}

.item-dot {

  position: absolute;

  top: 10px;

  right: 8px;

  width: 6px;

  height: 6px;

  border-radius: 50%;

  background: var(--muted);

}

.item-dot.ok {

  background: var(--ok);

}

.item-dot.warn {

  background: var(--warn);

}

.item-dot.err {

  background: var(--danger);

}

.item-folder {

  position: absolute;

  top: 6px;

  right: 6px;

  border: none;

  background: transparent;

  cursor: pointer;

  font-size: 12px;

  line-height: 1;

  padding: 2px 4px;

  opacity: 0.7;

}

.item-folder:hover {

  opacity: 1;

}

.history-rail-item:has(.item-folder) .item-dot {

  right: 28px;

}

.history-rail-foot {

  flex-shrink: 0;

  padding: 10px 8px 14px;

  border-top: 1px solid var(--border);

}

.settings-btn {

  width: 100%;

  text-align: left;

  font-size: 13px;

  padding: 10px 12px;

}

.context-menu {

  position: fixed;

  z-index: 1000;

  min-width: 180px;

  padding: 6px;

  border-radius: 10px;

  background: var(--panel);

  border: 1px solid var(--border);

  box-shadow: var(--shadow);

}

.context-menu-item {

  display: block;

  width: 100%;

  text-align: left;

  border: none;

  background: transparent;

  color: var(--text);

  font: inherit;

  font-size: 13px;

  padding: 10px 12px;

  border-radius: 8px;

  cursor: pointer;

}

.context-menu-item:hover:not(:disabled) {

  background: var(--panel-2);

}

.context-menu-item.danger {

  color: var(--danger);

}

.context-menu-item:disabled {

  opacity: 0.45;

  cursor: not-allowed;

}

</style>


