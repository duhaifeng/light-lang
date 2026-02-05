// Package diag provides diagnostic (error/warning) types for the compiler.
package diag

import (
	"fmt"
	"light-lang/internal/span"
)

// Severity indicates the severity of a diagnostic.
type Severity int

const (
	Error   Severity = iota
	Warning
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	default:
		return "unknown"
	}
}

// Diagnostic represents a compiler diagnostic message.
type Diagnostic struct {
	Code     string    `json:"code"`               // stable error code, e.g. "E0001"
	Severity Severity  `json:"severity"`            // error or warning
	Message  string    `json:"message"`             // human-readable description
	Span     span.Span `json:"span"`                // source location
	Hint     string    `json:"hint,omitempty"`       // optional hint
}

// String returns a human-readable representation of the diagnostic.
func (d Diagnostic) String() string {
	prefix := d.Severity.String()
	loc := fmt.Sprintf("%d:%d", d.Span.Start.Line, d.Span.Start.Column)
	msg := fmt.Sprintf("[%s] %s at %s: %s", d.Code, prefix, loc, d.Message)
	if d.Hint != "" {
		msg += " (hint: " + d.Hint + ")"
	}
	return msg
}

// Errorf creates an error diagnostic at the given span.
func Errorf(code string, s span.Span, format string, args ...interface{}) Diagnostic {
	return Diagnostic{
		Code:     code,
		Severity: Error,
		Message:  fmt.Sprintf(format, args...),
		Span:     s,
	}
}

// Warningf creates a warning diagnostic at the given span.
func Warningf(code string, s span.Span, format string, args ...interface{}) Diagnostic {
	return Diagnostic{
		Code:     code,
		Severity: Warning,
		Message:  fmt.Sprintf(format, args...),
		Span:     s,
	}
}
