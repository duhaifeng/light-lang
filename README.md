# Light Lang

English | [中文](README_CN.md) | [日本語](README_JA.md)

A lightweight, dynamically typed programming language interpreter built from scratch in Go.

Light Lang features a clean, expressive syntax inspired by JavaScript/TypeScript, with support for classes, closures, higher-order functions, error handling, and more — all implemented without any third-party dependencies.

## Features

- **Dynamic Typing** — variables can hold any type: `int`, `float`, `string`, `bool`, `null`, `array`, `map`
- **First-Class Functions** — functions as values, closures, and arrow functions `(x) => x * 2`
- **Object-Oriented** — classes with constructors, methods, single inheritance (`extends`), and `super`
- **Error Handling** — `try` / `catch` / `throw` for structured exception handling
- **Collections** — arrays `[1, 2, 3]` and maps `{ key: "value" }` with built-in methods
- **Control Flow** — `if/else`, `while`, C-style `for`, `for-of` iteration, `break`, `continue`
- **Ternary Operator** — `condition ? then : else`
- **Compound Assignment** — `+=`, `-=`, `*=`, `/=`
- **Interactive REPL** — experiment with the language interactively
- **Toolchain** — tokenizer, parser (AST output as JSON), and interpreter

## Quick Start

### Prerequisites

