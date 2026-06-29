/** 桌面 WebView 无边框窗口控制（由 Go flowDesktop Bind 注入）。 */
export function flowDesktop(action: 'minimize' | 'maximize' | 'close' | 'drag') {
  window.flowDesktop?.(action)
}

export function onDesktopDragMouseDown(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (target.closest('button, a, input, label, select, textarea')) return
  flowDesktop('drag')
}
