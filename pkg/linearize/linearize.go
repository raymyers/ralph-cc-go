// Package linearize transforms LTL (CFG-based) to Linear (sequential code).
// This involves ordering basic blocks and inserting explicit labels and branches.
// Also includes branch tunneling and label cleanup optimizations.
package linearize

import (
	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
)

// Linearize transforms an LTL function to Linear code.
// It orders blocks via reverse postorder and adds explicit branches where needed.
func Linearize(fn *ltl.Function) *linear.Function {
	l := &linearizer{
		fn:        fn,
		nodeToLbl: make(map[ltl.Node]linear.Label),
	}
	return l.linearize()
}

// linearizer holds state during linearization
type linearizer struct {
	fn        *ltl.Function
	order     []ltl.Node             // block ordering (reverse postorder)
	nodeToLbl map[ltl.Node]linear.Label // maps CFG nodes to Linear labels
	nextLabel linear.Label
}

// linearize performs the transformation
func (l *linearizer) linearize() *linear.Function {
	result := linear.NewFunction(l.fn.Name, l.fn.Sig)
	result.Stacksize = l.fn.Stacksize

	if len(l.fn.Code) == 0 {
		return result
	}

	// Step 1: Compute block ordering (reverse postorder for natural fall-through)
	l.computeOrder()

	// Step 2: Assign labels to each block
	l.assignLabels()

	// Step 3: Emit linearized code for each block in order
	for i, node := range l.order {
		l.emitBlock(result, node, i)
	}

	return result
}

// computeOrder computes reverse postorder traversal of the CFG
func (l *linearizer) computeOrder() {
	visited := make(map[ltl.Node]bool)
	var postorder []ltl.Node

	var dfs func(n ltl.Node)
	dfs = func(n ltl.Node) {
		if visited[n] {
			return
		}
		visited[n] = true

		block := l.fn.Code[n]
		if block == nil {
			return
		}

		// Visit successors first
		for _, succ := range l.blockSuccessors(block) {
			dfs(succ)
		}

		// Add to postorder after visiting successors
		postorder = append(postorder, n)
	}

	// Start from entry point
	dfs(l.fn.Entrypoint)

	// Also visit any unreachable blocks (shouldn't happen in well-formed code)
	for n := range l.fn.Code {
		if !visited[n] {
			dfs(n)
		}
	}

	// Reverse postorder
	l.order = make([]ltl.Node, len(postorder))
	for i, n := range postorder {
		l.order[len(postorder)-1-i] = n
	}
}

// blockSuccessors returns the successor nodes of a basic block
func (l *linearizer) blockSuccessors(block *ltl.BBlock) []ltl.Node {
	if len(block.Body) == 0 {
		return nil
	}

	// The terminator is the last instruction
	term := block.Body[len(block.Body)-1]

	switch t := term.(type) {
	case ltl.Lbranch:
		return []ltl.Node{t.Succ}
	case ltl.Lcond:
		return []ltl.Node{t.IfSo, t.IfNot}
	case ltl.Ljumptable:
		nodes := make([]ltl.Node, len(t.Targets))
		for i, target := range t.Targets {
			nodes[i] = target
		}
		return nodes
	case ltl.Lreturn, ltl.Ltailcall:
		return nil
	default:
		return nil
	}
}

// assignLabels assigns a Linear label to each CFG node
func (l *linearizer) assignLabels() {
	l.nextLabel = 1
	for _, n := range l.order {
		l.nodeToLbl[n] = l.nextLabel
		l.nextLabel++
	}
}

// emitBlock emits linearized code for a single block
func (l *linearizer) emitBlock(result *linear.Function, node ltl.Node, orderIdx int) {
	block := l.fn.Code[node]
	if block == nil {
		return
	}

	lbl := l.nodeToLbl[node]

	// Emit label
	result.Append(linear.Llabel{Lbl: lbl})

	// Emit body instructions (except terminator)
	for i := 0; i < len(block.Body)-1; i++ {
		result.Append(l.convertInstruction(block.Body[i]))
	}

	// Handle terminator specially for fall-through optimization
	if len(block.Body) > 0 {
		term := block.Body[len(block.Body)-1]
		l.emitTerminator(result, term, orderIdx)
	}
}

