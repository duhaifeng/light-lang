// Package ast defines the abstract syntax tree for light-lang.
package ast

import (
	"light-lang/internal/span"
	"light-lang/internal/token"
)

// ============================================================
// Node interfaces
// ============================================================

// Node is the interface implemented by all AST nodes.
type Node interface {
	nodeNode()
	GetSpan() span.Span
}

// Expr is the interface for expression nodes.
type Expr interface {
	Node
	exprNode()
}

// Stmt is the interface for statement nodes.
type Stmt interface {
	Node
	stmtNode()
}

// ============================================================
// Base types (embedded to provide common fields)
// ============================================================

// NodeBase provides the common Span field for all AST nodes.
type NodeBase struct {
	Span span.Span
}

func (n NodeBase) nodeNode()          {}
func (n NodeBase) GetSpan() span.Span { return n.Span }

// ExprBase is embedded by all expression nodes.
type ExprBase struct{ NodeBase }

func (ExprBase) exprNode() {}

// StmtBase is embedded by all statement nodes.
type StmtBase struct{ NodeBase }

func (StmtBase) stmtNode() {}

// ============================================================
// File (top-level AST root)
// ============================================================

// File represents the entire source file.
type File struct {
	NodeBase
	Body []Node // top-level statements and declarations
}

// ============================================================
// Expressions
// ============================================================

// IdentExpr represents an identifier reference.
type IdentExpr struct {
	ExprBase
	Name string
}

// IntLiteral represents an integer literal.
type IntLiteral struct {
	ExprBase
	Value int64
}

// FloatLiteral represents a floating-point literal.
type FloatLiteral struct {
	ExprBase
	Value float64
}

// StringLiteral represents a string literal.
type StringLiteral struct {
	ExprBase
	Value string
}

// BoolLiteral represents true or false.
type BoolLiteral struct {
	ExprBase
	Value bool
}

// NullLiteral represents null.
type NullLiteral struct {
	ExprBase
}

// ThisExpr represents the 'this' keyword.
type ThisExpr struct {
	ExprBase
}

// UnaryExpr represents a unary operation: !x, -x.
type UnaryExpr struct {
	ExprBase
	Op      token.Kind
	Operand Expr
}

// BinaryExpr represents a binary operation: a + b, x == y.
type BinaryExpr struct {
	ExprBase
	Op    token.Kind
	Left  Expr
	Right Expr
}

// CallExpr represents a function call: f(a, b).
type CallExpr struct {
	ExprBase
	Callee Expr
	Args   []Expr
}

// IndexExpr represents indexing: a[i].
type IndexExpr struct {
	ExprBase
	Object Expr
	Index  Expr
}

// MemberExpr represents member access: a.b.
type MemberExpr struct {
	ExprBase
	Object   Expr
	Property string
}

// NewExpr represents object creation: new ClassName(args).
type NewExpr struct {
	ExprBase
	ClassName string
	Args      []Expr
}

// ArrayLiteral represents an array literal: [a, b, c].
type ArrayLiteral struct {
	ExprBase
	Elements []Expr
}

// FuncExpr represents a function expression: function(params) { body } or (x) => expr.
type FuncExpr struct {
	ExprBase
	Name   string // may be empty for anonymous / arrow functions
	Params []string
	Body   *BlockStmt
}

// TernaryExpr represents a ternary: cond ? then : else.
type TernaryExpr struct {
	ExprBase
	Condition Expr
	Then      Expr
	Else      Expr
}

// MapLiteral represents a map literal: { key: val, ... }.
type MapLiteral struct {
	ExprBase
	Keys   []Expr // StringLiteral (identifier keys are converted to strings)
	Values []Expr
}

// SuperExpr represents the 'super' keyword.
type SuperExpr struct {
	ExprBase
}

// TemplateLiteral represents a template string: `text ${expr} text`.
// Parts has len(Exprs)+1 elements; Parts[i] is the text before Exprs[i].
type TemplateLiteral struct {
	ExprBase
	Parts []string // static text segments
	Exprs []Expr   // interpolated expressions
}

// ============================================================
// Statements
// ============================================================

// ExprStmt wraps an expression used as a statement.
type ExprStmt struct {
	StmtBase
	Expr Expr
}

