package runtime

import (
	"fmt"
	"io"
	"light-lang/internal/ast"
	"light-lang/internal/span"
	"light-lang/internal/token"
	"sort"
	"strings"
)

// ============================================================
// Control flow signals
// ============================================================

// ExecSignal represents a control flow signal from statement execution.
type ExecSignal int

const (
	SigNone     ExecSignal = iota
	SigReturn              // return from function
	SigBreak               // break from loop
	SigContinue            // continue in loop
)

// ExecResult carries a control flow signal and an optional value (for return).
type ExecResult struct {
	Signal ExecSignal
	Value  Value
}

var resultNone = ExecResult{Signal: SigNone}

// ============================================================
// Runtime error
// ============================================================

// RuntimeError represents an error during interpretation.
type RuntimeError struct {
	Message string
	Span    span.Span
}

func (e *RuntimeError) Error() string {
	return fmt.Sprintf("runtime error at %d:%d: %s", e.Span.Start.Line, e.Span.Start.Column, e.Message)
}

func runtimeErr(s span.Span, format string, args ...interface{}) *RuntimeError {
	return &RuntimeError{Message: fmt.Sprintf(format, args...), Span: s}
}

// ThrownError represents a user-thrown error (via throw statement).
type ThrownError struct {
	Value Value
	Span  span.Span
}

func (e *ThrownError) Error() string {
	return fmt.Sprintf("uncaught throw at %d:%d: %s", e.Span.Start.Line, e.Span.Start.Column, e.Value.String())
}

// ============================================================
// Interpreter
// ============================================================

// Interpreter walks the AST and executes it.
type Interpreter struct {
	global *Environment
	env    *Environment
	output io.Writer
}

// NewInterpreter creates a new interpreter with built-in functions registered.
func NewInterpreter(output io.Writer) *Interpreter {
	global := NewEnvironment(nil)
	RegisterBuiltins(global, output)
	return &Interpreter{
		global: global,
		env:    global,
		output: output,
	}
}

// Run executes the entire AST file.
func (i *Interpreter) Run(file *ast.File) error {
	for _, node := range file.Body {
		result, err := i.execNode(node)
		if err != nil {
			return err
		}
		if result.Signal == SigReturn {
			return runtimeErr(node.GetSpan(), "return outside of function")
		}
		if result.Signal == SigBreak {
			return runtimeErr(node.GetSpan(), "break outside of loop")
		}
		if result.Signal == SigContinue {
			return runtimeErr(node.GetSpan(), "continue outside of loop")
		}
	}
	return nil
}

// Env returns the current environment (useful for REPL).
func (i *Interpreter) Env() *Environment {
	return i.env
}

// ============================================================
// Node dispatch
// ============================================================

func (i *Interpreter) execNode(node ast.Node) (ExecResult, error) {
	switch n := node.(type) {
	case *ast.FuncDecl:
		return i.execFuncDecl(n)
	case *ast.ClassDecl:
		return i.execClassDecl(n)
	case ast.Stmt:
		return i.execStmt(n)
	default:
		return resultNone, runtimeErr(node.GetSpan(), "unexpected node type: %T", node)
	}
}

// ============================================================
// Statement execution
// ============================================================

func (i *Interpreter) execStmt(stmt ast.Stmt) (ExecResult, error) {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		_, err := i.evalExpr(s.Expr)
		return resultNone, err

	case *ast.VarDeclStmt:
		return i.execVarDecl(s)

	case *ast.AssignStmt:
		return i.execAssign(s)

	case *ast.ReturnStmt:
		var val Value = NullVal{}
		if s.Value != nil {
			v, err := i.evalExpr(s.Value)
			if err != nil {
				return resultNone, err
			}
			val = v
		}
		return ExecResult{Signal: SigReturn, Value: val}, nil

	case *ast.BreakStmt:
		return ExecResult{Signal: SigBreak}, nil

	case *ast.ContinueStmt:
		return ExecResult{Signal: SigContinue}, nil

	case *ast.IfStmt:
		return i.execIf(s)

	case *ast.WhileStmt:
		return i.execWhile(s)

	case *ast.ForStmt:
		return i.execFor(s)

	case *ast.ForOfStmt:
		return i.execForOf(s)

	case *ast.TryStmt:
		return i.execTry(s)

	case *ast.ThrowStmt:
		return i.execThrow(s)

	case *ast.BlockStmt:
		return i.execBlock(s, NewEnvironment(i.env))

	case *ast.FuncDecl:
		return i.execFuncDecl(s)

	case *ast.ClassDecl:
		return i.execClassDecl(s)

	default:
		return resultNone, runtimeErr(stmt.GetSpan(), "unhandled statement type: %T", stmt)
	}
}

func (i *Interpreter) execVarDecl(s *ast.VarDeclStmt) (ExecResult, error) {
	var val Value = NullVal{}
	if s.Init != nil {
		v, err := i.evalExpr(s.Init)
		if err != nil {
			return resultNone, err
		}
		val = v
	}
	if err := i.env.Define(s.Name, val, s.IsConst); err != nil {
		return resultNone, runtimeErr(s.GetSpan(), "%s", err)
	}
	return resultNone, nil
}

