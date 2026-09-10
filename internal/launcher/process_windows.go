//go:build windows

package launcher

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func openTrackedProcess(pid int) (windows.Handle, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return 0, os.ErrProcessDone
	}
	return handle, err
}

func windowsProcessIdentity(handle windows.Handle) (string, error) {
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return "", err
	}
	if exited.HighDateTime != 0 || exited.LowDateTime != 0 {
		return "", os.ErrProcessDone
	}
	return fmt.Sprintf("%d:%d", created.HighDateTime, created.LowDateTime), nil
}

func processIdentity(pid int) (string, error) {
	handle, err := openTrackedProcess(pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	return windowsProcessIdentity(handle)
}

func requestProcessClose(pid int, expected string) error {
	handle, err := openTrackedProcess(pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	identity, err := windowsProcessIdentity(handle)
	if err != nil {
		return err
	}
	if identity != expected {
		return os.ErrProcessDone
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	enumWindows := user32.NewProc("EnumWindows")
	getPID := user32.NewProc("GetWindowThreadProcessId")
	postMessage := user32.NewProc("PostMessageW")
	count := 0
	var failure error
	callback := windows.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var owner uint32
		getPID.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
		if owner == uint32(pid) {
			result, _, callErr := postMessage.Call(hwnd, 0x0010 /* WM_CLOSE */, 0, 0)
			if result == 0 {
				failure = fmt.Errorf("send window close request: %w", callErr)
				return 0
			}
			count++
		}
		return 1
	})
	result, _, callErr := enumWindows.Call(callback, 0)
	if failure != nil {
		return failure
	}
	if result == 0 {
		return fmt.Errorf("enumerate application windows: %w", callErr)
	}
	if count == 0 {
		return errors.New("tracked process has no closable window; close it manually")
	}
	return nil
}
