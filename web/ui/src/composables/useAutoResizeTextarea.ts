import { nextTick, onMounted, watch, type Ref } from 'vue'

/** 将 textarea 高度随内容自动撑开（审阅区多字段复用）。 */
export function useAutoResizeTextarea(
  sources: Array<Ref<string> | Ref<unknown>>,
  rootRef: Ref<HTMLElement | null>,
) {
  function resizeAll() {
    const root = rootRef.value
    if (!root) return
    root.querySelectorAll('textarea').forEach((node) => {
      const el = node as HTMLTextAreaElement
      el.style.height = 'auto'
      el.style.height = `${Math.max(el.scrollHeight, 44)}px`
    })
  }

  onMounted(() => nextTick(resizeAll))

  for (const src of sources) {
    watch(
      src,
      () => nextTick(resizeAll),
      { deep: true },
    )
  }

  function onInput(event: Event) {
    const el = event.target as HTMLTextAreaElement | null
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.max(el.scrollHeight, 44)}px`
  }

  return { onInput, resizeAll }
}
