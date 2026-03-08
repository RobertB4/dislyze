// Package auditcheck is a golangci-lint plugin that ensures InsertAuditLog
// errors are always returned to the caller. Silently dropping audit log errors
// is a compliance violation (GDPR Article 30).
//
// Flagged patterns:
//
//	_ = q.InsertAuditLog(...)                              // error discarded
//	if err := q.InsertAuditLog(...); err != nil { log() }  // error not returned
//	err = q.InsertAuditLog(...)
//	if err != nil { log() }                                // error not returned
//
// Intentional exceptions (middleware denial paths, logout) should use
// //nolint:auditcheck with a justification comment.
package auditcheck

import (
	"go/ast"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("auditcheck", New)
}

func New(_ any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		{
			Name: "auditcheck",
			Doc:  "checks that InsertAuditLog errors are returned, not silently dropped",
			Run:  run,
		},
	}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				if isBlankAssignToInsertAuditLog(node) {
					pass.Report(analysis.Diagnostic{
						Pos:     node.Pos(),
						Message: "InsertAuditLog error discarded — audit log errors must be returned (compliance requirement)",
					})
				}
			case *ast.IfStmt:
				if isInsertAuditLogIfWithoutReturn(node) {
					pass.Report(analysis.Diagnostic{
						Pos:     node.Pos(),
						Message: "InsertAuditLog error not returned — audit log errors must be returned (compliance requirement).",
					})
				}
			case *ast.BlockStmt:
				checkConsecutiveStmts(pass, node)
			}
			return true
		})
	}
	return nil, nil
}

// isBlankAssignToInsertAuditLog checks for: _ = q.InsertAuditLog(...)
func isBlankAssignToInsertAuditLog(stmt *ast.AssignStmt) bool {
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return false
	}

	ident, ok := stmt.Lhs[0].(*ast.Ident)
	if !ok || ident.Name != "_" {
		return false
	}

	return isInsertAuditLogCall(stmt.Rhs[0])
}

// isInsertAuditLogIfWithoutReturn checks for:
// if err := q.InsertAuditLog(...); err != nil { /* no return */ }
func isInsertAuditLogIfWithoutReturn(ifStmt *ast.IfStmt) bool {
	if ifStmt.Init == nil {
		return false
	}

	assignStmt, ok := ifStmt.Init.(*ast.AssignStmt)
	if !ok || len(assignStmt.Rhs) != 1 {
		return false
	}

	if !isInsertAuditLogCall(assignStmt.Rhs[0]) {
		return false
	}

	return !containsReturn(ifStmt.Body)
}

// checkConsecutiveStmts checks for the pattern:
//
//	err = q.InsertAuditLog(...)
//	if err != nil { /* no return */ }
func checkConsecutiveStmts(pass *analysis.Pass, block *ast.BlockStmt) {
	for i := 0; i < len(block.List)-1; i++ {
		assignStmt, ok := block.List[i].(*ast.AssignStmt)
		if !ok || len(assignStmt.Rhs) != 1 {
			continue
		}

		if !isInsertAuditLogCall(assignStmt.Rhs[0]) {
			continue
		}

		// Check if LHS assigns to a named variable (not _; that's caught separately)
		if len(assignStmt.Lhs) != 1 {
			continue
		}
		lhsIdent, ok := assignStmt.Lhs[0].(*ast.Ident)
		if !ok || lhsIdent.Name == "_" {
			continue
		}

		// Look at the next statement — is it an if checking this error?
		ifStmt, ok := block.List[i+1].(*ast.IfStmt)
		if !ok || ifStmt.Init != nil {
			continue
		}

		if !containsReturn(ifStmt.Body) {
			pass.Report(analysis.Diagnostic{
				Pos:     assignStmt.Pos(),
				Message: "InsertAuditLog error not returned — audit log errors must be returned (compliance requirement).",
			})
		}
	}
}

func isInsertAuditLogCall(expr ast.Expr) bool {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return selectorExpr.Sel.Name == "InsertAuditLog"
}

func containsReturn(block *ast.BlockStmt) bool {
	if block == nil {
		return false
	}

	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		if _, ok := n.(*ast.ReturnStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}
