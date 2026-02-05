// Package lexer implements the lexical analysis (tokenization) for light-lang.
package lexer

import (
	"fmt"
	"light-lang/internal/diag"
	"light-lang/internal/span"
	"light-lang/internal/token"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes source code into a sequence of tokens.
type Lexer struct {
	source   string
	filename string

	pos  int // current read position in source
	line int // current line (1-based)
	col  int // current column (1-based)

	diags         []diag.Diagnostic
	templateStack []int // brace depth stack for template string expressions
}

// New creates a new Lexer for the given source text.
func New(source, filename string) *Lexer {
	return &Lexer{
		source:   source,
		filename: filename,
		pos:      0,
		line:     1,
		col:      1,
	}
}

// Tokenize scans the entire source and returns all tokens and diagnostics.
func (l *Lexer) Tokenize() ([]token.Token, []diag.Diagnostic) {
	var tokens []token.Token
	for {
		tok := l.nextToken()
		tokens = append(tokens, tok)
		if tok.Kind == token.EOF {
			break
		}
	}
	return tokens, l.diags
}

// ---- internal helpers ----

// peek returns the current character without advancing, or 0 if at end.
func (l *Lexer) peek() byte {
	if l.pos >= len(l.source) {
		return 0
	}
	return l.source[l.pos]
}

// peekNext returns the character after current, or 0 if at end.
func (l *Lexer) peekNext() byte {
	if l.pos+1 >= len(l.source) {
		return 0
	}
	return l.source[l.pos+1]
}

// advance consumes the current character and returns it.
func (l *Lexer) advance() byte {
	ch := l.source[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

// curPos returns the current position as a span.Position.
func (l *Lexer) curPos() span.Position {
	return span.Position{Offset: l.pos, Line: l.line, Column: l.col}
}

// makeSpan returns a span from start to current position.
func (l *Lexer) makeSpan(start span.Position) span.Span {
	return span.Span{Start: start, End: l.curPos()}
}

// skipWhitespace skips spaces and tabs (not newlines).
func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.source) {
		ch := l.source[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
		} else {
			break
		}
	}
}

// skipLineComment skips from // to end of line.
func (l *Lexer) skipLineComment() {
	for l.pos < len(l.source) && l.source[l.pos] != '\n' {
		l.advance()
	}
}

// addError records a diagnostic error.
func (l *Lexer) addError(code string, s span.Span, msg string) {
	l.diags = append(l.diags, diag.Errorf(code, s, "%s", msg))
}

// ---- token reading ----

func (l *Lexer) nextToken() token.Token {
	l.skipWhitespace()

	if l.pos >= len(l.source) {
		return token.Token{Kind: token.EOF, Lexeme: "", Span: l.makeSpan(l.curPos())}
	}

	start := l.curPos()
	ch := l.peek()

	// Newline
	if ch == '\n' {
		l.advance()
		return token.Token{Kind: token.NEWLINE, Lexeme: "\\n", Span: l.makeSpan(start)}
	}

	// Line comment: //
	if ch == '/' && l.peekNext() == '/' {
		l.skipLineComment()
		return l.nextToken() // skip comment, get next token
	}

	// Hash comment: #
	if ch == '#' {
		l.skipLineComment()
		return l.nextToken()
	}

	// String literal
	if ch == '"' {
		return l.readString(start)
	}

	// Template string literal
	if ch == '`' {
		return l.readTemplateStart(start)
	}

	// Number literal
	if isDigit(ch) {
		return l.readNumber(start)
	}

	// Identifier or keyword
	if isIdentStart(ch) {
		return l.readIdentifier(start)
	}

	// Operators and delimiters
	return l.readOperator(start)
}

// readString reads a string literal (double-quoted).
func (l *Lexer) readString(start span.Position) token.Token {
	l.advance() // skip opening "
	var value []byte

	for l.pos < len(l.source) {
		ch := l.peek()
		if ch == '"' {
			l.advance() // skip closing "
			return token.Token{
				Kind:   token.STRING,
				Lexeme: string(value),
				Span:   l.makeSpan(start),
			}
		}
		if ch == '\n' {
			l.addError("E1001", l.makeSpan(start), "unterminated string literal")
			return token.Token{Kind: token.STRING, Lexeme: string(value), Span: l.makeSpan(start)}
		}
		if ch == '\\' {
			l.advance()
			esc := l.peek()
			switch esc {
			case 'n':
				value = append(value, '\n')
			case 't':
				value = append(value, '\t')
			case '\\':
				value = append(value, '\\')
			case '"':
				value = append(value, '"')
			case '0':
				value = append(value, 0)
			default:
				l.addError("E1002", l.makeSpan(start), fmt.Sprintf("unknown escape sequence: \\%c", esc))
				value = append(value, esc)
			}
			l.advance()
			continue
		}
		value = append(value, ch)
		l.advance()
	}

	l.addError("E1001", l.makeSpan(start), "unterminated string literal")
	return token.Token{Kind: token.STRING, Lexeme: string(value), Span: l.makeSpan(start)}
}

// readNumber reads an integer or float literal.
func (l *Lexer) readNumber(start span.Position) token.Token {
	isFloat := false
	numStart := l.pos

	for l.pos < len(l.source) && isDigit(l.peek()) {
		l.advance()
	}

	// Check for decimal point
	if l.pos < len(l.source) && l.peek() == '.' && isDigit(l.peekNext()) {
		isFloat = true
		l.advance() // skip '.'
		for l.pos < len(l.source) && isDigit(l.peek()) {
			l.advance()
		}
	}

	lexeme := l.source[numStart:l.pos]
	kind := token.INT
	if isFloat {
		kind = token.FLOAT
	}
	return token.Token{Kind: kind, Lexeme: lexeme, Span: l.makeSpan(start)}
}

// readIdentifier reads an identifier or keyword.
func (l *Lexer) readIdentifier(start span.Position) token.Token {
	identStart := l.pos

	for l.pos < len(l.source) && isIdentPart(l.peek()) {
		l.advance()
	}

	lexeme := l.source[identStart:l.pos]
	kind := token.LookupIdent(lexeme)
	return token.Token{Kind: kind, Lexeme: lexeme, Span: l.makeSpan(start)}
}

// readOperator reads an operator or delimiter token.
func (l *Lexer) readOperator(start span.Position) token.Token {
	ch := l.advance()

	switch ch {
	case '(':
		return token.Token{Kind: token.LPAREN, Lexeme: "(", Span: l.makeSpan(start)}
	case ')':
		return token.Token{Kind: token.RPAREN, Lexeme: ")", Span: l.makeSpan(start)}
	case '{':
		if len(l.templateStack) > 0 {
			l.templateStack[len(l.templateStack)-1]++
		}
		return token.Token{Kind: token.LBRACE, Lexeme: "{", Span: l.makeSpan(start)}
	case '}':
		if len(l.templateStack) > 0 && l.templateStack[len(l.templateStack)-1] == 0 {
			// Closing a template expression — continue reading template text
			l.templateStack = l.templateStack[:len(l.templateStack)-1]
			text := l.readTemplateText()
			if l.peek() == '`' {
				l.advance()
				return token.Token{Kind: token.TEMPLATE_TAIL, Lexeme: text, Span: l.makeSpan(start)}
			}
			// Must be ${ — another expression follows
			l.advance() // $
			l.advance() // {
			l.templateStack = append(l.templateStack, 0)
			return token.Token{Kind: token.TEMPLATE_MIDDLE, Lexeme: text, Span: l.makeSpan(start)}
		}
		if len(l.templateStack) > 0 {
			l.templateStack[len(l.templateStack)-1]--
		}
		return token.Token{Kind: token.RBRACE, Lexeme: "}", Span: l.makeSpan(start)}
	case '[':
		return token.Token{Kind: token.LBRACKET, Lexeme: "[", Span: l.makeSpan(start)}
	case ']':
		return token.Token{Kind: token.RBRACKET, Lexeme: "]", Span: l.makeSpan(start)}
	case ',':
		return token.Token{Kind: token.COMMA, Lexeme: ",", Span: l.makeSpan(start)}
	case '.':
		return token.Token{Kind: token.DOT, Lexeme: ".", Span: l.makeSpan(start)}
	case ';':
		return token.Token{Kind: token.SEMICOLON, Lexeme: ";", Span: l.makeSpan(start)}
	case ':':
		return token.Token{Kind: token.COLON, Lexeme: ":", Span: l.makeSpan(start)}
	case '+':
		if l.peek() == '=' {
			l.advance()
			return token.Token{Kind: token.PLUS_ASSIGN, Lexeme: "+=", Span: l.makeSpan(start)}
		}
		return token.Token{Kind: token.PLUS, Lexeme: "+", Span: l.makeSpan(start)}
	case '-':
		if l.peek() == '=' {
			l.advance()
			return token.Token{Kind: token.MINUS_ASSIGN, Lexeme: "-=", Span: l.makeSpan(start)}
		}
		return token.Token{Kind: token.MINUS, Lexeme: "-", Span: l.makeSpan(start)}
	case '*':
		if l.peek() == '=' {
			l.advance()
			return token.Token{Kind: token.STAR_ASSIGN, Lexeme: "*=", Span: l.makeSpan(start)}
		}
		return token.Token{Kind: token.STAR, Lexeme: "*", Span: l.makeSpan(start)}
	case '/':
		if l.peek() == '=' {
			l.advance()
			return token.Token{Kind: token.SLASH_ASSIGN, Lexeme: "/=", Span: l.makeSpan(start)}
		}
		return token.Token{Kind: token.SLASH, Lexeme: "/", Span: l.makeSpan(start)}
	case '%':
		return token.Token{Kind: token.PERCENT, Lexeme: "%", Span: l.makeSpan(start)}
	case '!':
		if l.peek() == '=' {
			l.advance()
			return token.Token{Kind: token.NEQ, Lexeme: "!=", Span: l.makeSpan(start)}
		}
		return token.Token{Kind: token.BANG, Lexeme: "!", Span: l.makeSpan(start)}
	case '?':
		return token.Token{Kind: token.QUESTION, Lexeme: "?", Span: l.makeSpan(start)}
	case '=':
		if l.peek() == '=' {
			l.advance()
			return token.Token{Kind: token.EQ, Lexeme: "==", Span: l.makeSpan(start)}
		}
		if l.peek() == '>' {
			l.advance()
			return token.Token{Kind: token.ARROW, Lexeme: "=>", Span: l.makeSpan(start)}
		}
		return token.Token{Kind: token.ASSIGN, Lexeme: "=", Span: l.makeSpan(start)}
	case '<':
		if l.peek() == '=' {
			l.advance()
			return token.Token{Kind: token.LTE, Lexeme: "<=", Span: l.makeSpan(start)}
		}
		return token.Token{Kind: token.LT, Lexeme: "<", Span: l.makeSpan(start)}
	case '>':
		if l.peek() == '=' {
			l.advance()
			return token.Token{Kind: token.GTE, Lexeme: ">=", Span: l.makeSpan(start)}
		}
		return token.Token{Kind: token.GT, Lexeme: ">", Span: l.makeSpan(start)}
	case '&':
		if l.peek() == '&' {
			l.advance()
			return token.Token{Kind: token.AND, Lexeme: "&&", Span: l.makeSpan(start)}
		}
		l.addError("E1003", l.makeSpan(start), fmt.Sprintf("unexpected character: '%c', did you mean '&&'?", ch))
		return token.Token{Kind: token.ILLEGAL, Lexeme: string(ch), Span: l.makeSpan(start)}
	case '|':
		if l.peek() == '|' {
			l.advance()
			return token.Token{Kind: token.OR, Lexeme: "||", Span: l.makeSpan(start)}
		}
		l.addError("E1003", l.makeSpan(start), fmt.Sprintf("unexpected character: '%c', did you mean '||'?", ch))
		return token.Token{Kind: token.ILLEGAL, Lexeme: string(ch), Span: l.makeSpan(start)}
	default:
		l.addError("E1003", l.makeSpan(start), fmt.Sprintf("unexpected character: '%c'", ch))
		return token.Token{Kind: token.ILLEGAL, Lexeme: string(ch), Span: l.makeSpan(start)}
	}
}

// ---- template string helpers ----

// readTemplateStart is called when we encounter a backtick (`).
// It reads template text and determines if this is a simple literal or a head.
func (l *Lexer) readTemplateStart(start span.Position) token.Token {
	l.advance() // consume opening `
	text := l.readTemplateText()

	if l.peek() == '`' {
		l.advance() // consume closing `
		return token.Token{Kind: token.TEMPLATE_LITERAL, Lexeme: text, Span: l.makeSpan(start)}
	}

	// Must be ${ — template with expressions
	l.advance() // $
	l.advance() // {
	l.templateStack = append(l.templateStack, 0)
	return token.Token{Kind: token.TEMPLATE_HEAD, Lexeme: text, Span: l.makeSpan(start)}
}

// readTemplateText reads characters until ` or ${ is found (or EOF).
func (l *Lexer) readTemplateText() string {
	var text []byte
	for l.pos < len(l.source) {
		ch := l.peek()
		if ch == '`' {
			break
		}
		if ch == '$' && l.peekNext() == '{' {
			break
		}
		if ch == '\\' {
			l.advance()
			if l.pos >= len(l.source) {
				break
			}
			esc := l.peek()
			switch esc {
			case 'n':
				text = append(text, '\n')
			case 't':
				text = append(text, '\t')
			case '\\':
				text = append(text, '\\')
			case '`':
				text = append(text, '`')
			case '$':
				text = append(text, '$')
			default:
				text = append(text, '\\', esc)
			}
			l.advance()
			continue
		}
		if ch == '\n' {
			l.line++
			l.col = 0
		}
		text = append(text, ch)
		l.advance()
	}
	return string(text)
}

// ---- character classification ----

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentStart(ch byte) bool {
	if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
		return true
	}
	// Support non-ASCII letters (e.g. Chinese identifiers) via utf8
	if ch >= 0x80 {
		r, _ := utf8.DecodeRuneInString(string(ch))
		return unicode.IsLetter(r)
	}
	return false
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}
