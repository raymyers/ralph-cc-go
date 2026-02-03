package stacking

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
)

func TestIsCalleeSaved(t *testing.T) {
	tests := []struct {
		reg  ltl.MReg
		want bool
	}{
		{ltl.X0, false},
		{ltl.X18, false},
		{ltl.X19, true},
		{ltl.X28, true},
		{ltl.X29, false}, // FP is not callee-saved (handled specially)
		{ltl.X30, false}, // LR
		{ltl.D0, false},
		{ltl.D7, false},
		{ltl.D8, true},
		{ltl.D15, true},
		{ltl.D16, false},
	}

	for _, tt := range tests {
		got := IsCalleeSaved(tt.reg)
		if got != tt.want {
			t.Errorf("IsCalleeSaved(%v) = %v, want %v", tt.reg, got, tt.want)
		}
	}
}

func TestFindUsedCalleeSaveRegsEmpty(t *testing.T) {
	fn := linear.NewFunction("empty", linear.Sig{})
	regs := FindUsedCalleeSaveRegs(fn)

	if len(regs) != 0 {
		t.Errorf("expected no callee-saved regs, got %v", regs)
	}
}

func TestFindUsedCalleeSaveRegsNoCalleeSaved(t *testing.T) {
	fn := linear.NewFunction("noCalleeSaved", linear.Sig{})
	// Uses only caller-saved registers
	fn.Append(linear.Lop{
		Op:   nil,
		Args: []linear.Loc{linear.R{Reg: ltl.X0}},
		Dest: linear.R{Reg: ltl.X1},
	})

	regs := FindUsedCalleeSaveRegs(fn)

	if len(regs) != 0 {
		t.Errorf("expected no callee-saved regs, got %v", regs)
	}
}

func TestFindUsedCalleeSaveRegsWithCalleeSaved(t *testing.T) {
	fn := linear.NewFunction("withCalleeSaved", linear.Sig{})
	// Uses X19 (callee-saved)
	fn.Append(linear.Lop{
		Op:   nil,
		Args: []linear.Loc{linear.R{Reg: ltl.X19}},
		Dest: linear.R{Reg: ltl.X0},
	})

	regs := FindUsedCalleeSaveRegs(fn)

	if len(regs) != 1 || regs[0] != ltl.X19 {
		t.Errorf("expected [X19], got %v", regs)
	}
}

func TestFindUsedCalleeSaveRegsMultiple(t *testing.T) {
	fn := linear.NewFunction("multiple", linear.Sig{})
	fn.Append(linear.Lop{
		Op:   nil,
		Args: []linear.Loc{linear.R{Reg: ltl.X19}},
		Dest: linear.R{Reg: ltl.X20},
	})
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotLocal,
		Ofs:  0,
		Ty:   linear.Tlong,
		Dest: ltl.X21,
	})

	regs := FindUsedCalleeSaveRegs(fn)

	if len(regs) != 3 {
		t.Errorf("expected 3 callee-saved regs, got %d: %v", len(regs), regs)
	}
	// Should be sorted
	expected := []ltl.MReg{ltl.X19, ltl.X20, ltl.X21}
	for i, r := range expected {
		if regs[i] != r {
			t.Errorf("regs[%d] = %v, want %v", i, regs[i], r)
		}
	}
}

func TestFindUsedCalleeSaveRegsFloat(t *testing.T) {
	fn := linear.NewFunction("float", linear.Sig{})
	fn.Append(linear.Lop{
		Op:   nil,
		Args: []linear.Loc{linear.R{Reg: ltl.D8}},
		Dest: linear.R{Reg: ltl.D0},
	})

	regs := FindUsedCalleeSaveRegs(fn)

	if len(regs) != 1 || regs[0] != ltl.D8 {
		t.Errorf("expected [D8], got %v", regs)
	}
}

