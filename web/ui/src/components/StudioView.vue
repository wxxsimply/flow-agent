<script setup lang="ts">

import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import {

  createMicroMovie,

  fetchArtifactBlob,

  fetchRunStatus,

  iterateRun,

  openRunFolder,

  resumeRun,

  sleep,

} from '../api'

import {

  buildStageItems,

  STAGE_LABELS,

  type ChatMessage,

  type RunStatus,

} from '../constants'

import { useStudioCreativeOptions } from '../composables/useStudioCreativeOptions'

import { useStudioRunState } from '../composables/useStudioRunState'

import { useRunElapsedTimer } from '../composables/useRunElapsedTimer'

import ReviewCenterPanel from './ReviewCenterPanel.vue'



const props = defineProps<{

  loadRunId?: string | null

}>()



const { buildCreativeBody, initOptions, outputDir, persistWorkspaceDir, restoreWorkspaceDir, rememberWorkspaceFromRunDir } = useStudioCreativeOptions()

const { runPhase, polling, currentRunId } = useStudioRunState()

const { label: runElapsedLabel } = useRunElapsedTimer(polling)



const messages = ref<ChatMessage[]>([

  {

    role: 'agent',

    text: '你好，我是 FlowAgent。请告诉我你想拍什么——可以是一镜画面、一段旁白，或完整剧情梗概。\n示例：雨夜天台，少年握紧手机，霓虹在雨幕里模糊……',

  },

])



const prompt = ref('')

const dryRun = ref(false)

const autoGate = ref(false)



const runMeta = ref('尚未开始')

const currentStage = ref('')

const masterVideo = ref('')

const currentRunDir = ref('')

let masterVideoObjectUrl: string | null = null

function revokeMasterVideoUrl() {
  if (masterVideoObjectUrl) {
    URL.revokeObjectURL(masterVideoObjectUrl)
    masterVideoObjectUrl = null
  }
  masterVideo.value = ''
}

async function loadMasterVideo(url: string) {
  revokeMasterVideoUrl()
  try {
    const blob = await fetchArtifactBlob(`${url}?t=${Date.now()}`)
    masterVideoObjectUrl = URL.createObjectURL(blob)
    masterVideo.value = masterVideoObjectUrl
    showResult.value = true
  } catch {
    showResult.value = false
  }
}

const showResult = ref(false)

const reviewMode = ref(false)

const reviewBriefPath = ref('')

const reviewStoryboardPath = ref('')

const reviewExpandPath = ref('')

const reviewBriefRunesMin = ref(2000)

const messagesEl = ref<HTMLElement | null>(null)

const lastAnnouncedStage = ref('')

const resumeFromStage = ref('')

let pollGeneration = 0



const stageItems = computed(() => buildStageItems(currentStage.value))

const showProduceWaitHint = computed(() => polling.value && currentStage.value === 'produce')

const PRODUCE_WAIT_MSG = '视频生成通常需要 5–10 分钟，属正常现象，请勿关闭窗口或结束进程，请耐心等待'



const submitLabel = computed(() => {

  if (polling.value) return '生成中…'

  if (reviewMode.value) return '请先完成审阅'

  if (runPhase.value === 'interrupted' && resumeFromStage.value) return '继续生成'

  if (runPhase.value === 'failed' && resumeFromStage.value === 'produce') return '继续生成'

  if (runPhase.value === 'finished' || runPhase.value === 'failed') return '继续修改'

  return '开始创作'

})



const canSubmit = computed(() => {

  if (reviewMode.value || polling.value) return false

  if (runPhase.value === 'interrupted' && resumeFromStage.value) return true

  if (runPhase.value === 'failed' && resumeFromStage.value === 'produce' && !prompt.value.trim()) return true

  if (runPhase.value === 'finished' || runPhase.value === 'failed') return prompt.value.trim() !== ''

  return prompt.value.trim() !== '' && outputDir.value.trim() !== ''

})



