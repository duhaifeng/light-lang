package runtime

import "fmt"

// Environment represents a variable scope with a parent chain.
type Environment struct {
	values map[string]Value
	consts map[string]bool // tracks which names are const
	parent *Environment
}

// NewEnvironment creates a new environment with an optional parent scope.
func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		values: make(map[string]Value),
		consts: make(map[string]bool),
		parent: parent,
	}
}

// Define declares a new variable in the current scope.
func (e *Environment) Define(name string, value Value, isConst bool) error {
	if _, exists := e.values[name]; exists {
		return fmt.Errorf("variable '%s' already declared in this scope", name)
	}
	e.values[name] = value
	if isConst {
		e.consts[name] = true
	}
	return nil
}

// Get looks up a variable by walking the scope chain.
func (e *Environment) Get(name string) (Value, bool) {
	for env := e; env != nil; env = env.parent {
		if val, exists := env.values[name]; exists {
			return val, true
		}
	}
	return nil, false
}

// Set assigns to an existing variable. Returns an error if not found or const.
func (e *Environment) Set(name string, value Value) error {
	for env := e; env != nil; env = env.parent {
		if _, exists := env.values[name]; exists {
			if env.consts[name] {
				return fmt.Errorf("cannot assign to constant '%s'", name)
			}
			env.values[name] = value
			return nil
		}
	}
	return fmt.Errorf("undefined variable '%s'", name)
}
