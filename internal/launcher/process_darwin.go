//go:build darwin

package launcher

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func processIdentity(pid int) (string, error) {
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if errors.Is(err, unix.ESRCH) || errors.Is(err, unix.ENOENT) {
		return "", os.ErrProcessDone
	}
	if err != nil {
		// A missing PID produces a zero-length sysctl result (EIO).
		if errors.Is(err, unix.EIO) && errors.Is(unix.Kill(pid, 0), unix.ESRCH) {
			return "", os.ErrProcessDone
		}
		return "", err
	}
	if info == nil || info.Proc.P_pid != int32(pid) || info.Proc.P_stat == 5 {
		return "", os.ErrProcessDone
	}
	return fmt.Sprintf("%d:%d", info.Proc.P_starttime.Sec, info.Proc.P_starttime.Usec), nil
}

func requestProcessClose(pid int, expected string) error {
	identity, err := processIdentity(pid)
	if err != nil {
		return err
	}
	if identity != expected {
		return os.ErrProcessDone
	}
	err = unix.Kill(pid, unix.SIGTERM)
	if errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