// convertInstruction converts an LTL instruction to Linear
func (l *linearizer) convertInstruction(inst ltl.Instruction) linear.Instruction {
	switch i := inst.(type) {
	case ltl.Lnop:
		// Skip nops in linearized code
		return nil
	case ltl.Lop:
		return linear.Lop{Op: i.Op, Args: i.Args, Dest: i.Dest}
	case ltl.Lload:
		return linear.Lload{Chunk: i.Chunk, Addr: i.Addr, Args: i.Args, Dest: i.Dest}
	case ltl.Lstore:
		return linear.Lstore{Chunk: i.Chunk, Addr: i.Addr, Args: i.Args, Src: i.Src}
	case ltl.Lcall:
		return linear.Lcall{Sig: i.Sig, Fn: l.convertFunRef(i.Fn)}
	case ltl.Lbuiltin:
		return linear.Lbuiltin{Builtin: i.Builtin, Args: i.Args, Dest: i.Dest}
	default:
		// Terminal instructions are handled separately
		return nil
	}
}

// convertFunRef converts an LTL function reference to Linear
func (l *linearizer) convertFunRef(fn ltl.FunRef) linear.FunRef {
	switch f := fn.(type) {
	case ltl.FunReg:
		return linear.FunReg{Loc: f.Loc}
	case ltl.FunSymbol:
		return linear.FunSymbol{Name: f.Name}
	default:
		return nil
	}
}

// emitTerminator emits code for a block terminator, optimizing fall-through
func (l *linearizer) emitTerminator(result *linear.Function, term ltl.Instruction, orderIdx int) {
	// Determine the next block in linear order (if any)
	var nextNode *ltl.Node
	if orderIdx+1 < len(l.order) {
		n := l.order[orderIdx+1]
		nextNode = &n
	}

	switch t := term.(type) {
	case ltl.Lbranch:
		// If target is the next block, we can fall through
		if nextNode != nil && t.Succ == *nextNode {
			// Omit goto, fall through
			return
		}
		result.Append(linear.Lgoto{Target: l.nodeToLbl[t.Succ]})

	case ltl.Lcond:
		ifSoLbl := l.nodeToLbl[t.IfSo]
		ifNotLbl := l.nodeToLbl[t.IfNot]

		// Check if either branch can fall through
		if nextNode != nil && t.IfNot == *nextNode {
			// "if not" falls through - emit conditional for "if so" only
			result.Append(linear.Lcond{Cond: t.Cond, Args: t.Args, IfSo: ifSoLbl})
		} else if nextNode != nil && t.IfSo == *nextNode {
			// "if so" falls through - need to negate condition and emit for "if not"
			// For now, emit both branches (can add condition negation later)
			result.Append(linear.Lcond{Cond: t.Cond, Args: t.Args, IfSo: ifSoLbl})
			result.Append(linear.Lgoto{Target: ifNotLbl})
		} else {
			// Neither falls through - emit conditional and goto
			result.Append(linear.Lcond{Cond: t.Cond, Args: t.Args, IfSo: ifSoLbl})
			result.Append(linear.Lgoto{Target: ifNotLbl})
		}

	case ltl.Ljumptable:
		targets := make([]linear.Label, len(t.Targets))
		for i, target := range t.Targets {
			targets[i] = l.nodeToLbl[target]
		}
		result.Append(linear.Ljumptable{Arg: t.Arg, Targets: targets})

	case ltl.Ltailcall:
		result.Append(linear.Ltailcall{Sig: t.Sig, Fn: l.convertFunRef(t.Fn)})

	case ltl.Lreturn:
		result.Append(linear.Lreturn{})
	}
}
