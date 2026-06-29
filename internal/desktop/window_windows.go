//go:build windows

package desktop

import (
	"syscall"
	"unsafe"

	webview "github.com/jchv/go-webview2"
)

const (
	gwlStyle        = ^uintptr(15) // GWL_STYLE = -16
	wsCaption       = 0x00C00000
	wsThickFrame    = 0x00040000
	wsBorder        = 0x00800000
	wsDlgFrame      = 0x00400000
	wsMinimizeBox   = 0x00020000
	wsMaximizeBox   = 0x00010000
	swMinimize      = 6
	swMaximize      = 3
	swRestore       = 9
	wmClose         = 0x0010
	wmNCLButtonDown = 0x00A1
	htCaption       = 2
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	dwmapi             = syscall.NewLazyDLL("dwmapi.dll")
	procGetWindowLong  = user32.NewProc("GetWindowLongW")
	procSetWindowLong  = user32.NewProc("SetWindowLongW")
	procSetWindowPos   = user32.NewProc("SetWindowPos")
	procShowWindow     = user32.NewProc("ShowWindow")
	procIsZoomed       = user32.NewProc("IsZoomed")
	procReleaseCapture = user32.NewProc("ReleaseCapture")
	procSendMessage    = user32.NewProc("SendMessageW")
	procPostMessage    = user32.NewProc("PostMessageW")
	procDwmSetWindowAttribute      = dwmapi.NewProc("DwmSetWindowAttribute")
	procDwmExtendFrameIntoClientArea = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
)

type dwmMargins struct {
	cxLeftWidth, cxRightWidth, cyTopHeight, cyBottomHeight int32
}

func showWindow(url string) error {
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("FlowAgent Studio")
	w.SetSize(1280, 860, webview.HintNone)

	hwnd := uintptr(w.Window())
	removeWindowFrame(hwnd)

	_ = w.Bind("flowDesktop", func(action string) {
		switch action {
		case "minimize":
			_, _, _ = procShowWindow.Call(hwnd, swMinimize)
		case "maximize":
			if zoomed, _, _ := procIsZoomed.Call(hwnd); zoomed != 0 {
				_, _, _ = procShowWindow.Call(hwnd, swRestore)
			} else {
				_, _, _ = procShowWindow.Call(hwnd, swMaximize)
			}
		case "close":
			_, _, _ = procPostMessage.Call(hwnd, wmClose, 0, 0)
		case "drag":
			_, _, _ = procReleaseCapture.Call()
			_, _, _ = procSendMessage.Call(hwnd, wmNCLButtonDown, htCaption, 0)
		}
	})

	w.Navigate(url)
	w.Run()
	return nil
}

func removeWindowFrame(hwnd uintptr) {
	style, _, _ := procGetWindowLong.Call(hwnd, gwlStyle)
	newStyle := style &^ (wsCaption | wsBorder | wsDlgFrame)
	newStyle |= wsThickFrame | wsMinimizeBox | wsMaximizeBox
	_, _, _ = procSetWindowLong.Call(hwnd, gwlStyle, newStyle)
	// SWP_FRAMECHANGED | SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_NOACTIVATE
	_, _, _ = procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, 0x0020|0x0002|0x0001|0x0004)

	// 深色无边框：去掉 DWM 浅色描边
	val := int32(1)
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&val)), unsafe.Sizeof(val))
	// DWMWA_WINDOW_CORNER_PREFERENCE = 33, DWMWCP_DONOTROUND = 1
	corner := int32(1)
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&corner)), unsafe.Sizeof(corner))
	margins := dwmMargins{-1, -1, -1, -1}
	_, _, _ = procDwmExtendFrameIntoClientArea.Call(hwnd, uintptr(unsafe.Pointer(&margins)))
}