func (i *Interpreter) execAssign(s *ast.AssignStmt) (ExecResult, error) {
	val, err := i.evalExpr(s.Value)
	if err != nil {
		return resultNone, err
	}

	switch target := s.Target.(type) {
	case *ast.IdentExpr:
		if err := i.env.Set(target.Name, val); err != nil {
			return resultNone, runtimeErr(s.GetSpan(), "%s", err)
		}
	case *ast.MemberExpr:
		obj, err := i.evalExpr(target.Object)
		if err != nil {
			return resultNone, err
		}
		switch o := obj.(type) {
		case *ObjectVal:
			o.Props[target.Property] = val
		case *MapVal:
			key := target.Property
			if _, exists := o.Values[key]; !exists {
				o.Keys = append(o.Keys, key)
			}
			o.Values[key] = val
		default:
			return resultNone, runtimeErr(s.GetSpan(), "cannot set property on value of type '%s'", obj.TypeName())
		}
	case *ast.IndexExpr:
		obj, err := i.evalExpr(target.Object)
		if err != nil {
			return resultNone, err
		}
		idx, err := i.evalExpr(target.Index)
		if err != nil {
			return resultNone, err
		}
		switch o := obj.(type) {
		case *ArrayVal:
			idxInt, ok := ToInt64(idx)
			if !ok {
				return resultNone, runtimeErr(s.GetSpan(), "array index must be an integer")
			}
			if idxInt < 0 || int(idxInt) >= len(o.Elements) {
				return resultNone, runtimeErr(s.GetSpan(), "array index %d out of range (length %d)", idxInt, len(o.Elements))
			}
			o.Elements[idxInt] = val
		case *MapVal:
			keyStr, ok := idx.(StringVal)
			if !ok {
				return resultNone, runtimeErr(s.GetSpan(), "map key must be a string, got '%s'", idx.TypeName())
			}
			key := string(keyStr)
			if _, exists := o.Values[key]; !exists {
				o.Keys = append(o.Keys, key)
			}
			o.Values[key] = val
		default:
			return resultNone, runtimeErr(s.GetSpan(), "cannot index-assign value of type '%s'", obj.TypeName())
		}
	default:
		return resultNone, runtimeErr(s.GetSpan(), "invalid assignment target")
	}
	return resultNone, nil
}

func (i *Interpreter) execIf(s *ast.IfStmt) (ExecResult, error) {
	cond, err := i.evalExpr(s.Condition)
	if err != nil {
		return resultNone, err
	}

	if IsTruthy(cond) {
		return i.execBlock(s.Body, NewEnvironment(i.env))
	}

	for _, elseIf := range s.ElseIfs {
		cond, err := i.evalExpr(elseIf.Condition)
		if err != nil {
			return resultNone, err
		}
		if IsTruthy(cond) {
			return i.execBlock(elseIf.Body, NewEnvironment(i.env))
		}
	}

	if s.ElseBody != nil {
		return i.execBlock(s.ElseBody, NewEnvironment(i.env))
	}

	return resultNone, nil
}

func (i *Interpreter) execWhile(s *ast.WhileStmt) (ExecResult, error) {
	for {
		cond, err := i.evalExpr(s.Condition)
		if err != nil {
			return resultNone, err
		}
		if !IsTruthy(cond) {
			break
		}

		result, err := i.execBlock(s.Body, NewEnvironment(i.env))
		if err != nil {
			return resultNone, err
		}
		if result.Signal == SigBreak {
			break
		}
		if result.Signal == SigReturn {
			return result, nil // propagate return
		}
		// SigContinue: just continue the loop
	}
	return resultNone, nil
}

func (i *Interpreter) execBlock(block *ast.BlockStmt, blockEnv *Environment) (ExecResult, error) {
	prevEnv := i.env
	i.env = blockEnv
	defer func() { i.env = prevEnv }()

	for _, node := range block.Stmts {
		result, err := i.execNode(node)
		if err != nil {
			return resultNone, err
		}
		if result.Signal != SigNone {
			return result, nil // propagate signal
		}
	}
	return resultNone, nil
}

func (i *Interpreter) execFuncDecl(s *ast.FuncDecl) (ExecResult, error) {
	fn := &FuncVal{
		Name:    s.Name,
		Params:  s.Params,
		Body:    s.Body,
		Closure: i.env,
	}
	if err := i.env.Define(s.Name, fn, false); err != nil {
		return resultNone, runtimeErr(s.GetSpan(), "%s", err)
	}
	return resultNone, nil
}

func (i *Interpreter) execClassDecl(s *ast.ClassDecl) (ExecResult, error) {
	cls := &ClassVal{Decl: s, Env: i.env}

	// Resolve super class if extends is specified
	if s.SuperClass != "" {
		superVal, ok := i.env.Get(s.SuperClass)
		if !ok {
			return resultNone, runtimeErr(s.GetSpan(), "undefined class '%s'", s.SuperClass)
		}
		superCls, ok := superVal.(*ClassVal)
		if !ok {
			return resultNone, runtimeErr(s.GetSpan(), "'%s' is not a class", s.SuperClass)
		}
		cls.Super = superCls
	}

	if err := i.env.Define(s.Name, cls, false); err != nil {
		return resultNone, runtimeErr(s.GetSpan(), "%s", err)
	}
	return resultNone, nil
}

// ============================================================
// Expression evaluation
// ============================================================

func (i *Interpreter) evalExpr(expr ast.Expr) (Value, error) {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return IntVal(e.Value), nil
	case *ast.FloatLiteral:
		return FloatVal(e.Value), nil
	case *ast.StringLiteral:
		return StringVal(e.Value), nil
	case *ast.BoolLiteral:
		return BoolVal(e.Value), nil
	case *ast.NullLiteral:
		return NullVal{}, nil
	case *ast.ThisExpr:
		return i.evalThis(e)
	case *ast.IdentExpr:
		return i.evalIdent(e)
	case *ast.UnaryExpr:
		return i.evalUnary(e)
	case *ast.BinaryExpr:
		return i.evalBinary(e)
	case *ast.CallExpr:
		return i.evalCall(e)
	case *ast.MemberExpr:
		return i.evalMember(e)
	case *ast.IndexExpr:
		return i.evalIndex(e)
	case *ast.NewExpr:
		return i.evalNew(e)
	case *ast.ArrayLiteral:
		return i.evalArrayLiteral(e)
	case *ast.FuncExpr:
		return i.evalFuncExpr(e)
	case *ast.TernaryExpr:
		return i.evalTernary(e)
	case *ast.MapLiteral:
		return i.evalMapLiteral(e)
	case *ast.TemplateLiteral:
		return i.evalTemplateLiteral(e)
	case *ast.SuperExpr:
		return nil, runtimeErr(e.GetSpan(), "super can only be used as super() or super.method()")
	default:
		return nil, runtimeErr(expr.GetSpan(), "unhandled expression type: %T", expr)
	}
}

func (i *Interpreter) evalThis(e *ast.ThisExpr) (Value, error) {
	val, ok := i.env.Get("this")
	if !ok {
		return nil, runtimeErr(e.GetSpan(), "'this' used outside of a class method or constructor")
	}
	return val, nil
}

