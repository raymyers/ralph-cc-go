package stacking

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/mach"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// Helper to create a minimal SlotTranslator for tests
func testSlotTranslator() *SlotTranslator {
	layout := &FrameLayout{
		CalleeSaveSize: 16,
		LocalSize:      16,
		OutgoingSize:   0,
		LocalOffset:    -32, // -CalleeSaveSize - LocalSize
		TotalSize:      48,  // 16 + 16 + 16 (FP/LR)
	}
	return NewSlotTranslator(layout)
}

func TestGenerateParamCopies(t *testing.T) {
	// Test 1: Parameter in X19 (callee-saved), should copy from X0
	params := []ltl.Loc{ltl.R{Reg: ltl.X19}}
	copies := GenerateParamCopies(params, testSlotTranslator())

	if len(copies) != 1 {
		t.Fatalf("expected 1 copy instruction, got %d", len(copies))
	}

	op, ok := copies[0].(mach.Mop)
	if !ok {
		t.Fatalf("expected Mop, got %T", copies[0])
	}
	if len(op.Args) != 1 || op.Args[0] != ltl.X0 {
		t.Errorf("expected Args=[X0], got %v", op.Args)
	}
	if op.Dest != ltl.X19 {
		t.Errorf("expected Dest=X19, got %v", op.Dest)
	}
}

func TestGenerateParamCopiesMultiple(t *testing.T) {
	// Test: Two parameters in callee-saved registers (no conflict)
	// First in X20, second in X19
	params := []ltl.Loc{ltl.R{Reg: ltl.X20}, ltl.R{Reg: ltl.X19}}
	copies := GenerateParamCopies(params, testSlotTranslator())

	if len(copies) != 2 {
		t.Fatalf("expected 2 copy instructions, got %d", len(copies))
	}

	// The order may vary due to map iteration, but both should be present
	found := make(map[string]bool)
	for _, c := range copies {
		op, _ := c.(mach.Mop)
		key := string(rune(op.Args[0])) + "->" + string(rune(op.Dest))
		found[key] = true
	}
	
	// Check X20 = X0 is present
	x20Copy := false
	x19Copy := false
	for _, c := range copies {
		op, _ := c.(mach.Mop)
		if op.Args[0] == ltl.X0 && op.Dest == ltl.X20 {
			x20Copy = true
		}
		if op.Args[0] == ltl.X1 && op.Dest == ltl.X19 {
			x19Copy = true
		}
	}
	if !x20Copy {
		t.Error("missing copy X20 = X0")
	}
	if !x19Copy {
		t.Error("missing copy X19 = X1")
	}
}

func TestGenerateParamCopiesCycle(t *testing.T) {
	// Test: Two parameters with a cycle (first in X1, second in X0)
	// This requires breaking the cycle with a temp register
	params := []ltl.Loc{ltl.R{Reg: ltl.X1}, ltl.R{Reg: ltl.X0}}
	copies := GenerateParamCopies(params, testSlotTranslator())

	// Should have at least 2 instructions (possibly 3 with temp)
	if len(copies) < 2 {
		t.Fatalf("expected at least 2 copy instructions, got %d", len(copies))
	}

	t.Logf("Generated copies for cycle case:")
	for i, c := range copies {
		t.Logf("  [%d] %v", i, c)
	}
	
	// Verify the cycle is properly broken by simulating execution
	regs := make(map[ltl.MReg]int)
	regs[ltl.X0] = 35 // Incoming first arg
	regs[ltl.X1] = 7  // Incoming second arg
	regs[ltl.X8] = 0  // Temp register
	
	for _, c := range copies {
		op, _ := c.(mach.Mop)
		regs[op.Dest] = regs[op.Args[0]]
	}
	
	// After all copies:
	// X1 should have original X0 value (35)
	// X0 should have original X1 value (7)
	if regs[ltl.X1] != 35 {
		t.Errorf("X1 = %d, want 35 (original X0)", regs[ltl.X1])
	}
	if regs[ltl.X0] != 7 {
		t.Errorf("X0 = %d, want 7 (original X1)", regs[ltl.X0])
	}
}

func TestTransformWithParams(t *testing.T) {
	// Test the full Transform function with a parameter
	fn := linear.NewFunction("test", ltl.Sig{Args: []string{"int"}, Return: "int"})
	fn.Params = []linear.Loc{ltl.R{Reg: ltl.X19}}

	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lop{Op: rtl.Omove{}, Args: []linear.Loc{linear.R{Reg: ltl.X19}}, Dest: linear.R{Reg: ltl.X0}})
	fn.Append(linear.Lreturn{})

	machFn := Transform(fn)

	// Look for the param copy instruction after prologue
	foundParamCopy := false
	for i, inst := range machFn.Code {
		if op, ok := inst.(mach.Mop); ok {
			if _, isMove := op.Op.(rtl.Omove); isMove {
				if len(op.Args) == 1 && op.Args[0] == ltl.X0 && op.Dest == ltl.X19 {
					foundParamCopy = true
					t.Logf("Found param copy at index %d", i)
					break
				}
			}
		}
	}

	if !foundParamCopy {
		t.Error("did not find param copy instruction (X19 = move X0)")
		t.Log("Generated Mach instructions:")
		for i, inst := range machFn.Code {
			t.Logf("  [%d] %T: %v", i, inst, inst)
		}
	}
}
