//go:build darwin || linux

package atproxy

import "syscall"

func init() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return
	}
	rLimit.Cur = rLimit.Max
	syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
}
