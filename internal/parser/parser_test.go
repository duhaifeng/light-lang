package parser

import (
	"encoding/json"
	"light-lang/internal/ast"
	"light-lang/internal/lexer"
	"testing"
)

// helper: parse source and return AST + check for no errors
func parseOK(t *testing.T, source string) *ast.File {
	t.Helper()
	l := lexer.New(source, "test.lt")
	tokens, lexDiags := l.Tokenize()
	if len(lexDiags) > 0 {
		t.Fatalf("lex errors: %v", lexDiags)
	}
	p := New(tokens)
	file, parseDiags := p.ParseFile()
	if len(parseDiags) > 0 {
		t.Fatalf("parse errors: %v", parseDiags)
	}
	return file
}

// helper: parse and return JSON string (for golden-test style checks)
func parseToJSON(t *testing.T, source string) string {
	t.Helper()
	file := parseOK(t, source)
	m := ast.NodeToMap(file)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("json error: %v", err)
	}
	return string(data)
}

func TestParseVarDecl(t *testing.T) {
	file := parseOK(t, `var x = 42`)
	if len(file.Body) != 1 {
		t.Fatalf("expected 1 node, got %d", len(file.Body))
	}
	decl, ok := file.Body[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", file.Body[0])
	}
	if decl.Name != "x" {
		t.Errorf("expected name 'x', got %q", decl.Name)
	}
	if decl.IsConst {
		t.Error("expected var, got const")
	}
}

func TestParseConstDecl(t *testing.T) {
	file := parseOK(t, `const PI = 3.14`)
	decl, ok := file.Body[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("expected VarDeclStmt, got %T", file.Body[0])
	}
	if !decl.IsConst {
		t.Error("expected const")
	}
	if decl.Name != "PI" {
		t.Errorf("expected name 'PI', got %q", decl.Name)
	}
}

func TestParseBinaryExpr(t *testing.T) {
	file := parseOK(t, `var z = 1 + 2 * 3`)
	decl := file.Body[0].(*ast.VarDeclStmt)
	// init should be BinaryExpr: 1 + (2 * 3)
	binExpr, ok := decl.Init.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", decl.Init)
	}
	if binExpr.Op.String() != "+" {
		t.Errorf("expected '+', got %q", binExpr.Op.String())
	}
	// right should be BinaryExpr: 2 * 3
	rightBin, ok := binExpr.Right.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected right BinaryExpr, got %T", binExpr.Right)
	}
	if rightBin.Op.String() != "*" {
		t.Errorf("expected '*', got %q", rightBin.Op.String())
	}
}

func TestParseIfStmt(t *testing.T) {
	source := `if (x > 0) {
  print(x)
} else if (x == 0) {
  print(0)
} else {
  print(-1)
}`
	file := parseOK(t, source)
	ifStmt, ok := file.Body[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("expected IfStmt, got %T", file.Body[0])
	}
	if ifStmt.Condition == nil {
		t.Fatal("condition is nil")
	}
	if len(ifStmt.ElseIfs) != 1 {
		t.Errorf("expected 1 else-if, got %d", len(ifStmt.ElseIfs))
	}
	if ifStmt.ElseBody == nil {
		t.Error("else body is nil")
	}
}

func TestParseWhileStmt(t *testing.T) {
	source := `while (i < 10) {
  i = i + 1
}`
	file := parseOK(t, source)
	whileStmt, ok := file.Body[0].(*ast.WhileStmt)
	if !ok {
		t.Fatalf("expected WhileStmt, got %T", file.Body[0])
	}
	if whileStmt.Condition == nil {
		t.Fatal("condition is nil")
	}
	if whileStmt.Body == nil {
		t.Fatal("body is nil")
	}
}

func TestParseFuncDecl(t *testing.T) {
	source := `function add(a, b) {
  return a + b
}`
	file := parseOK(t, source)
	fn, ok := file.Body[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected FuncDecl, got %T", file.Body[0])
	}
	if fn.Name != "add" {
		t.Errorf("expected name 'add', got %q", fn.Name)
	}
	if len(fn.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fn.Params))
	}
}

func TestParseClassDecl(t *testing.T) {
	source := `class Point {
  constructor(x, y) {
    this.x = x
    this.y = y
  }
  move(dx, dy) {
    this.x = this.x + dx
  }
}`
	file := parseOK(t, source)
	cls, ok := file.Body[0].(*ast.ClassDecl)
	if !ok {
		t.Fatalf("expected ClassDecl, got %T", file.Body[0])
	}
	if cls.Name != "Point" {
		t.Errorf("expected name 'Point', got %q", cls.Name)
	}
	if cls.Constructor == nil {
		t.Fatal("constructor is nil")
	}
	if len(cls.Constructor.Params) != 2 {
		t.Errorf("expected 2 constructor params, got %d", len(cls.Constructor.Params))
	}
	if len(cls.Methods) != 1 {
		t.Errorf("expected 1 method, got %d", len(cls.Methods))
	}
}

func TestParseCallExpr(t *testing.T) {
	file := parseOK(t, `print(1, 2, 3)`)
	stmt, ok := file.Body[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("expected ExprStmt, got %T", file.Body[0])
	}
	call, ok := stmt.Expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", stmt.Expr)
	}
	if len(call.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(call.Args))
	}
}

func TestParseMemberExpr(t *testing.T) {
	file := parseOK(t, `obj.method(1).prop`)
	stmt := file.Body[0].(*ast.ExprStmt)
	member, ok := stmt.Expr.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("expected MemberExpr, got %T", stmt.Expr)
	}
	if member.Property != "prop" {
		t.Errorf("expected property 'prop', got %q", member.Property)
	}
}

func TestParseNewExpr(t *testing.T) {
	file := parseOK(t, `var p = new Point(1, 2)`)
	decl := file.Body[0].(*ast.VarDeclStmt)
	newExpr, ok := decl.Init.(*ast.NewExpr)
	if !ok {
		t.Fatalf("expected NewExpr, got %T", decl.Init)
	}
	if newExpr.ClassName != "Point" {
		t.Errorf("expected 'Point', got %q", newExpr.ClassName)
	}
	if len(newExpr.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(newExpr.Args))
	}
}

func TestParseAssignment(t *testing.T) {
	file := parseOK(t, `x = 42`)
	assign, ok := file.Body[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("expected AssignStmt, got %T", file.Body[0])
	}
	ident, ok := assign.Target.(*ast.IdentExpr)
	if !ok {
		t.Fatalf("expected IdentExpr target, got %T", assign.Target)
	}
	if ident.Name != "x" {
		t.Errorf("expected 'x', got %q", ident.Name)
	}
}

func TestParseJSONOutput(t *testing.T) {
	jsonStr := parseToJSON(t, `var x = 1`)
	// Just make sure it's valid JSON and has the right structure
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["kind"] != "File" {
		t.Errorf("expected kind 'File', got %v", m["kind"])
	}
}

func TestParseErrorRecovery(t *testing.T) {
	// Missing closing paren - parser should still produce some output
	source := `var x = add(1, 2
var y = 3`
	l := lexer.New(source, "test.lt")
	tokens, _ := l.Tokenize()
	p := New(tokens)
	file, diags := p.ParseFile()

	if len(diags) == 0 {
		t.Error("expected parse errors")
	}
	// Should still parse something
	if file == nil {
		t.Fatal("file is nil")
	}
}
