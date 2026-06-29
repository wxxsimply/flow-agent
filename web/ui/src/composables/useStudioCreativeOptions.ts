import { computed, ref, watch } from 'vue'
import { fetchConfigStatus, fetchThemes, fetchUserPrefs, fetchVoices, saveUserPrefs } from '../api'
import {
  STUDIO_STACK,
  durationBoundsForStack,
  type ThemeOption,
  type VoiceOption,
} from '../constants'

const style = ref('2d')
const orientation = ref('portrait')
const visualTheme = ref('generic')
const narratorVoice = ref('')
const targetDurationSec = ref(18)
/** 工作区目录（新建项目 API 使用；勿与 run_dir 混用） */
const workspaceDir = ref('')
const stackProfile = ref(STUDIO_STACK)
const mediaConfigured = ref(false)
const activeMediaProviderLabel = ref('')
const desktopMode = ref(false)
const desktopWindowControls = ref(false)
const voices = ref<VoiceOption[]>([])
const themes = ref<ThemeOption[]>([])
let initialized = false

const durationBounds = computed(() => durationBoundsForStack(stackProfile.value))

function clampDurationForStack() {
  const b = durationBounds.value
  let v = targetDurationSec.value
  if (v < b.min) v = b.min
  if (v > b.max) v = b.max
  const steps = Math.round((v - b.min) / b.step)
  targetDurationSec.value = b.min + steps * b.step
}

watch(stackProfile, () => {
  const b = durationBounds.value
  if (targetDurationSec.value < b.min || targetDurationSec.value > b.max) {
    targetDurationSec.value = b.defaultSec
  } else {
    clampDurationForStack()
  }
})

/** 从项目子目录推断工作区（父目录） */
function parentWorkspace(runDir: string): string {
  const trimmed = runDir.trim().replace(/[\\/]+$/, '')
  const sep = trimmed.includes('\\') ? '\\' : '/'
  const idx = trimmed.lastIndexOf(sep)
  if (idx <= 0) return trimmed
  return trimmed.slice(0, idx)
}

export function useStudioCreativeOptions() {
  async function initOptions() {
    if (initialized) return
    initialized = true
    try {
      const cfg = await fetchConfigStatus()
      desktopMode.value = cfg.desktop_mode
      desktopWindowControls.value = !!cfg.desktop_window_controls
      if (cfg.stack_profile) stackProfile.value = cfg.stack_profile
      mediaConfigured.value = cfg.media_configured
      if (cfg.active_media_provider_label) {
        activeMediaProviderLabel.value = cfg.active_media_provider_label
      }
    } catch {
      desktopMode.value = false
    }
    await restoreWorkspaceDir()
    clampDurationForStack()
    voices.value = await fetchVoices()
    themes.value = await fetchThemes()
    const preferred = voices.value.find((v) => v.id === 'epic_male')
    narratorVoice.value = preferred?.id || voices.value[0]?.id || ''
    visualTheme.value = themes.value[0]?.id || 'generic'
  }

  async function refreshFromConfig() {
    try {
      const cfg = await fetchConfigStatus()
      if (cfg.stack_profile) stackProfile.value = cfg.stack_profile
      mediaConfigured.value = cfg.media_configured
      if (cfg.active_media_provider_label) {
        activeMediaProviderLabel.value = cfg.active_media_provider_label
      }
      clampDurationForStack()
    } catch {
      // ignore
    }
  }

  function applyWorkspace(path: string) {
    const ws = path.trim()
    if (!ws) return
    workspaceDir.value = ws
  }

  async function restoreWorkspaceDir() {
    try {
      const prefs = await fetchUserPrefs()
      const ws = prefs.workspace_dir || prefs.default_output_dir || ''
      applyWorkspace(ws)
      if (prefs.stack_profile) stackProfile.value = prefs.stack_profile
      if (prefs.media_configured != null) mediaConfigured.value = prefs.media_configured
      if (prefs.active_media_provider_label) {
        activeMediaProviderLabel.value = prefs.active_media_provider_label
      }
      clampDurationForStack()
    } catch {
      // ignore
    }
  }

  /** 加载历史/完成任务时，从 run_dir 记住工作区，不覆盖 picker 为项目路径 */
  function rememberWorkspaceFromRunDir(runDir?: string) {
    if (!runDir?.trim()) return
    const parent = parentWorkspace(runDir.trim())
    if (parent) {
      applyWorkspace(parent)
    }
  }

  async function persistWorkspaceDir(path: string) {
    const ws = path.trim()
    if (!ws) return
    applyWorkspace(ws)
    try {
      const existing = await fetchUserPrefs()
      await saveUserPrefs({
        workspace_dir: ws,
        default_output_dir: ws,
        stack_profile: existing.stack_profile || stackProfile.value,
        providers: existing.providers,
        volcengine: existing.volcengine || existing.providers?.volcengine,
      })
    } catch {
      // ignore
    }
  }

  function buildCreativeBody(plot: string) {
    clampDurationForStack()
    return {
      plot,
      style: style.value,
      orientation: orientation.value,
      theme: visualTheme.value,
      narrator_voice: narratorVoice.value,
      target_duration_sec: targetDurationSec.value,
      stack_profile: stackProfile.value || STUDIO_STACK,
    }
  }

  return {
    style,
    orientation,
    visualTheme,
    narratorVoice,
    targetDurationSec,
    stackProfile,
    durationBounds,
    mediaConfigured,
    activeMediaProviderLabel,
    /** @deprecated 绑定工作区；历史命名保留兼容 CreationOptionsPanel */
    outputDir: workspaceDir,
    workspaceDir,
    desktopMode,
    desktopWindowControls,
    voices,
    themes,
    initOptions,
    refreshFromConfig,
    restoreWorkspaceDir,
    rememberWorkspaceFromRunDir,
    persistWorkspaceDir,
    buildCreativeBody,
    clampDurationForStack,
  }
}