- [Go](https://go.dev/) 1.21+ installed

### Build

```bash
git clone https://github.com/duhaifeng/light-lang.git
cd light-lang
go build -o light ./cmd/light
```

### Run a Program

```bash
./light run testdata/hello.lt
```

### Start the REPL

```bash
./light repl
```

```
light-lang REPL (type 'exit' to quit)

light> print("hello, light-lang!")
hello, light-lang!
light> var x = 1 + 2 * 3
light> print(x)
7
light> exit
```

## Language Tour

### Variables

```javascript
var name = "Alice"
var age = 30
var pi = 3.14
var active = true
const MAX = 100
```

### Functions

```javascript
// Regular function
function greet(name) {
  return "Hello, " + name + "!"
}
print(greet("World"))  // Hello, World!

// Arrow functions
var add = (a, b) => a + b
var square = x => x * x
print(add(3, 4))   // 7
print(square(6))    // 36
```

### Control Flow

```javascript
// If / Else
if (x > 10) {
  print("big")
} else if (x > 5) {
  print("medium")
} else {
  print("small")
}

// While loop
var i = 0
while (i < 5) {
  print(i)
  i += 1
}

// C-style for loop
for (var i = 0; i < 10; i += 1) {
  print(i)
}

// For-of loop
var items = [10, 20, 30]
for (var item of items) {
  print(item)
}
```

### Arrays

```javascript
var arr = [1, 2, 3, 4, 5]
arr.push(6)
print(arr.length)   // 6
print(arr[0])       // 1
print(arr.pop())    // 6
```

### Maps (Dictionaries)

```javascript
var person = {
  name: "Alice",
  age: 30,
  active: true,
}

print(person.name)       // Alice
print(person["age"])     // 30
person.role = "admin"

var ks = keys(person)
var vs = values(person)

// Iterate over map keys
for (var key of person) {
  print(key + " = " + person[key])
}
```

### Classes & Inheritance

```javascript
class Animal {
  constructor(name) {
    this.name = name
  }

  speak() {
    return this.name + " makes a sound"
  }
}

class Dog extends Animal {
  constructor(name, breed) {
    super(name)
    this.breed = breed
  }

  speak() {
    return this.name + " barks"
  }
}

var dog = new Dog("Rex", "Labrador")
print(dog.speak())   // Rex barks
```

### Closures

```javascript
function makeCounter(start) {
  var count = start
  function increment() {
    count += 1
    return count
  }
  return increment
}

var counter = makeCounter(0)
print(counter())  // 1
print(counter())  // 2
print(counter())  // 3
```

### Error Handling

```javascript
function safeDivide(a, b) {
  if (b == 0) {
    throw "division by zero"
  }
  return a / b
}

try {
  print(safeDivide(10, 2))   // 5
  print(safeDivide(10, 0))   // throws
} catch (e) {
  print("caught: " + e)      // caught: division by zero
}
```

### Higher-Order Functions

```javascript
function applyTwice(fn, val) {
  return fn(fn(val))
}

print(applyTwice(x => x + 1, 10))  // 12
print(applyTwice(x => x * 2, 3))   // 12
```

### Ternary Operator & String Coercion

```javascript
var x = 5
print(x > 3 ? "big" : "small")     // big
print("value: " + 42)               // value: 42
print(100 + " dollars")             // 100 dollars
```

## CLI Usage

```
Usage:
  light tokens <file> [--json]   Tokenize and print tokens
  light parse  <file>            Parse and print AST (JSON)
  light run    <file>            Run a source file
  light repl                     Start interactive REPL
```

### Examples

```bash
# Run a program
./light run testdata/fib.lt

# View tokens
./light tokens testdata/hello.lt

# View tokens as JSON
./light tokens testdata/hello.lt --json

# View AST as JSON
./light parse testdata/hello.lt

# Interactive mode
./light repl
```

## Built-in Functions

| Function | Description |
|---|---|
| `print(...)` | Print values separated by spaces |
| `println(...)` | Same as `print` |
| `typeOf(value)` | Return the type name as a string |
| `toString(value)` | Convert a value to its string representation |
| `len(value)` | Return the length of a string, array, or map |
| `push(array, value)` | Append a value to an array, returns the new length |
| `pop(array)` | Remove and return the last element of an array |
| `keys(map)` | Return an array of a map's keys |
| `values(map)` | Return an array of a map's values |

Arrays also support method-style calls: `arr.push(val)`, `arr.pop()`, `arr.length`.

## Architecture

Light Lang follows a classic interpreter pipeline:

```
Source Code → Lexer → Tokens → Parser → AST → Interpreter → Output
```

```
light-lang/
├── cmd/light/           # CLI entry point (tokens, parse, run, repl)
├── internal/
│   ├── token/           # Token type definitions and keywords
│   ├── span/            # Source position tracking (line, column, offset)
│   ├── lexer/           # Lexical analysis — source text to tokens
│   ├── parser/          # Syntax analysis — Pratt parsing + recursive descent
│   ├── ast/             # Abstract Syntax Tree node definitions
│   ├── diag/            # Diagnostic / error reporting
│   └── runtime/         # Tree-walking interpreter
│       ├── interpreter.go   # AST execution engine
│       ├── value.go         # Runtime value types
│       ├── env.go           # Lexical scoping / environment chain
│       └── builtin.go       # Built-in functions
├── testdata/            # Example programs and test cases
└── docs/                # Design documents (Chinese)
```

### Key Design Decisions

- **Pratt Parsing** for expressions — clean, extensible precedence handling
- **Recursive Descent** for statements — straightforward and easy to extend
- **Tree-Walking Interpreter** — directly executes the AST without compilation
- **Lexical Scoping** — environment chain with parent pointers for closures
- **Zero Dependencies** — pure Go standard library, no third-party packages

## Example Programs

The `testdata/` directory contains several example programs:

| File | Description |
|---|---|
| `hello.lt` | Hello world, basic arithmetic |
| `fib.lt` | Recursive Fibonacci sequence |
| `class.lt` | Class definition and method calls |
| `golden_array.lt` | Array operations and iteration |
| `golden_for.lt` | Various for loop patterns |
| `golden_features.lt` | Comprehensive feature showcase |
| `golden_complex.lt` | Advanced: sorting, stacks, higher-order functions, closures, matrix ops |

## Contributing

Contributions are welcome! Feel free to open issues and pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is open source. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project was created as an exploration of language design and interpreter implementation. It serves as both a learning resource and a foundation for further language experimentation.
