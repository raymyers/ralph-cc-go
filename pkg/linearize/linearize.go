// Package linearize transforms LTL (CFG-based) to Linear (sequential code).
// This involves ordering basic blocks and inserting explicit labels and branches.
// Also includes branch tunneling and label cleanup optimizations.
package linearize

import (
	"runtime"

	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// knownVariadicFuncs lists known variadic C library functions.
// On macOS ARM64, variadic arguments must be passed on the stack.
var knownVariadicFuncs = map[string]int{
	// printf family - first arg is format string in x0, rest on stack
	"printf":   1, // 1 fixed arg (format)
	"fprintf":  2, // 2 fixed args (file, format)
	"sprintf":  2, // 2 fixed args (buf, format)
	"snprintf": 3, // 3 fixed args (buf, size, format)
	"dprintf":  2, // 2 fixed args (fd, format)

	// scanf family
	"scanf":  1,
	"fscanf": 2,
	"sscanf": 2,

	// Other variadic functions
	"execl":   1,
	"execlp":  1,
	"execle":  1,
	"fcntl":   2,
	"ioctl":   2,
	"open":    2,
	"openat":  3,
	"semctl":  3,
	"syslog":  2,
}

// TransformProgram transforms an entire LTL program to Linear
func TransformProgram(prog *ltl.Program) *linear.Program {
	linearProg := &linear.Program{
		Globals: make([]linear.GlobVar, len(prog.Globals)),
	}

	// Copy globals
	for i, g := range prog.Globals {
		linearProg.Globals[i] = linear.GlobVar{
			Name:     g.Name,
			Size:     g.Size,
			Init:     g.Init,
			ReadOnly: g.ReadOnly,
		}
	}

	// Transform each function
	for _, fn := range prog.Functions {
		linearFn := Transform(&fn)
		linearProg.Functions = append(linearProg.Functions, *linearFn)
	}

	return linearProg
}

// Transform applies all linearization passes to a single function:
// 1. Linearize (CFG to sequential)
// 2. Tunnel (shortcut jump chains)
// 3. CleanupLabels (remove unused labels)
// 4. ComputeStackSize
func Transform(fn *ltl.Function) *linear.Function {
	result := Linearize(fn)
	Tunnel(result)
	CleanupLabels(result)
	ComputeStackSize(result)
	return result
}

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
	result.Params = l.fn.Params // Propagate parameter locations

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
// Only includes blocks reachable from the entry point.
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

	// Start from entry point - only include reachable blocks
	dfs(l.fn.Entrypoint)

	// NOTE: We intentionally do NOT include unreachable blocks.
	// Unreachable code (e.g., orphan exit nodes from RTL generation)
	// should not be emitted to the final output.

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
		for _, inst := range l.convertInstruction(block.Body[i]) {
			result.Append(inst)
		}
	}

	// Handle terminator specially for fall-through optimization
	if len(block.Body) > 0 {
		term := block.Body[len(block.Body)-1]
		l.emitTerminator(result, term, orderIdx)
	}
}

// convertInstruction converts an LTL instruction to Linear
// Returns a slice since some instructions expand to multiple (e.g., call with arg moves)
func (l *linearizer) convertInstruction(inst ltl.Instruction) []linear.Instruction {
	switch i := inst.(type) {
	case ltl.Lnop:
		// Skip nops in linearized code
		return nil
	case ltl.Lop:
		return []linear.Instruction{linear.Lop{Op: i.Op, Args: i.Args, Dest: i.Dest}}
	case ltl.Lload:
		return []linear.Instruction{linear.Lload{Chunk: i.Chunk, Addr: i.Addr, Args: i.Args, Dest: i.Dest}}
	case ltl.Lstore:
		return []linear.Instruction{linear.Lstore{Chunk: i.Chunk, Addr: i.Addr, Args: i.Args, Src: i.Src}}
	case ltl.Lcall:
		// Generate moves to place arguments in X0, X1, X2, ... before the call
		return l.convertCall(i)
	case ltl.Lbuiltin:
		return []linear.Instruction{linear.Lbuiltin{Builtin: i.Builtin, Args: i.Args, Dest: i.Dest}}
	default:
		// Terminal instructions are handled separately
		return nil
	}
}

// IntArgRegs are the argument registers for ARM64 calling convention
var intArgRegs = []ltl.MReg{ltl.X0, ltl.X1, ltl.X2, ltl.X3, ltl.X4, ltl.X5, ltl.X6, ltl.X7}

// tempReg is a scratch register used for parallel moves
// X8 is caller-saved and not used for arguments
const tempReg = ltl.X8

// isVariadicCall checks if a function call is to a known variadic function
// and returns the number of fixed arguments (0 if not variadic)
func isVariadicCall(fn ltl.FunRef) (bool, int) {
	switch f := fn.(type) {
	case ltl.FunSymbol:
		if fixedArgs, known := knownVariadicFuncs[f.Name]; known {
			return true, fixedArgs
		}
	}
	return false, 0
}

