package ast

import (
	"light-lang/internal/span"
	"light-lang/internal/token"
)

// NodeToMap converts an AST node to a map suitable for JSON serialization.
// This produces a tagged-union structure: every node has a "kind" field.
func NodeToMap(node Node) map[string]interface{} {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *File:
		return m("File", n.Span, "body", nodeSlice(n.Body))

	// ---- Expressions ----
	case *IdentExpr:
		return m("IdentExpr", n.Span, "name", n.Name)
	case *IntLiteral:
		return m("IntLiteral", n.Span, "value", n.Value)
	case *FloatLiteral:
		return m("FloatLiteral", n.Span, "value", n.Value)
	case *StringLiteral:
		return m("StringLiteral", n.Span, "value", n.Value)
	case *BoolLiteral:
		return m("BoolLiteral", n.Span, "value", n.Value)
	case *NullLiteral:
		return m("NullLiteral", n.Span)
	case *ThisExpr:
		return m("ThisExpr", n.Span)
	case *UnaryExpr:
		return m("UnaryExpr", n.Span, "op", opStr(n.Op), "operand", NodeToMap(n.Operand))
	case *BinaryExpr:
		return m("BinaryExpr", n.Span,
			"op", opStr(n.Op),
			"left", NodeToMap(n.Left),
			"right", NodeToMap(n.Right))
	case *CallExpr:
		return m("CallExpr", n.Span,
			"callee", NodeToMap(n.Callee),
			"args", exprSlice(n.Args))
	case *IndexExpr:
		return m("IndexExpr", n.Span,
			"object", NodeToMap(n.Object),
			"index", NodeToMap(n.Index))
	case *MemberExpr:
		return m("MemberExpr", n.Span,
			"object", NodeToMap(n.Object),
			"property", n.Property)
	case *NewExpr:
		return m("NewExpr", n.Span,
			"className", n.ClassName,
			"args", exprSlice(n.Args))
	case *ArrayLiteral:
		return m("ArrayLiteral", n.Span, "elements", exprSlice(n.Elements))
	case *FuncExpr:
		return m("FuncExpr", n.Span, "name", n.Name, "params", n.Params, "body", NodeToMap(n.Body))

	// ---- Statements ----
	case *ExprStmt:
		return m("ExprStmt", n.Span, "expr", NodeToMap(n.Expr))
	case *AssignStmt:
		return m("AssignStmt", n.Span,
			"target", NodeToMap(n.Target),
			"value", NodeToMap(n.Value))
	case *VarDeclStmt:
		result := m("VarDeclStmt", n.Span, "name", n.Name, "isConst", n.IsConst)
		if n.Init != nil {
			result["init"] = NodeToMap(n.Init)
		}
		return result
	case *ReturnStmt:
		result := m("ReturnStmt", n.Span)
		if n.Value != nil {
			result["value"] = NodeToMap(n.Value)
		}
		return result
	case *BreakStmt:
		return m("BreakStmt", n.Span)
	case *ContinueStmt:
		return m("ContinueStmt", n.Span)
	case *BlockStmt:
		return m("BlockStmt", n.Span, "stmts", nodeSlice(n.Stmts))
	case *IfStmt:
		result := m("IfStmt", n.Span,
			"condition", NodeToMap(n.Condition),
			"body", NodeToMap(n.Body))
		if len(n.ElseIfs) > 0 {
			elseIfs := make([]interface{}, len(n.ElseIfs))
			for i, ei := range n.ElseIfs {
				elseIfs[i] = map[string]interface{}{
					"kind":      "ElseIfClause",
					"span":      spanToMap(ei.Span),
					"condition": NodeToMap(ei.Condition),
					"body":      NodeToMap(ei.Body),
				}
			}
			result["elseIfs"] = elseIfs
		}
		if n.ElseBody != nil {
			result["elseBody"] = NodeToMap(n.ElseBody)
		}
		return result
	case *WhileStmt:
		return m("WhileStmt", n.Span,
			"condition", NodeToMap(n.Condition),
			"body", NodeToMap(n.Body))
	case *ForStmt:
		result := m("ForStmt", n.Span, "body", NodeToMap(n.Body))
		if n.Init != nil {
			result["init"] = NodeToMap(n.Init)
		}
		if n.Condition != nil {
			result["condition"] = NodeToMap(n.Condition)
		}
		if n.Update != nil {
			result["update"] = NodeToMap(n.Update)
		}
		return result
	case *ForOfStmt:
		return m("ForOfStmt", n.Span,
			"varName", n.VarName,
			"iterable", NodeToMap(n.Iterable),
			"body", NodeToMap(n.Body))

	// ---- Declarations ----
	case *FuncDecl:
		return m("FuncDecl", n.Span,
			"name", n.Name,
			"params", n.Params,
			"body", NodeToMap(n.Body))
	case *ClassDecl:
		result := m("ClassDecl", n.Span, "name", n.Name)
		if n.Constructor != nil {
			result["constructor"] = map[string]interface{}{
				"kind":   "ConstructorDecl",
				"span":   spanToMap(n.Constructor.Span),
				"params": n.Constructor.Params,
				"body":   NodeToMap(n.Constructor.Body),
			}
		}
		if len(n.Methods) > 0 {
			methods := make([]interface{}, len(n.Methods))
			for i, md := range n.Methods {
				methods[i] = map[string]interface{}{
					"kind":   "MethodDecl",
					"span":   spanToMap(md.Span),
					"name":   md.Name,
					"params": md.Params,
					"body":   NodeToMap(md.Body),
				}
			}
			result["methods"] = methods
		}
		return result

	default:
		return map[string]interface{}{"kind": "Unknown"}
	}
}

// ---- helpers ----

// m builds a map with kind, span, and extra key-value pairs.
func m(kind string, s span.Span, kvs ...interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"kind": kind,
		"span": spanToMap(s),
	}
	for i := 0; i+1 < len(kvs); i += 2 {
		key := kvs[i].(string)
		result[key] = kvs[i+1]
	}
	return result
}

func spanToMap(s span.Span) map[string]interface{} {
	return map[string]interface{}{
		"start": map[string]interface{}{
			"offset": s.Start.Offset,
			"line":   s.Start.Line,
			"column": s.Start.Column,
		},
		"end": map[string]interface{}{
			"offset": s.End.Offset,
			"line":   s.End.Line,
			"column": s.End.Column,
		},
	}
}

func nodeSlice(nodes []Node) []interface{} {
	result := make([]interface{}, len(nodes))
	for i, n := range nodes {
		result[i] = NodeToMap(n)
	}
	return result
}

func exprSlice(exprs []Expr) []interface{} {
	result := make([]interface{}, len(exprs))
	for i, e := range exprs {
		result[i] = NodeToMap(e)
	}
	return result
}

func opStr(kind token.Kind) string {
	return kind.String()
}
