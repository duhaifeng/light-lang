// Command light is the CLI entry point for the light-lang toolchain.
//
// Usage:
//
//	light tokens <file>            Print tokens
//	light tokens <file> --json     Print tokens as JSON
//	light parse  <file>            Print AST as JSON
//	light run    <file>            Run a source file
//	light repl                     Start interactive REPL
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"light-lang/internal/ast"
	"light-lang/internal/diag"
	"light-lang/internal/lexer"
	"light-lang/internal/parser"
	"light-lang/internal/runtime"
	"light-lang/internal/token"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "tokens":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "error: missing file argument")
			os.Exit(1)
		}
		source := readFile(os.Args[2])
		jsonMode := hasFlag("--json")
		cmdTokens(source, os.Args[2], jsonMode)
	case "parse":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "error: missing file argument")
			os.Exit(1)
		}
		source := readFile(os.Args[2])
		cmdParse(source, os.Args[2])
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "error: missing file argument")
			os.Exit(1)
		}
		source := readFile(os.Args[2])
		cmdRun(source, os.Args[2])
	case "repl":
		cmdRepl()
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
	fmt.Fprintln(os.Stderr, "  light run    <file>            Run a source file")
	fmt.Fprintln(os.Stderr, "  light repl                     Start interactive REPL")
}

func readFile(filename string) string {
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read file %s: %v\n", filename, err)
		os.Exit(1)
	}
	return string(source)
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
	l := lexer.New(source, filename)
	tokens, lexDiags := l.Tokenize()

	p := parser.New(tokens)
	file, parseDiags := p.ParseFile()

	allDiags := append(lexDiags, parseDiags...)

	output := map[string]interface{}{
		"ast":         ast.NodeToMap(file),
		"diagnostics": diagsToSlice(allDiags),
	}
	printJSON(output)

	if len(allDiags) > 0 {
		os.Exit(1)
	}
}

// ---- run command ----

func cmdRun(source, filename string) {
	// Tokenize
	l := lexer.New(source, filename)
	tokens, lexDiags := l.Tokenize()
	if len(lexDiags) > 0 {
		printDiagsText(lexDiags)
		os.Exit(1)
	}

	// Parse
	p := parser.New(tokens)
	file, parseDiags := p.ParseFile()
	if len(parseDiags) > 0 {
		printDiagsText(parseDiags)
		os.Exit(1)
	}

	// Interpret
	interp := runtime.NewInterpreter(os.Stdout)
	if err := interp.Run(file); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ---- repl command ----

func cmdRepl() {
	fmt.Println("light-lang REPL (type 'exit' to quit)")
	fmt.Println()

	interp := runtime.NewInterpreter(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	var accumulated strings.Builder
	braceDepth := 0

	for {
		// Prompt
		if braceDepth > 0 {
			fmt.Print("...   ")
		} else {
			fmt.Print("light> ")
		}

		if !scanner.Scan() {
			fmt.Println()
			break
		}

		line := scanner.Text()

		// Exit
		if braceDepth == 0 && strings.TrimSpace(line) == "exit" {
			break
		}

		// Count braces for multi-line input
		braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
		accumulated.WriteString(line)
		accumulated.WriteString("\n")

		// If braces are unbalanced, keep reading
		if braceDepth > 0 {
			continue
		}
		braceDepth = 0

		source := accumulated.String()
		accumulated.Reset()

		// Skip empty input
		if strings.TrimSpace(source) == "" {
			continue
		}

		// Tokenize
		l := lexer.New(source, "<repl>")
		tokens, lexDiags := l.Tokenize()
		if len(lexDiags) > 0 {
			printDiagsText(lexDiags)
			continue
		}

		// Parse
		p := parser.New(tokens)
		file, parseDiags := p.ParseFile()
		if len(parseDiags) > 0 {
			printDiagsText(parseDiags)
			continue
		}

		// Execute
		if err := interp.Run(file); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
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
