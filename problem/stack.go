package problem

import (
	"fmt"
	"io"
	"runtime"
)

// captureStack records the call stack at the point New is invoked, skipping runtime.Callers,
// captureStack itself, and New — so the first recorded frame is New's caller (e.g. a
// Register-ed ErrorFunc's caller).
func captureStack() []uintptr {
	const maxDepth = 32
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(3, pcs)
	return pcs[:n]
}

// Frame is one call frame captured in an Error's stack trace.
type Frame struct {
	Function string
	File     string
	Line     int
}

// String formats the frame as "file:line function".
func (f Frame) String() string {
	return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Function)
}

// StackTrace returns the call stack captured when the Error was created (by New, or by a
// Register-ed ErrorFunc), innermost frame — the call site — first. It is nil for an Error
// built by any means other than New (e.g. a zero-value Error or one decoded from JSON).
func (e *Error) StackTrace() []Frame {
	if len(e.stack) == 0 {
		return nil
	}

	frames := runtime.CallersFrames(e.stack)
	trace := make([]Frame, 0, len(e.stack))
	for {
		f, more := frames.Next()
		trace = append(trace, Frame{Function: f.Function, File: f.File, Line: f.Line})
		if !more {
			break
		}
	}
	return trace
}

// Format implements fmt.Formatter, matching the ergonomics of github.com/pkg/errors: "%+v"
// prints the error message followed by its captured stack trace, one frame per line; "%v"
// and "%s" print just the error message.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.Error())
			for _, f := range e.StackTrace() {
				_, _ = fmt.Fprintf(s, "\n%s\n\t%s:%d", f.Function, f.File, f.Line)
			}
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}
