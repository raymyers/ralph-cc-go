package regalloc

import (
	"sort"

	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// TransformFunction transforms an RTL function to LTL by applying register allocation
func TransformFunction(rtlFn *rtl.Function) *ltl.Function {
	// Perform register allocation
	allocation := AllocateFunction(rtlFn)

	ltlFn := ltl.NewFunction(rtlFn.Name, rtlFn.Sig)
	ltlFn.Stacksize = rtlFn.Stacksize + allocation.StackSize

	// Build parameter entry locations (X0-X7 for first 8 args)
	// These are the locations where arguments arrive
	for i := range rtlFn.Params {
		ltlFn.Params = append(ltlFn.Params, ArgLocation(i, false))
	}

	// Group instructions into basic blocks
	// For simplicity, we create one block per RTL node initially
	sortedNodes := getSortedNodes(rtlFn)
	for _, node := range sortedNodes {
		instr := rtlFn.Code[node]
		ltlBlock := transformInstruction(instr, allocation)
		ltlFn.Code[ltl.Node(node)] = ltlBlock
	}

	// At function entry, we need to copy parameters from their argument
	// registers (X0-X7) to their allocated locations.
	// This is necessary because the register allocator may assign other
	// variables to X0-X7, which could clobber parameters before they're used.
	entryBlock := ltlFn.Code[ltl.Node(rtlFn.Entrypoint)]
	if entryBlock != nil {
		var paramMoves []ltl.Instruction

		// Generate moves from argument registers to allocated locations
		argLocs := make([]ltl.Loc, len(rtlFn.Params))
		allocLocs := make([]ltl.Loc, len(rtlFn.Params))
		for i, param := range rtlFn.Params {
			argLocs[i] = ArgLocation(i, false)
			allocLocs[i] = allocation.RegToLoc[param]
		}
		paramMoves = resolveParallelMoves(argLocs, allocLocs)

		// Prepend parameter moves to the entry block
		if len(paramMoves) > 0 {
			newBody := make([]ltl.Instruction, 0, len(paramMoves)+len(entryBlock.Body))
			newBody = append(newBody, paramMoves...)
			newBody = append(newBody, entryBlock.Body...)
			entryBlock.Body = newBody
		}
	}

	ltlFn.Entrypoint = ltl.Node(rtlFn.Entrypoint)
	return ltlFn
}

func getSortedNodes(fn *rtl.Function) []rtl.Node {
	nodes := make([]rtl.Node, 0, len(fn.Code))
	for n := range fn.Code {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i] < nodes[j]
	})
	return nodes
}

func transformInstruction(instr rtl.Instruction, alloc *AllocationResult) *ltl.BBlock {
	switch i := instr.(type) {
	case rtl.Inop:
		return &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Lbranch{Succ: ltl.Node(i.Succ)},
			},
		}

	case rtl.Iop:
		args := transformRegs(i.Args, alloc)
		dest := alloc.RegToLoc[i.Dest]
		return &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Lop{Op: i.Op, Args: args, Dest: dest},
				ltl.Lbranch{Succ: ltl.Node(i.Succ)},
			},
		}

	case rtl.Iload:
		args := transformRegs(i.Args, alloc)
		dest := alloc.RegToLoc[i.Dest]
		return &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Lload{Chunk: i.Chunk, Addr: i.Addr, Args: args, Dest: dest},
				ltl.Lbranch{Succ: ltl.Node(i.Succ)},
			},
		}

	case rtl.Istore:
		args := transformRegs(i.Args, alloc)
		src := alloc.RegToLoc[i.Src]
		return &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Lstore{Chunk: i.Chunk, Addr: i.Addr, Args: args, Src: src},
				ltl.Lbranch{Succ: ltl.Node(i.Succ)},
			},
		}

	case rtl.Icall:
		args := transformRegs(i.Args, alloc)
		var fn ltl.FunRef
		switch f := i.Fn.(type) {
		case rtl.FunSymbol:
			fn = ltl.FunSymbol{Name: f.Name}
		case rtl.FunReg:
			fn = ltl.FunReg{Loc: alloc.RegToLoc[f.Reg]}
		}
		body := []ltl.Instruction{
			ltl.Lcall{Sig: i.Sig, Fn: fn, Args: args},
		}
		// If call has a destination, move the return value (X0) to it
		if i.Dest != 0 {
			destLoc := alloc.RegToLoc[i.Dest]
			retLoc := ReturnLocation(false) // TODO: handle float returns
			// Only add move if destination is not already X0
			if destLoc != retLoc {
				body = append(body, ltl.Lop{
					Op:   rtl.Omove{},
					Args: []ltl.Loc{retLoc},
					Dest: destLoc,
				})
			}
		}
		body = append(body, ltl.Lbranch{Succ: ltl.Node(i.Succ)})
		return &ltl.BBlock{Body: body}

	case rtl.Itailcall:
		args := transformRegs(i.Args, alloc)
		var fn ltl.FunRef
		switch f := i.Fn.(type) {
		case rtl.FunSymbol:
			fn = ltl.FunSymbol{Name: f.Name}
		case rtl.FunReg:
			fn = ltl.FunReg{Loc: alloc.RegToLoc[f.Reg]}
		}
		return &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Ltailcall{Sig: i.Sig, Fn: fn, Args: args},
			},
		}

	case rtl.Ibuiltin:
		args := transformRegs(i.Args, alloc)
		var dest *ltl.Loc
		if i.Dest != nil {
			loc := alloc.RegToLoc[*i.Dest]
			dest = &loc
		}
		return &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Lbuiltin{Builtin: i.Builtin, Args: args, Dest: dest},
				ltl.Lbranch{Succ: ltl.Node(i.Succ)},
			},
		}

	case rtl.Icond:
		args := transformRegs(i.Args, alloc)
		return &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Lcond{
					Cond:  i.Cond,
					Args:  args,
					IfSo:  ltl.Node(i.IfSo),
					IfNot: ltl.Node(i.IfNot),
				},
			},
		}

	case rtl.Ijumptable:
		arg := alloc.RegToLoc[i.Arg]
		targets := make([]ltl.Node, len(i.Targets))
		for j, t := range i.Targets {
			targets[j] = ltl.Node(t)
		}
		return &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Ljumptable{Arg: arg, Targets: targets},
			},
		}

	case rtl.Ireturn:
		var instrs []ltl.Instruction
		// If there's a return value, move it to the return register
		if i.Arg != nil {
			srcLoc := alloc.RegToLoc[*i.Arg]
			destLoc := ReturnLocation(false) // TODO: handle float returns
			// Only add move if not already in return register
			if srcLoc != destLoc {
				instrs = append(instrs, ltl.Lop{
					Op:   rtl.Omove{},
					Args: []ltl.Loc{srcLoc},
					Dest: destLoc,
				})
			}
		}
		instrs = append(instrs, ltl.Lreturn{})
		return &ltl.BBlock{Body: instrs}
	}

	// Fallback: empty block with nop
	return &ltl.BBlock{Body: []ltl.Instruction{ltl.Lnop{}}}
}