func (i *Interpreter) evalIdent(e *ast.IdentExpr) (Value, error) {
	val, ok := i.env.Get(e.Name)
	if !ok {
		return nil, runtimeErr(e.GetSpan(), "undefined variable '%s'", e.Name)
	}
	return val, nil
}

func (i *Interpreter) evalUnary(e *ast.UnaryExpr) (Value, error) {
	operand, err := i.evalExpr(e.Operand)
	if err != nil {
		return nil, err
	}

	switch e.Op {
	case token.BANG:
		return BoolVal(!IsTruthy(operand)), nil
	case token.MINUS:
		switch v := operand.(type) {
		case IntVal:
			return IntVal(-int64(v)), nil
		case FloatVal:
			return FloatVal(-float64(v)), nil
		default:
			return nil, runtimeErr(e.GetSpan(), "cannot negate value of type '%s'", operand.TypeName())
		}
	default:
		return nil, runtimeErr(e.GetSpan(), "unknown unary operator: %s", e.Op)
	}
}

func (i *Interpreter) evalBinary(e *ast.BinaryExpr) (Value, error) {
	// Short-circuit for logical operators
	if e.Op == token.AND || e.Op == token.OR {
		return i.evalLogical(e)
	}

	left, err := i.evalExpr(e.Left)
	if err != nil {
		return nil, err
	}
	right, err := i.evalExpr(e.Right)
	if err != nil {
		return nil, err
	}

	// String concatenation (auto-convert if one side is string)
	if e.Op == token.PLUS {
		_, leftIsStr := left.(StringVal)
		_, rightIsStr := right.(StringVal)
		if leftIsStr || rightIsStr {
			return StringVal(left.String() + right.String()), nil
		}
	}

	// Equality (works for all types)
	if e.Op == token.EQ {
		return BoolVal(valuesEqual(left, right)), nil
	}
	if e.Op == token.NEQ {
		return BoolVal(!valuesEqual(left, right)), nil
	}

	// Numeric operations
	leftF, leftOk := ToFloat64(left)
	rightF, rightOk := ToFloat64(right)
	if !leftOk || !rightOk {
		return nil, runtimeErr(e.GetSpan(), "cannot apply '%s' to '%s' and '%s'", e.Op, left.TypeName(), right.TypeName())
	}

	// Check if both are ints (for integer arithmetic)
	_, leftIsInt := left.(IntVal)
	_, rightIsInt := right.(IntVal)
	bothInt := leftIsInt && rightIsInt

	switch e.Op {
	case token.PLUS:
		if bothInt {
			return IntVal(int64(leftF) + int64(rightF)), nil
		}
		return FloatVal(leftF + rightF), nil
	case token.MINUS:
		if bothInt {
			return IntVal(int64(leftF) - int64(rightF)), nil
		}
		return FloatVal(leftF - rightF), nil
	case token.STAR:
		if bothInt {
			return IntVal(int64(leftF) * int64(rightF)), nil
		}
		return FloatVal(leftF * rightF), nil
	case token.SLASH:
		if rightF == 0 {
			return nil, runtimeErr(e.GetSpan(), "division by zero")
		}
		if bothInt {
			return IntVal(int64(leftF) / int64(rightF)), nil
		}
		return FloatVal(leftF / rightF), nil
	case token.PERCENT:
		if !bothInt {
			return nil, runtimeErr(e.GetSpan(), "modulo requires integer operands")
		}
		if int64(rightF) == 0 {
			return nil, runtimeErr(e.GetSpan(), "division by zero")
		}
		return IntVal(int64(leftF) % int64(rightF)), nil
	case token.LT:
		return BoolVal(leftF < rightF), nil
	case token.LTE:
		return BoolVal(leftF <= rightF), nil
	case token.GT:
		return BoolVal(leftF > rightF), nil
	case token.GTE:
		return BoolVal(leftF >= rightF), nil
	default:
		return nil, runtimeErr(e.GetSpan(), "unknown binary operator: %s", e.Op)
	}
}

func (i *Interpreter) evalLogical(e *ast.BinaryExpr) (Value, error) {
	left, err := i.evalExpr(e.Left)
	if err != nil {
		return nil, err
	}
	if e.Op == token.OR {
		if IsTruthy(left) {
			return left, nil // short-circuit
		}
		return i.evalExpr(e.Right)
	}
	// AND
	if !IsTruthy(left) {
		return left, nil // short-circuit
	}
	return i.evalExpr(e.Right)
}

func (i *Interpreter) evalCall(e *ast.CallExpr) (Value, error) {
	// Evaluate arguments
	args := make([]Value, len(e.Args))
	for idx, argExpr := range e.Args {
		val, err := i.evalExpr(argExpr)
		if err != nil {
			return nil, err
		}
		args[idx] = val
	}

	// Check for super() or super.method() calls
	if _, isSuper := e.Callee.(*ast.SuperExpr); isSuper {
		return i.callSuperConstructor(args, e.GetSpan())
	}
	if member, ok := e.Callee.(*ast.MemberExpr); ok {
		if _, isSuper := member.Object.(*ast.SuperExpr); isSuper {
			return i.callSuperMethod(member.Property, args, e.GetSpan())
		}
	}

	// Check for method call: obj.method(args)
	if member, ok := e.Callee.(*ast.MemberExpr); ok {
		obj, err := i.evalExpr(member.Object)
		if err != nil {
			return nil, err
		}

		switch o := obj.(type) {
		case *ObjectVal:
			return i.callMethod(o, member.Property, args, e.GetSpan())
		case *ArrayVal:
			return i.callArrayMethod(o, member.Property, args, e.GetSpan())
		case StringVal:
			return i.callStringMethod(string(o), member.Property, args, e.GetSpan())
		default:
			return nil, runtimeErr(e.GetSpan(), "cannot call method on value of type '%s'", obj.TypeName())
		}
	}

	// Regular call
	callee, err := i.evalExpr(e.Callee)
	if err != nil {
		return nil, err
	}

	return i.callValue(callee, args, e.GetSpan())
}

