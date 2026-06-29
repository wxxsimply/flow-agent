export const STAGES = [
  'assemble',
  'expand',
  'script',
  'character',
  'storyboard',
  'produce',
  'comply',
  'publish',
  'finished',
] as const

export const STAGE_LABELS: Record<string, string> = {
  expand: '剧情扩写',
  script: '剧本',
  character: '角色设计',
  storyboard: '分镜',
  produce: '素材生成',
  assemble: '镜头组装',
  comply: '合规检查',
  publish: '导出发布',
  finished: '完成',
  created: '已创建',
  awaiting_review: '待审阅',
}

export const STUDIO_STACK_SAMPLE = 'micro-movie-cap5'
export const STUDIO_STACK_FINAL = 'micro-movie-seedance'
/** Studio 默认视频方案（样片） */
export const STUDIO_STACK = STUDIO_STACK_SAMPLE

export interface ProviderUserCreds {
  api_key?: string
  base_url?: string
  access_key?: string
  secret_key?: string
  app_id?: string
  region?: string
}

export interface ProvidersUserCreds {
  volcengine?: ProviderUserCreds
  dashscope?: ProviderUserCreds
  kling?: ProviderUserCreds
  gemini?: ProviderUserCreds
  openai?: ProviderUserCreds
  deepseek?: ProviderUserCreds
}

/** @deprecated use ProviderUserCreds */
export type VolcengineUserCreds = ProviderUserCreds

export interface StackSummary {
  name: string
  label?: string
  description: string
  cost_hint?: string
  cost_mode?: 'cap' | 'per_30_sec' | string
  target_duration_sec: number
  image_provider: string
  video_provider: string
  tts_provider: string
  required_hints: string[]
}

const SAMPLE_STACK_IDS = new Set([
  STUDIO_STACK_SAMPLE,
  'micro-movie-economy',
  'micro-movie-wan-quick',
])

export function isSampleStack(profile?: string): boolean {
  if (!profile) return true
  return SAMPLE_STACK_IDS.has(profile)
}

export function stackProfileLabel(profile?: string, stacks?: StackSummary[]): string {
  if (!profile) return '样片'
  const fromApi = stacks?.find((s) => s.name === profile)?.label
  if (fromApi) return fromApi
  if (isSampleStack(profile)) return '样片'
  if (profile.startsWith('micro-movie-')) return '成片'
  return profile
}

export function durationBoundsForStack(profile?: string): {
  min: number
  max: number
  step: number
  defaultSec: number
} {
  if (isSampleStack(profile)) {
    return { min: 12, max: 24, step: 6, defaultSec: 18 }
  }
  return { min: 30, max: 180, step: 6, defaultSec: 30 }
}

export function budgetHintForStack(profile: string | undefined, durationSec: number): string {
  if (isSampleStack(profile)) {
    return '预算封顶 5 元'
  }
  const yuan = Math.round((durationSec / 30) * 5 * 10) / 10
  return `预算约 ${yuan} 元（每 30 秒 5 元）`
}

export interface ThemeOption {
  id: string
  label: string
  hint?: string
}

export interface VoiceOption {
  id: string
  label: string
  description: string
  speed_ratio: number
}

export interface CustomMediaProvider {
  id: string
  label: string
  adapter: string
  api_key?: string
  secret_key?: string
  base_url?: string
  image_model?: string
  video_model?: string
}

export interface MediaAdapterInfo {
  id: string
  label: string
  default_base_url?: string
  needs_secret_key?: boolean
}

export const MEDIA_ADAPTER_OPTIONS: MediaAdapterInfo[] = [
  { id: 'openai', label: 'OpenAI / Sora', default_base_url: 'https://api.openai.com/v1' },
  { id: 'volcengine', label: '火山 Seedance', default_base_url: 'https://ark.cn-beijing.volces.com/api/v3' },
  { id: 'kling', label: '可灵 Kling', default_base_url: 'https://api-beijing.klingai.com', needs_secret_key: true },
  { id: 'gemini', label: 'Google Gemini / Veo', default_base_url: 'https://generativelanguage.googleapis.com/v1beta' },
  { id: 'dashscope', label: '百炼 / 万相', default_base_url: '' },
]

