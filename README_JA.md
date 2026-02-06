# Light Lang

[English](README.md) | [中文](README_CN.md) | 日本語

Go でゼロから実装した軽量な動的型付けプログラミング言語インタプリタです。

Light Lang は JavaScript/TypeScript にインスパイアされた簡潔で表現力豊かな構文を持ち、クラス、クロージャ、高階関数、例外処理などをサポートしています。サードパーティ依存なし、Go 標準ライブラリのみで実装されています。

## 特徴

- **動的型付け** — 変数は任意の型を保持可能：`int`、`float`、`string`、`bool`、`null`、`array`、`map`
- **第一級関数** — 関数を値として扱い、クロージャ、アロー関数 `(x) => x * 2` をサポート
- **オブジェクト指向** — クラス、コンストラクタ、メソッド、単一継承（`extends`）、`super` 呼び出し
- **例外処理** — `try` / `catch` / `throw` による構造化例外処理
- **コレクション** — 配列 `[1, 2, 3]` と辞書 `{ key: "value" }`、組み込みメソッド付き
- **制御フロー** — `if/else`、`while`、C スタイル `for`、`for-of` イテレーション、`break`、`continue`
- **三項演算子** — `condition ? then : else`
- **複合代入** — `+=`、`-=`、`*=`、`/=`
- **対話型 REPL** — インタラクティブに言語を試せる環境
- **完全なツールチェーン** — 字句解析器、構文解析器（AST を JSON で出力）、インタプリタ

## クイックスタート

### 前提条件