func (i *Interpreter) callValue(callee Value, args []Value, s span.Span) (Value, error) {
	switch fn := callee.(type) {
	case *FuncVal:
		return i.callFunc(fn, args, s)
	case *BuiltinVal:
		return fn.Fn(args)
	default:
		return nil, runtimeErr(s, "cannot call value of type '%s'", callee.TypeName())
	}
}

func (i *Interpreter) callFunc(fn *FuncVal, args []Value, s span.Span) (Value, error) {
	if len(args) != len(fn.Params) {
		return nil, runtimeErr(s, "%s() expects %d arguments, got %d", fn.Name, len(fn.Params), len(args))
	}

	// Create new scope from closure
	funcEnv := NewEnvironment(fn.Closure)
	for idx, param := range fn.Params {
		funcEnv.Define(param, args[idx], false)
	}

	result, err := i.execBlock(fn.Body, funcEnv)
	if err != nil {
		return nil, err
	}

	if result.Signal == SigReturn {
		return result.Value, nil
	}
	return NullVal{}, nil
}

func (i *Interpreter) callMethod(obj *ObjectVal, methodName string, args []Value, s span.Span) (Value, error) {
	// Walk the prototype chain to find the method
	method, methodClass := findMethod(obj.Class, methodName)
	if method != nil {
		if len(args) != len(method.Params) {
			return nil, runtimeErr(s, "%s.%s() expects %d arguments, got %d",
				obj.Class.Decl.Name, methodName, len(method.Params), len(args))
		}

		methodEnv := NewEnvironment(methodClass.Env)
		methodEnv.Define("this", obj, true)
		methodEnv.Define("__class__", methodClass, true)
		for idx, param := range method.Params {
			methodEnv.Define(param, args[idx], false)
		}

		result, err := i.execBlock(method.Body, methodEnv)
		if err != nil {
			return nil, err
		}
		if result.Signal == SigReturn {
			return result.Value, nil
		}
		return NullVal{}, nil
	}

	// Check if it's a property that's callable
	if propVal, exists := obj.Props[methodName]; exists {
		return i.callValue(propVal, args, s)
	}

	return nil, runtimeErr(s, "undefined method '%s' on class '%s'", methodName, obj.Class.Decl.Name)
}

// findMethod walks the class inheritance chain to find a method.
func findMethod(cls *ClassVal, name string) (*ast.MethodDecl, *ClassVal) {
	for cls != nil {
		for _, m := range cls.Decl.Methods {
			if m.Name == name {
				return m, cls
			}
		}
		cls = cls.Super
	}
	return nil, nil
}

// findConstructor walks the chain to find the nearest constructor.
func findConstructor(cls *ClassVal) (*ast.ConstructorDecl, *ClassVal) {
	for cls != nil {
		if cls.Decl.Constructor != nil {
			return cls.Decl.Constructor, cls
		}
		cls = cls.Super
	}
	return nil, nil
}

func (i *Interpreter) evalMember(e *ast.MemberExpr) (Value, error) {
	obj, err := i.evalExpr(e.Object)
	if err != nil {
		return nil, err
	}

	switch o := obj.(type) {
	case *ObjectVal:
		if val, exists := o.Props[e.Property]; exists {
			return val, nil
		}
		return NullVal{}, nil
	case *ArrayVal:
		if e.Property == "length" {
			return IntVal(len(o.Elements)), nil
		}
		return nil, runtimeErr(e.GetSpan(), "array has no property '%s'", e.Property)
	case *MapVal:
		if val, exists := o.Values[e.Property]; exists {
			return val, nil
		}
		return NullVal{}, nil
	case StringVal:
		if e.Property == "length" {
			return IntVal(len(string(o))), nil
		}
		return nil, runtimeErr(e.GetSpan(), "string has no property '%s'", e.Property)
	default:
		return nil, runtimeErr(e.GetSpan(), "cannot access property '%s' on value of type '%s'",
			e.Property, obj.TypeName())
	}
}

func (i *Interpreter) evalIndex(e *ast.IndexExpr) (Value, error) {
	obj, err := i.evalExpr(e.Object)
	if err != nil {
		return nil, err
	}
	idx, err := i.evalExpr(e.Index)
	if err != nil {
		return nil, err
	}

	switch o := obj.(type) {
	case StringVal:
		idxInt, ok := ToInt64(idx)
		if !ok {
			return nil, runtimeErr(e.GetSpan(), "string index must be an integer")
		}
		s := string(o)
		if idxInt < 0 || int(idxInt) >= len(s) {
			return nil, runtimeErr(e.GetSpan(), "string index %d out of range (length %d)", idxInt, len(s))
		}
		return StringVal(string(s[idxInt])), nil
	case *ArrayVal:
		idxInt, ok := ToInt64(idx)
		if !ok {
			return nil, runtimeErr(e.GetSpan(), "array index must be an integer")
		}
		if idxInt < 0 || int(idxInt) >= len(o.Elements) {
			return nil, runtimeErr(e.GetSpan(), "array index %d out of range (length %d)", idxInt, len(o.Elements))
		}
		return o.Elements[idxInt], nil
	case *MapVal:
		keyStr, ok := idx.(StringVal)
		if !ok {
			return nil, runtimeErr(e.GetSpan(), "map key must be a string, got '%s'", idx.TypeName())
		}
		if val, exists := o.Values[string(keyStr)]; exists {
			return val, nil
		}
		return NullVal{}, nil
	default:
		return nil, runtimeErr(e.GetSpan(), "cannot index value of type '%s'", obj.TypeName())
	}
}

