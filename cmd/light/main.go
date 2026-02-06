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
	"fmt"
	"light-lang/internal/ast"
	"light-lang/internal/lexer"
	"light-lang/internal/parser"
	"light-lang/internal/runtime"
	"os"
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