function addMessage(role: ChatMessage['role'], text: string) {
  messages.value.push({ role, text })
  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
  })
}

function announceStage(stage: string) {

  if (!stage || stage === lastAnnouncedStage.value) return

  lastAnnouncedStage.value = stage

  if (stage === 'created') return

  const label = STAGE_LABELS[stage] || stage

  addMessage('agent', `▸ ${label}`)

}



function applyStatus(status: RunStatus) {

  runMeta.value = `run_id: ${status.run_id}\nstage: ${status.stage}`

  if (status.run_dir) {
    currentRunDir.value = status.run_dir
    runMeta.value += `\n输出: ${status.run_dir}`
  }

  currentStage.value = status.stage

  if (status.resume_stage) {
    resumeFromStage.value = status.resume_stage
  } else if (status.finished) {
    resumeFromStage.value = ''
  }

  if (status.interrupted) {

    runPhase.value = 'interrupted'

    reviewMode.value = false

    return 'interrupted'

  }

  if (status.failed) {

    runPhase.value = 'failed'

    reviewMode.value = false

    return 'failed'

  }

  if (status.awaiting_review || status.stage === 'awaiting_review') {

    runPhase.value = 'review'

    reviewMode.value = true

    reviewBriefPath.value = status.brief_path || ''

    reviewStoryboardPath.value = status.storyboard_path || ''

    reviewExpandPath.value = status.expand_path || ''

    reviewBriefRunesMin.value = status.brief_runes_min || 2000

    return 'review'

  }

  if (status.finished) {

    runPhase.value = 'finished'

    reviewMode.value = false

    if (status.master_video) {

      void loadMasterVideo(status.master_video)

    } else {

      revokeMasterVideoUrl()

      showResult.value = false

    }

    return 'finished'

  }

  runPhase.value = 'running'

  return 'running'

}



async function loadExistingRun(runId: string) {

  currentRunId.value = runId

  showResult.value = false

  reviewMode.value = false

  revokeMasterVideoUrl()

  lastAnnouncedStage.value = ''

  addMessage('agent', `正在加载历史任务 ${runId.slice(0, 8)}…`)

  try {

    const status: RunStatus = await fetchRunStatus(runId)

    if (status.run_dir) {
      rememberWorkspaceFromRunDir(status.run_dir)
    }

    const outcome = applyStatus(status)

    announceStage(status.stage)

    if (outcome === 'review') {

      addMessage('agent', '扩写与分镜已生成，请在下方审阅区编辑，确认后再生成视频。')

      return

    }

    if (outcome === 'finished') {

      addMessage('agent', '任务已完成，可在右侧预览成片。你可以继续输入修改意见并重新生成。')

      return

    }

    if (outcome === 'failed') {

      addMessage('agent', `任务失败：${status.error || status.stage}`)

      return

    }

    if (outcome === 'interrupted') {

      addMessage(

        'agent',

        `任务已中断（可能因欠费、关闭应用或进程异常）。分镜与已生成素材已保留，点击「继续生成」从 ${status.resume_stage || status.stage} 阶段恢复；或点「新建项目」换目录开始新创作。`,

      )

      return

    }

    addMessage('agent', '任务进行中，正在同步状态…')

    await pollRun(runId)

  } catch (err) {

    const msg = err instanceof Error ? err.message : String(err)

    addMessage('agent', `加载失败：${msg}`)

  }

}



watch(

  () => props.loadRunId,

  (id, prev) => {

    if (!id) return

    if (id === prev && id === currentRunId.value) {

      void loadExistingRun(id)

      return

    }

    loadExistingRun(id)

  },

)



watch(polling, (active) => {
  if (!active) return
  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
  })
})



onUnmounted(() => {

  revokeMasterVideoUrl()

})



