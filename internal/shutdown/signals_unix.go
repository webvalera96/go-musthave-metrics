//go:build unix

package shutdown

import (
	"os"
	"syscall"
)

// Signals — SIGINT, SIGTERM, SIGQUIT (штатное завершение).
func Signals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}
}
