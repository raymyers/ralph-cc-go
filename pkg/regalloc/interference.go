package regalloc

import (
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// InterferenceGraph represents the register interference graph.
// Two registers interfere if they are both live at the same point.
type InterferenceGraph struct {
	// Nodes are pseudo-registers
	Nodes RegSet
	// Edges maps each register to its interfering neighbors
	Edges map[rtl.Reg]RegSet
	// Preferences maps each register to preferred registers (for coalescing)
	Preferences map[rtl.Reg]RegSet
	// LiveAcrossCalls tracks registers that are live across function calls
	// These must be assigned to callee-saved registers or spilled
	LiveAcrossCalls RegSet
}

// NewInterferenceGraph creates an empty interference graph
func NewInterferenceGraph() *InterferenceGraph {
	return &InterferenceGraph{
		Nodes:           NewRegSet(),
		Edges:           make(map[rtl.Reg]RegSet),
		Preferences:     make(map[rtl.Reg]RegSet),
		LiveAcrossCalls: NewRegSet(),
	}
}

// AddNode adds a register to the graph
func (g *InterferenceGraph) AddNode(r rtl.Reg) {
	g.Nodes.Add(r)
	if g.Edges[r] == nil {
		g.Edges[r] = NewRegSet()
	}
	if g.Preferences[r] == nil {
		g.Preferences[r] = NewRegSet()
	}
}

// AddEdge adds an interference edge between two registers
func (g *InterferenceGraph) AddEdge(r1, r2 rtl.Reg) {
	if r1 == r2 {
		return // No self-edges
	}
	g.AddNode(r1)
	g.AddNode(r2)
	g.Edges[r1].Add(r2)
	g.Edges[r2].Add(r1)
}

// AddPreference adds a preference edge (for move coalescing)
func (g *InterferenceGraph) AddPreference(r1, r2 rtl.Reg) {
	if r1 == r2 {
		return
	}
	g.AddNode(r1)
	g.AddNode(r2)
	g.Preferences[r1].Add(r2)
	g.Preferences[r2].Add(r1)
}

// HasEdge returns true if there is an interference edge
func (g *InterferenceGraph) HasEdge(r1, r2 rtl.Reg) bool {
	if edges, ok := g.Edges[r1]; ok {
		return edges.Contains(r2)
	}
	return false
}

// Degree returns the number of neighbors for a register
func (g *InterferenceGraph) Degree(r rtl.Reg) int {
	if edges, ok := g.Edges[r]; ok {
		return len(edges)
	}
	return 0
}

// Neighbors returns the interfering neighbors of a register
func (g *InterferenceGraph) Neighbors(r rtl.Reg) RegSet {
	if edges, ok := g.Edges[r]; ok {
		return edges.Copy()
	}
	return NewRegSet()
}

// RemoveNode removes a register from the graph
func (g *InterferenceGraph) RemoveNode(r rtl.Reg) {
	// Remove edges from neighbors
	if edges, ok := g.Edges[r]; ok {
		for neighbor := range edges {
			delete(g.Edges[neighbor], r)
		}
	}
	// Remove from preferences
	if prefs, ok := g.Preferences[r]; ok {
		for neighbor := range prefs {
			delete(g.Preferences[neighbor], r)
		}
	}
	// Remove the node
	delete(g.Nodes, r)
	delete(g.Edges, r)
	delete(g.Preferences, r)
}

// BuildInterferenceGraph constructs the interference graph from liveness info
func BuildInterferenceGraph(fn *rtl.Function, liveness *LivenessInfo) *InterferenceGraph {
	g := NewInterferenceGraph()

	// First, add all registers as nodes (including parameters!)
	for _, param := range fn.Params {
		g.AddNode(param)
	}
	for node, def := range liveness.Def {
		for r := range def {
			g.AddNode(r)
		}
		for r := range liveness.Use[node] {
			g.AddNode(r)
		}
	}

	// Build interference edges
	// Rule: A defined register interferes with all registers live at exit
	// (except itself, and except when it's a move instruction copying from that register)
	for node, instr := range fn.Code {
		def := liveness.Def[node]
		liveOut := liveness.LiveOut[node]

		// For each defined register
		for defReg := range def {
			// It interferes with all live-out registers
			for liveReg := range liveOut {
				// Special case: move instruction - no interference with source
				if isMove(instr) && isMoveSource(instr, liveReg) {
					continue
				}
				g.AddEdge(defReg, liveReg)
			}
		}

		// Track registers live across function calls
		// These must be allocated to callee-saved registers or spilled
		if isCall(instr) {
			for liveReg := range liveOut {
				g.LiveAcrossCalls.Add(liveReg)
			}
		}
	}

	// IMPORTANT: Parameters need special handling for interference.
	// Parameters arrive in specific registers (X0-X7) and if they're used later,
	// no other variable can use those registers until the parameter's last use.
	// Add interference between used parameters and ALL other pseudo-registers.
	for _, param := range fn.Params {
		// Check if this parameter is used anywhere
		var paramUsed bool
		for _, use := range liveness.Use {
			if use.Contains(param) {
				paramUsed = true
				break
			}
		}
		if !paramUsed {
			continue
		}

		// Add interference with ALL other registers (conservative approach)
		for other := range g.Nodes {
			if other != param {
				g.AddEdge(param, other)
			}
		}
	}

	// Build preference edges for moves
	for _, instr := range fn.Code {
		if iop, ok := instr.(rtl.Iop); ok {
			if _, isMove := iop.Op.(rtl.Omove); isMove && len(iop.Args) == 1 {
				g.AddPreference(iop.Dest, iop.Args[0])
			}
		}
	}

	return g
}

// isCall returns true if the instruction is a function call
func isCall(instr rtl.Instruction) bool {
	switch instr.(type) {
	case rtl.Icall:
		return true
	default:
		return false
	}
}

// isMove returns true if the instruction is a move operation
func isMove(instr rtl.Instruction) bool {
	if iop, ok := instr.(rtl.Iop); ok {
		_, isMove := iop.Op.(rtl.Omove)
		return isMove
	}
	return false
}

// isMoveSource returns true if reg is the source of a move instruction
func isMoveSource(instr rtl.Instruction, reg rtl.Reg) bool {
	if iop, ok := instr.(rtl.Iop); ok {
		if _, isMove := iop.Op.(rtl.Omove); isMove && len(iop.Args) == 1 {
			return iop.Args[0] == reg
		}
	}
	return false
}

// MoveRelated returns true if the register is involved in a move
func (g *InterferenceGraph) MoveRelated(r rtl.Reg) bool {
	prefs := g.Preferences[r]
	return len(prefs) > 0
}
