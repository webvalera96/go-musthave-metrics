//go:build windows

package shutdown

import (
	"os"
	"syscall"
)

// Signals — на Windows нет SIGQUIT.
func Signals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM}
}