export interface ConfigStatus {
  root: string
  stack_default: string
  stack_profile?: string
  media_configured: boolean
  volcengine_configured?: boolean
  required_providers?: string[]
  available_stacks?: StackSummary[]
  active_media_provider_label?: string
  providers_file: string
  providers_exists: boolean
  ffmpeg_available: boolean
  wmreward_ready: boolean
  desktop_mode: boolean
  desktop_window_controls?: boolean
  auth_enabled?: boolean
}

export interface UserPrefs {
  workspace_dir?: string
  default_output_dir?: string
  stack_profile?: string
  providers?: ProvidersUserCreds
  volcengine?: ProviderUserCreds
  volcengine_configured?: boolean
  media_configured?: boolean
  custom_media_providers?: CustomMediaProvider[]
  active_media_provider_id?: string
  active_media_provider_label?: string
  required_providers?: string[]
}

export type RunPhase = 'idle' | 'running' | 'review' | 'finished' | 'failed' | 'interrupted'

export interface AuthStatus {
  auth_enabled: boolean
  logged_in: boolean
}

export interface UserInfo {
  id: string
  phone: string
  created_at: string
}

export interface RunHistoryItem {
  run_id: string
  title: string
  stage: string
  stack_profile?: string
  started_at: string
  updated_at: string
  finished: boolean
  failed: boolean
  awaiting_review: boolean
  has_master_video: boolean
  run_dir?: string
}

export interface CreateRunResponse {
  run_id: string
  run_dir: string
}

export interface RunStatus {
  run_id: string
  run_dir?: string
  stage: string
  finished: boolean
  failed?: boolean
  awaiting_review?: boolean
  interrupted?: boolean
  workflow_active?: boolean
  resume_stage?: string
  dry_run: boolean
  stack_profile?: string
  master_video?: string
  brief_path?: string
  storyboard_path?: string
  expand_path?: string
  brief_runes_min?: number
  error?: string
}

export interface ReviewShot {
  id?: string
  shot_size?: string
  duration_sec?: number
  narrative_beat?: string
  brief_excerpt?: string
  camera_angle?: string
  scene_background?: string
  character_motion?: string
  micro_expression?: string
  action_behavior?: string
  narration?: string
  visual_prompt?: string
  action_beats?: string[]
  physics_cues?: string
  forbidden_physics?: string
}

export interface ShotLanguageExpandPayload {
  opening_shot?: string
  shot_language_brief?: string
  story_background?: string
  mood?: string
  tone?: string
  shots?: ReviewShot[]
}

export interface StoryboardShot {
  id?: string
  duration_sec?: number
  narration?: string
  visual_prompt?: string
  physics_cues?: string
  forbidden_physics?: string
}

export interface ChatMessage {
  role: 'user' | 'agent'
  text: string
}

export interface StageItem {
  id: string
  label: string
  state: 'pending' | 'active' | 'done'
}

export function buildStageItems(current: string): StageItem[] {
  const reviewMode = current === 'awaiting_review'
  const idx = STAGES.indexOf(current as (typeof STAGES)[number])
  const items: StageItem[] = []

  for (let i = 0; i < STAGES.length; i++) {
    const st = STAGES[i]
    if (st === 'finished' && current !== 'finished') continue
    let state: StageItem['state'] = 'pending'
    if (st === current) state = 'active'
    else if (reviewMode && st === 'assemble') state = 'done'
    else if (idx >= 0 && i < idx) state = 'done'
    items.push({ id: st, label: STAGE_LABELS[st] || st, state })
  }

  if (reviewMode) {
    items.push({ id: 'awaiting_review', label: STAGE_LABELS.awaiting_review, state: 'active' })
  } else if (current && !STAGES.includes(current as (typeof STAGES)[number]) && current !== 'created') {
    items.unshift({ id: current, label: STAGE_LABELS[current] || current, state: 'active' })
  }
  return items
}