func (i *Interpreter) evalNew(e *ast.NewExpr) (Value, error) {
	// Look up class
	classVal, ok := i.env.Get(e.ClassName)
	if !ok {
		return nil, runtimeErr(e.GetSpan(), "undefined class '%s'", e.ClassName)
	}
	cls, ok := classVal.(*ClassVal)
	if !ok {
		return nil, runtimeErr(e.GetSpan(), "'%s' is not a class", e.ClassName)
	}

	// Evaluate arguments
	args := make([]Value, len(e.Args))
	for idx, argExpr := range e.Args {
		val, err := i.evalExpr(argExpr)
		if err != nil {
			return nil, err
		}
		args[idx] = val
	}

	// Create new object
	obj := &ObjectVal{
		Class: cls,
		Props: make(map[string]Value),
	}

	// Find constructor (walk inheritance chain)
	ctor, ctorClass := findConstructor(cls)
	if ctor != nil {
		if len(args) != len(ctor.Params) {
			return nil, runtimeErr(e.GetSpan(), "%s constructor expects %d arguments, got %d",
				e.ClassName, len(ctor.Params), len(args))
		}
		ctorEnv := NewEnvironment(ctorClass.Env)
		ctorEnv.Define("this", obj, true)
		ctorEnv.Define("__class__", ctorClass, true)
		for idx, param := range ctor.Params {
			ctorEnv.Define(param, args[idx], false)
		}

		result, err := i.execBlock(ctor.Body, ctorEnv)
		if err != nil {
			return nil, err
		}
		if result.Signal == SigReturn {
			// constructor should not return a value, but we allow it to end early
		}
	} else if len(args) > 0 {
		return nil, runtimeErr(e.GetSpan(), "%s has no constructor but was called with %d arguments",
			e.ClassName, len(args))
	}

	return obj, nil
}

// ============================================================
// For loop execution
// ============================================================

func (i *Interpreter) execFor(s *ast.ForStmt) (ExecResult, error) {
	// Create scope for the for loop (init vars are scoped to the loop)
	forEnv := NewEnvironment(i.env)
	prevEnv := i.env
	i.env = forEnv
	defer func() { i.env = prevEnv }()

	// Execute init
	if s.Init != nil {
		result, err := i.execNode(s.Init)
		if err != nil {
			return resultNone, err
		}
		if result.Signal != SigNone {
			return result, nil
		}
	}

	for {
		// Check condition
		if s.Condition != nil {
			cond, err := i.evalExpr(s.Condition)
			if err != nil {
				return resultNone, err
			}
			if !IsTruthy(cond) {
				break
			}
		}

		// Execute body (new scope for each iteration)
		result, err := i.execBlock(s.Body, NewEnvironment(i.env))
		if err != nil {
			return resultNone, err
		}
		if result.Signal == SigBreak {
			break
		}
		if result.Signal == SigReturn {
			return result, nil
		}
		// SigContinue: skip to update

		// Execute update
		if s.Update != nil {
			_, err := i.execNode(s.Update)
			if err != nil {
				return resultNone, err
			}
		}
	}

	return resultNone, nil
}

func (i *Interpreter) execForOf(s *ast.ForOfStmt) (ExecResult, error) {
	iterable, err := i.evalExpr(s.Iterable)
	if err != nil {
		return resultNone, err
	}

	var items []Value
	switch it := iterable.(type) {
	case *ArrayVal:
		items = it.Elements
	case *MapVal:
		items = make([]Value, len(it.Keys))
		for idx, k := range it.Keys {
			items[idx] = StringVal(k)
		}
	default:
		return resultNone, runtimeErr(s.GetSpan(), "for-of requires an array or map, got '%s'", iterable.TypeName())
	}

	for _, elem := range items {
		loopEnv := NewEnvironment(i.env)
		loopEnv.Define(s.VarName, elem, false)

		result, err := i.execBlock(s.Body, loopEnv)
		if err != nil {
			return resultNone, err
		}
		if result.Signal == SigBreak {
			break
		}
		if result.Signal == SigReturn {
			return result, nil
		}
		// SigContinue: continue
	}

	return resultNone, nil
}

// ============================================================
// Array methods
// ============================================================

func (i *Interpreter) evalArrayLiteral(e *ast.ArrayLiteral) (Value, error) {
	elements := make([]Value, len(e.Elements))
	for idx, elemExpr := range e.Elements {
		val, err := i.evalExpr(elemExpr)
		if err != nil {
			return nil, err
		}
		elements[idx] = val
	}
	return &ArrayVal{Elements: elements}, nil
}

func (i *Interpreter) evalTernary(e *ast.TernaryExpr) (Value, error) {
	cond, err := i.evalExpr(e.Condition)
	if err != nil {
		return nil, err
	}
	if IsTruthy(cond) {
		return i.evalExpr(e.Then)
	}
	return i.evalExpr(e.Else)
}

func (i *Interpreter) evalMapLiteral(e *ast.MapLiteral) (Value, error) {
	m := &MapVal{
		Keys:   make([]string, 0, len(e.Keys)),
		Values: make(map[string]Value, len(e.Keys)),
	}
	for idx, keyExpr := range e.Keys {
		keyStr, ok := keyExpr.(*ast.StringLiteral)
		if !ok {
			return nil, runtimeErr(keyExpr.GetSpan(), "map key must be a string")
		}
		key := keyStr.Value
		val, err := i.evalExpr(e.Values[idx])
		if err != nil {
			return nil, err
		}
		if _, exists := m.Values[key]; !exists {
			m.Keys = append(m.Keys, key)
		}
		m.Values[key] = val
	}
	return m, nil
}

func (i *Interpreter) execTry(s *ast.TryStmt) (ExecResult, error) {
	result, err := i.execBlock(s.Body, NewEnvironment(i.env))
	if err == nil {
		return result, nil
	}

	// Error occurred - catch it
	if s.CatchBody != nil {
		catchEnv := NewEnvironment(i.env)
		var errVal Value
		switch e := err.(type) {
		case *ThrownError:
			errVal = e.Value
		case *RuntimeError:
			errVal = StringVal(e.Message)
		default:
			errVal = StringVal(err.Error())
		}
		if s.CatchParam != "" {
			catchEnv.Define(s.CatchParam, errVal, false)
		}
		return i.execBlock(s.CatchBody, catchEnv)
	}

	return resultNone, err // re-throw if no catch
}

func (i *Interpreter) execThrow(s *ast.ThrowStmt) (ExecResult, error) {
	val, err := i.evalExpr(s.Value)
	if err != nil {
		return resultNone, err
	}
	return resultNone, &ThrownError{Value: val, Span: s.GetSpan()}
}

