import { ref } from 'vue'
import type { RunPhase } from '../constants'

const runPhase = ref<RunPhase>('idle')
const polling = ref(false)
const currentRunId = ref<string | null>(null)

export function useStudioRunState() {
  return {
    runPhase,
    polling,
    currentRunId,
  }
}
