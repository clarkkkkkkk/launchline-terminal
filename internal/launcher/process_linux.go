//go:build linux

package launcher

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func processIdentity(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if errors.Is(err, os.ErrNotExist) {
		return "", os.ErrProcessDone
	}
	if err != nil {
		return "", err
	}
	end := strings.LastIndexByte(string(data), ')')
	if end < 0 {
		return "", errors.New("invalid process stat")
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 20 {
		return "", errors.New("incomplete process stat")
	}
	if fields[0] == "Z" || fields[0] == "X" {
		return "", os.ErrProcessDone
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return "", err
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(boot)) + ":" + fields[19], nil
}

func requestProcessClose(pid int, expected string) error {
	fd, err := unix.PidfdOpen(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	}
	if err != nil {
		return fmt.Errorf("open process handle (Linux 5.3+ required): %w", err)
	}
	defer unix.Close(fd)
	identity, err := processIdentity(pid)
	if err != nil {
		return err
	}
	if identity != expected {
		return os.ErrProcessDone
	}
	err = unix.PidfdSendSignal(fd, unix.SIGTERM, nil, 0)
	if errors.Is(err, unix.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
