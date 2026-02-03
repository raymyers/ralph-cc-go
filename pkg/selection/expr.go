// Package selection - Expression selection for instruction selection pass.
// This file transforms Cminor expressions to CminorSel expressions,
// selecting addressing modes for loads and machine operators for arithmetic.
package selection

import (
	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/cminorsel"
)

// SelectionContext holds context needed during expression selection.
type SelectionContext struct {
	// Globals is a set of global variable names
	Globals map[string]bool
	// StackVars maps stack variable names to their stack offsets
	StackVars map[string]int64
}

// NewSelectionContext creates a new selection context.
func NewSelectionContext(globals map[string]bool, stackVars map[string]int64) *SelectionContext {
	if globals == nil {
		globals = make(map[string]bool)
	}
	if stackVars == nil {
		stackVars = make(map[string]int64)
	}
	return &SelectionContext{
		Globals:   globals,
		StackVars: stackVars,
	}
}

// SelectExpr transforms a Cminor expression to a CminorSel expression.
// It selects addressing modes for loads and machine operators for operations.
func (ctx *SelectionContext) SelectExpr(e cminor.Expr) cminorsel.Expr {
	switch expr := e.(type) {
	case cminor.Evar:
		return ctx.selectVar(expr)

	case cminor.Econst:
		return ctx.selectConst(expr)

	case cminor.Eunop:
		return ctx.selectUnop(expr)

	case cminor.Ebinop:
		return ctx.selectBinop(expr)

	case cminor.Ecmp:
		return ctx.selectCmp(expr)

	case cminor.Eload:
		return ctx.selectLoad(expr)

	default:
		// Unknown expression type - return as variable for safety
		return cminorsel.Evar{Name: "?unknown?"}
	}
}

// selectVar handles variable references.
func (ctx *SelectionContext) selectVar(v cminor.Evar) cminorsel.Expr {
	// Check if it's a global symbol
	if ctx.Globals[v.Name] {
		return cminorsel.Econst{Const: cminorsel.Oaddrsymbol{Symbol: v.Name, Offset: 0}}
	}
	// Check if it's a stack variable
	if off, ok := ctx.StackVars[v.Name]; ok {
		return cminorsel.Econst{Const: cminorsel.Oaddrstack{Offset: off}}
	}
	// Regular local variable
	return cminorsel.Evar{Name: v.Name}
}

// selectConst handles constant expressions.
func (ctx *SelectionContext) selectConst(c cminor.Econst) cminorsel.Expr {
	switch cnst := c.Const.(type) {
	case cminor.Ointconst:
		return cminorsel.Econst{Const: cminorsel.Ointconst{Value: cnst.Value}}
	case cminor.Ofloatconst:
		return cminorsel.Econst{Const: cminorsel.Ofloatconst{Value: cnst.Value}}
	case cminor.Olongconst:
		return cminorsel.Econst{Const: cminorsel.Olongconst{Value: cnst.Value}}
	case cminor.Osingleconst:
		return cminorsel.Econst{Const: cminorsel.Osingleconst{Value: cnst.Value}}
	case cminor.Oaddrsymbol:
		return cminorsel.Econst{Const: cminorsel.Oaddrsymbol{Symbol: cnst.Name, Offset: cnst.Offset}}
	default:
		return cminorsel.Econst{Const: cminorsel.Ointconst{Value: 0}}
	}
}

// selectUnop handles unary operations.
func (ctx *SelectionContext) selectUnop(u cminor.Eunop) cminorsel.Expr {
	arg := ctx.SelectExpr(u.Arg)
	return cminorsel.Eunop{
		Op:  cminorsel.UnaryOp(u.Op),
		Arg: arg,
	}
}

