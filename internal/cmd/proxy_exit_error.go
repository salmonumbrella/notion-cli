package cmd

import "errors"

// proxiedCommandExitError preserves exit codes from delegated external commands.
type proxiedCommandExitError struct {
	Code int
}

func (e *proxiedCommandExitError) Error() string {
	return "proxied command failed"
}

func proxiedCommandExitStatus(err error) (int, bool) {
	var proxiedErr *proxiedCommandExitError
	if !errors.As(err, &proxiedErr) {
		return 0, false
	}
	if proxiedErr.Code <= 0 {
		return ExitSystem, true
	}
	return proxiedErr.Code, true
}
