# Light Lang

[English](README.md) | 中文 | [日本語](README_JA.md)

一个使用 Go 从零实现的轻量级动态类型编程语言解释器。

Light Lang 拥有简洁、富有表现力的语法，灵感来源于 JavaScript/TypeScript，支持类、闭包、高阶函数、异常处理等特性 —— 全部使用 Go 标准库实现，零第三方依赖。

## 特性

- **动态类型** — 变量可持有任意类型：`int`、`float`、`string`、`bool`、`null`、`array`、`map`
- **一等函数** — 函数作为值传递、闭包、箭头函数 `(x) => x * 2`
- **面向对象** — 类、构造函数、方法、单继承（`extends`）、`super` 调用
- **异常处理** — `try` / `catch` / `throw` 结构化异常机制
- **集合类型** — 数组 `[1, 2, 3]` 和字典 `{ key: "value" }`，附带内置方法
- **控制流** — `if/else`、`while`、C 风格 `for`、`for-of` 迭代、`break`、`continue`
- **三元运算符** — `condition ? then : else`
- **复合赋值** — `+=`、`-=`、`*=`、`/=`
- **交互式 REPL** — 交互式探索语言特性
- **完整工具链** — 词法分析器、语法分析器（AST 以 JSON 输出）、解释器

## 快速开始

### 环境要求

- 已安装 [Go](https://go.dev/) 1.21+

### 构建

```bash
git clone https://github.com/duhaifeng/light-lang.git
cd light-lang
go build -o light ./cmd/light
```

### 运行程序

```bash
./light run testdata/hello.lt
```

### 启动 REPL

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

## 语言概览

### 变量

```javascript
var name = "Alice"
var age = 30
var pi = 3.14
var active = true
const MAX = 100
```

### 函数

```javascript
// 普通函数
function greet(name) {
  return "Hello, " + name + "!"
}
print(greet("World"))  // Hello, World!

// 箭头函数
var add = (a, b) => a + b
var square = x => x * x
print(add(3, 4))   // 7
print(square(6))    // 36
```

### 控制流

```javascript
// 条件分支
if (x > 10) {
  print("big")
} else if (x > 5) {
  print("medium")
} else {
  print("small")
}

// While 循环
var i = 0
while (i < 5) {
  print(i)
  i += 1
}

// C 风格 for 循环
for (var i = 0; i < 10; i += 1) {
  print(i)
}

// For-of 迭代
var items = [10, 20, 30]
for (var item of items) {
  print(item)
}
```

### 数组

```javascript
var arr = [1, 2, 3, 4, 5]
arr.push(6)
print(arr.length)   // 6
print(arr[0])       // 1
print(arr.pop())    // 6
```

### 字典（Map）

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

// 遍历字典的键
for (var key of person) {
  print(key + " = " + person[key])
}
```

### 类与继承

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

### 闭包

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

### 异常处理

```javascript
function safeDivide(a, b) {
  if (b == 0) {
    throw "division by zero"
  }
  return a / b
}

try {
  print(safeDivide(10, 2))   // 5
  print(safeDivide(10, 0))   // 抛出异常
} catch (e) {
  print("caught: " + e)      // caught: division by zero
}
```

### 高阶函数

```javascript
function applyTwice(fn, val) {
  return fn(fn(val))
}

print(applyTwice(x => x + 1, 10))  // 12
print(applyTwice(x => x * 2, 3))   // 12
```

### 三元运算符与字符串自动拼接

```javascript
var x = 5
print(x > 3 ? "big" : "small")     // big
print("value: " + 42)               // value: 42
print(100 + " dollars")             // 100 dollars
```

## 命令行用法

```
用法:
  light tokens <file> [--json]   词法分析并打印 Token
  light parse  <file>            语法分析并打印 AST（JSON 格式）
  light run    <file>            运行源文件
  light repl                     启动交互式 REPL
```

### 示例

```bash
# 运行程序
./light run testdata/fib.lt

# 查看 Token
./light tokens testdata/hello.lt

# 以 JSON 格式查看 Token
./light tokens testdata/hello.lt --json

# 查看 AST（JSON 格式）
./light parse testdata/hello.lt

# 交互模式
./light repl
```

## 内置函数

| 函数 | 说明 |
|---|---|
| `print(...)` | 打印值，多个参数以空格分隔 |
| `println(...)` | 同 `print` |
| `typeOf(value)` | 返回值的类型名称（字符串） |
| `toString(value)` | 将值转换为字符串 |
| `len(value)` | 返回字符串、数组或字典的长度 |
| `push(array, value)` | 向数组末尾追加元素，返回新长度 |
| `pop(array)` | 移除并返回数组最后一个元素 |
| `keys(map)` | 返回字典所有键组成的数组 |
| `values(map)` | 返回字典所有值组成的数组 |

数组还支持方法风格调用：`arr.push(val)`、`arr.pop()`、`arr.length`。

## 架构设计

Light Lang 遵循经典的解释器管线：

```
源代码 → 词法分析器 → Token 流 → 语法分析器 → AST → 解释器 → 输出
```

```
light-lang/
├── cmd/light/           # CLI 入口（tokens、parse、run、repl）
├── internal/
│   ├── token/           # Token 类型定义与关键字
│   ├── span/            # 源码位置追踪（行号、列号、偏移量）
│   ├── lexer/           # 词法分析 — 源码文本转 Token
│   ├── parser/          # 语法分析 — Pratt 解析 + 递归下降
│   ├── ast/             # 抽象语法树节点定义
│   ├── diag/            # 诊断 / 错误报告
│   └── runtime/         # 树遍历解释器
│       ├── interpreter.go   # AST 执行引擎
│       ├── value.go         # 运行时值类型
│       ├── env.go           # 词法作用域 / 环境链
│       └── builtin.go       # 内置函数
├── testdata/            # 示例程序与测试用例
└── docs/                # 设计文档（中文）
```

### 关键设计决策

- **Pratt 解析** 处理表达式 — 简洁、可扩展的优先级处理
- **递归下降** 处理语句 — 直观且易于扩展
- **树遍历解释器** — 直接执行 AST，无需编译步骤
- **词法作用域** — 通过父指针环境链实现闭包
- **零依赖** — 纯 Go 标准库实现，无第三方包

## 示例程序

`testdata/` 目录包含多个示例程序：

| 文件 | 说明 |
|---|---|
| `hello.lt` | Hello World，基本算术运算 |
| `fib.lt` | 递归斐波那契数列 |
| `class.lt` | 类定义与方法调用 |
| `golden_array.lt` | 数组操作与迭代 |
| `golden_for.lt` | 各种 for 循环模式 |
| `golden_features.lt` | 综合特性展示 |
| `golden_complex.lt` | 进阶：排序、栈、高阶函数、闭包、矩阵操作 |

## 参与贡献

欢迎贡献代码！请随时提交 Issue 和 Pull Request。

1. Fork 本仓库
2. 创建特性分支（`git checkout -b feature/amazing-feature`）
3. 提交更改（`git commit -m '添加某个很棒的特性'`）
4. 推送到分支（`git push origin feature/amazing-feature`）
5. 发起 Pull Request

## 许可证

本项目为开源项目，详情请参阅 [LICENSE](LICENSE) 文件。

## 致谢

本项目是对编程语言设计与解释器实现的探索。它既可作为学习资源，也可作为进一步语言实验的基础。
