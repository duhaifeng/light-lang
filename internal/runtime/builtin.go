package runtime

import (
	"fmt"
	"io"
)

// RegisterBuiltins adds built-in functions to the given environment.
func RegisterBuiltins(env *Environment, w io.Writer) {
	env.Define("print", &BuiltinVal{
		Name: "print",
		Fn: func(args []Value) (Value, error) {
			fmt.Fprintln(w, ValuesString(args, " "))
			return NullVal{}, nil
		},
	}, true)

	env.Define("println", &BuiltinVal{
		Name: "println",
		Fn: func(args []Value) (Value, error) {
			fmt.Fprintln(w, ValuesString(args, " "))
			return NullVal{}, nil
		},
	}, true)

	env.Define("typeOf", &BuiltinVal{
		Name: "typeOf",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("typeOf() expects 1 argument, got %d", len(args))
			}
			return StringVal(args[0].TypeName()), nil
		},
	}, true)

	env.Define("toString", &BuiltinVal{
		Name: "toString",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("toString() expects 1 argument, got %d", len(args))
			}
			return StringVal(args[0].String()), nil
		},
	}, true)

	env.Define("len", &BuiltinVal{
		Name: "len",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("len() expects 1 argument, got %d", len(args))
			}
			switch v := args[0].(type) {
			case StringVal:
				return IntVal(len(string(v))), nil
			default:
				return nil, fmt.Errorf("len() not supported for type '%s'", args[0].TypeName())
			}
		},
	}, true)
}
