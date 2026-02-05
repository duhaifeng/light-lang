package runtime

import (
	"fmt"
	"io"
	"light-lang/internal/ast"
	"light-lang/internal/span"
	"light-lang/internal/token"
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
		objVal, ok := obj.(*ObjectVal)
		if !ok {
			return resultNone, runtimeErr(s.GetSpan(), "cannot set property on non-object value of type '%s'", obj.TypeName())
		}
		objVal.Props[target.Property] = val
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

	// String concatenation
	if e.Op == token.PLUS {
		if ls, ok := left.(StringVal); ok {
			if rs, ok := right.(StringVal); ok {
				return StringVal(string(ls) + string(rs)), nil
			}
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
	// Look up method in class declaration
	cls := obj.Class.Decl

	for _, m := range cls.Methods {
		if m.Name == methodName {
			if len(args) != len(m.Params) {
				return nil, runtimeErr(s, "%s.%s() expects %d arguments, got %d",
					cls.Name, methodName, len(m.Params), len(args))
			}

			methodEnv := NewEnvironment(obj.Class.Env)
			methodEnv.Define("this", obj, true)
			for idx, param := range m.Params {
				methodEnv.Define(param, args[idx], false)
			}

			result, err := i.execBlock(m.Body, methodEnv)
			if err != nil {
				return nil, err
			}
			if result.Signal == SigReturn {
				return result.Value, nil
			}
			return NullVal{}, nil
		}
	}

	// Check if it's a property that's callable
	if propVal, exists := obj.Props[methodName]; exists {
		return i.callValue(propVal, args, s)
	}

	return nil, runtimeErr(s, "undefined method '%s' on class '%s'", methodName, cls.Name)
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

	// Run constructor if defined
	ctor := cls.Decl.Constructor
	if ctor != nil {
		if len(args) != len(ctor.Params) {
			return nil, runtimeErr(e.GetSpan(), "%s constructor expects %d arguments, got %d",
				e.ClassName, len(ctor.Params), len(args))
		}
		ctorEnv := NewEnvironment(cls.Env)
		ctorEnv.Define("this", obj, true)
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

	arr, ok := iterable.(*ArrayVal)
	if !ok {
		return resultNone, runtimeErr(s.GetSpan(), "for-of requires an array, got '%s'", iterable.TypeName())
	}

	for _, elem := range arr.Elements {
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
	default:
		return nil, runtimeErr(s, "array has no method '%s'", name)
	}
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
