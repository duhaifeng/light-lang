package main

import (
	"fmt"
	"io"
	"light-lang/internal/diag"
	"light-lang/internal/lexer"
	"light-lang/internal/parser"
	"light-lang/internal/runtime"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
)

// ---- ANSI colors ----

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// ---- repl command ----

func cmdRepl() {
	// Determine history file path (~/.light_history)
	historyFile := ""
	if home, err := os.UserHomeDir(); err == nil {
		historyFile = filepath.Join(home, ".light_history")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            colorGreen + "light> " + colorReset,
		HistoryFile:       historyFile,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "readline init failed: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	// Welcome banner
	fmt.Fprintf(rl.Stdout(), "%s%slight-lang REPL%s %s(type 'exit' or Ctrl+D to quit)%s\n\n",
		colorBold, colorCyan, colorReset, colorGray, colorReset)

	interp := runtime.NewInterpreter(rl.Stdout())
	var accumulated strings.Builder
	braceDepth := 0

	for {
		// Update prompt based on multi-line state
		if braceDepth > 0 {
			rl.SetPrompt(colorGray + "...   " + colorReset)
		} else {
			rl.SetPrompt(colorGreen + "light> " + colorReset)
		}

		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if braceDepth > 0 {
					// Cancel multi-line input
					accumulated.Reset()
					braceDepth = 0
					continue
				}
				// Show hint instead of exiting
				fmt.Fprintf(rl.Stdout(), "\n%s(use 'exit' or Ctrl+D to quit)%s\n", colorGray, colorReset)
				continue
			}
			// EOF (Ctrl+D) or other error → exit
			if err == io.EOF {
				fmt.Fprintln(rl.Stdout())
			}
			break
		}

		// Exit command
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
			printDiagsColored(rl.Stderr(), lexDiags)
			continue
		}

		// Parse
		p := parser.New(tokens)
		file, parseDiags := p.ParseFile()
		if len(parseDiags) > 0 {
			printDiagsColored(rl.Stderr(), parseDiags)
			continue
		}

		// Execute
		if err := interp.Run(file); err != nil {
			fmt.Fprintf(rl.Stderr(), "%serror: %s%s\n", colorRed, err, colorReset)
			continue
		}
	}
}

// printDiagsColored prints diagnostics with red color for REPL display.
func printDiagsColored(w io.Writer, diags []diag.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintf(w, "%s%s%s\n", colorRed, d.String(), colorReset)
	}
}
