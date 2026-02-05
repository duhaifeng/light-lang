package runtime

import (
	"bytes"
	"light-lang/internal/lexer"
	"light-lang/internal/parser"
	"strings"
	"testing"
)

// runSource parses and executes source code, returning captured stdout and any error.
func runSource(source string) (string, error) {
	l := lexer.New(source, "test.lt")
	tokens, _ := l.Tokenize()
	p := parser.New(tokens)
	file, _ := p.ParseFile()

	var buf bytes.Buffer
	interp := NewInterpreter(&buf)
	err := interp.Run(file)
	return buf.String(), err
}

func expectOutput(t *testing.T, source, expected string) {
	t.Helper()
	out, err := runSource(source)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	if strings.TrimRight(out, "\n") != strings.TrimRight(expected, "\n") {
		t.Errorf("output mismatch:\nexpected: %q\ngot:      %q", expected, out)
	}
}

func expectError(t *testing.T, source, contains string) {
	t.Helper()
	_, err := runSource(source)
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", contains)
	}
	if !strings.Contains(err.Error(), contains) {
		t.Errorf("expected error containing %q, got: %v", contains, err)
	}
}

// ---- Tests ----

func TestPrintLiteral(t *testing.T) {
	expectOutput(t, `print(42)`, "42\n")
}

func TestPrintString(t *testing.T) {
	expectOutput(t, `print("hello")`, "hello\n")
}

func TestArithmetic(t *testing.T) {
	expectOutput(t, `print(1 + 2 * 3)`, "7\n")
	expectOutput(t, `print((1 + 2) * 3)`, "9\n")
	expectOutput(t, `print(10 / 3)`, "3\n")    // integer division
	expectOutput(t, `print(10 % 3)`, "1\n")
	expectOutput(t, `print(10.0 / 3.0)`, "3.3333333333333335\n")
}

func TestVarDecl(t *testing.T) {
	expectOutput(t, `
var x = 10
print(x)
`, "10\n")
}

func TestConstDecl(t *testing.T) {
	expectOutput(t, `
const PI = 3.14
print(PI)
`, "3.14\n")
}

func TestConstReassignError(t *testing.T) {
	expectError(t, `
const x = 1
x = 2
`, "cannot assign to constant")
}

func TestVarReassign(t *testing.T) {
	expectOutput(t, `
var x = 1
x = 2
print(x)
`, "2\n")
}

func TestUndefinedVarError(t *testing.T) {
	expectError(t, `print(y)`, "undefined variable 'y'")
}

func TestIfElse(t *testing.T) {
	expectOutput(t, `
var x = 10
if (x > 5) {
  print("big")
} else {
  print("small")
}
`, "big\n")

	expectOutput(t, `
var x = 3
if (x > 5) {
  print("big")
} else if (x > 1) {
  print("medium")
} else {
  print("small")
}
`, "medium\n")
}

func TestWhileLoop(t *testing.T) {
	expectOutput(t, `
var i = 0
var sum = 0
while (i < 5) {
  sum = sum + i
  i = i + 1
}
print(sum)
`, "10\n")
}

func TestBreak(t *testing.T) {
	expectOutput(t, `
var i = 0
while (i < 100) {
  if (i == 3) {
    break
  }
  i = i + 1
}
print(i)
`, "3\n")
}

func TestContinue(t *testing.T) {
	expectOutput(t, `
var i = 0
var sum = 0
while (i < 5) {
  i = i + 1
  if (i == 3) {
    continue
  }
  sum = sum + i
}
print(sum)
`, "12\n")
}

func TestFunction(t *testing.T) {
	expectOutput(t, `
function add(a, b) {
  return a + b
}
print(add(3, 4))
`, "7\n")
}

func TestRecursion(t *testing.T) {
	expectOutput(t, `
function fib(n) {
  if (n <= 1) {
    return n
  }
  return fib(n - 1) + fib(n - 2)
}
print(fib(10))
`, "55\n")
}

func TestClosure(t *testing.T) {
	expectOutput(t, `
function makeCounter() {
  var count = 0
  function inc() {
    count = count + 1
    return count
  }
  return inc
}
var counter = makeCounter()
print(counter())
print(counter())
print(counter())
`, "1\n2\n3\n")
}

func TestClass(t *testing.T) {
	expectOutput(t, `
class Point {
  constructor(x, y) {
    this.x = x
    this.y = y
  }
  move(dx, dy) {
    this.x = this.x + dx
    this.y = this.y + dy
  }
}
var p = new Point(1, 2)
p.move(3, 4)
print(p.x)
print(p.y)
`, "4\n6\n")
}

func TestStringConcat(t *testing.T) {
	expectOutput(t, `print("hello" + " " + "world")`, "hello world\n")
}

func TestLogicalOps(t *testing.T) {
	expectOutput(t, `print(true && false)`, "false\n")
	expectOutput(t, `print(true || false)`, "true\n")
	expectOutput(t, `print(!true)`, "false\n")
}

func TestComparison(t *testing.T) {
	expectOutput(t, `print(1 == 1)`, "true\n")
	expectOutput(t, `print(1 != 2)`, "true\n")
	expectOutput(t, `print(3 > 2)`, "true\n")
	expectOutput(t, `print(2 <= 2)`, "true\n")
}

func TestDivisionByZero(t *testing.T) {
	expectError(t, `print(1 / 0)`, "division by zero")
}

func TestBuiltinTypeOf(t *testing.T) {
	expectOutput(t, `print(typeOf(42))`, "int\n")
	expectOutput(t, `print(typeOf("hi"))`, "string\n")
	expectOutput(t, `print(typeOf(true))`, "bool\n")
	expectOutput(t, `print(typeOf(null))`, "null\n")
}

func TestBuiltinLen(t *testing.T) {
	expectOutput(t, `print(len("hello"))`, "5\n")
}

func TestBuiltinToString(t *testing.T) {
	expectOutput(t, `print(toString(42))`, "42\n")
}

func TestStringIndex(t *testing.T) {
	expectOutput(t, `
var s = "hello"
print(s[0])
print(s[4])
`, "h\no\n")
}

func TestNullEquality(t *testing.T) {
	expectOutput(t, `print(null == null)`, "true\n")
	expectOutput(t, `print(null != 1)`, "true\n")
}

func TestUnaryMinus(t *testing.T) {
	expectOutput(t, `print(-5)`, "-5\n")
	expectOutput(t, `print(-3.14)`, "-3.14\n")
}

func TestMultipleArgs(t *testing.T) {
	expectOutput(t, `print(1, 2, 3)`, "1 2 3\n")
}

func TestNestedFunction(t *testing.T) {
	expectOutput(t, `
function outer() {
  var x = 10
  function inner() {
    return x + 1
  }
  return inner()
}
print(outer())
`, "11\n")
}

func TestFibonacci(t *testing.T) {
	source := `
function fib(n) {
  if (n <= 1) {
    return n
  }
  return fib(n - 1) + fib(n - 2)
}
var i = 0
while (i < 10) {
  print(fib(i))
  i = i + 1
}
`
	expectOutput(t, source, "0\n1\n1\n2\n3\n5\n8\n13\n21\n34\n")
}
