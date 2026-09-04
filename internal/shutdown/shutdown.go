// Package shutdown turns an OS termination signal into a channel close, the
// shape the rest of the program uses to know when to stop.
package shutdown

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// Signal returns a channel that closes the moment the process receives a
// termination signal, so callers can select on it (or pass it to something
// that waits on a <-chan struct{}) instead of handling signals themselves.
func Signal() <-chan struct{} {
	stop := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			syscall.SIGINT,  // Ctrl+C
			syscall.SIGTERM, // Termination Request
			syscall.SIGSEGV, // FullDerp
			syscall.SIGABRT, // Abnormal termination
			syscall.SIGILL,  // illegal instruction
			syscall.SIGFPE)  // floating point - this is why we can't have nice things
		sig := <-c
		slog.Warn("signal detected, shutting down", "signal", sig)
		close(stop)
	}()
	return stop
}
