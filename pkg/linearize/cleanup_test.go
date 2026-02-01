package linearize

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestCleanupLabelsEmpty(t *testing.T) {
	fn := linear.NewFunction("empty", linear.Sig{})
	CleanupLabels(fn)
	// Should not panic
	if len(fn.Code) != 0 {
		t.Errorf("Expected empty code")
	}
}

func TestCleanupLabelsKeepsEntry(t *testing.T) {
	// L1: return
	// Entry label should be kept even if unreferenced
	fn := linear.NewFunction("entry", linear.Sig{})
	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lreturn{})

	CleanupLabels(fn)

	// Entry label should still be there
	if len(fn.Code) != 2 {
		t.Errorf("Code length = %d, want 2", len(fn.Code))
	}
	if lbl, ok := fn.Code[0].(linear.Llabel); !ok || lbl.Lbl != 1 {
		t.Errorf("Entry label should be preserved")
	}
}

func TestCleanupLabelsRemovesUnreferenced(t *testing.T) {
	// L1: goto L3
	// L2: nop (unreferenced)
	// L3: return
	fn := linear.NewFunction("cleanup", linear.Sig{})
	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lgoto{Target: 3})
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lop{Op: rtl.Ointconst{Value: 0}, Dest: linear.R{Reg: ltl.X0}})
	fn.Append(linear.Llabel{Lbl: 3})
	fn.Append(linear.Lreturn{})

	CleanupLabels(fn)

	// L2 should be removed (unreferenced)
	labelCount := 0
	labels := []linear.Label{}
	for _, inst := range fn.Code {
		if lbl, ok := inst.(linear.Llabel); ok {
			labelCount++
			labels = append(labels, lbl.Lbl)
		}
	}

	if labelCount != 2 {
		t.Errorf("Label count = %d, want 2 (got labels %v)", labelCount, labels)
	}

	// L1 and L3 should remain
	seenL1 := false
	seenL3 := false
	for _, lbl := range labels {
		if lbl == 1 {
			seenL1 = true
		}
		if lbl == 3 {
			seenL3 = true
		}
	}
	if !seenL1 || !seenL3 {
		t.Errorf("Expected L1 and L3 to remain, got labels %v", labels)
	}
}

func TestCleanupLabelsPreservesConditionalTargets(t *testing.T) {
	// L1: if cond then L2
	// L2: return
	fn := linear.NewFunction("cond", linear.Sig{})
	cond := rtl.Ccompimm{Cond: rtl.Ceq, N: 0}

	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lcond{Cond: cond, Args: []linear.Loc{linear.R{Reg: ltl.X0}}, IfSo: 2})
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lreturn{})

	CleanupLabels(fn)

	// Both labels should be preserved
	labelCount := 0
	for _, inst := range fn.Code {
		if _, ok := inst.(linear.Llabel); ok {
			labelCount++
		}
	}
	if labelCount != 2 {
		t.Errorf("Label count = %d, want 2", labelCount)
	}
}

func TestCleanupLabelsPreservesJumptableTargets(t *testing.T) {
	// L1: jumptable [L2, L3]
	// L2: return
	// L3: return
	fn := linear.NewFunction("jt", linear.Sig{})

	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Ljumptable{Arg: linear.R{Reg: ltl.X0}, Targets: []linear.Label{2, 3}})
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lreturn{})
	fn.Append(linear.Llabel{Lbl: 3})
	fn.Append(linear.Lreturn{})

	CleanupLabels(fn)

	// All labels should be preserved
	labelCount := 0
	for _, inst := range fn.Code {
		if _, ok := inst.(linear.Llabel); ok {
			labelCount++
		}
	}
	if labelCount != 3 {
		t.Errorf("Label count = %d, want 3", labelCount)
	}
}

func TestCleanupLabelsMultipleUnreferenced(t *testing.T) {
	// L1: goto L5
	// L2: unreferenced
	// L3: unreferenced
	// L4: unreferenced
	// L5: return
	fn := linear.NewFunction("multi", linear.Sig{})

	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lgoto{Target: 5})
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lop{Op: rtl.Ointconst{Value: 0}, Dest: linear.R{Reg: ltl.X0}})
	fn.Append(linear.Llabel{Lbl: 3})
	fn.Append(linear.Lop{Op: rtl.Ointconst{Value: 0}, Dest: linear.R{Reg: ltl.X0}})
	fn.Append(linear.Llabel{Lbl: 4})
	fn.Append(linear.Lop{Op: rtl.Ointconst{Value: 0}, Dest: linear.R{Reg: ltl.X0}})
	fn.Append(linear.Llabel{Lbl: 5})
	fn.Append(linear.Lreturn{})

	CleanupLabels(fn)

	// Only L1 and L5 should remain
	labels := []linear.Label{}
	for _, inst := range fn.Code {
		if lbl, ok := inst.(linear.Llabel); ok {
			labels = append(labels, lbl.Lbl)
		}
	}

	if len(labels) != 2 {
		t.Errorf("Label count = %d, want 2 (got labels %v)", len(labels), labels)
	}
}

func TestCleanupLabelsAfterTunneling(t *testing.T) {
	// After tunneling, some labels may become unreferenced
	// L1: goto L3 (after tunneling from L1 -> L2 -> L3)
	// L2: now unreferenced!
	// L3: return
	fn := linear.NewFunction("posttunnel", linear.Sig{})

	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lgoto{Target: 3}) // Already tunneled
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lgoto{Target: 3})
	fn.Append(linear.Llabel{Lbl: 3})
	fn.Append(linear.Lreturn{})

	CleanupLabels(fn)

	// L2 should be removed
	labels := []linear.Label{}
	for _, inst := range fn.Code {
		if lbl, ok := inst.(linear.Llabel); ok {
			labels = append(labels, lbl.Lbl)
		}
	}

	if len(labels) != 2 {
		t.Errorf("Label count = %d, want 2 (got labels %v)", len(labels), labels)
	}

	// L2 should not be present
	for _, lbl := range labels {
		if lbl == 2 {
			t.Errorf("L2 should be removed but was found")
		}
	}
}
