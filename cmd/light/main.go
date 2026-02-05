// Command light is the CLI entry point for the light-lang toolchain.
//
// Usage:
//
//	light tokens <file>            Print tokens
//	light tokens <file> --json     Print tokens as JSON
//	light parse  <file>            Print AST as JSON
package main

import (
	"encoding/json"
	"fmt"
	"light-lang/internal/ast"
	"light-lang/internal/diag"
	"light-lang/internal/lexer"
	"light-lang/internal/parser"
	"light-lang/internal/token"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]
	filename := os.Args[2]

	// Read source file
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read file %s: %v\n", filename, err)
		os.Exit(1)
	}

	switch command {
	case "tokens":
		jsonMode := hasFlag("--json")
		cmdTokens(string(source), filename, jsonMode)
	case "parse":
		cmdParse(string(source), filename)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command '%s'\n", command)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  light tokens <file> [--json]   Tokenize and print tokens")
	fmt.Fprintln(os.Stderr, "  light parse  <file>            Parse and print AST (JSON)")
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[3:] {
		if arg == flag {
			return true
		}
	}
	return false
}

// ---- tokens command ----

func cmdTokens(source, filename string, jsonMode bool) {
	l := lexer.New(source, filename)
	tokens, diags := l.Tokenize()

	if jsonMode {
		printTokensJSON(tokens, diags)
	} else {
		printTokensText(tokens, diags)
	}

	if len(diags) > 0 {
		os.Exit(1)
	}
}

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

// ---- parse command ----

func cmdParse(source, filename string) {
	// Tokenize
	l := lexer.New(source, filename)
	tokens, lexDiags := l.Tokenize()

	// Parse
	p := parser.New(tokens)
	file, parseDiags := p.ParseFile()

	// Combine diagnostics
	allDiags := append(lexDiags, parseDiags...)

	// Output
	output := map[string]interface{}{
		"ast":         ast.NodeToMap(file),
		"diagnostics": diagsToSlice(allDiags),
	}
	printJSON(output)

	if len(allDiags) > 0 {
		os.Exit(1)
	}
}

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
