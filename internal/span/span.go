// Package span provides source position and span types used across the compiler.
package span

import "fmt"

// Position represents a position in source code.
type Position struct {
	Offset int `json:"offset"` // byte offset from beginning of source
	Line   int `json:"line"`   // 1-based line number
	Column int `json:"column"` // 1-based column number
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Span represents a range in source code [Start, End).
type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

func (s Span) String() string {
	return fmt.Sprintf("%s..%s", s.Start, s.End)
}

// Len returns the byte length of the span.
func (s Span) Len() int {
	return s.End.Offset - s.Start.Offset
}
