//go:build windows

package game

import (
	"errors"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	clipboardUser32           = windows.NewLazySystemDLL("user32.dll")
	clipboardKernel32         = windows.NewLazySystemDLL("kernel32.dll")
	clipboardOpen             = clipboardUser32.NewProc("OpenClipboard")
	clipboardClose            = clipboardUser32.NewProc("CloseClipboard")
	clipboardGet              = clipboardUser32.NewProc("GetClipboardData")
	clipboardSet              = clipboardUser32.NewProc("SetClipboardData")
	clipboardEmpty            = clipboardUser32.NewProc("EmptyClipboard")
	clipboardForegroundWindow = clipboardUser32.NewProc("GetForegroundWindow")
	clipboardWindowProcess    = clipboardUser32.NewProc("GetWindowThreadProcessId")
	clipboardGlobalSize       = clipboardKernel32.NewProc("GlobalSize")
	clipboardGlobalLock       = clipboardKernel32.NewProc("GlobalLock")
	clipboardGlobalUnlock     = clipboardKernel32.NewProc("GlobalUnlock")
	clipboardGlobalAlloc      = clipboardKernel32.NewProc("GlobalAlloc")
	clipboardGlobalFree       = clipboardKernel32.NewProc("GlobalFree")
	clipboardMoveMemory       = clipboardKernel32.NewProc("RtlMoveMemory")
)

const clipboardUnicodeText = 13 // CF_UNICODETEXT

// These calls occur only after an explicit paste shortcut or Copy Invite click.
// See Microsoft OpenClipboard/GetClipboardData/SetClipboardData documentation.
func readClipboardText() (string, error) {
	if err := clipboardMoveMemory.Find(); err != nil {
		return "", errors.New("clipboard memory copy unavailable")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if ok, _, _ := clipboardOpen.Call(0); ok == 0 {
		return "", errors.New("clipboard busy")
	}
	defer clipboardClose.Call()
	handle, _, _ := clipboardGet.Call(clipboardUnicodeText)
	if handle == 0 {
		return "", errors.New("clipboard has no text")
	}
	size, _, _ := clipboardGlobalSize.Call(handle)
	if size < 2 || size > maxClipboardTextBytes || size%2 != 0 {
		return "", errors.New("clipboard text size invalid")
	}
	ptr, _, _ := clipboardGlobalLock.Call(handle)
	if ptr == 0 {
		return "", errors.New("clipboard text unavailable")
	}
	defer clipboardGlobalUnlock.Call(handle)
	// Copy exactly the checked GlobalSize into an equally sized Go allocation.
	// Native handles/pointers remain native integers, never Go pointer views.
	// Pin the Go buffer during the synchronous Win32 call, then decode the copy
	// before unlocking. Clipboard-owned memory is never freed by this app.
	units := make([]uint16, int(size/2))
	var pin runtime.Pinner
	pin.Pin(&units[0])
	clipboardMoveMemory.Call(uintptr(unsafe.Pointer(&units[0])), ptr, size)
	pin.Unpin()
	return decodeClipboardTextUTF16(units)
}

func writeClipboardText(value string) error {
	if err := clipboardMoveMemory.Find(); err != nil {
		return errors.New("clipboard memory copy unavailable")
	}
	value, err := normalizePastedMultiplayerAddress(value)
	if err != nil {
		return err
	}
	units, err := syscall.UTF16FromString(value)
	if err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// SetClipboardData needs a non-NULL owner. Use only a foreground window
	// owned by this process; a focus switch must not borrow another app's HWND.
	hwnd, _, _ := clipboardForegroundWindow.Call()
	var pid uint32
	if hwnd == 0 {
		return errors.New("game window not focused")
	}
	clipboardWindowProcess.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid != uint32(os.Getpid()) {
		return errors.New("game window not focused")
	}
	handle, _, _ := clipboardGlobalAlloc.Call(0x0002, uintptr(len(units)*2)) // GMEM_MOVEABLE
	if handle == 0 {
		return errors.New("clipboard allocation failed")
	}
	owned := true
	defer func() {
		if owned {
			clipboardGlobalFree.Call(handle)
		}
	}()
	ptr, _, _ := clipboardGlobalLock.Call(handle)
	if ptr == 0 {
		return errors.New("clipboard allocation unavailable")
	}
	// GlobalAlloc above reserved this exact byte count. The synchronous native
	// copy avoids converting its uintptr address into a Go-managed pointer.
	var pin runtime.Pinner
	pin.Pin(&units[0])
	clipboardMoveMemory.Call(ptr, uintptr(unsafe.Pointer(&units[0])), uintptr(len(units)*2))
	pin.Unpin()
	clipboardGlobalUnlock.Call(handle)
	if ok, _, _ := clipboardOpen.Call(hwnd); ok == 0 {
		return errors.New("clipboard busy")
	}
	defer clipboardClose.Call()
	if ok, _, _ := clipboardEmpty.Call(); ok == 0 {
		return errors.New("clipboard could not be cleared")
	}
	if result, _, _ := clipboardSet.Call(clipboardUnicodeText, handle); result == 0 {
		return errors.New("clipboard write failed")
	}
	owned = false // Windows owns this allocation after successful transfer.
	return nil
}