func (i *Interpreter) callSuperConstructor(args []Value, s span.Span) (Value, error) {
	classVal, ok := i.env.Get("__class__")
	if !ok {
		return nil, runtimeErr(s, "super() used outside of a constructor")
	}
	cls := classVal.(*ClassVal)
	if cls.Super == nil {
		return nil, runtimeErr(s, "class '%s' has no super class", cls.Decl.Name)
	}

	ctor, ctorClass := findConstructor(cls.Super)
	if ctor == nil {
		if len(args) > 0 {
			return nil, runtimeErr(s, "super class has no constructor but was called with %d arguments", len(args))
		}
		return NullVal{}, nil
	}
	if len(args) != len(ctor.Params) {
		return nil, runtimeErr(s, "super constructor expects %d arguments, got %d", len(ctor.Params), len(args))
	}

	thisVal, _ := i.env.Get("this")
	ctorEnv := NewEnvironment(ctorClass.Env)
	ctorEnv.Define("this", thisVal, true)
	ctorEnv.Define("__class__", ctorClass, true)
	for idx, param := range ctor.Params {
		ctorEnv.Define(param, args[idx], false)
	}

	_, err := i.execBlock(ctor.Body, ctorEnv)
	return NullVal{}, err
}

func (i *Interpreter) callSuperMethod(methodName string, args []Value, s span.Span) (Value, error) {
	classVal, ok := i.env.Get("__class__")
	if !ok {
		return nil, runtimeErr(s, "super used outside of a class")
	}
	cls := classVal.(*ClassVal)
	if cls.Super == nil {
		return nil, runtimeErr(s, "class has no super class")
	}

	thisVal, _ := i.env.Get("this")
	obj := thisVal.(*ObjectVal)

	method, methodClass := findMethod(cls.Super, methodName)
	if method == nil {
		return nil, runtimeErr(s, "super class has no method '%s'", methodName)
	}
	if len(args) != len(method.Params) {
		return nil, runtimeErr(s, "super.%s() expects %d arguments, got %d", methodName, len(method.Params), len(args))
	}

	methodEnv := NewEnvironment(methodClass.Env)
	methodEnv.Define("this", obj, true)
	methodEnv.Define("__class__", methodClass, true)
	for idx, param := range method.Params {
		methodEnv.Define(param, args[idx], false)
	}

	result, err := i.execBlock(method.Body, methodEnv)
	if err != nil {
		return nil, err
	}
	if result.Signal == SigReturn {
		return result.Value, nil
	}
	return NullVal{}, nil
}

func (i *Interpreter) evalFuncExpr(e *ast.FuncExpr) (Value, error) {
	name := e.Name
	if name == "" {
		name = "<anonymous>"
	}
	fn := &FuncVal{
		Name:    name,
		Params:  e.Params,
		Body:    e.Body,
		Closure: i.env,
	}
	return fn, nil
}

// ============================================================
// Template literal evaluation
// ============================================================

func (i *Interpreter) evalTemplateLiteral(e *ast.TemplateLiteral) (Value, error) {
	var sb strings.Builder
	for idx, part := range e.Parts {
		sb.WriteString(part)
		if idx < len(e.Exprs) {
			val, err := i.evalExpr(e.Exprs[idx])
			if err != nil {
				return nil, err
			}
			sb.WriteString(val.String())
		}
	}
	return StringVal(sb.String()), nil
}

// ============================================================
// String methods
// ============================================================