// selectBinop handles binary operations, including combined operation recognition.
func (ctx *SelectionContext) selectBinop(b cminor.Ebinop) cminorsel.Expr {
	// Try to recognize combined shift+arithmetic patterns (ARM64)
	combined := TrySelectCombinedOp(b.Op, b.Left, b.Right)
	if combined.IsCombined {
		return ctx.buildCombinedOp(combined)
	}

	// Standard binary operation
	left := ctx.SelectExpr(b.Left)
	right := ctx.SelectExpr(b.Right)
	return cminorsel.Ebinop{
		Op:    cminorsel.BinaryOp(b.Op),
		Left:  left,
		Right: right,
	}
}

// buildCombinedOp builds a CminorSel expression for a combined operation.
func (ctx *SelectionContext) buildCombinedOp(c CombinedOpResult) cminorsel.Expr {
	base := ctx.SelectExpr(c.Base)
	index := ctx.SelectExpr(c.Index)

	// Determine shift type (ARM64 uses logical shift left for combined ops)
	shiftOp := cminorsel.Slsl

	// Select the appropriate combined expression type
	switch c.Op {
	case cminorsel.MOaddshift, cminorsel.MOaddlshift:
		return cminorsel.Eaddshift{
			Op:    shiftOp,
			Shift: c.Shift,
			Left:  base,
			Right: index,
		}
	case cminorsel.MOsubshift, cminorsel.MOsublshift:
		return cminorsel.Esubshift{
			Op:    shiftOp,
			Shift: c.Shift,
			Left:  base,
			Right: index,
		}
	default:
		// For and/or/xor shifts, fall back to regular binop
		// (CminorSel AST doesn't have Eandshift etc.)
		return cminorsel.Ebinop{
			Op:    cminorsel.BinaryOp(cminor.Oand), // fallback
			Left:  base,
			Right: index,
		}
	}
}

// selectCmp handles comparison expressions.
func (ctx *SelectionContext) selectCmp(c cminor.Ecmp) cminorsel.Expr {
	left := ctx.SelectExpr(c.Left)
	right := ctx.SelectExpr(c.Right)

	// Comparisons produce int 0 or 1, represented as Ecmp in CminorSel
	// preserving the comparison condition (Ceq, Clt, etc.)
	return cminorsel.Ecmp{
		Op:    cminorsel.BinaryOp(c.Op),
		Cmp:   cminorsel.Comparison(c.Cmp),
		Left:  left,
		Right: right,
	}
}

// selectLoad handles memory load with addressing mode selection.
func (ctx *SelectionContext) selectLoad(ld cminor.Eload) cminorsel.Expr {
	// Select addressing mode for the address expression
	addrResult := SelectAddressing(ld.Addr, ctx.Globals, ctx.StackVars)

	// For address arguments, we need to fully select them
	selectedArgs := make([]cminorsel.Expr, len(addrResult.Args))
	for i, arg := range addrResult.Args {
		// Args are already in CminorSel form from SelectAddressing
		// but we need to recursively select any nested expressions
		selectedArgs[i] = ctx.reSelectExpr(arg)
	}

	return cminorsel.Eload{
		Chunk: cminorsel.Chunk(ld.Chunk),
		Mode:  addrResult.Mode,
		Args:  selectedArgs,
	}
}

// reSelectExpr handles CminorSel expressions that may need further selection.
// This is needed because SelectAddressing returns partially-selected expressions.
func (ctx *SelectionContext) reSelectExpr(e cminorsel.Expr) cminorsel.Expr {
	switch expr := e.(type) {
	case cminorsel.Evar:
		// Check if this var should be a global or stack address
		if ctx.Globals[expr.Name] {
			return cminorsel.Econst{Const: cminorsel.Oaddrsymbol{Symbol: expr.Name, Offset: 0}}
		}
		if off, ok := ctx.StackVars[expr.Name]; ok {
			return cminorsel.Econst{Const: cminorsel.Oaddrstack{Offset: off}}
		}
		return expr

	case cminorsel.Econst:
		return expr

	case cminorsel.Eunop:
		return cminorsel.Eunop{
			Op:  expr.Op,
			Arg: ctx.reSelectExpr(expr.Arg),
		}

	case cminorsel.Ebinop:
		return cminorsel.Ebinop{
			Op:    expr.Op,
			Left:  ctx.reSelectExpr(expr.Left),
			Right: ctx.reSelectExpr(expr.Right),
		}

	case cminorsel.Eload:
		args := make([]cminorsel.Expr, len(expr.Args))
		for i, arg := range expr.Args {
			args[i] = ctx.reSelectExpr(arg)
		}
		return cminorsel.Eload{
			Chunk: expr.Chunk,
			Mode:  expr.Mode,
			Args:  args,
		}

	default:
		return expr
	}
}