// convertCall generates move instructions to place arguments in the correct
// registers (X0, X1, ...) before emitting the call instruction.
// This implements the ARM64 calling convention for function arguments.
// Handles the parallel move problem by using a temp register for cycles.
//
// On macOS ARM64, variadic arguments must be passed on the stack, not in registers.
// We detect known variadic functions and generate stack stores for their varargs.
func (l *linearizer) convertCall(call ltl.Lcall) []linear.Instruction {
	// Check if this is a variadic call on macOS
	isVariadic, fixedArgs := isVariadicCall(call.Fn)
	useDarwinVariadicConvention := isVariadic && runtime.GOOS == "darwin"

	// Build a mapping from target register to source location
	// moves[dest] = src means we need to do: dest = src
	moves := make(map[ltl.MReg]ltl.Loc)

	// For variadic on macOS: only fixed args go in registers
	maxRegArgs := len(intArgRegs)
	if useDarwinVariadicConvention && fixedArgs < maxRegArgs {
		maxRegArgs = fixedArgs
	}

	for i, argLoc := range call.Args {
		if i >= maxRegArgs {
			// Arguments beyond register limit go on the stack
			break
		}
		targetReg := intArgRegs[i]

		// If the argument is already in the right register, no move needed
		if regLoc, ok := argLoc.(ltl.R); ok && regLoc.Reg == targetReg {
			continue
		}

		moves[targetReg] = argLoc
	}

	// Generate moves using a simple algorithm that handles cycles
	var result []linear.Instruction

	// Store overflow arguments on the stack (for both variadic and non-variadic calls)
	overflowStart := maxRegArgs
	if useDarwinVariadicConvention {
		overflowStart = fixedArgs
	}
	if len(call.Args) > overflowStart {
		// Store each overflow argument on the stack at [SP + offset]
		// Each argument takes 8 bytes (padded)
		for i := overflowStart; i < len(call.Args); i++ {
			stackOfs := int64((i - overflowStart) * 8)
			argLoc := call.Args[i]

			// Lsetstack requires an MReg source, so ensure arg is in a register
			var srcReg ltl.MReg
			if regLoc, ok := argLoc.(ltl.R); ok {
				srcReg = regLoc.Reg
			} else {
				// Move to temp register first
				srcReg = tempReg
				result = append(result, linear.Lop{
					Op:   rtl.Omove{},
					Args: []ltl.Loc{argLoc},
					Dest: ltl.R{Reg: tempReg},
				})
			}
			result = append(result, linear.Lsetstack{
				Src:  srcReg,
				Slot: ltl.SlotOutgoing,
				Ofs:  stackOfs,
				Ty:   ltl.Tlong, // All args padded to 8 bytes
			})
		}
	}

	done := make(map[ltl.MReg]bool)

	// isSourceOfPendingMove returns true if reg is the source of a move that hasn't been done
	isSourceOfPendingMove := func(reg ltl.MReg) bool {
		for dest, src := range moves {
			if done[dest] {
				continue
			}
			if srcReg, ok := src.(ltl.R); ok && srcReg.Reg == reg {
				return true
			}
		}
		return false
	}

	for len(done) < len(moves) {
		madeProgress := false

		// Try to find a move where the destination is not needed as a source
		for dest, src := range moves {
			if done[dest] {
				continue
			}

			// Check if this destination register is used as source by another pending move
			if isSourceOfPendingMove(dest) {
				continue
			}

			// Safe to do this move
			result = append(result, linear.Lop{
				Op:   rtl.Omove{},
				Args: []ltl.Loc{src},
				Dest: ltl.R{Reg: dest},
			})
			done[dest] = true
			madeProgress = true
		}

		// If no progress, we have a cycle - break it with temp register
		if !madeProgress {
			// Find any pending move and save its source to temp
			for dest, src := range moves {
				if done[dest] {
					continue
				}

				srcReg, ok := src.(ltl.R)
				if !ok {
					// Non-register source - just do the move
					result = append(result, linear.Lop{
						Op:   rtl.Omove{},
						Args: []ltl.Loc{src},
						Dest: ltl.R{Reg: dest},
					})
					done[dest] = true
					madeProgress = true
					break
				}

				// Save the source value in temp
				result = append(result, linear.Lop{
					Op:   rtl.Omove{},
					Args: []ltl.Loc{ltl.R{Reg: srcReg.Reg}},
					Dest: ltl.R{Reg: tempReg},
				})

				// Update moves map: any move that uses srcReg as source now uses temp
				for dest2, src2 := range moves {
					if srcReg2, ok := src2.(ltl.R); ok && srcReg2.Reg == srcReg.Reg {
						moves[dest2] = ltl.R{Reg: tempReg}
					}
				}
				break
			}
		}
	}

	// Add the call instruction
	result = append(result, linear.Lcall{Sig: call.Sig, Fn: l.convertFunRef(call.Fn)})

	return result
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
