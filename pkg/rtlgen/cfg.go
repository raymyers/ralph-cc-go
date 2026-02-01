// Package rtlgen transforms CminorSel to RTL.
// RTLgen builds a control flow graph with infinite pseudo-registers and 3-address code.
// This mirrors CompCert's backend/RTLgen.v
package rtlgen

import (
	"github.com/raymyers/ralph-cc/pkg/cminorsel"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// CFGBuilder constructs an RTL control flow graph.
// It maintains state for node allocation and maps labels to nodes.
type CFGBuilder struct {
	nextNode   rtl.Node                   // next available node ID
	code       map[rtl.Node]rtl.Instruction // CFG: node -> instruction
	labelNodes map[string]rtl.Node        // label -> node mapping
	varToReg   map[string]rtl.Reg         // variable -> register mapping
	nextReg    rtl.Reg                    // next available register
	stackVars  map[string]int64           // stack-allocated variable offsets
	stackSize  int64                      // total stack size
}

// NewCFGBuilder creates a new CFG builder.
func NewCFGBuilder() *CFGBuilder {
	return &CFGBuilder{
		nextNode:   1, // Node IDs start at 1 (positive integers)
		code:       make(map[rtl.Node]rtl.Instruction),
		labelNodes: make(map[string]rtl.Node),
		varToReg:   make(map[string]rtl.Reg),
		nextReg:    1, // Register IDs start at 1
		stackVars:  make(map[string]int64),
	}
}

// AllocNode allocates a fresh node ID.
func (b *CFGBuilder) AllocNode() rtl.Node {
	n := b.nextNode
	b.nextNode++
	return n
}

// AddInstr adds an instruction at the given node.
func (b *CFGBuilder) AddInstr(node rtl.Node, instr rtl.Instruction) {
	b.code[node] = instr
}

// EmitInstr allocates a node and adds an instruction to it.
// Returns the node ID.
func (b *CFGBuilder) EmitInstr(instr rtl.Instruction) rtl.Node {
	n := b.AllocNode()
	b.AddInstr(n, instr)
	return n
}

// GetCode returns the completed CFG.
func (b *CFGBuilder) GetCode() map[rtl.Node]rtl.Instruction {
	return b.code
}

// AllocReg allocates a fresh pseudo-register.
func (b *CFGBuilder) AllocReg() rtl.Reg {
	r := b.nextReg
	b.nextReg++
	return r
}

// AllocRegs allocates n fresh pseudo-registers.
func (b *CFGBuilder) AllocRegs(n int) []rtl.Reg {
	regs := make([]rtl.Reg, n)
	for i := 0; i < n; i++ {
		regs[i] = b.AllocReg()
	}
	return regs
}

// MapVar maps a variable name to a register.
// If already mapped, returns the existing register.
func (b *CFGBuilder) MapVar(name string) rtl.Reg {
	if r, ok := b.varToReg[name]; ok {
		return r
	}
	r := b.AllocReg()
	b.varToReg[name] = r
	return r
}

// GetVarReg returns the register for a variable, or 0 if not mapped.
func (b *CFGBuilder) GetVarReg(name string) (rtl.Reg, bool) {
	r, ok := b.varToReg[name]
	return r, ok
}

// SetStackVar records a stack-allocated variable's offset.
func (b *CFGBuilder) SetStackVar(name string, offset int64) {
	b.stackVars[name] = offset
}

// GetStackVar returns the offset for a stack-allocated variable.
func (b *CFGBuilder) GetStackVar(name string) (int64, bool) {
	offset, ok := b.stackVars[name]
	return offset, ok
}

// SetStackSize sets the total stack frame size.
func (b *CFGBuilder) SetStackSize(size int64) {
	b.stackSize = size
}

// GetStackSize returns the total stack frame size.
func (b *CFGBuilder) GetStackSize() int64 {
	return b.stackSize
}

// GetOrCreateLabel returns the node for a label, creating it if needed.
func (b *CFGBuilder) GetOrCreateLabel(label string) rtl.Node {
	if n, ok := b.labelNodes[label]; ok {
		return n
	}
	n := b.AllocNode()
	b.labelNodes[label] = n
	return n
}

// GetLabel returns the node for a label if it exists.
func (b *CFGBuilder) GetLabel(label string) (rtl.Node, bool) {
	n, ok := b.labelNodes[label]
	return n, ok
}

// ExitContext tracks exit targets for Sblock/Sexit.
// Sexit(n) exits n+1 nested blocks.
type ExitContext struct {
	targets []rtl.Node // stack of exit targets (innermost first)
}

// NewExitContext creates a new exit context.
func NewExitContext() *ExitContext {
	return &ExitContext{
		targets: nil,
	}
}

// Push adds an exit target for a new block.
func (e *ExitContext) Push(target rtl.Node) {
	e.targets = append(e.targets, target)
}

// Pop removes the innermost exit target.
func (e *ExitContext) Pop() {
	if len(e.targets) > 0 {
		e.targets = e.targets[:len(e.targets)-1]
	}
}

// Get returns the exit target for Sexit(n).
// Returns (0, false) if n is out of range.
func (e *ExitContext) Get(n int) (rtl.Node, bool) {
	// Sexit(0) = innermost block, Sexit(1) = next outer, etc.
	idx := len(e.targets) - 1 - n
	if idx < 0 || idx >= len(e.targets) {
		return 0, false
	}
	return e.targets[idx], true
}

// Depth returns the current nesting depth.
func (e *ExitContext) Depth() int {
	return len(e.targets)
}

// TranslateCondition converts a CminorSel condition to RTL condition code.
// Returns the condition code and the argument expressions to evaluate.
func TranslateCondition(cond cminorsel.Condition) (rtl.ConditionCode, []cminorsel.Expr) {
	switch c := cond.(type) {
	case cminorsel.CondTrue:
		// Always true: use comparison 0 != 0 would be wrong, 
		// use 1 != 0 (or just return a special constant comparison)
		return rtl.Ccompimm{Cond: rtl.Cne, N: 0}, []cminorsel.Expr{
			cminorsel.Econst{Const: cminorsel.Ointconst{Value: 1}},
		}
	case cminorsel.CondFalse:
		// Always false: 0 != 0 is false
		return rtl.Ccompimm{Cond: rtl.Cne, N: 0}, []cminorsel.Expr{
			cminorsel.Econst{Const: cminorsel.Ointconst{Value: 0}},
		}
	case cminorsel.CondCmp:
		return rtl.Ccomp{Cond: translateComparison(c.Cmp)}, []cminorsel.Expr{c.Left, c.Right}
	case cminorsel.CondCmpu:
		return rtl.Ccompu{Cond: translateComparison(c.Cmp)}, []cminorsel.Expr{c.Left, c.Right}
	case cminorsel.CondCmpf:
		return rtl.Ccompf{Cond: translateComparison(c.Cmp)}, []cminorsel.Expr{c.Left, c.Right}
	case cminorsel.CondCmps:
		return rtl.Ccomps{Cond: translateComparison(c.Cmp)}, []cminorsel.Expr{c.Left, c.Right}
	case cminorsel.CondCmpl:
		return rtl.Ccompl{Cond: translateComparison(c.Cmp)}, []cminorsel.Expr{c.Left, c.Right}
	case cminorsel.CondCmplu:
		return rtl.Ccomplu{Cond: translateComparison(c.Cmp)}, []cminorsel.Expr{c.Left, c.Right}
	case cminorsel.CondNot:
		// Negate the inner condition
		inner, args := TranslateCondition(c.Cond)
		return negateConditionCode(inner), args
	default:
		// For compound conditions (CondAnd, CondOr), we need special handling
		// in the statement translation to build proper control flow.
		// Return a placeholder that will be handled specially.
		return nil, nil
	}
}

// translateComparison converts CminorSel comparison to RTL condition.
func translateComparison(cmp cminorsel.Comparison) rtl.Condition {
	switch cmp {
	case cminorsel.Ceq:
		return rtl.Ceq
	case cminorsel.Cne:
		return rtl.Cne
	case cminorsel.Clt:
		return rtl.Clt
	case cminorsel.Cle:
		return rtl.Cle
	case cminorsel.Cgt:
		return rtl.Cgt
	case cminorsel.Cge:
		return rtl.Cge
	default:
		return rtl.Ceq
	}
}

// negateConditionCode returns the negated condition code.
func negateConditionCode(cc rtl.ConditionCode) rtl.ConditionCode {
	switch c := cc.(type) {
	case rtl.Ccomp:
		return rtl.Ccomp{Cond: c.Cond.Negate()}
	case rtl.Ccompu:
		return rtl.Ccompu{Cond: c.Cond.Negate()}
	case rtl.Ccompimm:
		return rtl.Ccompimm{Cond: c.Cond.Negate(), N: c.N}
	case rtl.Ccompuimm:
		return rtl.Ccompuimm{Cond: c.Cond.Negate(), N: c.N}
	case rtl.Ccompl:
		return rtl.Ccompl{Cond: c.Cond.Negate()}
	case rtl.Ccomplu:
		return rtl.Ccomplu{Cond: c.Cond.Negate()}
	case rtl.Ccomplimm:
		return rtl.Ccomplimm{Cond: c.Cond.Negate(), N: c.N}
	case rtl.Ccompluimm:
		return rtl.Ccompluimm{Cond: c.Cond.Negate(), N: c.N}
	case rtl.Ccompf:
		return rtl.Cnotcompf{Cond: c.Cond}
	case rtl.Cnotcompf:
		return rtl.Ccompf{Cond: c.Cond}
	case rtl.Ccomps:
		return rtl.Cnotcomps{Cond: c.Cond}
	case rtl.Cnotcomps:
		return rtl.Ccomps{Cond: c.Cond}
	default:
		return cc
	}
}
