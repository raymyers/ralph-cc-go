// Expression translation for RTLgen.
// Translates CminorSel expressions to RTL instruction sequences.
// Expressions are evaluated into destination registers with proper
// evaluation order (left-to-right for arguments).

package rtlgen

import (
	"github.com/raymyers/ralph-cc/pkg/cminorsel"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// ExprTranslator translates CminorSel expressions to RTL instructions.
// It builds instruction sequences that evaluate expressions into registers.
type ExprTranslator struct {
	ib   *InstrBuilder
	regs *RegAllocator
	cfg  *CFGBuilder
	ctx  *ExitContext
}

// NewExprTranslator creates an expression translator.
func NewExprTranslator(cfg *CFGBuilder, regs *RegAllocator) *ExprTranslator {
	return &ExprTranslator{
		ib:   NewInstrBuilder(cfg, regs),
		regs: regs,
		cfg:  cfg,
		ctx:  NewExitContext(),
	}
}

// TranslateExpr translates an expression to RTL instructions.
// Returns the register containing the result and the entry node.
// Instructions chain backward from succ.
//
// The "backward chaining" approach: we allocate successor first,
// then emit instructions that branch TO that successor.
func (t *ExprTranslator) TranslateExpr(e cminorsel.Expr, dest rtl.Reg, succ rtl.Node) rtl.Node {
	switch expr := e.(type) {
	case cminorsel.Evar:
		return t.translateVar(expr, dest, succ)
	case cminorsel.Econst:
		return t.translateConst(expr, dest, succ)
	case cminorsel.Eunop:
		return t.translateUnop(expr, dest, succ)
	case cminorsel.Ebinop:
		return t.translateBinop(expr, dest, succ)
	case cminorsel.Eload:
		return t.translateLoad(expr, dest, succ)
	case cminorsel.Econdition:
		return t.translateCondition(expr, dest, succ)
	case cminorsel.Elet:
		return t.translateLet(expr, dest, succ)
	case cminorsel.Eletvar:
		return t.translateLetvar(expr, dest, succ)
	case cminorsel.Eaddshift:
		return t.translateAddshift(expr, dest, succ)
	case cminorsel.Esubshift:
		return t.translateSubshift(expr, dest, succ)
	default:
		// Unknown expression - emit nop
		return t.ib.EmitNop(succ)
	}
}

// TranslateExprList translates a list of expressions left-to-right.
// Returns the list of result registers and the entry node.
func (t *ExprTranslator) TranslateExprList(exprs []cminorsel.Expr, succ rtl.Node) ([]rtl.Reg, rtl.Node) {
	if len(exprs) == 0 {
		return nil, succ
	}
	
	// Allocate registers for all results
	regs := make([]rtl.Reg, len(exprs))
	for i := range exprs {
		regs[i] = t.regs.Fresh()
	}
	
	// Translate right-to-left (backward chaining)
	entry := succ
	for i := len(exprs) - 1; i >= 0; i-- {
		entry = t.TranslateExpr(exprs[i], regs[i], entry)
	}
	
	return regs, entry
}

func (t *ExprTranslator) translateVar(e cminorsel.Evar, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Look up the variable's register
	src, ok := t.regs.LookupVar(e.Name)
	if !ok {
		// Variable not mapped - map it now
		src = t.regs.MapVar(e.Name)
	}
	
	if src == dest {
		// Same register - just continue
		return succ
	}
	
	// Move from src to dest
	return t.ib.EmitMove(src, dest, succ)
}

func (t *ExprTranslator) translateConst(e cminorsel.Econst, dest rtl.Reg, succ rtl.Node) rtl.Node {
	return t.ib.EmitConst(e.Const, dest, succ)
}

func (t *ExprTranslator) translateUnop(e cminorsel.Eunop, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Allocate temporary for argument
	argReg := t.regs.Fresh()
	
	// Emit the operation: dest = op(arg)
	op := TranslateUnaryOp(e.Op)
	opNode := t.ib.EmitOp(op, []rtl.Reg{argReg}, dest, succ)
	
	// Translate argument (chains to op)
	return t.TranslateExpr(e.Arg, argReg, opNode)
}

func (t *ExprTranslator) translateBinop(e cminorsel.Ebinop, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Allocate temporaries for arguments
	leftReg := t.regs.Fresh()
	rightReg := t.regs.Fresh()
	
	// Emit the operation: dest = op(left, right)
	op := TranslateBinaryOp(e.Op)
	opNode := t.ib.EmitOp(op, []rtl.Reg{leftReg, rightReg}, dest, succ)
	
	// Translate right operand first (chains to op)
	rightEntry := t.TranslateExpr(e.Right, rightReg, opNode)
	
	// Translate left operand (chains to right translation)
	return t.TranslateExpr(e.Left, leftReg, rightEntry)
}

func (t *ExprTranslator) translateLoad(e cminorsel.Eload, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Translate addressing mode arguments
	argRegs, entry := t.TranslateExprList(e.Args, succ)
	
	// Insert load instruction before the argument evaluation
	// Actually we need to emit load AFTER args are computed
	// So: args -> load -> succ
	
	// Re-think: we want args evaluated, then load executed, then succ
	// With backward chaining: we need load to chain to succ, args to chain to load
	
	// Emit load instruction
	addr := TranslateAddressingMode(e.Mode)
	chunk := TranslateChunk(e.Chunk)
	
	if len(e.Args) == 0 {
		// No address args - just emit load directly
		return t.ib.EmitLoad(chunk, addr, nil, dest, succ)
	}
	
	// We already translated args chaining to succ, but that's wrong.
	// We need: args -> load -> succ
	// Let's re-do this properly:
	
	// First, emit load -> succ
	loadNode := t.ib.EmitLoad(chunk, addr, argRegs, dest, succ)
	
	// Now translate args -> loadNode
	// But we already got argRegs... we need to redo TranslateExprList
	// Actually, the args were already translated incorrectly. Let's fix:
	
	// Ignore the entry we got (it was wrong)
	_ = entry
	
	// Re-translate args chaining to loadNode
	_, argsEntry := t.translateExprListToRegs(e.Args, argRegs, loadNode)
	return argsEntry
}

// translateExprListToRegs translates expressions into specific registers.
func (t *ExprTranslator) translateExprListToRegs(exprs []cminorsel.Expr, regs []rtl.Reg, succ rtl.Node) ([]rtl.Reg, rtl.Node) {
	if len(exprs) == 0 {
		return regs, succ
	}
	
	entry := succ
	for i := len(exprs) - 1; i >= 0; i-- {
		entry = t.TranslateExpr(exprs[i], regs[i], entry)
	}
	return regs, entry
}

func (t *ExprTranslator) translateCondition(e cminorsel.Econdition, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Conditional expression: if cond then then_expr else else_expr
	// We need:
	//   cond -> true: eval_then -> join
	//        -> false: eval_else -> join
	//   join: succ
	
	// The join point stores result in dest and goes to succ
	// Both branches evaluate to dest
	
	// Translate else branch -> succ
	elseEntry := t.TranslateExpr(e.Else, dest, succ)
	
	// Translate then branch -> succ
	thenEntry := t.TranslateExpr(e.Then, dest, succ)
	
	// Translate condition -> branches
	return t.TranslateCond(e.Cond, thenEntry, elseEntry)
}

// TranslateCond translates a condition for branching.
// Returns the entry node.
func (t *ExprTranslator) TranslateCond(cond cminorsel.Condition, ifso, ifnot rtl.Node) rtl.Node {
	cc, args := TranslateCondition(cond)
	
	if cc == nil {
		// Compound condition (CondAnd, CondOr) - handle specially
		return t.translateCompoundCond(cond, ifso, ifnot)
	}
	
	// Translate condition arguments
	argRegs, entry := t.TranslateExprList(args, t.ib.AllocNode())
	
	// Emit the conditional branch
	condNode := t.ib.EmitCond(cc, argRegs, ifso, ifnot)
	
	// Link args to cond
	if entry != t.ib.cfg.nextNode-1 {
		// Args were translated, need to fix the chain
		// Find the last instruction in args and point it to condNode
		// Actually with our approach, entry should chain correctly
		// Let's re-translate
	}
	
	// Re-do: translate args -> condNode
	_, argsEntry := t.translateExprListToRegs(args, argRegs, condNode)
	return argsEntry
}

func (t *ExprTranslator) translateCompoundCond(cond cminorsel.Condition, ifso, ifnot rtl.Node) rtl.Node {
	switch c := cond.(type) {
	case cminorsel.CondAnd:
		// c1 && c2: if c1 then (if c2 then ifso else ifnot) else ifnot
		inner := t.TranslateCond(c.Right, ifso, ifnot)
		return t.TranslateCond(c.Left, inner, ifnot)
		
	case cminorsel.CondOr:
		// c1 || c2: if c1 then ifso else (if c2 then ifso else ifnot)
		inner := t.TranslateCond(c.Right, ifso, ifnot)
		return t.TranslateCond(c.Left, ifso, inner)
		
	default:
		// Fallback - evaluate to register and compare to 0
		// This shouldn't happen for CondAnd/CondOr
		return ifso
	}
}

func (t *ExprTranslator) translateLet(e cminorsel.Elet, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Let binding: evaluate Bind, then evaluate Body with binding available
	// The bound value is accessed via Eletvar with index
	
	// Create a temporary for the bound value
	boundReg := t.regs.Fresh()
	
	// Push the bound register onto a let stack (we'd need to track this)
	// For simplicity, we'll use the letvar index to look up
	t.pushLetBinding(boundReg)
	defer t.popLetBinding()
	
	// Translate body -> succ
	bodyEntry := t.TranslateExpr(e.Body, dest, succ)
	
	// Translate bind -> body
	return t.TranslateExpr(e.Bind, boundReg, bodyEntry)
}

func (t *ExprTranslator) translateLetvar(e cminorsel.Eletvar, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Reference to let-bound variable by index
	src := t.getLetBinding(e.Index)
	if src == 0 {
		// No binding found - should not happen
		return t.ib.EmitNop(succ)
	}
	
	if src == dest {
		return succ
	}
	return t.ib.EmitMove(src, dest, succ)
}

// Let binding stack
var letBindings []rtl.Reg

func (t *ExprTranslator) pushLetBinding(r rtl.Reg) {
	letBindings = append(letBindings, r)
}

func (t *ExprTranslator) popLetBinding() {
	if len(letBindings) > 0 {
		letBindings = letBindings[:len(letBindings)-1]
	}
}

func (t *ExprTranslator) getLetBinding(index int) rtl.Reg {
	// Index 0 = innermost binding
	idx := len(letBindings) - 1 - index
	if idx < 0 || idx >= len(letBindings) {
		return 0
	}
	return letBindings[idx]
}

func (t *ExprTranslator) translateAddshift(e cminorsel.Eaddshift, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Add with shifted operand: left + (right << shift)
	leftReg := t.regs.Fresh()
	rightReg := t.regs.Fresh()
	
	// We could emit a combined add-shift operation if target supports it
	// For now, emit: tmp = right << shift; dest = left + tmp
	
	// Shift right operand
	shiftOp := translateShiftOp(e.Op, e.Shift)
	shiftedReg := t.regs.Fresh()
	
	// dest = left + shifted
	addNode := t.ib.EmitOp(rtl.Oadd{}, []rtl.Reg{leftReg, shiftedReg}, dest, succ)
	
	// shifted = right << amount
	shiftNode := t.ib.EmitOp(shiftOp, []rtl.Reg{rightReg}, shiftedReg, addNode)
	
	// Translate right -> shift
	rightEntry := t.TranslateExpr(e.Right, rightReg, shiftNode)
	
	// Translate left -> right
	return t.TranslateExpr(e.Left, leftReg, rightEntry)
}

func (t *ExprTranslator) translateSubshift(e cminorsel.Esubshift, dest rtl.Reg, succ rtl.Node) rtl.Node {
	// Sub with shifted operand: left - (right << shift)
	leftReg := t.regs.Fresh()
	rightReg := t.regs.Fresh()
	
	shiftOp := translateShiftOp(e.Op, e.Shift)
	shiftedReg := t.regs.Fresh()
	
	// dest = left - shifted
	subNode := t.ib.EmitOp(rtl.Osub{}, []rtl.Reg{leftReg, shiftedReg}, dest, succ)
	
	// shifted = right << amount
	shiftNode := t.ib.EmitOp(shiftOp, []rtl.Reg{rightReg}, shiftedReg, subNode)
	
	// Translate right -> shift
	rightEntry := t.TranslateExpr(e.Right, rightReg, shiftNode)
	
	// Translate left -> right
	return t.TranslateExpr(e.Left, leftReg, rightEntry)
}

func translateShiftOp(op cminorsel.ShiftOp, amount int) rtl.Operation {
	switch op {
	case cminorsel.Slsl:
		return rtl.Oshlimm{N: int32(amount)}
	case cminorsel.Slsr:
		return rtl.Oshruimm{N: int32(amount)}
	case cminorsel.Sasr:
		return rtl.Oshrimm{N: int32(amount)}
	default:
		return rtl.Oshlimm{N: int32(amount)}
	}
}

// ResetLetBindings clears the let binding stack.
func ResetLetBindings() {
	letBindings = nil
}
