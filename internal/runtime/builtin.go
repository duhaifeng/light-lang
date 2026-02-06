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
			case *ArrayVal:
				return IntVal(len(v.Elements)), nil
			case *MapVal:
				return IntVal(len(v.Keys)), nil
			default:
				return nil, fmt.Errorf("len() not supported for type '%s'", args[0].TypeName())
			}
		},
	}, true)

	env.Define("push", &BuiltinVal{
		Name: "push",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("push() expects 2 arguments, got %d", len(args))
			}
			arr, ok := args[0].(*ArrayVal)
			if !ok {
				return nil, fmt.Errorf("push() first argument must be an array, got '%s'", args[0].TypeName())
			}
			arr.Elements = append(arr.Elements, args[1])
			return IntVal(len(arr.Elements)), nil
		},
	}, true)

	env.Define("pop", &BuiltinVal{
		Name: "pop",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("pop() expects 1 argument, got %d", len(args))
			}
			arr, ok := args[0].(*ArrayVal)
			if !ok {
				return nil, fmt.Errorf("pop() first argument must be an array, got '%s'", args[0].TypeName())
			}
			if len(arr.Elements) == 0 {
				return nil, fmt.Errorf("pop() on empty array")
			}
			last := arr.Elements[len(arr.Elements)-1]
			arr.Elements = arr.Elements[:len(arr.Elements)-1]
			return last, nil
		},
	}, true)

	env.Define("keys", &BuiltinVal{
		Name: "keys",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("keys() expects 1 argument, got %d", len(args))
			}
			m, ok := args[0].(*MapVal)
			if !ok {
				return nil, fmt.Errorf("keys() expects a map argument, got '%s'", args[0].TypeName())
			}
			elements := make([]Value, len(m.Keys))
			for i, k := range m.Keys {
				elements[i] = StringVal(k)
			}
			return &ArrayVal{Elements: elements}, nil
		},
	}, true)

	env.Define("implements", &BuiltinVal{
		Name: "implements",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("implements() expects 2 arguments, got %d", len(args))
			}
			obj, ok := args[0].(*ObjectVal)
			if !ok {
				return BoolVal(false), nil
			}
			iface, ok := args[1].(*InterfaceVal)
			if !ok {
				return nil, fmt.Errorf("implements() second argument must be an interface, got '%s'", args[1].TypeName())
			}
			for _, sig := range iface.Decl.Methods {
				method, _ := findMethod(obj.Class, sig.Name)
				if method == nil || len(method.Params) != sig.ParamCount {
					return BoolVal(false), nil
				}
			}
			return BoolVal(true), nil
		},
	}, true)

	env.Define("values", &BuiltinVal{
		Name: "values",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("values() expects 1 argument, got %d", len(args))
			}
			m, ok := args[0].(*MapVal)
			if !ok {
				return nil, fmt.Errorf("values() expects a map argument, got '%s'", args[0].TypeName())
			}
			elements := make([]Value, len(m.Keys))
			for i, k := range m.Keys {
				elements[i] = m.Values[k]
			}
			return &ArrayVal{Elements: elements}, nil
		},
	}, true)
}
