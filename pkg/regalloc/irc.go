package regalloc

import (
	"sort"

	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// Allocator performs register allocation using the Iterated Register Coalescing algorithm.
// This is a simplified version that focuses on correctness over optimality.
type Allocator struct {
	graph     *InterferenceGraph
	liveness  *LivenessInfo
	fn        *rtl.Function
	K         int // Number of allocatable registers
	colors    map[rtl.Reg]int // Assigned color (machine register index)
	spillSlot map[rtl.Reg]int // Spill slot offset for spilled registers

	// IRC worklists
	simplifyWorklist []rtl.Reg // Low-degree non-move-related nodes
	freezeWorklist   []rtl.Reg // Low-degree move-related nodes
	spillWorklist    []rtl.Reg // High-degree nodes (potential spills)
	coalescedNodes   RegSet    // Nodes that have been coalesced
	coloredNodes     RegSet    // Successfully colored nodes
	spilledNodes     RegSet    // Nodes that must be spilled
	selectStack      []rtl.Reg // Stack of nodes removed during simplify/spill

	// For coalescing
	alias map[rtl.Reg]rtl.Reg // Maps coalesced node to its representative

	// Move worklists
	coalescedMoves   [][2]rtl.Reg // Successfully coalesced
	constrainedMoves [][2]rtl.Reg // Moves between interfering nodes
	frozenMoves      [][2]rtl.Reg // Frozen (no longer candidates for coalescing)
	worklistMoves    [][2]rtl.Reg // Active move candidates
	activeMoves      [][2]rtl.Reg // Moves not yet ready to coalesce

	nextSpillSlot int64 // Next available spill slot offset

	// Precolored registers for parameters (maps param index to its fixed location)
	precoloredParams map[rtl.Reg]ltl.Loc
}

// AllocationResult holds the result of register allocation
type AllocationResult struct {
	// RegToLoc maps pseudo-registers to their assigned locations
	RegToLoc map[rtl.Reg]ltl.Loc
	// SpilledRegs is the set of registers that were spilled
	SpilledRegs RegSet
	// StackSize is the size of the stack frame needed for spills
	StackSize int64
}

// NewAllocator creates a new register allocator
func NewAllocator(fn *rtl.Function, graph *InterferenceGraph, liveness *LivenessInfo) *Allocator {
	a := &Allocator{
		fn:               fn,
		graph:            graph,
		liveness:         liveness,
		K:                NumAllocatableIntRegs,
		colors:           make(map[rtl.Reg]int),
		spillSlot:        make(map[rtl.Reg]int),
		coalescedNodes:   NewRegSet(),
		coloredNodes:     NewRegSet(),
		spilledNodes:     NewRegSet(),
		alias:            make(map[rtl.Reg]rtl.Reg),
		precoloredParams: make(map[rtl.Reg]ltl.Loc),
	}

	// Precolor parameters according to calling convention
	// Parameters 0-7 go to X0-X7, parameters 8+ go on the stack
	// IMPORTANT: Do NOT precolor parameters that are live across calls.
	// Those parameters need to be moved to callee-saved registers.
	for i, param := range fn.Params {
		// Check if this parameter is live across any call
		if graph.LiveAcrossCalls.Contains(param) {
			// Don't precolor - let it be allocated to a callee-saved register
			continue
		}
		a.precoloredParams[param] = ArgLocation(i, false)
	}

	return a
}

// Allocate performs register allocation and returns the result
func (a *Allocator) Allocate() *AllocationResult {
	a.buildWorklists()

	// Main loop
	for {
		if len(a.simplifyWorklist) > 0 {
			a.simplify()
		} else if len(a.worklistMoves) > 0 {
			a.coalesce()
		} else if len(a.freezeWorklist) > 0 {
			a.freeze()
		} else if len(a.spillWorklist) > 0 {
			a.selectSpill()
		} else {
			break
		}
	}

	// Assign colors
	a.assignColors()

	// Build result
	return a.buildResult()
}

func (a *Allocator) buildWorklists() {
	// First, mark precolored params as already colored
	// They should not be in any worklist and their colors are fixed
	for param, loc := range a.precoloredParams {
		if regLoc, ok := loc.(ltl.R); ok {
			// Find the color index for this register
			for i, mreg := range AllocatableIntRegs {
				if mreg == regLoc.Reg {
					a.colors[param] = i
					a.coloredNodes.Add(param)
					break
				}
			}
		}
		// Stack-slot params don't get colored - they'll be handled in buildResult
	}

	// Categorize non-precolored nodes into worklists
	for r := range a.graph.Nodes {
		// Skip precolored params - they're already colored
		if _, isParam := a.precoloredParams[r]; isParam {
			continue
		}
		if a.degree(r) >= a.K {
			a.spillWorklist = append(a.spillWorklist, r)
		} else if a.graph.MoveRelated(r) {
			a.freezeWorklist = append(a.freezeWorklist, r)
		} else {
			a.simplifyWorklist = append(a.simplifyWorklist, r)
		}
	}

	// Build initial move worklist from preferences
	for r, prefs := range a.graph.Preferences {
		for p := range prefs {
			if r < p { // Avoid duplicates
				a.worklistMoves = append(a.worklistMoves, [2]rtl.Reg{r, p})
			}
		}
	}
}

func (a *Allocator) degree(r rtl.Reg) int {
	// Don't count coalesced nodes
	deg := 0
	for neighbor := range a.graph.Edges[r] {
		if !a.coalescedNodes.Contains(neighbor) {
			deg++
		}
	}
	return deg
}

func (a *Allocator) simplify() {
	// Pop a node from simplify worklist
	n := len(a.simplifyWorklist) - 1
	r := a.simplifyWorklist[n]
	a.simplifyWorklist = a.simplifyWorklist[:n]

	// Push onto select stack
	a.selectStack = append(a.selectStack, r)

	// Update neighbors' degrees
	for neighbor := range a.graph.Edges[r] {
		a.decrementDegree(neighbor)
	}
}

func (a *Allocator) decrementDegree(r rtl.Reg) {
	if a.coalescedNodes.Contains(r) {
		return
	}

	// If degree drops below K, move to appropriate worklist
	if a.degree(r) == a.K-1 {
		// Remove from spill worklist
		a.removeFromWorklist(r, &a.spillWorklist)

		// Add to freeze or simplify worklist
		if a.graph.MoveRelated(r) {
			a.freezeWorklist = append(a.freezeWorklist, r)
		} else {
			a.simplifyWorklist = append(a.simplifyWorklist, r)
		}
	}
}

func (a *Allocator) removeFromWorklist(r rtl.Reg, list *[]rtl.Reg) {
	for i, reg := range *list {
		if reg == r {
			*list = append((*list)[:i], (*list)[i+1:]...)
			return
		}
	}
}

func (a *Allocator) coalesce() {
	// Pop a move from worklist
	n := len(a.worklistMoves) - 1
	m := a.worklistMoves[n]
	a.worklistMoves = a.worklistMoves[:n]

	x := a.getAlias(m[0])
	y := a.getAlias(m[1])

	// Make sure we merge into the lower-numbered register (arbitrary choice)
	var u, v rtl.Reg
	if x < y {
		u, v = x, y
	} else {
		u, v = y, x
	}

	if u == v {
		// Already coalesced
		a.coalescedMoves = append(a.coalescedMoves, m)
		a.addToWorklist(u)
	} else if a.graph.HasEdge(u, v) {
		// Interfere - can't coalesce
		a.constrainedMoves = append(a.constrainedMoves, m)
		a.addToWorklist(u)
		a.addToWorklist(v)
	} else if a.conservativeCoalesce(u, v) {
		// Safe to coalesce
		a.coalescedMoves = append(a.coalescedMoves, m)
		a.combine(u, v)
		a.addToWorklist(u)
	} else {
		// Not safe yet - keep active
		a.activeMoves = append(a.activeMoves, m)
	}
}

func (a *Allocator) getAlias(r rtl.Reg) rtl.Reg {
	if a.coalescedNodes.Contains(r) {
		return a.getAlias(a.alias[r])
	}
	return r
}

func (a *Allocator) conservativeCoalesce(u, v rtl.Reg) bool {
	// Conservative coalescing (Briggs criterion):
	// Safe to coalesce if combined node has < K high-degree neighbors
	highDegreeNeighbors := 0
	neighbors := NewRegSet()

	for n := range a.graph.Edges[u] {
		if !a.coalescedNodes.Contains(n) {
			neighbors.Add(n)
		}
	}
	for n := range a.graph.Edges[v] {
		if !a.coalescedNodes.Contains(n) {
			neighbors.Add(n)
		}
	}

	for n := range neighbors {
		if a.degree(n) >= a.K {
			highDegreeNeighbors++
		}
	}

	return highDegreeNeighbors < a.K
}

func (a *Allocator) combine(u, v rtl.Reg) {
	// Remove v from worklists
	a.removeFromWorklist(v, &a.freezeWorklist)
	a.removeFromWorklist(v, &a.spillWorklist)

	// Mark v as coalesced
	a.coalescedNodes.Add(v)
	a.alias[v] = u

	// If either u or v is live across calls, the combined node must be too
	if a.graph.LiveAcrossCalls.Contains(v) {
		a.graph.LiveAcrossCalls.Add(u)
	}

	// Merge edges
	for n := range a.graph.Edges[v] {
		if !a.coalescedNodes.Contains(n) && n != u {
			a.graph.AddEdge(u, n)
			a.decrementDegree(n)
		}
	}

	// Merge preferences
	for n := range a.graph.Preferences[v] {
		if n != u {
			a.graph.AddPreference(u, n)
		}
	}

	// If u now has high degree, move to spill worklist
	if a.degree(u) >= a.K {
		a.removeFromWorklist(u, &a.freezeWorklist)
		a.spillWorklist = append(a.spillWorklist, u)
	}
}

func (a *Allocator) addToWorklist(r rtl.Reg) {
	if a.coalescedNodes.Contains(r) {
		return
	}
	if a.degree(r) < a.K && !a.graph.MoveRelated(r) {
		a.removeFromWorklist(r, &a.freezeWorklist)
		a.simplifyWorklist = append(a.simplifyWorklist, r)
	}
}

func (a *Allocator) freeze() {
	// Pop a node from freeze worklist
	n := len(a.freezeWorklist) - 1
	r := a.freezeWorklist[n]
	a.freezeWorklist = a.freezeWorklist[:n]

	// Move to simplify worklist
	a.simplifyWorklist = append(a.simplifyWorklist, r)

	// Freeze all moves involving this node
	a.freezeMovesFor(r)
}

func (a *Allocator) freezeMovesFor(r rtl.Reg) {
	var remaining [][2]rtl.Reg
	for _, m := range a.activeMoves {
		if m[0] == r || m[1] == r {
			a.frozenMoves = append(a.frozenMoves, m)

			// Enable coalescing for the other node
			var other rtl.Reg
			if m[0] == r {
				other = m[1]
			} else {
				other = m[0]
			}
			a.addToWorklist(other)
		} else {
			remaining = append(remaining, m)
		}
	}
	a.activeMoves = remaining
}

func (a *Allocator) selectSpill() {
	// Select a node to spill using simple heuristic
	// Choose the one with highest degree
	var maxDeg int
	var maxReg rtl.Reg
	maxIdx := -1

	for i, r := range a.spillWorklist {
		d := a.degree(r)
		if d > maxDeg || maxIdx == -1 {
			maxDeg = d
			maxReg = r
			maxIdx = i
		}
	}

	if maxIdx >= 0 {
		a.spillWorklist = append(a.spillWorklist[:maxIdx], a.spillWorklist[maxIdx+1:]...)
		a.simplifyWorklist = append(a.simplifyWorklist, maxReg)
		a.freezeMovesFor(maxReg)
	}
}

func (a *Allocator) assignColors() {
	// Pop nodes from select stack and assign colors
	for len(a.selectStack) > 0 {
		n := len(a.selectStack) - 1
		r := a.selectStack[n]
		a.selectStack = a.selectStack[:n]

		// Determine which colors are used by neighbors
		usedColors := make(map[int]bool)
		for neighbor := range a.graph.Edges[r] {
			alias := a.getAlias(neighbor)
			if a.coloredNodes.Contains(alias) {
				usedColors[a.colors[alias]] = true
			}
		}

		// Determine the starting color based on whether this register is live across calls
		// If live across a call, only use callee-saved registers (colors FirstCalleeSavedColor and above)
		startColor := 0
		if a.graph.LiveAcrossCalls.Contains(r) {
			startColor = FirstCalleeSavedColor
		}

		// Try to assign a color
		color := -1
		for c := startColor; c < a.K; c++ {
			if !usedColors[c] {
				color = c
				break
			}
		}

		if color >= 0 {
			a.coloredNodes.Add(r)
			a.colors[r] = color
		} else {
			// Must spill
			a.spilledNodes.Add(r)
			a.spillSlot[r] = int(a.nextSpillSlot)
			a.nextSpillSlot += 8 // 8 bytes per spill slot
		}
	}

	// Copy colors to coalesced nodes
	for r := range a.coalescedNodes {
		alias := a.getAlias(r)
		if a.coloredNodes.Contains(alias) {
			a.colors[r] = a.colors[alias]
			a.coloredNodes.Add(r)
		} else if a.spilledNodes.Contains(alias) {
			a.spilledNodes.Add(r)
			a.spillSlot[r] = a.spillSlot[alias]
		} else if _, isParam := a.precoloredParams[alias]; isParam {
			// Alias is a precolored stack-slot param (not in coloredNodes)
			// r inherits the same stack slot location
			// We don't add to coloredNodes or spilledNodes - buildResult will handle via precoloredParams
			a.precoloredParams[r] = a.precoloredParams[alias]
		}
	}
}

func (a *Allocator) buildResult() *AllocationResult {
	result := &AllocationResult{
		RegToLoc:    make(map[rtl.Reg]ltl.Loc),
		SpilledRegs: a.spilledNodes.Copy(),
		StackSize:   a.nextSpillSlot,
	}

	// First, map precolored parameters (these have fixed locations)
	for param, loc := range a.precoloredParams {
		result.RegToLoc[param] = loc
	}

	// Map colored registers to physical registers
	for r := range a.coloredNodes {
		// Skip precolored params - they already have their locations
		if _, isParam := a.precoloredParams[r]; isParam {
			continue
		}
		color := a.colors[r]
		if color < len(AllocatableIntRegs) {
			result.RegToLoc[r] = ltl.R{Reg: AllocatableIntRegs[color]}
		}
	}

	// Map spilled registers to stack slots
	for r := range a.spilledNodes {
		// Skip precolored params - they already have their locations
		if _, isParam := a.precoloredParams[r]; isParam {
			continue
		}
		slot := a.spillSlot[r]
		result.RegToLoc[r] = ltl.S{
			Slot: ltl.SlotLocal,
			Ofs:  int64(slot),
			Ty:   ltl.Tlong,
		}
	}

	return result
}

// AllocateFunction performs register allocation for a function
func AllocateFunction(fn *rtl.Function) *AllocationResult {
	liveness := AnalyzeLiveness(fn)
	graph := BuildInterferenceGraph(fn, liveness)
	allocator := NewAllocator(fn, graph, liveness)
	return allocator.Allocate()
}

// GetAllRegisters returns all pseudo-registers used in the function
func GetAllRegisters(fn *rtl.Function) RegSet {
	regs := NewRegSet()
	for _, param := range fn.Params {
		regs.Add(param)
	}
	for _, instr := range fn.Code {
		switch i := instr.(type) {
		case rtl.Iop:
			for _, arg := range i.Args {
				regs.Add(arg)
			}
			regs.Add(i.Dest)
		case rtl.Iload:
			for _, arg := range i.Args {
				regs.Add(arg)
			}
			regs.Add(i.Dest)
		case rtl.Istore:
			for _, arg := range i.Args {
				regs.Add(arg)
			}
			regs.Add(i.Src)
		case rtl.Icall:
			for _, arg := range i.Args {
				regs.Add(arg)
			}
			if i.Dest != 0 {
				regs.Add(i.Dest)
			}
			if fr, ok := i.Fn.(rtl.FunReg); ok {
				regs.Add(fr.Reg)
			}
		case rtl.Itailcall:
			for _, arg := range i.Args {
				regs.Add(arg)
			}
			if fr, ok := i.Fn.(rtl.FunReg); ok {
				regs.Add(fr.Reg)
			}
		case rtl.Ibuiltin:
			for _, arg := range i.Args {
				regs.Add(arg)
			}
			if i.Dest != nil {
				regs.Add(*i.Dest)
			}
		case rtl.Icond:
			for _, arg := range i.Args {
				regs.Add(arg)
			}
		case rtl.Ijumptable:
			regs.Add(i.Arg)
		case rtl.Ireturn:
			if i.Arg != nil {
				regs.Add(*i.Arg)
			}
		}
	}
	return regs
}

// SortedRegSlice returns a sorted slice of registers (for deterministic output)
func SortedRegSlice(s RegSet) []rtl.Reg {
	result := s.Slice()
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}
