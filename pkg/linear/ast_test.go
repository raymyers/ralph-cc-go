package linear

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestLabelValid(t *testing.T) {
	tests := []struct {
		label Label
		valid bool
	}{
		{Label(0), false},
		{Label(1), true},
		{Label(100), true},
		{Label(-1), false},
	}
	for _, tt := range tests {
		if got := tt.label.Valid(); got != tt.valid {
			t.Errorf("Label(%d).Valid() = %v, want %v", tt.label, got, tt.valid)
		}
	}
}

func TestNewFunction(t *testing.T) {
	sig := Sig{Args: []string{"int"}, Return: "int"}
	fn := NewFunction("test", sig)

	if fn.Name != "test" {
		t.Errorf("Name = %s, want test", fn.Name)
	}
	if fn.Code == nil {
		t.Error("Code should be initialized")
	}
	if len(fn.Code) != 0 {
		t.Errorf("Code should be empty, got %d instructions", len(fn.Code))
	}
}

func TestFunctionAppend(t *testing.T) {
	sig := Sig{}
	fn := NewFunction("test", sig)

	fn.Append(Llabel{Lbl: Label(1)})
	fn.Append(Lop{Op: rtl.Ointconst{Value: 42}, Dest: R{Reg: ltl.X0}})
	fn.Append(Lreturn{})

	if len(fn.Code) != 3 {
		t.Errorf("Code length = %d, want 3", len(fn.Code))
	}
}

func TestFunctionLabels(t *testing.T) {
	sig := Sig{}
	fn := NewFunction("test", sig)

	fn.Append(Llabel{Lbl: Label(1)})
	fn.Append(Lgoto{Target: Label(2)})
	fn.Append(Llabel{Lbl: Label(2)})
	fn.Append(Llabel{Lbl: Label(1)}) // duplicate
	fn.Append(Lreturn{})

	labels := fn.Labels()
	if len(labels) != 2 {
		t.Errorf("Labels() returned %d labels, want 2", len(labels))
	}

	// Should contain 1 and 2
	seen := make(map[Label]bool)
	for _, l := range labels {
		seen[l] = true
	}
	if !seen[Label(1)] || !seen[Label(2)] {
		t.Errorf("Labels() = %v, want [1, 2]", labels)
	}
}

func TestFunctionReferencedLabels(t *testing.T) {
	sig := Sig{}
	fn := NewFunction("test", sig)

	// Use a simple condition code type
	cond := rtl.Ccompimm{Cond: rtl.Ceq, N: 0}

	fn.Append(Llabel{Lbl: Label(1)})
	fn.Append(Lgoto{Target: Label(3)})
	fn.Append(Llabel{Lbl: Label(2)})
	fn.Append(Lcond{
		Cond: cond,
		IfSo: Label(1),
	})
	fn.Append(Ljumptable{
		Arg:     R{Reg: ltl.X0},
		Targets: []Label{Label(1), Label(2), Label(4)},
	})
	fn.Append(Lreturn{})

	refs := fn.ReferencedLabels()
	// Should reference: 3 (goto), 1 (cond), 1,2,4 (jumptable) = unique: 1,2,3,4
	if len(refs) != 4 {
		t.Errorf("ReferencedLabels() returned %d labels, want 4", len(refs))
	}

	seen := make(map[Label]bool)
	for _, l := range refs {
		seen[l] = true
	}
	for _, expected := range []Label{1, 2, 3, 4} {
		if !seen[expected] {
			t.Errorf("ReferencedLabels() missing label %d", expected)
		}
	}
}

func TestInstructionTypes(t *testing.T) {
	cond := rtl.Ccompimm{Cond: rtl.Ceq, N: 0}

	// Test that all instruction types implement the interface
	instructions := []Instruction{
		Lgetstack{Slot: SlotLocal, Ofs: 0, Ty: Tint, Dest: ltl.X0},
		Lsetstack{Src: ltl.X0, Slot: SlotLocal, Ofs: 0, Ty: Tint},
		Lop{Op: rtl.Ointconst{Value: 1}, Dest: R{Reg: ltl.X0}},
		Lload{Chunk: Mint64, Dest: R{Reg: ltl.X0}},
		Lstore{Chunk: Mint64, Src: R{Reg: ltl.X0}},
		Lcall{Sig: Sig{}, Fn: FunSymbol{Name: "foo"}},
		Ltailcall{Sig: Sig{}, Fn: FunSymbol{Name: "foo"}},
		Lbuiltin{Builtin: "memcpy"},
		Llabel{Lbl: Label(1)},
		Lgoto{Target: Label(1)},
		Lcond{Cond: cond, IfSo: Label(1)},
		Ljumptable{Arg: R{Reg: ltl.X0}, Targets: []Label{Label(1)}},
		Lreturn{},
	}

	for i, inst := range instructions {
		if inst == nil {
			t.Errorf("Instruction %d is nil", i)
		}
	}
}

func TestFunRefTypes(t *testing.T) {
	// Test that FunRef types work
	refs := []FunRef{
		FunReg{Loc: R{Reg: ltl.X0}},
		FunSymbol{Name: "test"},
	}

	for i, ref := range refs {
		if ref == nil {
			t.Errorf("FunRef %d is nil", i)
		}
	}
}