async function pollRun(runId: string) {

  const gen = ++pollGeneration

  polling.value = true

  runPhase.value = 'running'

  for (;;) {

    if (gen !== pollGeneration) break

    const status: RunStatus = await fetchRunStatus(runId)

    announceStage(status.stage)

    const outcome = applyStatus(status)

    if (outcome === 'failed') {

      addMessage('agent', `任务失败：${status.error || status.stage || '未知错误'}`)

      break

    }

    if (outcome === 'interrupted') {

      addMessage(

        'agent',

        `任务已中断（可能因欠费、关闭应用或进程异常）。点击「继续生成」恢复；或「新建项目」换目录开始新创作。`,

      )

      break

    }

    if (outcome === 'review') {

      addMessage('agent', '扩写与分镜已生成，请在下方审阅区编辑，确认后再生成视频。')

      break

    }

    if (outcome === 'finished') {

      addMessage(

        'agent',

        '创作完成！你可以在右侧预览成片，或打开输出目录查看全部产物。\n你可以继续输入修改意见，我会基于当前项目重新扩写分镜。',

      )

      break

    }

    await sleep(2000)

  }

  if (gen === pollGeneration) polling.value = false

}



async function retryProduce() {

  if (!currentRunId.value || !resumeFromStage.value) return

  addMessage('agent', '正在从上次进度继续生成视频…')

  showResult.value = false

  revokeMasterVideoUrl()

  lastAnnouncedStage.value = ''

  try {

    await resumeRun(currentRunId.value, {

      from_stage: resumeFromStage.value,

      auto_gate: autoGate.value,

    })

    await pollRun(currentRunId.value)

  } catch (err) {

    const msg = err instanceof Error ? err.message : String(err)

    addMessage('agent', `继续生成失败：${msg}`)

    polling.value = false

  }

}



async function startRun() {

  const plot = prompt.value.trim()

  if (!canSubmit.value) return



  if (runPhase.value === 'interrupted' && resumeFromStage.value && currentRunId.value) {

    await retryProduce()

    return

  }

  if (

    runPhase.value === 'failed' &&

    resumeFromStage.value === 'produce' &&

    currentRunId.value &&

    !plot

  ) {

    await retryProduce()

    return

  }

  if (!plot) return



  addMessage('user', plot)

  prompt.value = ''



  const isIterate = runPhase.value === 'finished' || runPhase.value === 'failed'

  if (isIterate && currentRunId.value) {

    addMessage('agent', '收到修改意见，正在基于当前项目重新扩写分镜…')

    showResult.value = false

    reviewMode.value = false

    revokeMasterVideoUrl()

    lastAnnouncedStage.value = ''

    try {

      await iterateRun(currentRunId.value, {

        ...buildCreativeBody(plot),

        dry_run: dryRun.value,

        auto_gate: autoGate.value,

        stop_after_stage: 'assemble',

      })

      await pollRun(currentRunId.value)

    } catch (err) {

      const msg = err instanceof Error ? err.message : String(err)

      addMessage('agent', `迭代失败：${msg}`)

    }

    return

  }



  if (!outputDir.value.trim()) {

    addMessage('agent', '请先在左侧边栏选择工作区目录。')

    return

  }



  addMessage('agent', '收到，正在扩写剧情并生成分镜…')

  showResult.value = false

  reviewMode.value = false

  revokeMasterVideoUrl()

  lastAnnouncedStage.value = ''



  try {

    const workspace = outputDir.value.trim()

    const created = await createMicroMovie({

      input_mode: 'director',

      shots: [],

      bgm: 'auto',

      output_dir: workspace,

      ...buildCreativeBody(plot),

      dry_run: dryRun.value,

      auto_gate: autoGate.value,

      stop_after_stage: 'assemble',

    })

    currentRunId.value = created.run_id
    if (workspace) {
      void persistWorkspaceDir(workspace)
    }

    runMeta.value = `run_id: ${created.run_id}\n输出: ${created.run_dir}`

    currentStage.value = 'created'

    runPhase.value = 'running'

    await pollRun(created.run_id)

  } catch (err) {

    const msg = err instanceof Error ? err.message : String(err)

    if (msg.includes('workspace') || msg.includes('project directory')) {

      addMessage(

        'agent',

        '请选择工作区目录（父文件夹），不要选择已有项目的子文件夹。可在设置中配置默认工作区。',

      )

    } else if (msg.includes('already contains a project')) {

      addMessage(

        'agent',

        '该输出目录已有进行中的项目。请点击「继续生成」恢复，或「新建项目」后选择空目录。',

      )

    } else {

      addMessage('agent', `创建失败：${msg}`)

    }

    runPhase.value = 'idle'

  }

}



