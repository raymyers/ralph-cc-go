package clightgen

import (
	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
	"github.com/raymyers/ralph-cc/pkg/simplexpr"
)

// transformStmt transforms a Cabs statement to a Clight statement.
func transformStmt(stmt cabs.Stmt, simplExpr *simplexpr.Transformer) clight.Stmt {
	switch s := stmt.(type) {
	case cabs.Return:
		if s.Expr == nil {
			return clight.Sreturn{Value: nil}
		}
		result := simplExpr.TransformExpr(s.Expr)
		return clight.Seq(append(result.Stmts, clight.Sreturn{Value: result.Expr})...)

	case cabs.Computation:
		result := simplExpr.TransformExpr(s.Expr)
		return clight.Seq(result.Stmts...)

	case cabs.If:
		condResult := simplExpr.TransformExpr(s.Cond)
		thenStmt := transformStmt(s.Then, simplExpr)
		var elseStmt clight.Stmt = clight.Sskip{}
		if s.Else != nil {
			elseStmt = transformStmt(s.Else, simplExpr)
		}
		ifStmt := clight.Sifthenelse{
			Cond: condResult.Expr,
			Then: thenStmt,
			Else: elseStmt,
		}
		return clight.Seq(append(condResult.Stmts, ifStmt)...)

	case cabs.While:
		// while (cond) body becomes: loop { if (cond) body else break }
		condResult := simplExpr.TransformExpr(s.Cond)
		bodyStmt := transformStmt(s.Body, simplExpr)
		loopBody := clight.Sifthenelse{
			Cond: condResult.Expr,
			Then: bodyStmt,
			Else: clight.Sbreak{},
		}
		// Prepend condition side-effects to loop body
		fullBody := clight.Seq(append(condResult.Stmts, loopBody)...)
		return clight.Sloop{Body: fullBody, Continue: clight.Sskip{}}

	case cabs.DoWhile:
		// do body while (cond) becomes: loop { body; if (!cond) break }
		bodyStmt := transformStmt(s.Body, simplExpr)
		condResult := simplExpr.TransformExpr(s.Cond)
		checkCond := clight.Sifthenelse{
			Cond: clight.Eunop{Op: clight.Onotbool, Arg: condResult.Expr, Typ: ctypes.Int()},
			Then: clight.Sbreak{},
			Else: clight.Sskip{},
		}
		fullBody := clight.Seq(append([]clight.Stmt{bodyStmt}, append(condResult.Stmts, checkCond)...)...)
		return clight.Sloop{Body: fullBody, Continue: clight.Sskip{}}

	case cabs.For:
		// for (init; cond; step) body becomes:
		// init; loop { if (cond) { body; step } else break }
		var initStmt clight.Stmt = clight.Sskip{}
		if s.Init != nil {
			initResult := simplExpr.TransformExpr(s.Init)
			initStmt = clight.Seq(initResult.Stmts...)
		}

		var condExpr clight.Expr = clight.Econst_int{Value: 1, Typ: ctypes.Int()} // default: true
		var condStmts []clight.Stmt
		if s.Cond != nil {
			condResult := simplExpr.TransformExpr(s.Cond)
			condExpr = condResult.Expr
			condStmts = condResult.Stmts
		}

		bodyStmt := transformStmt(s.Body, simplExpr)

		var stepStmt clight.Stmt = clight.Sskip{}
		if s.Step != nil {
			stepResult := simplExpr.TransformExpr(s.Step)
			stepStmt = clight.Seq(stepResult.Stmts...)
		}

		loopBody := clight.Sifthenelse{
			Cond: condExpr,
			Then: clight.Seq(bodyStmt, stepStmt),
			Else: clight.Sbreak{},
		}
		fullBody := clight.Seq(append(condStmts, loopBody)...)
		return clight.Seq(initStmt, clight.Sloop{Body: fullBody, Continue: clight.Sskip{}})

	case cabs.Break:
		return clight.Sbreak{}

	case cabs.Continue:
		return clight.Scontinue{}

	case cabs.Switch:
		exprResult := simplExpr.TransformExpr(s.Expr)
		var cases []clight.SwitchCase
		var defaultStmt clight.Stmt = clight.Sskip{}
		for _, c := range s.Cases {
			if c.Expr == nil {
				// default case
				var stmts []clight.Stmt
				for _, st := range c.Stmts {
					stmts = append(stmts, transformStmt(st, simplExpr))
				}
				defaultStmt = clight.Seq(stmts...)
			} else {
				// case with value
				var stmts []clight.Stmt
				for _, st := range c.Stmts {
					stmts = append(stmts, transformStmt(st, simplExpr))
				}
				if constExpr, ok := c.Expr.(cabs.Constant); ok {
					cases = append(cases, clight.SwitchCase{
						Value: constExpr.Value,
						Body:  clight.Seq(stmts...),
					})
				}
			}
		}
		return clight.Seq(append(exprResult.Stmts, clight.Sswitch{
			Expr:    exprResult.Expr,
			Cases:   cases,
			Default: defaultStmt,
		})...)

	case cabs.Goto:
		return clight.Sgoto{Label: s.Label}

	case cabs.Label:
		innerStmt := transformStmt(s.Stmt, simplExpr)
		return clight.Slabel{Label: s.Name, Stmt: innerStmt}

	case cabs.Block:
		return transformBlock(&s, simplExpr)

	case *cabs.Block:
		return transformBlock(s, simplExpr)

	case cabs.DeclStmt:
		// Declarations with initializers become assignments
		var stmts []clight.Stmt
		for _, decl := range s.Decls {
			if decl.Initializer != nil {
				typ := TypeFromString(decl.TypeSpec)
				result := simplExpr.TransformExpr(decl.Initializer)
				stmts = append(stmts, result.Stmts...)
				stmts = append(stmts, clight.Sassign{
					LHS: clight.Evar{Name: decl.Name, Typ: typ},
					RHS: result.Expr,
				})
			}
		}
		return clight.Seq(stmts...)

	default:
		return clight.Sskip{}
	}
}
