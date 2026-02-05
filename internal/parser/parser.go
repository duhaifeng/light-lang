// Package parser implements the syntax analysis for light-lang.
// It uses Pratt parsing for expressions and recursive descent for statements/declarations.
package parser

import (
	"fmt"
	"light-lang/internal/ast"
	"light-lang/internal/diag"
	"light-lang/internal/span"
	"light-lang/internal/token"
	"strconv"
)

// ============================================================
// Binding power (precedence) levels
// ============================================================

const (
	bpNone       = 0
	bpOr         = 10 // ||
	bpAnd        = 20 // &&
	bpEquality   = 30 // == !=
	bpComparison = 40 // < <= > >=
	bpAdditive   = 50 // + -
	bpMultiply   = 60 // * / %
	bpPrefix     = 70 // ! -
	bpPostfix    = 80 // () [] .
)

// infixBP returns the left binding power for an infix/postfix operator.
func infixBP(kind token.Kind) int {
	switch kind {
	case token.OR:
		return bpOr
	case token.AND:
		return bpAnd
	case token.EQ, token.NEQ:
		return bpEquality
	case token.LT, token.LTE, token.GT, token.GTE:
		return bpComparison
	case token.PLUS, token.MINUS:
		return bpAdditive
	case token.STAR, token.SLASH, token.PERCENT:
		return bpMultiply
	case token.LPAREN, token.LBRACKET, token.DOT:
		return bpPostfix
	default:
		return bpNone
	}
}

// ============================================================
// Parser
// ============================================================

// Parser performs syntax analysis on a stream of tokens.
type Parser struct {
	tokens []token.Token
	pos    int
	diags  []diag.Diagnostic
}

