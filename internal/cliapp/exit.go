// Package cliapp contains the plumbing shared by every CLI handler: app
// construction, MCP client/tool resolution, and a unified error-to-exit-code
// mapping.
package cliapp

import (
	"errors"
	"fmt"
)

// Exit codes, aligned with the plan.
const (
	ExitOK      = 0
	ExitRuntime = 1
	ExitUsage   = 2
	ExitConfig  = 3
)

// Sentinel errors used by handlers. Wrap them with fmt.Errorf("...: %w", ...)
// to add context while preserving classification.
var (
	ErrMCPUnreachable = errors.New("mcp unreachable")
	ErrUsage          = errors.New("usage")
	ErrConfig         = errors.New("config invalid")
	ErrToolNotFound   = errors.New("tool not found")
	ErrNoResults      = errors.New("no results")
)

// ExitError carries an exit code alongside a wrapped cause. main.go inspects
// this with errors.As to convert handler errors into os.Exit values.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// Newf builds an ExitError with an inline-formatted message.
func Newf(code int, format string, args ...any) *ExitError {
	return &ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}

// Wrap attaches an exit code to an existing error.
func Wrap(code int, err error) *ExitError {
	if err == nil {
		return nil
	}
	return &ExitError{Code: code, Err: err}
}

// MapError inspects err for known sentinels and returns an ExitError with an
// appropriate exit code. Plain errors default to ExitRuntime.
func MapError(err error) *ExitError {
	if err == nil {
		return nil
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee
	}
	switch {
	case errors.Is(err, ErrUsage):
		return &ExitError{Code: ExitUsage, Err: err}
	case errors.Is(err, ErrConfig):
		return &ExitError{Code: ExitConfig, Err: err}
	default:
		return &ExitError{Code: ExitRuntime, Err: err}
	}
}
