// Branch tunneling optimization for Linear code.
// This pass shortcuts jumps that jump to other jumps.
// E.g., "goto L1" where L1 is "goto L2" becomes "goto L2"
package linearize

import "github.com/raymyers/ralph-cc/pkg/linear"

// Tunnel performs branch tunneling on a Linear function.
// It shortcuts chains of unconditional jumps.
func Tunnel(fn *linear.Function) {
	if len(fn.Code) == 0 {
		return
	}

	// Build map: label -> what it jumps to (if just a goto)
	jumpTargets := buildJumpTargetMap(fn)

	// Resolve chains
	resolved := resolveChains(jumpTargets)

	// Apply tunneling to all branch instructions
	for i, inst := range fn.Code {
		fn.Code[i] = tunnelInstruction(inst, resolved)
	}
}

// buildJumpTargetMap finds labels that are immediately followed by a goto
func buildJumpTargetMap(fn *linear.Function) map[linear.Label]linear.Label {
	result := make(map[linear.Label]linear.Label)

	// First pass: find all labels and what instruction follows them
	for i := 0; i < len(fn.Code)-1; i++ {
		lbl, ok := fn.Code[i].(linear.Llabel)
		if !ok {
			continue
		}

		// Check if next instruction is a goto
		if gt, ok := fn.Code[i+1].(linear.Lgoto); ok {
			result[lbl.Lbl] = gt.Target
		}
	}

	return result
}

// resolveChains follows jump chains to their ultimate target.
// Handles cycles by returning the label where a cycle is detected.
func resolveChains(jumpTargets map[linear.Label]linear.Label) map[linear.Label]linear.Label {
	result := make(map[linear.Label]linear.Label)

	for lbl := range jumpTargets {
		result[lbl] = resolveLabel(lbl, jumpTargets)
	}

	return result
}

// resolveLabel follows a jump chain to its ultimate target
func resolveLabel(lbl linear.Label, jumpTargets map[linear.Label]linear.Label) linear.Label {
	visited := make(map[linear.Label]bool)
	current := lbl

	for {
		if visited[current] {
			// Cycle detected - return current
			return current
		}
		visited[current] = true

		target, ok := jumpTargets[current]
		if !ok {
			// No further jump, this is the final target
			return current
		}
		current = target
	}
}

// tunnelInstruction applies tunneling to a single instruction
func tunnelInstruction(inst linear.Instruction, resolved map[linear.Label]linear.Label) linear.Instruction {
	switch i := inst.(type) {
	case linear.Lgoto:
		if target, ok := resolved[i.Target]; ok {
			return linear.Lgoto{Target: target}
		}
		return inst

	case linear.Lcond:
		newIfSo := i.IfSo
		if target, ok := resolved[i.IfSo]; ok {
			newIfSo = target
		}
		return linear.Lcond{Cond: i.Cond, Args: i.Args, IfSo: newIfSo}

	case linear.Ljumptable:
		newTargets := make([]linear.Label, len(i.Targets))
		for j, target := range i.Targets {
			if resolved, ok := resolved[target]; ok {
				newTargets[j] = resolved
			} else {
				newTargets[j] = target
			}
		}
		return linear.Ljumptable{Arg: i.Arg, Targets: newTargets}

	default:
		return inst
	}
}