func (i *Interpreter) callStringMethod(s string, name string, args []Value, sp span.Span) (Value, error) {
	switch name {
	case "split":
		if len(args) != 1 {
			return nil, runtimeErr(sp, "split() expects 1 argument, got %d", len(args))
		}
		sep, ok := args[0].(StringVal)
		if !ok {
			return nil, runtimeErr(sp, "split() separator must be a string")
		}
		parts := strings.Split(s, string(sep))
		elements := make([]Value, len(parts))
		for idx, p := range parts {
			elements[idx] = StringVal(p)
		}
		return &ArrayVal{Elements: elements}, nil

	case "trim":
		if len(args) != 0 {
			return nil, runtimeErr(sp, "trim() expects 0 arguments, got %d", len(args))
		}
		return StringVal(strings.TrimSpace(s)), nil

	case "indexOf":
		if len(args) != 1 {
			return nil, runtimeErr(sp, "indexOf() expects 1 argument, got %d", len(args))
		}
		sub, ok := args[0].(StringVal)
		if !ok {
			return nil, runtimeErr(sp, "indexOf() argument must be a string")
		}
		return IntVal(strings.Index(s, string(sub))), nil

	case "slice":
		if len(args) < 1 || len(args) > 2 {
			return nil, runtimeErr(sp, "slice() expects 1-2 arguments, got %d", len(args))
		}
		start, ok := ToInt64(args[0])
		if !ok {
			return nil, runtimeErr(sp, "slice() start must be an integer")
		}
		end := int64(len(s))
		if len(args) == 2 {
			end, ok = ToInt64(args[1])
			if !ok {
				return nil, runtimeErr(sp, "slice() end must be an integer")
			}
		}
		if start < 0 {
			start = int64(len(s)) + start
		}
		if end < 0 {
			end = int64(len(s)) + end
		}
		if start < 0 {
			start = 0
		}
		if end > int64(len(s)) {
			end = int64(len(s))
		}
		if start >= end {
			return StringVal(""), nil
		}
		return StringVal(s[start:end]), nil

	case "toUpperCase":
		return StringVal(strings.ToUpper(s)), nil

	case "toLowerCase":
		return StringVal(strings.ToLower(s)), nil

	case "replace":
		if len(args) != 2 {
			return nil, runtimeErr(sp, "replace() expects 2 arguments, got %d", len(args))
		}
		old, ok1 := args[0].(StringVal)
		newStr, ok2 := args[1].(StringVal)
		if !ok1 || !ok2 {
			return nil, runtimeErr(sp, "replace() arguments must be strings")
		}
		return StringVal(strings.Replace(s, string(old), string(newStr), 1)), nil

	case "replaceAll":
		if len(args) != 2 {
			return nil, runtimeErr(sp, "replaceAll() expects 2 arguments, got %d", len(args))
		}
		old, ok1 := args[0].(StringVal)
		newStr, ok2 := args[1].(StringVal)
		if !ok1 || !ok2 {
			return nil, runtimeErr(sp, "replaceAll() arguments must be strings")
		}
		return StringVal(strings.ReplaceAll(s, string(old), string(newStr))), nil

	case "startsWith":
		if len(args) != 1 {
			return nil, runtimeErr(sp, "startsWith() expects 1 argument, got %d", len(args))
		}
		prefix, ok := args[0].(StringVal)
		if !ok {
			return nil, runtimeErr(sp, "startsWith() argument must be a string")
		}
		return BoolVal(strings.HasPrefix(s, string(prefix))), nil

	case "endsWith":
		if len(args) != 1 {
			return nil, runtimeErr(sp, "endsWith() expects 1 argument, got %d", len(args))
		}
		suffix, ok := args[0].(StringVal)
		if !ok {
			return nil, runtimeErr(sp, "endsWith() argument must be a string")
		}
		return BoolVal(strings.HasSuffix(s, string(suffix))), nil

	case "includes":
		if len(args) != 1 {
			return nil, runtimeErr(sp, "includes() expects 1 argument, got %d", len(args))
		}
		sub, ok := args[0].(StringVal)
		if !ok {
			return nil, runtimeErr(sp, "includes() argument must be a string")
		}
		return BoolVal(strings.Contains(s, string(sub))), nil

	case "charAt":
		if len(args) != 1 {
			return nil, runtimeErr(sp, "charAt() expects 1 argument, got %d", len(args))
		}
		idx, ok := ToInt64(args[0])
		if !ok {
			return nil, runtimeErr(sp, "charAt() argument must be an integer")
		}
		if idx < 0 || int(idx) >= len(s) {
			return StringVal(""), nil
		}
		return StringVal(string(s[idx])), nil

	case "substring":
		if len(args) < 1 || len(args) > 2 {
			return nil, runtimeErr(sp, "substring() expects 1-2 arguments, got %d", len(args))
		}
		start, ok := ToInt64(args[0])
		if !ok {
			return nil, runtimeErr(sp, "substring() start must be an integer")
		}
		end := int64(len(s))
		if len(args) == 2 {
			end, ok = ToInt64(args[1])
			if !ok {
				return nil, runtimeErr(sp, "substring() end must be an integer")
			}
		}
		if start < 0 {
			start = 0
		}
		if end > int64(len(s)) {
			end = int64(len(s))
		}
		if start > end {
			start, end = end, start
		}
		return StringVal(s[start:end]), nil

	case "repeat":
		if len(args) != 1 {
			return nil, runtimeErr(sp, "repeat() expects 1 argument, got %d", len(args))
		}
		count, ok := ToInt64(args[0])
		if !ok || count < 0 {
			return nil, runtimeErr(sp, "repeat() count must be a non-negative integer")
		}
		return StringVal(strings.Repeat(s, int(count))), nil

	case "trimStart":
		return StringVal(strings.TrimLeft(s, " \t\n\r")), nil

	case "trimEnd":
		return StringVal(strings.TrimRight(s, " \t\n\r")), nil

	default:
		return nil, runtimeErr(sp, "string has no method '%s'", name)
	}
}

// ============================================================
// Array methods (extended)
// ============================================================

