package main

import (
	"encoding/json"
	"fmt"
	"light-lang/internal/diag"
	"light-lang/internal/token"
	"os"
)

// ---- output helpers ----

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "error: JSON encoding failed: %v\n", err)
		os.Exit(1)
	}
}

func printDiagsText(diags []diag.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintln(os.Stderr, d.String())
	}
}

func diagsToSlice(diags []diag.Diagnostic) []map[string]interface{} {
	result := make([]map[string]interface{}, len(diags))
	for i, d := range diags {
		result[i] = map[string]interface{}{
			"code":     d.Code,
			"severity": d.Severity.String(),
			"message":  d.Message,
			"line":     d.Span.Start.Line,
			"column":   d.Span.Start.Column,
			"offset":   d.Span.Start.Offset,
		}
		if d.Hint != "" {
			result[i]["hint"] = d.Hint
		}
	}
	return result
}

// ---- token output helpers ----

func printTokensText(tokens []token.Token, diags []diag.Diagnostic) {
	for _, tok := range tokens {
		if tok.Kind == token.NEWLINE {
			fmt.Printf("%-12s %-20s %d:%d\n", tok.Kind, "\\n", tok.Span.Start.Line, tok.Span.Start.Column)
		} else {
			fmt.Printf("%-12s %-20s %d:%d\n", tok.Kind, tok.Lexeme, tok.Span.Start.Line, tok.Span.Start.Column)
		}
	}
	printDiagsText(diags)
}

func printTokensJSON(tokens []token.Token, diags []diag.Diagnostic) {
	type tokenJSON struct {
		Kind   string `json:"kind"`
		Lexeme string `json:"lexeme"`
		Line   int    `json:"line"`
		Column int    `json:"column"`
		Offset int    `json:"offset"`
	}

	var toks []tokenJSON
	for _, tok := range tokens {
		toks = append(toks, tokenJSON{
			Kind:   tok.Kind.String(),
			Lexeme: tok.Lexeme,
			Line:   tok.Span.Start.Line,
			Column: tok.Span.Start.Column,
			Offset: tok.Span.Start.Offset,
		})
	}

	output := map[string]interface{}{
		"tokens":      toks,
		"diagnostics": diagsToSlice(diags),
	}
	printJSON(output)
}
