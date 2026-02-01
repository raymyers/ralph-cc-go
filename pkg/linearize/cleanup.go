// Label cleanup for Linear code.
// This pass removes labels that are not referenced by any branch.
package linearize

import "github.com/raymyers/ralph-cc/pkg/linear"

// CleanupLabels removes unreferenced labels from a Linear function.
// The first label (entry point) is always preserved.
func CleanupLabels(fn *linear.Function) {
	if len(fn.Code) == 0 {
		return
	}

	// Find all referenced labels
	used := collectUsedLabels(fn)

	// Find the entry label (first label in code)
	var entryLabel linear.Label
	for _, inst := range fn.Code {
		if lbl, ok := inst.(linear.Llabel); ok {
			entryLabel = lbl.Lbl
			break
		}
	}

	// Always keep the entry label
	used[entryLabel] = true

	// Filter out unreferenced labels
	newCode := make([]linear.Instruction, 0, len(fn.Code))
	for _, inst := range fn.Code {
		if lbl, ok := inst.(linear.Llabel); ok {
			if !used[lbl.Lbl] {
				// Skip this unreferenced label
				continue
			}
		}
		newCode = append(newCode, inst)
	}

	fn.Code = newCode
}

// collectUsedLabels returns all labels that are targets of branches
func collectUsedLabels(fn *linear.Function) map[linear.Label]bool {
	used := make(map[linear.Label]bool)

	for _, inst := range fn.Code {
		switch i := inst.(type) {
		case linear.Lgoto:
			used[i.Target] = true
		case linear.Lcond:
			used[i.IfSo] = true
		case linear.Ljumptable:
			for _, target := range i.Targets {
				used[target] = true
			}
		}
	}

	return used
}
