//go:build windows

package game

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Native Win32 clipboard access through golang.org/x/sys/windows (already a
// dependency). This avoids adding a clipboard module and keeps the Windows
// build cgo-free.

const (
	cfUnicodeText   = 13
	gmemMoveable    = 0x0002
	maxClipboardLen = 64 * 1024 // UTF-16 code units read from the clipboard
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	kernel32Clipboard      = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard      = user32.NewProc("OpenClipboard")
	procCloseClipboard     = user32.NewProc("CloseClipboard")
	procEmptyClipboard     = user32.NewProc("EmptyClipboard")
	procGetClipboardData   = user32.NewProc("GetClipboardData")
	procSetClipboardData   = user32.NewProc("SetClipboardData")
	procIsClipboardFormatA = user32.NewProc("IsClipboardFormatAvailable")
	procGlobalAlloc        = kernel32Clipboard.NewProc("GlobalAlloc")
	procGlobalFree         = kernel32Clipboard.NewProc("GlobalFree")
	procGlobalLock         = kernel32Clipboard.NewProc("GlobalLock")
	procGlobalUnlock       = kernel32Clipboard.NewProc("GlobalUnlock")
	procGlobalSize         = kernel32Clipboard.NewProc("GlobalSize")
)

// openClipboard retries briefly because another process (clipboard managers,
// RDP, the app that just copied) may hold the clipboard for a moment.
func openClipboard() error {
	var lastErr error
	for i := 0; i < 10; i++ {
		r, _, err := procOpenClipboard.Call(0)
		if r != 0 {
			return nil
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("open clipboard: %v", lastErr)
}

// uintptrToPointer converts a pointer returned by a Win32 call. Going through
// memory keeps go vet's unsafeptr check quiet; the memory is a locked global
// handle owned by the clipboard, not by the Go heap.
func uintptrToPointer(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

func readClipboardText() (string, error) {
	if r, _, _ := procIsClipboardFormatA.Call(cfUnicodeText); r == 0 {
		return "", errors.New("clipboard does not contain text")
	}
	if err := openClipboard(); err != nil {
		return "", err
	}
	defer procCloseClipboard.Call()

	handle, _, err := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", fmt.Errorf("read clipboard: %v", err)
	}
	ptr, _, err := procGlobalLock.Call(handle)
	if ptr == 0 {
		return "", fmt.Errorf("lock clipboard: %v", err)
	}
	defer procGlobalUnlock.Call(handle)

	size, _, _ := procGlobalSize.Call(handle)
	units := int(size / 2)
	if units <= 0 {
		return "", nil
	}
	if units > maxClipboardLen {
		units = maxClipboardLen
	}
	data := unsafe.Slice((*uint16)(uintptrToPointer(ptr)), units)
	n := 0
	for n < len(data) && data[n] != 0 {
		n++
	}
	return string(utf16.Decode(data[:n])), nil
}

func writeClipboardText(text string) error {
	encoded := utf16.Encode([]rune(text))
	encoded = append(encoded, 0)
	if err := openClipboard(); err != nil {
		return err
	}
	defer procCloseClipboard.Call()
	if r, _, err := procEmptyClipboard.Call(); r == 0 {
		return fmt.Errorf("empty clipboard: %v", err)
	}
	bytes := uintptr(len(encoded) * 2)
	handle, _, err := procGlobalAlloc.Call(gmemMoveable, bytes)
	if handle == 0 {
		return fmt.Errorf("allocate clipboard memory: %v", err)
	}
	ptr, _, err := procGlobalLock.Call(handle)
	if ptr == 0 {
		procGlobalFree.Call(handle)
		return fmt.Errorf("lock clipboard memory: %v", err)
	}
	copy(unsafe.Slice((*uint16)(uintptrToPointer(ptr)), len(encoded)), encoded)
	procGlobalUnlock.Call(handle)
	if r, _, err := procSetClipboardData.Call(cfUnicodeText, handle); r == 0 {
		procGlobalFree.Call(handle)
		return fmt.Errorf("set clipboard: %v", err)
	}
	// The system owns the memory after a successful SetClipboardData.
	return nil
}