// AssignStmt represents an assignment: target = value.
type AssignStmt struct {
	StmtBase
	Target Expr // must be a valid lvalue (ident, member, index)
	Value  Expr
}

// VarDeclStmt represents a variable declaration: var x = expr / const x = expr.
type VarDeclStmt struct {
	StmtBase
	Name    string
	IsConst bool
	Init    Expr // may be nil if no initializer
}

// ReturnStmt represents a return statement.
type ReturnStmt struct {
	StmtBase
	Value Expr // may be nil
}

// BreakStmt represents a break statement.
type BreakStmt struct {
	StmtBase
}

// ContinueStmt represents a continue statement.
type ContinueStmt struct {
	StmtBase
}

// BlockStmt represents a block of statements: { ... }.
type BlockStmt struct {
	StmtBase
	Stmts []Node
}

// IfStmt represents an if/else if/else chain.
type IfStmt struct {
	StmtBase
	Condition Expr
	Body      *BlockStmt
	ElseIfs   []ElseIfClause
	ElseBody  *BlockStmt // may be nil
}

// ElseIfClause represents a single "else if" branch.
type ElseIfClause struct {
	Span      span.Span
	Condition Expr
	Body      *BlockStmt
}

// WhileStmt represents a while loop.
type WhileStmt struct {
	StmtBase
	Condition Expr
	Body      *BlockStmt
}

// ForStmt represents a C-style for loop: for (init; condition; update) { body }.
type ForStmt struct {
	StmtBase
	Init      Node // VarDeclStmt, AssignStmt, ExprStmt, or nil
	Condition Expr // or nil (infinite loop)
	Update    Node // AssignStmt, ExprStmt, or nil
	Body      *BlockStmt
}

// ForOfStmt represents a for-of loop: for (var name of iterable) { body }.
type ForOfStmt struct {
	StmtBase
	VarName  string
	Iterable Expr
	Body     *BlockStmt
}

// TryStmt represents a try/catch block.
type TryStmt struct {
	StmtBase
	Body       *BlockStmt
	CatchParam string     // variable name in catch(e), may be empty
	CatchBody  *BlockStmt // may be nil if no catch
}

// ThrowStmt represents a throw statement.
type ThrowStmt struct {
	StmtBase
	Value Expr
}

// MatchStmt represents: match (subject) { case pattern => body, ... }.
type MatchStmt struct {
	StmtBase
	Subject Expr
	Arms    []MatchArm
}

// MatchArm represents a single arm in a match statement.
type MatchArm struct {
	Span      span.Span
	Patterns  []Expr     // value expressions to compare (nil for default/binding)
	BindVar   string     // variable binding name (empty if value pattern or default)
	Guard     Expr       // guard condition for binding arms (nil if no guard)
	Body      *BlockStmt // arm body
	IsDefault bool       // true for _ => ...
}

// ============================================================
// Declarations (also implement Stmt for top-level use)
// ============================================================

// FuncDecl represents a function declaration: function name(params) { ... }.
type FuncDecl struct {
	StmtBase
	Name   string
	Params []string
	Body   *BlockStmt
}

// ClassDecl represents a class declaration.
type ClassDecl struct {
	StmtBase
	Name        string
	SuperClass  string   // may be empty if no extends
	Implements  []string // interface names (may be empty)
	Constructor *ConstructorDecl // may be nil
	Methods     []*MethodDecl
}

// EnumDecl represents an enum declaration: enum Name { Variant1, Variant2, ... }.
type EnumDecl struct {
	StmtBase
	Name     string
	Variants []string
}

// InterfaceDecl represents an interface declaration.
type InterfaceDecl struct {
	StmtBase
	Name    string
	Methods []InterfaceMethodSig
}

// InterfaceMethodSig represents a method signature in an interface.
type InterfaceMethodSig struct {
	Name       string
	ParamCount int
}

// ConstructorDecl represents a constructor inside a class.
type ConstructorDecl struct {
	Span   span.Span
	Params []string
	Body   *BlockStmt
}

// MethodDecl represents a method inside a class.
type MethodDecl struct {
	Span   span.Span
	Name   string
	Params []string
	Body   *BlockStmt
}
