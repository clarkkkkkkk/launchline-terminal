//go:build !linux && !darwin && !windows

package launcher

import "errors"

func processIdentity(int) (string, error) {
	return "", errors.New("process tracking is unsupported on this platform")
}
func requestProcessClose(int, string) error {
	return errors.New("stopping is unsupported on this platform")
}
