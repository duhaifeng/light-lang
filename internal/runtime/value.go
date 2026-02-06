// Package runtime implements the interpreter and runtime value system for light-lang.
package runtime

import (
	"fmt"
	"light-lang/internal/ast"
	"strings"
)

// Value is the interface for all runtime values.
type Value interface {
	TypeName() string
	String() string
}

// ---- Primitive values ----

// IntVal represents an integer value.
type IntVal int64

func (v IntVal) TypeName() string { return "int" }
func (v IntVal) String() string   { return fmt.Sprintf("%d", int64(v)) }

// FloatVal represents a floating-point value.
type FloatVal float64

func (v FloatVal) TypeName() string { return "float" }
func (v FloatVal) String() string   { return fmt.Sprintf("%g", float64(v)) }

// StringVal represents a string value.
type StringVal string

func (v StringVal) TypeName() string { return "string" }
func (v StringVal) String() string   { return string(v) }

// BoolVal represents a boolean value.
type BoolVal bool

func (v BoolVal) TypeName() string { return "bool" }
func (v BoolVal) String() string   { return fmt.Sprintf("%t", bool(v)) }

// NullVal represents null.
type NullVal struct{}

func (v NullVal) TypeName() string { return "null" }
func (v NullVal) String() string   { return "null" }

// ---- Callable values ----

// FuncVal represents a user-defined function (closure).
type FuncVal struct {
	Name    string
	Params  []string
	Body    *ast.BlockStmt
	Closure *Environment
}

func (v *FuncVal) TypeName() string { return "function" }
func (v *FuncVal) String() string   { return fmt.Sprintf("<function %s>", v.Name) }

// BuiltinFn is the Go signature for built-in functions.
type BuiltinFn func(args []Value) (Value, error)

// BuiltinVal represents a built-in (native) function.
type BuiltinVal struct {
	Name string
	Fn   BuiltinFn
}

func (v *BuiltinVal) TypeName() string { return "builtin" }
func (v *BuiltinVal) String() string   { return fmt.Sprintf("<builtin %s>", v.Name) }

// ---- OOP values ----

// ClassVal represents a class definition stored in the environment.
type ClassVal struct {
	Decl  *ast.ClassDecl
	Env   *Environment // environment where the class was defined
	Super *ClassVal    // parent class (for extends), may be nil
}

func (v *ClassVal) TypeName() string { return "class" }
func (v *ClassVal) String() string   { return fmt.Sprintf("<class %s>", v.Decl.Name) }

// ObjectVal represents an instance of a class.
type ObjectVal struct {
	Class *ClassVal
	Props map[string]Value
}

func (v *ObjectVal) TypeName() string { return "object" }
func (v *ObjectVal) String() string {
	return fmt.Sprintf("<object %s>", v.Class.Decl.Name)
}

// ---- Array value ----

// ArrayVal represents an array value.
type ArrayVal struct {
	Elements []Value
}

func (v *ArrayVal) TypeName() string { return "array" }
func (v *ArrayVal) String() string {
	parts := make([]string, len(v.Elements))
	for i, elem := range v.Elements {
		if s, ok := elem.(StringVal); ok {
			parts[i] = fmt.Sprintf("\"%s\"", string(s))
		} else {
			parts[i] = elem.String()
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ---- Map value ----

// MapVal represents a map (dictionary) value with ordered keys.
type MapVal struct {
	Keys   []string
	Values map[string]Value
}

func (v *MapVal) TypeName() string { return "map" }
func (v *MapVal) String() string {
	parts := make([]string, len(v.Keys))
	for i, k := range v.Keys {
		val := v.Values[k]
		if s, ok := val.(StringVal); ok {
			parts[i] = fmt.Sprintf("\"%s\": \"%s\"", k, string(s))
		} else {
			parts[i] = fmt.Sprintf("\"%s\": %s", k, val.String())
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// ---- Enum values ----

// EnumTypeVal represents an enum type (e.g., Color).
type EnumTypeVal struct {
	Name     string
	Variants map[string]*EnumVariantVal
	Order    []string // ordered variant names
}

func (v *EnumTypeVal) TypeName() string { return "enum" }
func (v *EnumTypeVal) String() string   { return fmt.Sprintf("<enum %s>", v.Name) }

// EnumVariantVal represents a specific enum variant (e.g., Color.Red).
type EnumVariantVal struct {
	EnumName    string
	VariantName string
	Ordinal     int
}

func (v *EnumVariantVal) TypeName() string { return v.EnumName }
func (v *EnumVariantVal) String() string   { return v.EnumName + "." + v.VariantName }

// ---- Interface value ----

// InterfaceVal represents an interface definition stored in the environment.
type InterfaceVal struct {
	Decl *ast.InterfaceDecl
}

func (v *InterfaceVal) TypeName() string { return "interface" }
func (v *InterfaceVal) String() string   { return fmt.Sprintf("<interface %s>", v.Decl.Name) }

// ---- Truthiness ----

// IsTruthy returns the truthiness of a value (JS/Python style).
func IsTruthy(v Value) bool {
	switch val := v.(type) {
	case NullVal:
		return false
	case BoolVal:
		return bool(val)
	case IntVal:
		return int64(val) != 0
	case FloatVal:
		return float64(val) != 0
	case StringVal:
		return string(val) != ""
	default:
		return true
	}
}

// ---- Helpers ----

// ValuesString formats a slice of values with a separator.
func ValuesString(vals []Value, sep string) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = v.String()
	}
	return strings.Join(parts, sep)
}

// ToFloat64 attempts to convert a numeric value to float64.
func ToFloat64(v Value) (float64, bool) {
	switch val := v.(type) {
	case IntVal:
		return float64(int64(val)), true
	case FloatVal:
		return float64(val), true
	default:
		return 0, false
	}
}

// ToInt64 attempts to convert a value to int64.
func ToInt64(v Value) (int64, bool) {
	switch val := v.(type) {
	case IntVal:
		return int64(val), true
	case FloatVal:
		return int64(float64(val)), true
	default:
		return 0, false
	}
}
