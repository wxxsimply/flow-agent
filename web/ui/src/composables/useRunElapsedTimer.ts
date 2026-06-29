import { computed, onUnmounted, ref, watch, type Ref } from 'vue'

/** 任务进行中每秒递增，供对话区展示「已工作 x 分 y 秒」。 */
export function useRunElapsedTimer(active: Ref<boolean>) {
  const elapsedSec = ref(0)
  let interval: ReturnType<typeof setInterval> | null = null

  function stop() {
    if (interval) {
      clearInterval(interval)
      interval = null
    }
  }

  function start() {
    stop()
    elapsedSec.value = 0
    interval = setInterval(() => {
      elapsedSec.value++
    }, 1000)
  }

  watch(
    active,
    (on) => {
      if (on) start()
      else stop()
    },
    { immediate: true },
  )

  onUnmounted(stop)

  const label = computed(() => {
    const m = Math.floor(elapsedSec.value / 60)
    const s = elapsedSec.value % 60
    return `已工作 ${m} 分 ${s} 秒`
  })

  return { label }
}