- [Go](https://go.dev/) 1.21+ がインストール済みであること

### ビルド

```bash
git clone https://github.com/duhaifeng/light-lang.git
cd light-lang
go build -o light ./cmd/light
```

### プログラムの実行

```bash
./light run testdata/hello.lt
```

### REPL の起動

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

## 言語ツアー

### 変数

```javascript
var name = "Alice"
var age = 30
var pi = 3.14
var active = true
const MAX = 100
```

### 関数

```javascript
// 通常の関数
function greet(name) {
  return "Hello, " + name + "!"
}
print(greet("World"))  // Hello, World!

// アロー関数
var add = (a, b) => a + b
var square = x => x * x
print(add(3, 4))   // 7
print(square(6))    // 36
```

### 制御フロー

```javascript
// 条件分岐
if (x > 10) {
  print("big")
} else if (x > 5) {
  print("medium")
} else {
  print("small")
}

// While ループ
var i = 0
while (i < 5) {
  print(i)
  i += 1
}

// C スタイル for ループ
for (var i = 0; i < 10; i += 1) {
  print(i)
}

// For-of イテレーション
var items = [10, 20, 30]
for (var item of items) {
  print(item)
}
```

### 配列

```javascript
var arr = [1, 2, 3, 4, 5]
arr.push(6)
print(arr.length)   // 6
print(arr[0])       // 1
print(arr.pop())    // 6
```

### 辞書（Map）

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

// 辞書のキーを反復処理
for (var key of person) {
  print(key + " = " + person[key])
}
```

### クラスと継承

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

### クロージャ

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

### 例外処理

```javascript
function safeDivide(a, b) {
  if (b == 0) {
    throw "division by zero"
  }
  return a / b
}

try {
  print(safeDivide(10, 2))   // 5
  print(safeDivide(10, 0))   // 例外をスロー
} catch (e) {
  print("caught: " + e)      // caught: division by zero
}
```

### 高階関数

```javascript
function applyTwice(fn, val) {
  return fn(fn(val))
}

print(applyTwice(x => x + 1, 10))  // 12
print(applyTwice(x => x * 2, 3))   // 12
```

### 三項演算子と文字列の自動変換

```javascript
var x = 5
print(x > 3 ? "big" : "small")     // big
print("value: " + 42)               // value: 42
print(100 + " dollars")             // 100 dollars
```

## コマンドライン

```
使い方:
  light tokens <file> [--json]   字句解析してトークンを表示
  light parse  <file>            構文解析して AST を表示（JSON 形式）
  light run    <file>            ソースファイルを実行
  light repl                     対話型 REPL を起動
```

### 使用例

```bash
# プログラムを実行
./light run testdata/fib.lt

# トークンを表示
./light tokens testdata/hello.lt

# トークンを JSON 形式で表示
./light tokens testdata/hello.lt --json

# AST を JSON 形式で表示
./light parse testdata/hello.lt

# 対話モード
./light repl
```

## 組み込み関数

| 関数 | 説明 |
|---|---|
| `print(...)` | 値をスペース区切りで出力 |
| `println(...)` | `print` と同じ |
| `typeOf(value)` | 値の型名を文字列で返す |
| `toString(value)` | 値を文字列に変換 |
| `len(value)` | 文字列・配列・辞書の長さを返す |
| `push(array, value)` | 配列の末尾に要素を追加し、新しい長さを返す |
| `pop(array)` | 配列の最後の要素を削除して返す |
| `keys(map)` | 辞書のすべてのキーを配列で返す |
| `values(map)` | 辞書のすべての値を配列で返す |

配列はメソッド形式の呼び出しもサポート：`arr.push(val)`、`arr.pop()`、`arr.length`。

## アーキテクチャ

Light Lang は典型的なインタプリタパイプラインに従っています：

```
ソースコード → 字句解析器 → トークン列 → 構文解析器 → AST → インタプリタ → 出力
```

```
light-lang/
├── cmd/light/           # CLI エントリーポイント（tokens、parse、run、repl）
├── internal/
│   ├── token/           # トークン型定義とキーワード
│   ├── span/            # ソース位置の追跡（行番号、列番号、オフセット）
│   ├── lexer/           # 字句解析 — ソーステキストからトークンへ
│   ├── parser/          # 構文解析 — Pratt 解析 + 再帰下降
│   ├── ast/             # 抽象構文木のノード定義
│   ├── diag/            # 診断 / エラーレポート
│   └── runtime/         # ツリーウォーキングインタプリタ
│       ├── interpreter.go   # AST 実行エンジン
│       ├── value.go         # ランタイム値型
│       ├── env.go           # レキシカルスコープ / 環境チェーン
│       └── builtin.go       # 組み込み関数
├── testdata/            # サンプルプログラムとテストケース
└── docs/                # 設計ドキュメント（中国語）
```

### 主要な設計方針

- **Pratt 解析** で式を処理 — 簡潔で拡張可能な優先順位制御
- **再帰下降** で文を処理 — 直感的で拡張しやすい
- **ツリーウォーキングインタプリタ** — コンパイルなしで AST を直接実行
- **レキシカルスコープ** — 親ポインタ付き環境チェーンによるクロージャの実現
- **ゼロ依存** — Go 標準ライブラリのみ、サードパーティパッケージなし

## サンプルプログラム

`testdata/` ディレクトリにはいくつかのサンプルプログラムが含まれています：

| ファイル | 説明 |
|---|---|
| `hello.lt` | Hello World、基本的な算術演算 |
| `fib.lt` | 再帰フィボナッチ数列 |
| `class.lt` | クラス定義とメソッド呼び出し |
| `golden_array.lt` | 配列操作とイテレーション |
| `golden_for.lt` | さまざまな for ループパターン |
| `golden_features.lt` | 総合的な機能ショーケース |
| `golden_complex.lt` | 応用：ソート、スタック、高階関数、クロージャ、行列操作 |

## コントリビューション

コントリビューションを歓迎します！お気軽に Issue や Pull Request を送ってください。

1. リポジトリをフォーク
2. フィーチャーブランチを作成（`git checkout -b feature/amazing-feature`）
3. 変更をコミット（`git commit -m '素晴らしい機能を追加'`）
4. ブランチにプッシュ（`git push origin feature/amazing-feature`）
5. Pull Request を作成

## ライセンス

本プロジェクトはオープンソースです。詳細は [LICENSE](LICENSE) ファイルをご覧ください。

## 謝辞

本プロジェクトは、プログラミング言語設計とインタプリタ実装を探求するために作成されました。学習リソースとしても、さらなる言語実験の基盤としてもご活用いただけます。
