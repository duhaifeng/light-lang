// Package token defines the token types produced by the lexer.
package token

import (
	"fmt"
	"light-lang/internal/span"
)

// Kind represents the type of a token.
type Kind int

const (
	// Special tokens
	ILLEGAL Kind = iota
	EOF
	NEWLINE

	// Literals
	IDENT  // identifiers: x, foo, myVar
	INT    // integer literals: 123
	FLOAT  // float literals: 3.14
	STRING // string literals: "hello"

	// Operators
	ASSIGN  // =
	PLUS    // +
	MINUS   // -
	STAR    // *
	SLASH   // /
	PERCENT // %
	BANG    // !

	EQ  // ==
	NEQ // !=
	LT  // <
	LTE // <=
	GT  // >
	GTE // >=

	AND // &&
	OR  // ||

	// Compound assignment
	PLUS_ASSIGN  // +=
	MINUS_ASSIGN // -=
	STAR_ASSIGN  // *=
	SLASH_ASSIGN // /=

	// Misc operators
	QUESTION // ?
	ARROW    // =>

	// Delimiters
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
	COMMA     // ,
	DOT       // .
	SEMICOLON // ;
	COLON     // :

	// Keywords
	KW_IF
	KW_ELSE
	KW_WHILE
	KW_FOR
	KW_FUNCTION
	KW_RETURN
	KW_BREAK
	KW_CONTINUE
	KW_VAR
	KW_CONST
	KW_CLASS
	KW_NEW
	KW_CONSTRUCTOR
	KW_THIS
	KW_TRUE
	KW_FALSE
	KW_NULL
	KW_TRY
	KW_CATCH
	KW_THROW
	KW_EXTENDS
	KW_SUPER
	KW_OF
)

var kindNames = map[Kind]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	NEWLINE: "NEWLINE",

	IDENT:  "IDENT",
	INT:    "INT",
	FLOAT:  "FLOAT",
	STRING: "STRING",

	ASSIGN:  "=",
	PLUS:    "+",
	MINUS:   "-",
	STAR:    "*",
	SLASH:   "/",
	PERCENT: "%",
	BANG:    "!",
	EQ:      "==",
	NEQ:     "!=",
	LT:      "<",
	LTE:     "<=",
	GT:      ">",
	GTE:     ">=",
	AND:          "&&",
	OR:           "||",
	PLUS_ASSIGN:  "+=",
	MINUS_ASSIGN: "-=",
	STAR_ASSIGN:  "*=",
	SLASH_ASSIGN: "/=",
	QUESTION:     "?",
	ARROW:        "=>",

	LPAREN:    "(",
	RPAREN:    ")",
	LBRACE:    "{",
	RBRACE:    "}",
	LBRACKET:  "[",
	RBRACKET:  "]",
	COMMA:     ",",
	DOT:       ".",
	SEMICOLON: ";",
	COLON:     ":",

	KW_IF:          "if",
	KW_ELSE:        "else",
	KW_WHILE:       "while",
	KW_FOR:         "for",
	KW_FUNCTION:    "function",
	KW_RETURN:      "return",
	KW_BREAK:       "break",
	KW_CONTINUE:    "continue",
	KW_VAR:         "var",
	KW_CONST:       "const",
	KW_CLASS:       "class",
	KW_NEW:         "new",
	KW_CONSTRUCTOR: "constructor",
	KW_THIS:        "this",
	KW_TRUE:        "true",
	KW_FALSE:       "false",
	KW_NULL:        "null",
	KW_TRY:         "try",
	KW_CATCH:       "catch",
	KW_THROW:       "throw",
	KW_EXTENDS:     "extends",
	KW_SUPER:       "super",
	KW_OF:          "of",
}

// String returns the human-readable name for a token kind.
func (k Kind) String() string {
	if name, ok := kindNames[k]; ok {
		return name
	}
	return fmt.Sprintf("Kind(%d)", int(k))
}

// IsKeyword returns true if the kind is a keyword.
func (k Kind) IsKeyword() bool {
	return k >= KW_IF && k <= KW_OF
}

// IsLiteral returns true if the kind is a literal (ident/int/float/string).
func (k Kind) IsLiteral() bool {
	return k >= IDENT && k <= STRING
}

var keywords = map[string]Kind{
	"if":          KW_IF,
	"else":        KW_ELSE,
	"while":       KW_WHILE,
	"for":         KW_FOR,
	"function":    KW_FUNCTION,
	"return":      KW_RETURN,
	"break":       KW_BREAK,
	"continue":    KW_CONTINUE,
	"var":         KW_VAR,
	"const":       KW_CONST,
	"class":       KW_CLASS,
	"new":         KW_NEW,
	"constructor": KW_CONSTRUCTOR,
	"this":        KW_THIS,
	"true":        KW_TRUE,
	"false":       KW_FALSE,
	"null":        KW_NULL,
	"try":         KW_TRY,
	"catch":       KW_CATCH,
	"throw":       KW_THROW,
	"extends":     KW_EXTENDS,
	"super":       KW_SUPER,
	"of":          KW_OF,
}

// LookupIdent returns the keyword Kind for ident, or IDENT if it is not a keyword.
func LookupIdent(ident string) Kind {
	if kind, ok := keywords[ident]; ok {
		return kind
	}
	return IDENT
}

// Token represents a lexical token with its kind, text, and source location.
type Token struct {
	Kind   Kind      `json:"kind"`
	Lexeme string    `json:"lexeme"`
	Span   span.Span `json:"span"`
}

// String returns a human-readable representation of the token.
func (t Token) String() string {
	return fmt.Sprintf("%s %q %s", t.Kind, t.Lexeme, t.Span.Start)
}
