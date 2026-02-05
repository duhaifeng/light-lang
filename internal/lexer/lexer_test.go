package lexer

import (
	"light-lang/internal/token"
	"testing"
)

func TestTokenizeSimple(t *testing.T) {
	source := `var x = 1 + 2`
	l := New(source, "test.lt")
	tokens, diags := l.Tokenize()

	if len(diags) > 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}

	expected := []token.Kind{
		token.KW_VAR, token.IDENT, token.ASSIGN,
		token.INT, token.PLUS, token.INT, token.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token[%d]: expected %s, got %s (%q)", i, exp, tokens[i].Kind, tokens[i].Lexeme)
		}
	}
}

func TestTokenizeKeywords(t *testing.T) {
	source := `if else while function return break continue var const class new constructor this true false null`
	l := New(source, "test.lt")
	tokens, diags := l.Tokenize()

	if len(diags) > 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}

	expected := []token.Kind{
		token.KW_IF, token.KW_ELSE, token.KW_WHILE, token.KW_FUNCTION,
		token.KW_RETURN, token.KW_BREAK, token.KW_CONTINUE,
		token.KW_VAR, token.KW_CONST, token.KW_CLASS, token.KW_NEW,
		token.KW_CONSTRUCTOR, token.KW_THIS,
		token.KW_TRUE, token.KW_FALSE, token.KW_NULL,
		token.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token[%d]: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestTokenizeOperators(t *testing.T) {
	source := `= == != < <= > >= + - * / % ! && ||`
	l := New(source, "test.lt")
	tokens, diags := l.Tokenize()

	if len(diags) > 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}

	expected := []token.Kind{
		token.ASSIGN, token.EQ, token.NEQ,
		token.LT, token.LTE, token.GT, token.GTE,
		token.PLUS, token.MINUS, token.STAR, token.SLASH, token.PERCENT,
		token.BANG, token.AND, token.OR,
		token.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token[%d]: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestTokenizeDelimiters(t *testing.T) {
	source := `( ) { } [ ] , . ; :`
	l := New(source, "test.lt")
	tokens, diags := l.Tokenize()

	if len(diags) > 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}

	expected := []token.Kind{
		token.LPAREN, token.RPAREN, token.LBRACE, token.RBRACE,
		token.LBRACKET, token.RBRACKET, token.COMMA, token.DOT,
		token.SEMICOLON, token.COLON,
		token.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token[%d]: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestTokenizeString(t *testing.T) {
	source := `"hello" "line1\nline2"`
	l := New(source, "test.lt")
	tokens, diags := l.Tokenize()

	if len(diags) > 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}

	if tokens[0].Kind != token.STRING || tokens[0].Lexeme != "hello" {
		t.Errorf("expected STRING 'hello', got %s %q", tokens[0].Kind, tokens[0].Lexeme)
	}

	if tokens[1].Kind != token.STRING || tokens[1].Lexeme != "line1\nline2" {
		t.Errorf("expected STRING with newline, got %s %q", tokens[1].Kind, tokens[1].Lexeme)
	}
}

func TestTokenizeNumbers(t *testing.T) {
	source := `123 3.14 0 42`
	l := New(source, "test.lt")
	tokens, diags := l.Tokenize()

	if len(diags) > 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}

	if tokens[0].Kind != token.INT || tokens[0].Lexeme != "123" {
		t.Errorf("token[0]: expected INT '123', got %s %q", tokens[0].Kind, tokens[0].Lexeme)
	}
	if tokens[1].Kind != token.FLOAT || tokens[1].Lexeme != "3.14" {
		t.Errorf("token[1]: expected FLOAT '3.14', got %s %q", tokens[1].Kind, tokens[1].Lexeme)
	}
}

func TestTokenizeNewlines(t *testing.T) {
	source := "a\nb\n"
	l := New(source, "test.lt")
	tokens, _ := l.Tokenize()

	expected := []token.Kind{
		token.IDENT, token.NEWLINE, token.IDENT, token.NEWLINE, token.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token[%d]: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestTokenizeComment(t *testing.T) {
	source := "x // this is a comment\ny"
	l := New(source, "test.lt")
	tokens, _ := l.Tokenize()

	expected := []token.Kind{
		token.IDENT, token.NEWLINE, token.IDENT, token.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token[%d]: expected %s, got %s", i, exp, tokens[i].Kind)
		}
	}
}

func TestTokenizePositions(t *testing.T) {
	source := "var x = 1"
	l := New(source, "test.lt")
	tokens, _ := l.Tokenize()

	// "var" starts at line 1, col 1
	if tokens[0].Span.Start.Line != 1 || tokens[0].Span.Start.Column != 1 {
		t.Errorf("'var' position: expected 1:1, got %d:%d", tokens[0].Span.Start.Line, tokens[0].Span.Start.Column)
	}
	// "x" starts at line 1, col 5
	if tokens[1].Span.Start.Line != 1 || tokens[1].Span.Start.Column != 5 {
		t.Errorf("'x' position: expected 1:5, got %d:%d", tokens[1].Span.Start.Line, tokens[1].Span.Start.Column)
	}
}
