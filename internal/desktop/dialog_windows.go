//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

func showError(title, message string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	m, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	const mbIconError = 0x00000010
	_, _, _ = messageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), mbIconError)
}
