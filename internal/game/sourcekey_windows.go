//go:build windows

package game

import "golang.org/x/sys/windows"

var (
	user32GetKeyDLL  = windows.NewLazySystemDLL("user32.dll")
	getAsyncKeyState = user32GetKeyDLL.NewProc("GetAsyncKeyState")
)

// sourceKeyDown mirrors Flash Key.isDown(code) on Windows: both consume the
// Windows virtual-key number. In particular P1 shoot/grenade are VK 219/221.
func sourceKeyDown(code int) bool {
	if code <= 0 {
		return false
	}
	state, _, _ := getAsyncKeyState.Call(uintptr(code))
	return int16(state&0xffff) < 0
}