// SelectCondition transforms a Cminor expression used as a condition
// to a CminorSel condition. This handles comparisons specially.
func (ctx *SelectionContext) SelectCondition(e cminor.Expr) cminorsel.Condition {
	switch expr := e.(type) {
	case cminor.Ecmp:
		// Direct comparison - select as a proper condition
		left := ctx.SelectExpr(expr.Left)
		right := ctx.SelectExpr(expr.Right)
		return SelectComparison(expr.Op, expr.Cmp, left, right)

	case cminor.Econst:
		// Constant condition
		switch c := expr.Const.(type) {
		case cminor.Ointconst:
			if c.Value != 0 {
				return cminorsel.CondTrue{}
			}
			return cminorsel.CondFalse{}
		case cminor.Olongconst:
			if c.Value != 0 {
				return cminorsel.CondTrue{}
			}
			return cminorsel.CondFalse{}
		}
		return cminorsel.CondTrue{}

	case cminor.Eunop:
		// Handle logical not
		if expr.Op == cminor.Onotbool {
			return cminorsel.CondNot{Cond: ctx.SelectCondition(expr.Arg)}
		}
		// Other unary ops: compare result != 0
		selected := ctx.SelectExpr(e)
		return cminorsel.CondCmp{
			Cmp:   cminorsel.Cne,
			Left:  selected,
			Right: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 0}},
		}

	default:
		// General expression: compare result != 0
		selected := ctx.SelectExpr(e)
		return cminorsel.CondCmp{
			Cmp:   cminorsel.Cne,
			Left:  selected,
			Right: cminorsel.Econst{Const: cminorsel.Ointconst{Value: 0}},
		}
	}
}

// IsProfitableIfConversion checks if converting a simple if/else to
// a conditional move would be beneficial. This is a heuristic.
func IsProfitableIfConversion(thenExpr, elseExpr cminor.Expr) bool {
	// If-conversion is profitable when:
	// 1. Both branches are simple expressions (no side effects)
	// 2. The expressions are cheap to compute
	return isSimpleExpr(thenExpr) && isSimpleExpr(elseExpr)
}

// isSimpleExpr checks if an expression is simple enough for if-conversion.
func isSimpleExpr(e cminor.Expr) bool {
	switch expr := e.(type) {
	case cminor.Evar, cminor.Econst:
		return true
	case cminor.Eunop:
		return isSimpleExpr(expr.Arg)
	case cminor.Ebinop:
		// Avoid expensive operations
		switch expr.Op {
		case cminor.Odiv, cminor.Odivu, cminor.Omod, cminor.Omodu,
			cminor.Odivl, cminor.Odivlu, cminor.Omodl, cminor.Omodlu,
			cminor.Odivf, cminor.Odivs:
			return false
		}
		return isSimpleExpr(expr.Left) && isSimpleExpr(expr.Right)
	case cminor.Ecmp:
		return isSimpleExpr(expr.Left) && isSimpleExpr(expr.Right)
	case cminor.Eload:
		// Loads have side effects (memory access)
		return false
	default:
		return false
	}
}

// SelectConditionalExpr creates a conditional expression (ternary).
// Used when if-conversion is profitable.
func (ctx *SelectionContext) SelectConditionalExpr(cond cminor.Expr, thenExpr, elseExpr cminor.Expr) cminorsel.Expr {
	return cminorsel.Econdition{
		Cond: ctx.SelectCondition(cond),
		Then: ctx.SelectExpr(thenExpr),
		Else: ctx.SelectExpr(elseExpr),
	}
}
