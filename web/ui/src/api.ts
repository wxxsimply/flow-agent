import { authHeaders, clearToken, setToken } from './auth'
import type {
  AuthStatus,
  ConfigStatus,
  CreateRunResponse,
  MediaAdapterInfo,
  RunHistoryItem,
  RunStatus,
  ShotLanguageExpandPayload,
  ThemeOption,
  StackSummary,
  UserInfo,
  UserPrefs,
  VoiceOption,
} from './constants'

async function parseError(res: Response): Promise<string> {
  const text = await res.text()
  try {
    const j = JSON.parse(text)
    if (typeof j === 'string') return j
    if (j.error) return String(j.error)
  } catch {
    // plain text
  }
  return text || res.statusText
}

async function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = { ...authHeaders(), ...(init.headers as Record<string, string> | undefined) }
  return fetch(path, { ...init, headers })
}

export async function fetchAuthStatus(): Promise<AuthStatus> {
  const res = await fetch('/api/auth/status')
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function fetchMe(): Promise<UserInfo> {
  const res = await apiFetch('/api/auth/me')
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function sendSMSCode(phone: string): Promise<void> {
  const res = await fetch('/api/auth/sms/send', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone }),
  })
  if (!res.ok) throw new Error(await parseError(res))
}

export async function loginWithSMS(phone: string, code: string): Promise<UserInfo> {
  const res = await fetch('/api/auth/sms/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone, code }),
  })
  if (!res.ok) throw new Error(await parseError(res))
  const data = await res.json()
  if (data.token) setToken(data.token)
  return data.user
}

export function logout(): void {
  clearToken()
}

export async function fetchThemes(): Promise<ThemeOption[]> {
  const res = await fetch('/api/themes')
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function fetchVoices(): Promise<VoiceOption[]> {
  const res = await fetch('/api/voices')
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function fetchConfigStacks(): Promise<StackSummary[]> {
  const res = await fetch('/api/config/stacks')
  if (!res.ok) throw new Error(await parseError(res))
  const data = await res.json()
  return data.stacks || []
}

export async function fetchConfigStatus(): Promise<ConfigStatus> {
  const res = await fetch('/api/config/status')
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function fetchMediaAdapters(): Promise<MediaAdapterInfo[]> {
  const res = await fetch('/api/config/media-adapters')
  if (!res.ok) throw new Error(await parseError(res))
  const data = await res.json()
  return data.adapters || []
}

export async function fetchRunHistory(): Promise<RunHistoryItem[]> {
  const res = await apiFetch('/api/runs')
  if (!res.ok) throw new Error(await parseError(res))
  const data = await res.json()
  return data.runs || []
}

export async function deleteRun(runId: string): Promise<void> {
  const res = await apiFetch(`/api/runs/${runId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await parseError(res))
}

export async function createMicroMovie(body: Record<string, unknown>): Promise<CreateRunResponse> {
  const res = await apiFetch('/api/micro-movie', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function fetchRunStatus(runId: string): Promise<RunStatus> {
  const res = await apiFetch(`/api/runs/${runId}`)
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function fetchArtifactText(path: string): Promise<string> {
  const res = await apiFetch(path)
  if (!res.ok) throw new Error(await parseError(res))
  return res.text()
}

export async function patchRunArtifacts(
  runId: string,
  body: {
    brief?: string
    storyboard?: { shots: unknown[] }
    shot_language_expand?: ShotLanguageExpandPayload
  },
): Promise<void> {
  const res = await apiFetch(`/api/runs/${runId}/artifacts`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await parseError(res))
}

export async function resumeRun(
  runId: string,
  body: { from_stage?: string; auto_gate?: boolean },
): Promise<void> {
  const res = await apiFetch(`/api/runs/${runId}/resume`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await parseError(res))
}

export async function fetchArtifactBlob(url: string): Promise<Blob> {
  const res = await apiFetch(url)
  if (!res.ok) throw new Error(await parseError(res))
  return res.blob()
}

export async function openDirectory(path: string, kind: 'workspace' | 'project' = 'workspace'): Promise<void> {
  const res = await apiFetch('/api/utils/open-dir', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path, kind }),
  })
  if (!res.ok) throw new Error(await parseError(res))
}

export async function openRunFolder(runId: string): Promise<void> {
  const res = await apiFetch(`/api/runs/${runId}/open-folder`, { method: 'POST' })
  if (!res.ok) throw new Error(await parseError(res))
}

export async function fetchUserPrefs(): Promise<UserPrefs> {
  const res = await fetch('/api/user/prefs')
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function saveUserPrefs(prefs: UserPrefs): Promise<UserPrefs> {
  const res = await fetch('/api/user/prefs', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(prefs),
  })
  if (!res.ok) throw new Error(await parseError(res))
  return res.json()
}

export async function validateOutputDir(path: string, mode: 'workspace' | 'project' = 'workspace'): Promise<string> {
  const res = await fetch('/api/utils/validate-dir', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path, mode }),
  })
  if (!res.ok) throw new Error(await parseError(res))
  const data = await res.json()
  return data.path
}

export async function pickOutputFolder(title?: string): Promise<string | null> {
  const res = await fetch('/api/dialog/pick-folder', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: title || '选择视频工作区目录' }),
  })
  if (res.status === 404) return null
  if (!res.ok) throw new Error(await parseError(res))
  const data = await res.json()
  if (data.cancelled) return null
  return data.path
}

export async function iterateRun(
  runId: string,
  body: Record<string, unknown>,
): Promise<void> {
  const res = await apiFetch(`/api/runs/${runId}/iterate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await parseError(res))
}

export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