function startNewProject() {

  pollGeneration++

  polling.value = false

  currentRunId.value = null

  runPhase.value = 'idle'

  showResult.value = false

  reviewMode.value = false

  revokeMasterVideoUrl()

  currentRunDir.value = ''

  currentStage.value = ''

  runMeta.value = '尚未开始'

  lastAnnouncedStage.value = ''

  resumeFromStage.value = ''

  void restoreWorkspaceDir()

  addMessage('agent', '已切换到新项目。工作区目录已恢复，请输入新的创作指令。')

}



async function onReviewResumed(runId: string) {

  reviewMode.value = false

  runPhase.value = 'running'

  addMessage('agent', `已确认分镜，正在生成视频素材…${PRODUCE_WAIT_MSG}。`)

  currentStage.value = 'produce'

  lastAnnouncedStage.value = ''

  await pollRun(runId)

}



function onReviewSaved() {

  addMessage('agent', '草稿已保存。')

}



function exitReview() {

  reviewMode.value = false

  runPhase.value = 'idle'

  addMessage('agent', '已退出审阅，可随时重新加载任务继续。')

}



async function handleOpenFolder() {

  if (!currentRunId.value) return

  try {

    await openRunFolder(currentRunId.value)

  } catch (err) {

    const msg = err instanceof Error ? err.message : String(err)

    addMessage('agent', `打开目录失败：${msg}`)

  }

}



function onPromptKeydown(e: KeyboardEvent) {

  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {

    e.preventDefault()

    startRun()

  }

}



onMounted(async () => {

  await initOptions()

})

defineExpose({ startNewProject })

</script>