// New creates a new parser from a token slice.
func New(tokens []token.Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

// ParseFile parses the entire file and returns the AST root and diagnostics.
func (p *Parser) ParseFile() (*ast.File, []diag.Diagnostic) {
	file := &ast.File{}
	startPos := p.peek().Span.Start

	p.skipSep()
	for !p.isAtEnd() {
		node := p.parseTopLevel()
		if node != nil {
			file.Body = append(file.Body, node)
		}
		p.skipSep()
	}

	endPos := p.peek().Span.End
	file.Span = span.Span{Start: startPos, End: endPos}
	return file, p.diags
}

// ---- navigation helpers ----

func (p *Parser) peek() token.Token {
	if p.pos >= len(p.tokens) {
		return token.Token{Kind: token.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peekKind() token.Kind {
	return p.peek().Kind
}

func (p *Parser) advance() token.Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) check(kind token.Kind) bool {
	return p.peekKind() == kind
}

func (p *Parser) match(kinds ...token.Kind) bool {
	for _, k := range kinds {
		if p.check(k) {
			return true
		}
	}
	return false
}

func (p *Parser) expect(kind token.Kind) (token.Token, bool) {
	if p.check(kind) {
		return p.advance(), true
	}
	tok := p.peek()
	p.error("E2001", tok.Span, fmt.Sprintf("expected '%s', got '%s'", kind, tok.Kind))
	return tok, false
}

func (p *Parser) isAtEnd() bool {
	return p.peekKind() == token.EOF
}

// skipSep skips NEWLINE and SEMICOLON tokens (separators).
func (p *Parser) skipSep() {
	for p.match(token.NEWLINE, token.SEMICOLON) {
		p.advance()
	}
}

// skipNewlines skips NEWLINE tokens only.
func (p *Parser) skipNewlines() {
	for p.check(token.NEWLINE) {
		p.advance()
	}
}

func (p *Parser) error(code string, s span.Span, msg string) {
	p.diags = append(p.diags, diag.Errorf(code, s, "%s", msg))
}

// ============================================================
// Error recovery
// ============================================================

// synchronize skips tokens until a likely statement boundary.
func (p *Parser) synchronize() {
	for !p.isAtEnd() {
		// Stop at separators
		if p.match(token.NEWLINE, token.SEMICOLON) {
			p.advance()
			return
		}
		// Stop at closing brace
		if p.check(token.RBRACE) {
			return
		}
		// Stop at statement-starting keywords
		if p.match(token.KW_IF, token.KW_WHILE, token.KW_FOR, token.KW_FUNCTION, token.KW_CLASS,
			token.KW_VAR, token.KW_CONST, token.KW_RETURN, token.KW_BREAK, token.KW_CONTINUE) {
			return
		}
		p.advance()
	}
}

// ============================================================
// Top-level parsing
// ============================================================

func (p *Parser) parseTopLevel() ast.Node {
	switch p.peekKind() {
	case token.KW_FUNCTION:
		return p.parseFuncDecl()
	case token.KW_CLASS:
		return p.parseClassDecl()
	default:
		return p.parseStmt()
	}
}

// ============================================================
// Statement parsing
// ============================================================

func (p *Parser) parseStmt() ast.Stmt {
	switch p.peekKind() {
	case token.KW_IF:
		return p.parseIfStmt()
	case token.KW_WHILE:
		return p.parseWhileStmt()
	case token.KW_FOR:
		return p.parseForStmt()
	case token.KW_RETURN:
		return p.parseReturnStmt()
	case token.KW_BREAK:
		return p.parseBreakStmt()
	case token.KW_CONTINUE:
		return p.parseContinueStmt()
	case token.KW_VAR, token.KW_CONST:
		return p.parseVarDecl()
	default:
		return p.parseSimpleStmt()
	}
}

// parseIfStmt parses: if (expr) block { else if (expr) block } [ else block ]
func (p *Parser) parseIfStmt() *ast.IfStmt {
	start := p.advance() // consume 'if'
	stmt := &ast.IfStmt{}

	// condition
	if _, ok := p.expect(token.LPAREN); !ok {
		p.synchronize()
		stmt.Span = p.makeSpan(start.Span.Start)
		return stmt
	}
	stmt.Condition = p.parseExpr(bpNone)
	p.expect(token.RPAREN)

	// body
	stmt.Body = p.parseBlock()

	// else if / else
	for p.check(token.KW_ELSE) {
		p.advance() // consume 'else'
		if p.check(token.KW_IF) {
			// else if
			elseIfStart := p.advance() // consume 'if'
			clause := ast.ElseIfClause{}
			if _, ok := p.expect(token.LPAREN); ok {
				clause.Condition = p.parseExpr(bpNone)
				p.expect(token.RPAREN)
			}
			clause.Body = p.parseBlock()
			clause.Span = p.makeSpan(elseIfStart.Span.Start)
			stmt.ElseIfs = append(stmt.ElseIfs, clause)
		} else {
			// else
			stmt.ElseBody = p.parseBlock()
			break
		}
	}

	stmt.Span = p.makeSpan(start.Span.Start)
	return stmt
}

// parseWhileStmt parses: while (expr) block
func (p *Parser) parseWhileStmt() *ast.WhileStmt {
	start := p.advance() // consume 'while'
	stmt := &ast.WhileStmt{}

	if _, ok := p.expect(token.LPAREN); !ok {
		p.synchronize()
		stmt.Span = p.makeSpan(start.Span.Start)
		return stmt
	}
	stmt.Condition = p.parseExpr(bpNone)
	p.expect(token.RPAREN)
	stmt.Body = p.parseBlock()
	stmt.Span = p.makeSpan(start.Span.Start)
	return stmt
}

// parseReturnStmt parses: return [expr]
func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	start := p.advance() // consume 'return'
	stmt := &ast.ReturnStmt{}

	// return can be followed by an expression on the same line
	if !p.match(token.NEWLINE, token.SEMICOLON, token.RBRACE, token.EOF) {
		stmt.Value = p.parseExpr(bpNone)
	}

	stmt.Span = p.makeSpan(start.Span.Start)
	return stmt
}

func (p *Parser) parseBreakStmt() *ast.BreakStmt {
	start := p.advance()
	return &ast.BreakStmt{StmtBase: makeStmtBase(start.Span.Start, p.prevEnd())}
}

func (p *Parser) parseContinueStmt() *ast.ContinueStmt {
	start := p.advance()
	return &ast.ContinueStmt{StmtBase: makeStmtBase(start.Span.Start, p.prevEnd())}
}

// parseVarDecl parses: (var | const) IDENT [ = expr ]
func (p *Parser) parseVarDecl() *ast.VarDeclStmt {
	start := p.advance() // consume 'var' or 'const'
	isConst := start.Kind == token.KW_CONST
	stmt := &ast.VarDeclStmt{IsConst: isConst}

	nameTok, ok := p.expect(token.IDENT)
	if !ok {
		p.synchronize()
		stmt.Span = p.makeSpan(start.Span.Start)
		return stmt
	}
	stmt.Name = nameTok.Lexeme

	// optional initializer
	if p.check(token.ASSIGN) {
		p.advance()
		stmt.Init = p.parseExpr(bpNone)
	}

	stmt.Span = p.makeSpan(start.Span.Start)
	return stmt
}

// parseSimpleStmt parses an expression statement or assignment.
func (p *Parser) parseSimpleStmt() ast.Stmt {
	expr := p.parseExpr(bpNone)
	if expr == nil {
		// couldn't parse expression; synchronize
		tok := p.peek()
		p.error("E2002", tok.Span, fmt.Sprintf("unexpected token: '%s'", tok.Lexeme))
		p.synchronize()
		return &ast.ExprStmt{
			StmtBase: makeStmtBase(tok.Span.Start, tok.Span.End),
		}
	}

	// Check for assignment: expr = value
	if p.check(token.ASSIGN) {
		p.advance()
		value := p.parseExpr(bpNone)
		return &ast.AssignStmt{
			StmtBase: makeStmtBase(expr.GetSpan().Start, p.prevEnd()),
			Target:   expr,
			Value:    value,
		}
	}

	// Check for compound assignment: expr += / -= / *= / /= value
	if p.match(token.PLUS_ASSIGN, token.MINUS_ASSIGN, token.STAR_ASSIGN, token.SLASH_ASSIGN) {
		opTok := p.advance()
		rhs := p.parseExpr(bpNone)
		// Desugar: target op= rhs → target = target op rhs
		binOp := compoundToOp(opTok.Kind)
		value := &ast.BinaryExpr{
			ExprBase: makeExprBase(expr.GetSpan().Start, rhs.GetSpan().End),
			Op:       binOp,
			Left:     expr,
			Right:    rhs,
		}
		return &ast.AssignStmt{
			StmtBase: makeStmtBase(expr.GetSpan().Start, p.prevEnd()),
			Target:   expr,
			Value:    value,
		}
	}

	return &ast.ExprStmt{
		StmtBase: makeStmtBase(expr.GetSpan().Start, expr.GetSpan().End),
		Expr:     expr,
	}
}

// parseBlock parses: { stmts }
func (p *Parser) parseBlock() *ast.BlockStmt {
	start := p.peek()
	block := &ast.BlockStmt{}

	if _, ok := p.expect(token.LBRACE); !ok {
		p.synchronize()
		block.Span = p.makeSpan(start.Span.Start)
		return block
	}

	p.skipSep()
	for !p.check(token.RBRACE) && !p.isAtEnd() {
		node := p.parseTopLevel()
		if node != nil {
			block.Stmts = append(block.Stmts, node)
		}
		p.skipSep()
	}

	p.expect(token.RBRACE)
	block.Span = p.makeSpan(start.Span.Start)
	return block
}

// ============================================================
// Declaration parsing
// ============================================================

// parseFuncDecl parses: function IDENT ( params ) block
func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	start := p.advance() // consume 'function'
	decl := &ast.FuncDecl{}

	nameTok, ok := p.expect(token.IDENT)
	if !ok {
		p.synchronize()
		decl.Span = p.makeSpan(start.Span.Start)
		return decl
	}
	decl.Name = nameTok.Lexeme

	decl.Params = p.parseParamList()
	decl.Body = p.parseBlock()
	decl.Span = p.makeSpan(start.Span.Start)
	return decl
}

// parseClassDecl parses: class IDENT { constructor / methods }
func (p *Parser) parseClassDecl() *ast.ClassDecl {
	start := p.advance() // consume 'class'
	decl := &ast.ClassDecl{}

	nameTok, ok := p.expect(token.IDENT)
	if !ok {
		p.synchronize()
		decl.Span = p.makeSpan(start.Span.Start)
		return decl
	}
	decl.Name = nameTok.Lexeme

	if _, ok := p.expect(token.LBRACE); !ok {
		p.synchronize()
		decl.Span = p.makeSpan(start.Span.Start)
		return decl
	}

	p.skipSep()
	for !p.check(token.RBRACE) && !p.isAtEnd() {
		if p.check(token.KW_CONSTRUCTOR) {
			decl.Constructor = p.parseConstructorDecl()
		} else if p.check(token.IDENT) {
			decl.Methods = append(decl.Methods, p.parseMethodDecl())
		} else {
			tok := p.peek()
			p.error("E2003", tok.Span, fmt.Sprintf("expected method or constructor, got '%s'", tok.Lexeme))
			p.synchronize()
		}
		p.skipSep()
	}

	p.expect(token.RBRACE)
	decl.Span = p.makeSpan(start.Span.Start)
	return decl
}

func (p *Parser) parseConstructorDecl() *ast.ConstructorDecl {
	start := p.advance() // consume 'constructor'
	decl := &ast.ConstructorDecl{}
	decl.Params = p.parseParamList()
	decl.Body = p.parseBlock()
	decl.Span = p.makeSpan(start.Span.Start)
	return decl
}

func (p *Parser) parseMethodDecl() *ast.MethodDecl {
	start := p.advance() // consume method name (IDENT)
	decl := &ast.MethodDecl{Name: start.Lexeme}
	decl.Params = p.parseParamList()
	decl.Body = p.parseBlock()
	decl.Span = p.makeSpan(start.Span.Start)
	return decl
}

// parseParamList parses: ( ident, ident, ... )
func (p *Parser) parseParamList() []string {
	var params []string

	if _, ok := p.expect(token.LPAREN); !ok {
		return params
	}

	if !p.check(token.RPAREN) {
		nameTok, ok := p.expect(token.IDENT)
		if ok {
			params = append(params, nameTok.Lexeme)
		}
		for p.check(token.COMMA) {
			p.advance() // consume ','
			p.skipNewlines()
			nameTok, ok = p.expect(token.IDENT)
			if ok {
				params = append(params, nameTok.Lexeme)
			}
		}
	}

	p.expect(token.RPAREN)
	return params
}

// ============================================================
// Expression parsing (Pratt / precedence climbing)
// ============================================================

// parseExpr parses an expression with the given minimum binding power.
func (p *Parser) parseExpr(minBP int) ast.Expr {
	left := p.nud()
	if left == nil {
		return nil
	}

	for {
		kind := p.peekKind()
		bp := infixBP(kind)
		if bp <= minBP {
			break
		}
		left = p.led(left)
	}

	return left
}

// nud handles prefix (null denotation) parsing.
func (p *Parser) nud() ast.Expr {
	tok := p.peek()

	switch tok.Kind {
	case token.INT:
		p.advance()
		val, _ := strconv.ParseInt(tok.Lexeme, 10, 64)
		return &ast.IntLiteral{
			ExprBase: makeExprBase(tok.Span.Start, tok.Span.End),
			Value:    val,
		}

	case token.FLOAT:
		p.advance()
		val, _ := strconv.ParseFloat(tok.Lexeme, 64)
		return &ast.FloatLiteral{
			ExprBase: makeExprBase(tok.Span.Start, tok.Span.End),
			Value:    val,
		}

	case token.STRING:
		p.advance()
		return &ast.StringLiteral{
			ExprBase: makeExprBase(tok.Span.Start, tok.Span.End),
			Value:    tok.Lexeme,
		}

	case token.KW_TRUE:
		p.advance()
		return &ast.BoolLiteral{
			ExprBase: makeExprBase(tok.Span.Start, tok.Span.End),
			Value:    true,
		}

	case token.KW_FALSE:
		p.advance()
		return &ast.BoolLiteral{
			ExprBase: makeExprBase(tok.Span.Start, tok.Span.End),
			Value:    false,
		}

	case token.KW_NULL:
		p.advance()
		return &ast.NullLiteral{
			ExprBase: makeExprBase(tok.Span.Start, tok.Span.End),
		}

	case token.KW_THIS:
		p.advance()
		return &ast.ThisExpr{
			ExprBase: makeExprBase(tok.Span.Start, tok.Span.End),
		}

	case token.IDENT:
		p.advance()
		return &ast.IdentExpr{
			ExprBase: makeExprBase(tok.Span.Start, tok.Span.End),
			Name:     tok.Lexeme,
		}

	case token.LPAREN:
		// Grouped expression: ( expr )
		p.advance() // consume '('
		p.skipNewlines()
		expr := p.parseExpr(bpNone)
		p.skipNewlines()
		p.expect(token.RPAREN)
		return expr

	case token.BANG:
		// Unary: !expr
		p.advance()
		p.skipNewlines()
		operand := p.parseExpr(bpPrefix)
		return &ast.UnaryExpr{
			ExprBase: makeExprBase(tok.Span.Start, operand.GetSpan().End),
			Op:       token.BANG,
			Operand:  operand,
		}

	case token.MINUS:
		// Unary: -expr
		p.advance()
		p.skipNewlines()
		operand := p.parseExpr(bpPrefix)
		return &ast.UnaryExpr{
			ExprBase: makeExprBase(tok.Span.Start, operand.GetSpan().End),
			Op:       token.MINUS,
			Operand:  operand,
		}

	case token.KW_NEW:
		return p.parseNewExpr()

	case token.KW_FUNCTION:
		return p.parseFuncExpr()

	case token.LBRACKET:
		return p.parseArrayLiteral()

	default:
		return nil
	}
}

// led handles infix/postfix (left denotation) parsing.
func (p *Parser) led(left ast.Expr) ast.Expr {
	tok := p.peek()

	switch tok.Kind {
	case token.PLUS, token.MINUS, token.STAR, token.SLASH, token.PERCENT,
		token.EQ, token.NEQ, token.LT, token.LTE, token.GT, token.GTE,
		token.AND, token.OR:
		// Binary infix operator (left-associative)
		bp := infixBP(tok.Kind)
		p.advance()
		p.skipNewlines() // allow continuation on next line after operator
		right := p.parseExpr(bp)
		return &ast.BinaryExpr{
			ExprBase: makeExprBase(left.GetSpan().Start, right.GetSpan().End),
			Op:       tok.Kind,
			Left:     left,
			Right:    right,
		}

	case token.LPAREN:
		// Call expression: callee(args)
		return p.parseCallExpr(left)

	case token.LBRACKET:
		// Index expression: object[index]
		p.advance() // consume '['
		p.skipNewlines()
		index := p.parseExpr(bpNone)
		p.skipNewlines()
		end, _ := p.expect(token.RBRACKET)
		return &ast.IndexExpr{
			ExprBase: makeExprBase(left.GetSpan().Start, end.Span.End),
			Object:   left,
			Index:    index,
		}

	case token.DOT:
		// Member access: object.property
		p.advance() // consume '.'
		p.skipNewlines()
		propTok, _ := p.expect(token.IDENT)
		return &ast.MemberExpr{
			ExprBase: makeExprBase(left.GetSpan().Start, propTok.Span.End),
			Object:   left,
			Property: propTok.Lexeme,
		}

	default:
		return left
	}
}

// parseCallExpr parses: callee ( args )
func (p *Parser) parseCallExpr(callee ast.Expr) *ast.CallExpr {
	p.advance() // consume '('
	var args []ast.Expr

	p.skipNewlines()
	if !p.check(token.RPAREN) {
		args = append(args, p.parseExpr(bpNone))
		for p.check(token.COMMA) {
			p.advance() // consume ','
			p.skipNewlines()
			args = append(args, p.parseExpr(bpNone))
		}
	}
	p.skipNewlines()
	end, _ := p.expect(token.RPAREN)

	return &ast.CallExpr{
		ExprBase: makeExprBase(callee.GetSpan().Start, end.Span.End),
		Callee:   callee,
		Args:     args,
	}
}

// parseNewExpr parses: new ClassName(args)
func (p *Parser) parseNewExpr() *ast.NewExpr {
	start := p.advance() // consume 'new'

	nameTok, ok := p.expect(token.IDENT)
	if !ok {
		return &ast.NewExpr{
			ExprBase: makeExprBase(start.Span.Start, p.prevEnd()),
		}
	}

	var args []ast.Expr
	if _, ok := p.expect(token.LPAREN); ok {
		p.skipNewlines()
		if !p.check(token.RPAREN) {
			args = append(args, p.parseExpr(bpNone))
			for p.check(token.COMMA) {
				p.advance()
				p.skipNewlines()
				args = append(args, p.parseExpr(bpNone))
			}
		}
		p.skipNewlines()
		p.expect(token.RPAREN)
	}

	return &ast.NewExpr{
		ExprBase:  makeExprBase(start.Span.Start, p.prevEnd()),
		ClassName: nameTok.Lexeme,
		Args:      args,
	}
}

// ============================================================
// For loop parsing
// ============================================================

// parseForStmt dispatches between C-style for and for-of.
func (p *Parser) parseForStmt() ast.Stmt {
	start := p.advance() // consume 'for'

	if _, ok := p.expect(token.LPAREN); !ok {
		p.synchronize()
		return &ast.ExprStmt{StmtBase: makeStmtBase(start.Span.Start, p.prevEnd())}
	}

	p.skipNewlines()

	// Detect for-of: for (var IDENT of expr)
	if p.check(token.KW_VAR) && p.pos+2 < len(p.tokens) &&
		p.tokens[p.pos+1].Kind == token.IDENT &&
		p.tokens[p.pos+2].Kind == token.KW_OF {
		return p.parseForOfBody(start)
	}

	// C-style for loop: for (init; cond; update)
	return p.parseCStyleFor(start)
}

// parseForOfBody parses the rest of: for ( var IDENT of expr ) block
func (p *Parser) parseForOfBody(start token.Token) *ast.ForOfStmt {
	p.advance() // consume 'var'
	nameTok := p.advance() // consume IDENT
	p.advance() // consume 'of'
	p.skipNewlines()

	iterable := p.parseExpr(bpNone)

	p.skipNewlines()
	p.expect(token.RPAREN)

	body := p.parseBlock()

	return &ast.ForOfStmt{
		StmtBase: makeStmtBase(start.Span.Start, p.prevEnd()),
		VarName:  nameTok.Lexeme,
		Iterable: iterable,
		Body:     body,
	}
}

// parseCStyleFor parses: for ( [init]; [cond]; [update] ) block
func (p *Parser) parseCStyleFor(start token.Token) *ast.ForStmt {
	stmt := &ast.ForStmt{}

	// Init (optional)
	p.skipNewlines()
	if !p.check(token.SEMICOLON) {
		if p.match(token.KW_VAR, token.KW_CONST) {
			stmt.Init = p.parseVarDecl()
		} else {
			stmt.Init = p.parseSimpleStmt()
		}
	}
	p.expect(token.SEMICOLON)

	// Condition (optional)
	p.skipNewlines()
	if !p.check(token.SEMICOLON) {
		stmt.Condition = p.parseExpr(bpNone)
	}
	p.expect(token.SEMICOLON)

	// Update (optional)
	p.skipNewlines()
	if !p.check(token.RPAREN) {
		stmt.Update = p.parseSimpleStmt()
	}
	p.expect(token.RPAREN)

	stmt.Body = p.parseBlock()
	stmt.StmtBase = makeStmtBase(start.Span.Start, p.prevEnd())
	return stmt
}

// parseFuncExpr parses: function [name] ( params ) block
func (p *Parser) parseFuncExpr() *ast.FuncExpr {
	start := p.advance() // consume 'function'
	expr := &ast.FuncExpr{}

	// Optional name
	if p.check(token.IDENT) {
		expr.Name = p.advance().Lexeme
	}

	expr.Params = p.parseParamList()
	expr.Body = p.parseBlock()
	expr.ExprBase = makeExprBase(start.Span.Start, p.prevEnd())
	return expr
}

// parseArrayLiteral parses: [ expr, expr, ... ]
func (p *Parser) parseArrayLiteral() *ast.ArrayLiteral {
	start := p.advance() // consume '['
	var elements []ast.Expr

	p.skipNewlines()
	if !p.check(token.RBRACKET) {
		elements = append(elements, p.parseExpr(bpNone))
		for p.check(token.COMMA) {
			p.advance() // consume ','
			p.skipNewlines()
			if p.check(token.RBRACKET) {
				break // trailing comma
			}
			elements = append(elements, p.parseExpr(bpNone))
		}
	}
	p.skipNewlines()
	end, _ := p.expect(token.RBRACKET)

	return &ast.ArrayLiteral{
		ExprBase: makeExprBase(start.Span.Start, end.Span.End),
		Elements: elements,
	}
}

// compoundToOp maps compound assignment token to binary operator.
func compoundToOp(kind token.Kind) token.Kind {
	switch kind {
	case token.PLUS_ASSIGN:
		return token.PLUS
	case token.MINUS_ASSIGN:
		return token.MINUS
	case token.STAR_ASSIGN:
		return token.STAR
	case token.SLASH_ASSIGN:
		return token.SLASH
	default:
		return token.PLUS
	}
}

// ============================================================
// Span helpers
// ============================================================

func (p *Parser) prevEnd() span.Position {
	if p.pos > 0 && p.pos-1 < len(p.tokens) {
		return p.tokens[p.pos-1].Span.End
	}
	return p.peek().Span.Start
}

func (p *Parser) makeSpan(start span.Position) span.Span {
	return span.Span{Start: start, End: p.prevEnd()}
}

func makeExprBase(start, end span.Position) ast.ExprBase {
	return ast.ExprBase{NodeBase: ast.NodeBase{Span: span.Span{Start: start, End: end}}}
}

func makeStmtBase(start, end span.Position) ast.StmtBase {
	return ast.StmtBase{NodeBase: ast.NodeBase{Span: span.Span{Start: start, End: end}}}
}
