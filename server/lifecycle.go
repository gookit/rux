package server

import (
	"context"
	"os"
	"os/signal"
)

// runHooks executes a slice of hooks in order. If stopOnError is true, the
// first error returned aborts the loop and is returned. When false, all hooks
// run and the first error (if any) is returned.
func runHooks(ctx context.Context, hooks []func(ctx context.Context) error, stopOnError bool) error {
	var firstErr error
	for _, h := range hooks {
		if h == nil {
			continue
		}
		if err := h(ctx); err != nil {
			if stopOnError {
				return err
			}
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// runHooksLogged runs hooks for their side effects, logging (but not returning)
// any errors. Used for non-fatal hooks (PostStart, PreShutdown, PostShutdown).
func (s *Server) runHooksLogged(ctx context.Context, hooks []func(ctx context.Context) error) {
	for i, h := range hooks {
		if h == nil {
			continue
		}
		if err := h(ctx); err != nil {
			s.logf("hook %d returned error: %v", i, err)
		}
	}
}

// newSignalChan returns a buffered channel that receives OS signals from the
// supplied list. Falls back to SIGINT/SIGTERM if the list is empty.
func newSignalChan(sigs []os.Signal) chan os.Signal {
	ch := make(chan os.Signal, 1)
	if len(sigs) == 0 {
		// Use the same defaults as New, but don't import syscall here to keep
		// this file portable. New always populates StopSignals so this branch
		// is only hit when the caller explicitly cleared the slice.
		signal.Notify(ch)
	} else {
		signal.Notify(ch, sigs...)
	}
	return ch
}

// stopSignalChan detaches a channel from the signal subsystem.
func stopSignalChan(ch chan os.Signal) {
	signal.Stop(ch)
}