<template>

  <section class="view active">

    <header class="topbar">

      <div class="topbar-main">

        <h1>指挥 Agent 创作微电影</h1>

        <p class="muted">描述第一镜或剧情意图，Agent 将扩写并生成分镜；确认后再合成成片。</p>

      </div>

      <div class="topbar-actions">

        <button

          v-if="currentRunId"

          type="button"

          class="ghost"

          @click="startNewProject"

        >

          新建项目

        </button>

        <label class="toggle"><input v-model="dryRun" type="checkbox" /> Dry-run</label>

        <label class="toggle"><input v-model="autoGate" type="checkbox" /> 自动门禁</label>

      </div>

    </header>



    <div class="workspace" :class="{ 'review-mode': reviewMode }">

      <ReviewCenterPanel
        v-if="reviewMode && currentRunId"
        class="chat-panel review-panel"
        :run-id="currentRunId"
        :brief-path="reviewBriefPath"
        :storyboard-path="reviewStoryboardPath"
        :expand-path="reviewExpandPath"
        :brief-runes-min="reviewBriefRunesMin"
        @saved="onReviewSaved"
        @resumed="onReviewResumed"
        @cancel="exitReview"
      />

      <div v-else class="chat-panel">

        <div ref="messagesEl" class="messages">

          <div v-for="(msg, i) in messages" :key="i" class="msg" :class="msg.role">

            <div class="msg-avatar">{{ msg.role === 'user' ? '你' : 'A' }}</div>

            <div class="msg-body">

              <p v-for="(line, j) in msg.text.split('\n')" :key="j">{{ line }}</p>

            </div>

          </div>

          <div v-if="polling" class="msg agent work-timer">

            <div class="msg-avatar work-timer-avatar">A</div>

            <div class="msg-body work-timer-body">

              <p v-if="showProduceWaitHint" class="work-timer-wait">

                {{ PRODUCE_WAIT_MSG }}

              </p>

              <p class="work-timer-text">{{ runElapsedLabel }}</p>

              <p v-if="currentStage" class="work-timer-stage muted">

                当前阶段：{{ STAGE_LABELS[currentStage] || currentStage }}

              </p>

            </div>

          </div>

        </div>



        <div class="composer">

          <textarea

            v-model="prompt"

            rows="3"

            :placeholder="

              polling

                ? '视频生成中，可先输入修改意见，完成后点击提交…'

                : runPhase === 'finished' || runPhase === 'failed'

                  ? '输入修改意见或新剧情…'

                  : '输入创作指令或第一镜文本…'

            "

            @keydown="onPromptKeydown"

          />

          <div class="composer-bar">

            <p v-if="polling" class="composer-hint muted">

              生成进行中，对话记录会保留；完成后可一键继续修改

            </p>

            <p v-else-if="runPhase === 'finished'" class="composer-hint muted">

              继续修改将基于当前项目重跑分镜与成片

            </p>

            <button class="primary" :disabled="!canSubmit" @click="startRun">

              {{ submitLabel }}

            </button>

          </div>

        </div>

      </div>



      <aside class="run-panel">

        <h2>任务进度</h2>

        <div class="run-meta muted">{{ runMeta }}</div>

        <div v-if="currentRunId" class="run-dir-actions">

          <button type="button" class="ghost" @click="handleOpenFolder">打开项目目录</button>

        </div>

        <p v-if="runPhase === 'interrupted' || (runPhase === 'failed' && resumeFromStage)" class="review-hint warn-hint">

          任务未完成。点击「继续生成」从 {{ resumeFromStage || '上次' }} 阶段恢复，或「新建项目」换目录。

        </p>

        <p v-else-if="reviewMode" class="review-hint muted">

          在中间区域编辑扩写正文或分镜，完成后点击确认生成视频。

        </p>

        <p v-else-if="showProduceWaitHint && !reviewMode" class="produce-wait-hint">

          {{ PRODUCE_WAIT_MSG }}

        </p>

        <ol class="stages">

          <li v-for="st in stageItems" :key="st.id" :class="st.state">{{ st.label }}</li>

        </ol>

        <div v-if="showResult && !reviewMode" class="result">

          <video :src="masterVideo" controls playsinline />

          <div class="result-actions">

            <button class="ghost" @click="handleOpenFolder">打开输出目录</button>

          </div>

        </div>

      </aside>

    </div>

  </section>

</template>



<style scoped>

.stages li.pending {

  opacity: 0.65;

}

.composer-bar {

  flex-direction: column;

  align-items: stretch;

}

.composer-hint {

  margin: 0 0 8px;

  font-size: 12px;

}

.work-timer-body {
  border-color: rgba(110, 168, 254, 0.35);
  background: rgba(110, 168, 254, 0.08);
}

.work-timer-text {
  margin: 0;
  font-weight: 600;
  color: var(--accent);
}

.work-timer-stage {
  margin: 6px 0 0;
  font-size: 12px;
}

.work-timer-wait {
  margin: 10px 0 0;
  padding: 8px 10px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--warn);
  background: rgba(255, 204, 102, 0.12);
  border: 1px solid rgba(255, 204, 102, 0.45);
}

.produce-wait-hint {
  font-size: 12px;
  line-height: 1.5;
  margin: 0 0 12px;
  padding: 10px 12px;
  border-radius: 10px;
  font-weight: 600;
  color: var(--warn);
  background: rgba(255, 204, 102, 0.12);
  border: 1px solid rgba(255, 204, 102, 0.45);
}

.topbar-main {
  flex: 1;
  min-width: 0;
}

.work-timer-avatar {
  animation: work-pulse 1.6s ease-in-out infinite;
}

@keyframes work-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.55;
  }
}

.warn-hint {

  color: var(--warn, #e8a838);

  font-size: 13px;

  line-height: 1.45;

}

</style>