func TestFindUsedCalleeSaveRegsLstore(t *testing.T) {
	fn := linear.NewFunction("store", linear.Sig{})
	fn.Append(linear.Lstore{
		Chunk: linear.Mint64,
		Addr:  nil,
		Args:  []linear.Loc{linear.R{Reg: ltl.X0}},
		Src:   linear.R{Reg: ltl.X19},
	})

	regs := FindUsedCalleeSaveRegs(fn)

	if len(regs) != 1 || regs[0] != ltl.X19 {
		t.Errorf("expected [X19], got %v", regs)
	}
}

func TestFindUsedCalleeSaveRegsLcond(t *testing.T) {
	fn := linear.NewFunction("cond", linear.Sig{})
	fn.Append(linear.Lcond{
		Args: []linear.Loc{linear.R{Reg: ltl.X19}},
		IfSo: 1,
	})

	regs := FindUsedCalleeSaveRegs(fn)

	if len(regs) != 1 || regs[0] != ltl.X19 {
		t.Errorf("expected [X19], got %v", regs)
	}
}

func TestFindUsedCalleeSaveRegsLjumptable(t *testing.T) {
	fn := linear.NewFunction("jump", linear.Sig{})
	fn.Append(linear.Ljumptable{
		Arg:     linear.R{Reg: ltl.X20},
		Targets: []linear.Label{1, 2},
	})

	regs := FindUsedCalleeSaveRegs(fn)

	if len(regs) != 1 || regs[0] != ltl.X20 {
		t.Errorf("expected [X20], got %v", regs)
	}
}

func TestComputeCalleeSaveInfo(t *testing.T) {
	fn := linear.NewFunction("test", linear.Sig{})
	layout := ComputeLayout(fn, 2) // 2 callee-save regs

	usedRegs := []ltl.MReg{ltl.X19, ltl.X20}
	info := ComputeCalleeSaveInfo(layout, usedRegs)

	if len(info.Regs) != 2 {
		t.Errorf("expected 2 regs, got %d", len(info.Regs))
	}
	if len(info.SaveOffsets) != 2 {
		t.Errorf("expected 2 offsets, got %d", len(info.SaveOffsets))
	}

	// First reg at CalleeSaveOffset
	if info.SaveOffsets[0] != layout.CalleeSaveOffset {
		t.Errorf("SaveOffsets[0] = %d, want %d", info.SaveOffsets[0], layout.CalleeSaveOffset)
	}
	// Second reg 8 bytes higher (positive offsets from FP)
	if info.SaveOffsets[1] != layout.CalleeSaveOffset+8 {
		t.Errorf("SaveOffsets[1] = %d, want %d", info.SaveOffsets[1], layout.CalleeSaveOffset+8)
	}
}

func TestPadToEvenEven(t *testing.T) {
	regs := []ltl.MReg{ltl.X19, ltl.X20}
	padded := PadToEven(regs)

	if len(padded) != 2 {
		t.Errorf("expected 2 regs, got %d", len(padded))
	}
}

func TestPadToEvenOdd(t *testing.T) {
	regs := []ltl.MReg{ltl.X19}
	padded := PadToEven(regs)

	if len(padded) != 2 {
		t.Errorf("expected 2 regs, got %d", len(padded))
	}
	if padded[0] != ltl.X19 || padded[1] != ltl.X19 {
		t.Errorf("expected [X19, X19], got %v", padded)
	}
}

func TestPadToEvenEmpty(t *testing.T) {
	regs := []ltl.MReg{}
	padded := PadToEven(regs)

	if len(padded) != 0 {
		t.Errorf("expected 0 regs, got %d", len(padded))
	}
}

func TestSortRegs(t *testing.T) {
	regs := []ltl.MReg{ltl.X21, ltl.X19, ltl.X20}
	sortRegs(regs)

	expected := []ltl.MReg{ltl.X19, ltl.X20, ltl.X21}
	for i, r := range expected {
		if regs[i] != r {
			t.Errorf("regs[%d] = %v, want %v", i, regs[i], r)
		}
	}
}