func (i *Interpreter) callArrayMethod(arr *ArrayVal, name string, args []Value, s span.Span) (Value, error) {
	switch name {
	case "push":
		if len(args) != 1 {
			return nil, runtimeErr(s, "push() expects 1 argument, got %d", len(args))
		}
		arr.Elements = append(arr.Elements, args[0])
		return IntVal(len(arr.Elements)), nil

	case "pop":
		if len(args) != 0 {
			return nil, runtimeErr(s, "pop() expects 0 arguments, got %d", len(args))
		}
		if len(arr.Elements) == 0 {
			return nil, runtimeErr(s, "pop() on empty array")
		}
		last := arr.Elements[len(arr.Elements)-1]
		arr.Elements = arr.Elements[:len(arr.Elements)-1]
		return last, nil

	case "map":
		if len(args) != 1 {
			return nil, runtimeErr(s, "map() expects 1 argument, got %d", len(args))
		}
		fn := args[0]
		result := make([]Value, len(arr.Elements))
		for idx, elem := range arr.Elements {
			val, err := i.callValue(fn, []Value{elem}, s)
			if err != nil {
				return nil, err
			}
			result[idx] = val
		}
		return &ArrayVal{Elements: result}, nil

	case "filter":
		if len(args) != 1 {
			return nil, runtimeErr(s, "filter() expects 1 argument, got %d", len(args))
		}
		fn := args[0]
		var result []Value
		for _, elem := range arr.Elements {
			val, err := i.callValue(fn, []Value{elem}, s)
			if err != nil {
				return nil, err
			}
			if IsTruthy(val) {
				result = append(result, elem)
			}
		}
		if result == nil {
			result = []Value{}
		}
		return &ArrayVal{Elements: result}, nil

	case "reduce":
		if len(args) < 1 || len(args) > 2 {
			return nil, runtimeErr(s, "reduce() expects 1-2 arguments, got %d", len(args))
		}
		fn := args[0]
		var acc Value
		startIdx := 0
		if len(args) == 2 {
			acc = args[1]
		} else {
			if len(arr.Elements) == 0 {
				return nil, runtimeErr(s, "reduce() of empty array with no initial value")
			}
			acc = arr.Elements[0]
			startIdx = 1
		}
		for idx := startIdx; idx < len(arr.Elements); idx++ {
			val, err := i.callValue(fn, []Value{acc, arr.Elements[idx]}, s)
			if err != nil {
				return nil, err
			}
			acc = val
		}
		return acc, nil

	case "forEach":
		if len(args) != 1 {
			return nil, runtimeErr(s, "forEach() expects 1 argument, got %d", len(args))
		}
		fn := args[0]
		for _, elem := range arr.Elements {
			_, err := i.callValue(fn, []Value{elem}, s)
			if err != nil {
				return nil, err
			}
		}
		return NullVal{}, nil

	case "find":
		if len(args) != 1 {
			return nil, runtimeErr(s, "find() expects 1 argument, got %d", len(args))
		}
		fn := args[0]
		for _, elem := range arr.Elements {
			val, err := i.callValue(fn, []Value{elem}, s)
			if err != nil {
				return nil, err
			}
			if IsTruthy(val) {
				return elem, nil
			}
		}
		return NullVal{}, nil

	case "sort":
		if len(args) > 1 {
			return nil, runtimeErr(s, "sort() expects 0-1 arguments, got %d", len(args))
		}
		if len(args) == 0 {
			sort.SliceStable(arr.Elements, func(a, b int) bool {
				return compareValues(arr.Elements[a], arr.Elements[b]) < 0
			})
		} else {
			fn := args[0]
			var sortErr error
			sort.SliceStable(arr.Elements, func(a, b int) bool {
				if sortErr != nil {
					return false
				}
				result, err := i.callValue(fn, []Value{arr.Elements[a], arr.Elements[b]}, s)
				if err != nil {
					sortErr = err
					return false
				}
				n, ok := ToFloat64(result)
				if !ok {
					sortErr = runtimeErr(s, "sort comparator must return a number")
					return false
				}
				return n < 0
			})
			if sortErr != nil {
				return nil, sortErr
			}
		}
		return arr, nil

	case "reverse":
		for left, right := 0, len(arr.Elements)-1; left < right; left, right = left+1, right-1 {
			arr.Elements[left], arr.Elements[right] = arr.Elements[right], arr.Elements[left]
		}
		return arr, nil

	case "join":
		sep := ","
		if len(args) == 1 {
			sepVal, ok := args[0].(StringVal)
			if !ok {
				return nil, runtimeErr(s, "join() separator must be a string")
			}
			sep = string(sepVal)
		} else if len(args) > 1 {
			return nil, runtimeErr(s, "join() expects 0-1 arguments, got %d", len(args))
		}
		parts := make([]string, len(arr.Elements))
		for idx, elem := range arr.Elements {
			parts[idx] = elem.String()
		}
		return StringVal(strings.Join(parts, sep)), nil

	case "slice":
		if len(args) < 1 || len(args) > 2 {
			return nil, runtimeErr(s, "slice() expects 1-2 arguments, got %d", len(args))
		}
		start, ok := ToInt64(args[0])
		if !ok {
			return nil, runtimeErr(s, "slice() start must be an integer")
		}
		end := int64(len(arr.Elements))
		if len(args) == 2 {
			end, ok = ToInt64(args[1])
			if !ok {
				return nil, runtimeErr(s, "slice() end must be an integer")
			}
		}
		if start < 0 {
			start = int64(len(arr.Elements)) + start
		}
		if end < 0 {
			end = int64(len(arr.Elements)) + end
		}
		if start < 0 {
			start = 0
		}
		if end > int64(len(arr.Elements)) {
			end = int64(len(arr.Elements))
		}
		if start >= end {
			return &ArrayVal{Elements: []Value{}}, nil
		}
		// Return a new copy of the slice
		newElems := make([]Value, end-start)
		copy(newElems, arr.Elements[start:end])
		return &ArrayVal{Elements: newElems}, nil

	case "indexOf":
		if len(args) != 1 {
			return nil, runtimeErr(s, "indexOf() expects 1 argument, got %d", len(args))
		}
		for idx, elem := range arr.Elements {
			if valuesEqual(elem, args[0]) {
				return IntVal(idx), nil
			}
		}
		return IntVal(-1), nil

	case "includes":
		if len(args) != 1 {
			return nil, runtimeErr(s, "includes() expects 1 argument, got %d", len(args))
		}
		for _, elem := range arr.Elements {
			if valuesEqual(elem, args[0]) {
				return BoolVal(true), nil
			}
		}
		return BoolVal(false), nil

	case "concat":
		if len(args) != 1 {
			return nil, runtimeErr(s, "concat() expects 1 argument, got %d", len(args))
		}
		other, ok := args[0].(*ArrayVal)
		if !ok {
			return nil, runtimeErr(s, "concat() argument must be an array")
		}
		newElems := make([]Value, len(arr.Elements)+len(other.Elements))
		copy(newElems, arr.Elements)
		copy(newElems[len(arr.Elements):], other.Elements)
		return &ArrayVal{Elements: newElems}, nil

	case "flat":
		var result []Value
		for _, elem := range arr.Elements {
			if inner, ok := elem.(*ArrayVal); ok {
				result = append(result, inner.Elements...)
			} else {
				result = append(result, elem)
			}
		}
		if result == nil {
			result = []Value{}
		}
		return &ArrayVal{Elements: result}, nil

	default:
		return nil, runtimeErr(s, "array has no method '%s'", name)
	}
}

// compareValues compares two values for sorting.
func compareValues(a, b Value) int {
	af, aOk := ToFloat64(a)
	bf, bOk := ToFloat64(b)
	if aOk && bOk {
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0
	}
	as, bs := a.String(), b.String()
	if as < bs {
		return -1
	}
	if as > bs {
		return 1
	}
	return 0
}

// ============================================================
// Value equality
// ============================================================

func valuesEqual(a, b Value) bool {
	switch av := a.(type) {
	case IntVal:
		if bv, ok := b.(IntVal); ok {
			return int64(av) == int64(bv)
		}
		if bv, ok := b.(FloatVal); ok {
			return float64(int64(av)) == float64(bv)
		}
	case FloatVal:
		if bv, ok := b.(FloatVal); ok {
			return float64(av) == float64(bv)
		}
		if bv, ok := b.(IntVal); ok {
			return float64(av) == float64(int64(bv))
		}
	case StringVal:
		if bv, ok := b.(StringVal); ok {
			return string(av) == string(bv)
		}
	case BoolVal:
		if bv, ok := b.(BoolVal); ok {
			return bool(av) == bool(bv)
		}
	case NullVal:
		_, ok := b.(NullVal)
		return ok
	}
	// Reference equality for objects/functions
	return a == b
}