func transformRegs(regs []rtl.Reg, alloc *AllocationResult) []ltl.Loc {
	result := make([]ltl.Loc, len(regs))
	for i, r := range regs {
		result[i] = alloc.RegToLoc[r]
	}
	return result
}

// TransformProgram transforms an RTL program to LTL
func TransformProgram(rtlProg *rtl.Program) *ltl.Program {
	ltlProg := &ltl.Program{}

	// Transform globals
	for _, g := range rtlProg.Globals {
		ltlProg.Globals = append(ltlProg.Globals, ltl.GlobVar{
			Name:     g.Name,
			Size:     g.Size,
			Init:     g.Init,
			ReadOnly: g.ReadOnly,
		})
	}

	// Transform functions
	for _, fn := range rtlProg.Functions {
		ltlFn := TransformFunction(&fn)
		ltlProg.Functions = append(ltlProg.Functions, *ltlFn)
	}

	return ltlProg
}

// resolveParallelMoves generates a sequence of moves that correctly implements
// a parallel assignment from srcLocs to dstLocs. It uses a simple but correct
// strategy: save all source values that would be clobbered, then do all moves.
func resolveParallelMoves(srcLocs, dstLocs []ltl.Loc) []ltl.Instruction {
	n := len(srcLocs)
	if n == 0 {
		return nil
	}

	// Find moves that are actually needed
	type move struct {
		src, dst ltl.Loc
	}
	var moves []move
	for i := 0; i < n; i++ {
		if srcLocs[i] != dstLocs[i] {
			moves = append(moves, move{srcLocs[i], dstLocs[i]})
		}
	}

	if len(moves) == 0 {
		return nil
	}

	// Check which sources will be clobbered by destinations
	dstSet := make(map[ltl.Loc]bool)
	for _, m := range moves {
		dstSet[m.dst] = true
	}

	// For each source that is also a destination (potential clobber),
	// we need to save it first. Use X8-X15 as temporary storage.
	tmpIndex := 0
	savedLocs := make(map[ltl.Loc]ltl.Loc) // original loc -> temp loc
	var result []ltl.Instruction

	// First pass: save any source that would be clobbered
	for _, m := range moves {
		if dstSet[m.src] {
			if _, alreadySaved := savedLocs[m.src]; !alreadySaved {
				// Allocate a temp register (X8, X9, ... X15)
				tmp := ltl.R{Reg: ltl.MReg(ltl.X8 + ltl.MReg(tmpIndex))}
				tmpIndex++
				if tmpIndex > 8 {
					// Ran out of temp registers - fall back to simple sequential
					// This should rarely happen
					break
				}
				// Save the source value
				result = append(result, ltl.Lop{
					Op:   rtl.Omove{},
					Args: []ltl.Loc{m.src},
					Dest: tmp,
				})
				savedLocs[m.src] = tmp
			}
		}
	}

	// Second pass: emit all moves, using saved temps where needed
	for _, m := range moves {
		src := m.src
		if savedSrc, ok := savedLocs[m.src]; ok {
			src = savedSrc
		}
		result = append(result, ltl.Lop{
			Op:   rtl.Omove{},
			Args: []ltl.Loc{src},
			Dest: m.dst,
		})
	}

	return result
}
