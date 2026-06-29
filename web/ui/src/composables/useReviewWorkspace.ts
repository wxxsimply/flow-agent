import { computed, ref, watch, type Ref } from 'vue'
import { fetchArtifactText, patchRunArtifacts, resumeRun } from '../api'
import type { ReviewShot, ShotLanguageExpandPayload } from '../constants'

export type ReviewTab = 'brief' | 'storyboard'

function extractBriefBody(raw: string): string {
  const text = raw.trim()
  if (!text) return ''
  const sep = '\n---\n'
  const idx = text.indexOf(sep)
  if (idx >= 0) return text.slice(idx + sep.length).trim()
  return text
}

function countRunes(text: string): number {
  return [...text].length
}

function transitionHint(current: ReviewShot, next?: ReviewShot): string {
  if (!next) return ''
  const beat = (current.narrative_beat || '').trim()
  const nextBeat = (next.narrative_beat || '').trim()
  if (beat && nextBeat && beat !== nextBeat) {
    return `剧情过渡：${beat} → ${nextBeat}`
  }
  const cam = (current.camera_angle || '').trim()
  const nextScene = (next.scene_background || '').trim()
  const curScene = (current.scene_background || '').trim()
  if (cam && nextScene && curScene && nextScene !== curScene) {
    return `过渡：${cam} → 硬切至「${nextScene.slice(0, 40)}…」`
  }
  if (cam) return `过渡：${cam} → 下一镜`
  return ''
}

export function useReviewWorkspace(
  runId: Ref<string>,
  briefPath: Ref<string | undefined>,
  storyboardPath: Ref<string | undefined>,
  expandPath: Ref<string | undefined>,
  briefRunesMin: Ref<number>,
) {
  const briefText = ref('')
  const expandPayload = ref<ShotLanguageExpandPayload>({})
  const storyBackground = ref('')
  const mood = ref('')
  const tone = ref('')
  const shots = ref<ReviewShot[]>([])
  const loading = ref(true)
  const saving = ref(false)
  const resuming = ref(false)
  const error = ref('')
  const activeTab = ref<ReviewTab>('brief')

  const briefRuneCount = computed(() => countRunes(briefText.value))
  const briefBelowMin = computed(
    () => briefRuneCount.value < briefRunesMin.value && briefRuneCount.value > 0,
  )

  async function loadArtifacts() {
    loading.value = true
    error.value = ''
    try {
      let expand: ShotLanguageExpandPayload | null = null
      if (expandPath.value) {
        const raw = await fetchArtifactText(expandPath.value)
        expand = JSON.parse(raw) as ShotLanguageExpandPayload
        expandPayload.value = { ...expand }
        storyBackground.value = expand.story_background || ''
        mood.value = expand.mood || ''
        tone.value = expand.tone || ''
        if (expand.shot_language_brief) {
          briefText.value = expand.shot_language_brief
        }
        shots.value = (expand.shots || []).map((s) => ({ ...s }))
      }
      if (!briefText.value && briefPath.value) {
        const md = await fetchArtifactText(briefPath.value)
        briefText.value = extractBriefBody(md)
      }
      if (shots.value.length === 0 && storyboardPath.value) {
        const raw = await fetchArtifactText(storyboardPath.value)
        const sb = JSON.parse(raw) as { shots?: ReviewShot[] }
        shots.value = (sb.shots || []).map((s) => ({ ...s }))
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  watch(
    () => [runId.value, briefPath.value, storyboardPath.value, expandPath.value],
    () => loadArtifacts(),
    { immediate: true },
  )

  function buildPatchBody() {
    const payload: ShotLanguageExpandPayload = {
      ...expandPayload.value,
      shot_language_brief: briefText.value,
      story_background: storyBackground.value,
      mood: mood.value,
      tone: tone.value,
      shots: shots.value,
    }
    return {
      brief: briefText.value,
      shot_language_expand: payload,
    }
  }

  async function saveDraft(): Promise<boolean> {
    if (!runId.value) return false
    saving.value = true
    error.value = ''
    try {
      await patchRunArtifacts(runId.value, buildPatchBody())
      expandPayload.value = { ...expandPayload.value, ...buildPatchBody().shot_language_expand }
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    } finally {
      saving.value = false
    }
  }

  async function confirmAndProduce(): Promise<boolean> {
    if (!runId.value) return false
    resuming.value = true
    error.value = ''
    try {
      await patchRunArtifacts(runId.value, buildPatchBody())
      await resumeRun(runId.value, { from_stage: 'produce', auto_gate: true })
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    } finally {
      resuming.value = false
    }
  }

  function updateShotField(index: number, field: keyof ReviewShot, value: string | string[] | number) {
    const next = [...shots.value]
    next[index] = { ...next[index], [field]: value }
    shots.value = next
  }

  function updateExpandMeta(field: 'story_background' | 'mood' | 'tone', value: string) {
    if (field === 'story_background') storyBackground.value = value
    if (field === 'mood') mood.value = value
    if (field === 'tone') tone.value = value
  }

  function updateActionBeat(shotIndex: number, beatIndex: number, value: string) {
    const next = [...shots.value]
    const beats = [...(next[shotIndex].action_beats || [])]
    while (beats.length < 3) beats.push('')
    beats[beatIndex] = value
    next[shotIndex] = { ...next[shotIndex], action_beats: beats }
    shots.value = next
  }

  function shotTransition(index: number): string {
    return transitionHint(shots.value[index], shots.value[index + 1])
  }

  return {
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
    loadArtifacts,
    saveDraft,
    confirmAndProduce,
    updateShotField,
    updateExpandMeta,
    updateActionBeat,
    shotTransition,
  }
}
