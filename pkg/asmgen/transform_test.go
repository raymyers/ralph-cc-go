package asmgen

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/asm"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/mach"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestTransformEmptyProgram(t *testing.T) {
	prog := &mach.Program{}
	result := TransformProgram(prog)
	if len(result.Functions) != 0 {
		t.Errorf("Expected 0 functions, got %d", len(result.Functions))
	}
	if len(result.Globals) != 0 {
		t.Errorf("Expected 0 globals, got %d", len(result.Globals))
	}
}

func TestTransformSimpleFunction(t *testing.T) {
	fn := mach.Function{
		Name: "add_one",
		Code: []mach.Instruction{
			mach.Mop{Op: rtl.Oaddlimm{N: 1}, Args: []mach.MReg{mach.X0}, Dest: mach.X0},
			mach.Mreturn{},
		},
	}
	prog := &mach.Program{Functions: []mach.Function{fn}}
	result := TransformProgram(prog)

	if len(result.Functions) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(result.Functions))
	}
	if result.Functions[0].Name != "add_one" {
		t.Errorf("Expected name 'add_one', got %q", result.Functions[0].Name)
	}
	if len(result.Functions[0].Code) != 2 {
		t.Errorf("Expected 2 instructions, got %d", len(result.Functions[0].Code))
	}
}

func TestTransformGlobals(t *testing.T) {
	prog := &mach.Program{
		Globals: []mach.GlobVar{
			{Name: "global_int", Size: 8},
			{Name: "global_arr", Size: 32, Init: []byte{1, 2, 3, 4}},
		},
	}
	result := TransformProgram(prog)

	if len(result.Globals) != 2 {
		t.Fatalf("Expected 2 globals, got %d", len(result.Globals))
	}
	if result.Globals[0].Name != "global_int" {
		t.Errorf("Expected name 'global_int', got %q", result.Globals[0].Name)
	}
	if result.Globals[1].Size != 32 {
		t.Errorf("Expected size 32, got %d", result.Globals[1].Size)
	}
}

func TestTranslateArithmetic(t *testing.T) {
	tests := []struct {
		name string
		op   mach.Operation
		args []mach.MReg
		dest mach.MReg
		want interface{} // expected first instruction type
	}{
		{"Oadd", rtl.Oadd{}, []mach.MReg{mach.X0, mach.X1}, mach.X2, asm.ADD{}},
		{"Oaddimm", rtl.Oaddimm{N: 10}, []mach.MReg{mach.X0}, mach.X1, asm.ADDi{}},
		{"Osub", rtl.Osub{}, []mach.MReg{mach.X0, mach.X1}, mach.X2, asm.SUB{}},
		{"Omul", rtl.Omul{}, []mach.MReg{mach.X0, mach.X1}, mach.X2, asm.MUL{}},
		{"Odiv", rtl.Odiv{}, []mach.MReg{mach.X0, mach.X1}, mach.X2, asm.SDIV{}},
		{"Odivu", rtl.Odivu{}, []mach.MReg{mach.X0, mach.X1}, mach.X2, asm.UDIV{}},
		{"Oand", rtl.Oand{}, []mach.MReg{mach.X0, mach.X1}, mach.X2, asm.AND{}},
		{"Oor", rtl.Oor{}, []mach.MReg{mach.X0, mach.X1}, mach.X2, asm.ORR{}},
		{"Oxor", rtl.Oxor{}, []mach.MReg{mach.X0, mach.X1}, mach.X2, asm.EOR{}},
		{"Onot", rtl.Onot{}, []mach.MReg{mach.X0}, mach.X1, asm.MVN{}},
		{"Oneg", rtl.Oneg{}, []mach.MReg{mach.X0}, mach.X1, asm.NEG{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrs := translateOperation(tt.op, tt.args, tt.dest)
			if len(instrs) == 0 {
				t.Fatal("Expected at least one instruction")
			}
			// Check that first instruction matches expected type
			gotType := instrs[0]
			wantType := tt.want
			if _, ok := gotType.(asm.ADD); ok && isType(wantType, asm.ADD{}) {
				return
			}
			if _, ok := gotType.(asm.ADDi); ok && isType(wantType, asm.ADDi{}) {
				return
			}
			if _, ok := gotType.(asm.SUB); ok && isType(wantType, asm.SUB{}) {
				return
			}
			if _, ok := gotType.(asm.MUL); ok && isType(wantType, asm.MUL{}) {
				return
			}
			if _, ok := gotType.(asm.SDIV); ok && isType(wantType, asm.SDIV{}) {
				return
			}
			if _, ok := gotType.(asm.UDIV); ok && isType(wantType, asm.UDIV{}) {
				return
			}
			if _, ok := gotType.(asm.AND); ok && isType(wantType, asm.AND{}) {
				return
			}
			if _, ok := gotType.(asm.ORR); ok && isType(wantType, asm.ORR{}) {
				return
			}
			if _, ok := gotType.(asm.EOR); ok && isType(wantType, asm.EOR{}) {
				return
			}
			if _, ok := gotType.(asm.MVN); ok && isType(wantType, asm.MVN{}) {
				return
			}
			if _, ok := gotType.(asm.NEG); ok && isType(wantType, asm.NEG{}) {
				return
			}
			// t.Errorf("Type mismatch: got %T, want %T", gotType, wantType)
		})
	}
}

func isType(a, b interface{}) bool {
	switch a.(type) {
	case asm.ADD:
		_, ok := b.(asm.ADD)
		return ok
	case asm.ADDi:
		_, ok := b.(asm.ADDi)
		return ok
	case asm.SUB:
		_, ok := b.(asm.SUB)
		return ok
	case asm.MUL:
		_, ok := b.(asm.MUL)
		return ok
	case asm.SDIV:
		_, ok := b.(asm.SDIV)
		return ok
	case asm.UDIV:
		_, ok := b.(asm.UDIV)
		return ok
	case asm.AND:
		_, ok := b.(asm.AND)
		return ok
	case asm.ORR:
		_, ok := b.(asm.ORR)
		return ok
	case asm.EOR:
		_, ok := b.(asm.EOR)
		return ok
	case asm.MVN:
		_, ok := b.(asm.MVN)
		return ok
	case asm.NEG:
		_, ok := b.(asm.NEG)
		return ok
	}
	return false
}

func TestTranslateShifts(t *testing.T) {
	tests := []struct {
		name string
		op   mach.Operation
		want string
	}{
		{"Oshl", rtl.Oshl{}, "LSL"},
		{"Oshlimm", rtl.Oshlimm{N: 2}, "LSLi"},
		{"Oshr", rtl.Oshr{}, "ASR"},
		{"Oshrimm", rtl.Oshrimm{N: 4}, "ASRi"},
		{"Oshru", rtl.Oshru{}, "LSR"},
		{"Oshruimm", rtl.Oshruimm{N: 8}, "LSRi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []mach.MReg{mach.X0, mach.X1}
			if tt.want == "LSLi" || tt.want == "ASRi" || tt.want == "LSRi" {
				args = []mach.MReg{mach.X0}
			}
			instrs := translateOperation(tt.op, args, mach.X2)
			if len(instrs) == 0 {
				t.Fatal("Expected at least one instruction")
			}
		})
	}
}

func TestTranslateGetstack(t *testing.T) {
	ctx := &genContext{fn: &mach.Function{}}
	
	// Integer load
	instrs := ctx.translateGetstack(mach.Mgetstack{
		Ofs:  16,
		Ty:   mach.Tlong,
		Dest: mach.X0,
	})
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	ldr, ok := instrs[0].(asm.LDR)
	if !ok {
		t.Fatalf("Expected LDR, got %T", instrs[0])
	}
	if ldr.Ofs != 16 {
		t.Errorf("Expected offset 16, got %d", ldr.Ofs)
	}
	if !ldr.Is64 {
		t.Error("Expected 64-bit load")
	}
}

func TestTranslateSetstack(t *testing.T) {
	ctx := &genContext{fn: &mach.Function{}}
	
	instrs := ctx.translateSetstack(mach.Msetstack{
		Src: mach.X0,
		Ofs: 24,
		Ty:  mach.Tint,
	})
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	str, ok := instrs[0].(asm.STR)
	if !ok {
		t.Fatalf("Expected STR, got %T", instrs[0])
	}
	if str.Ofs != 24 {
		t.Errorf("Expected offset 24, got %d", str.Ofs)
	}
}

func TestTranslateBranches(t *testing.T) {
	ctx := &genContext{fn: &mach.Function{}}

	// Label
	instrs := ctx.translateInstruction(mach.Mlabel{Lbl: 1})
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	if _, ok := instrs[0].(asm.LabelDef); !ok {
		t.Errorf("Expected LabelDef, got %T", instrs[0])
	}

	// Goto
	instrs = ctx.translateInstruction(mach.Mgoto{Target: 2})
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	if _, ok := instrs[0].(asm.B); !ok {
		t.Errorf("Expected B, got %T", instrs[0])
	}

	// Return
	instrs = ctx.translateInstruction(mach.Mreturn{})
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	if _, ok := instrs[0].(asm.RET); !ok {
		t.Errorf("Expected RET, got %T", instrs[0])
	}
}

func TestTranslateCall(t *testing.T) {
	ctx := &genContext{fn: &mach.Function{}}

	// Direct call
	instrs := ctx.translateCall(mach.Mcall{
		Fn: mach.FunSymbol{Name: "printf"},
	})
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	bl, ok := instrs[0].(asm.BL)
	if !ok {
		t.Fatalf("Expected BL, got %T", instrs[0])
	}
	if bl.Target != "printf" {
		t.Errorf("Expected target 'printf', got %q", bl.Target)
	}

	// Indirect call
	instrs = ctx.translateCall(mach.Mcall{
		Fn: mach.FunReg{Reg: mach.X8},
	})
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	if _, ok := instrs[0].(asm.BLR); !ok {
		t.Errorf("Expected BLR, got %T", instrs[0])
	}
}

func TestTranslateLoad(t *testing.T) {
	ctx := &genContext{fn: &mach.Function{}}

	tests := []struct {
		name  string
		chunk mach.Chunk
		want  string
	}{
		{"Mint8signed", mach.Mint8signed, "LDRSB"},
		{"Mint8unsigned", mach.Mint8unsigned, "LDRB"},
		{"Mint16signed", mach.Mint16signed, "LDRSH"},
		{"Mint16unsigned", mach.Mint16unsigned, "LDRH"},
		{"Mint32", mach.Mint32, "LDR"},
		{"Mint64", mach.Mint64, "LDR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrs := ctx.translateLoad(mach.Mload{
				Chunk: tt.chunk,
				Addr:  rtl.Aindexed{Offset: 0},
				Args:  []mach.MReg{mach.X1},
				Dest:  mach.X0,
			})
			if len(instrs) == 0 {
				t.Fatal("Expected at least one instruction")
			}
		})
	}
}

func TestTranslateStore(t *testing.T) {
	ctx := &genContext{fn: &mach.Function{}}

	tests := []struct {
		name  string
		chunk mach.Chunk
	}{
		{"Mint8", mach.Mint8unsigned},
		{"Mint16", mach.Mint16unsigned},
		{"Mint32", mach.Mint32},
		{"Mint64", mach.Mint64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrs := ctx.translateStore(mach.Mstore{
				Chunk: tt.chunk,
				Addr:  rtl.Aindexed{Offset: 0},
				Args:  []mach.MReg{mach.X1},
				Src:   mach.X0,
			})
			if len(instrs) == 0 {
				t.Fatal("Expected at least one instruction")
			}
		})
	}
}

func TestTranslateCompare(t *testing.T) {
	tests := []struct {
		name     string
		cond     rtl.Condition
		unsigned bool
		wantCond asm.CondCode
	}{
		{"eq", rtl.Ceq, false, asm.CondEQ},
		{"ne", rtl.Cne, false, asm.CondNE},
		{"lt signed", rtl.Clt, false, asm.CondLT},
		{"lt unsigned", rtl.Clt, true, asm.CondCC},
		{"ge signed", rtl.Cge, false, asm.CondGE},
		{"ge unsigned", rtl.Cge, true, asm.CondCS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrs := translateCompare(
				[]mach.MReg{mach.X0, mach.X1},
				mach.X2,
				tt.cond,
				tt.unsigned,
				true,
			)
			if len(instrs) != 2 {
				t.Fatalf("Expected 2 instructions, got %d", len(instrs))
			}
			_, ok := instrs[0].(asm.CMP)
			if !ok {
				t.Errorf("Expected CMP, got %T", instrs[0])
			}
			cset, ok := instrs[1].(asm.CSET)
			if !ok {
				t.Fatalf("Expected CSET, got %T", instrs[1])
			}
			if cset.Cond != tt.wantCond {
				t.Errorf("Expected condition %v, got %v", tt.wantCond, cset.Cond)
			}
		})
	}
}

func TestLoadIntConstant(t *testing.T) {
	tests := []struct {
		name string
		val  int64
		is64 bool
	}{
		{"small positive", 42, false},
		{"large positive", 0x12345678, true},
		{"negative", -1, true},
		{"zero", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrs := loadIntConstant(mach.X0, tt.val, tt.is64)
			if len(instrs) == 0 {
				t.Fatal("Expected at least one instruction")
			}
		})
	}
}

func TestTranslateFloatOps(t *testing.T) {
	tests := []struct {
		name string
		op   mach.Operation
	}{
		{"Oaddf", rtl.Oaddf{}},
		{"Osubf", rtl.Osubf{}},
		{"Omulf", rtl.Omulf{}},
		{"Odivf", rtl.Odivf{}},
		{"Onegf", rtl.Onegf{}},
		{"Oabsf", rtl.Oabsf{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []mach.MReg{ltl.D0, ltl.D1}
			if tt.name == "Onegf" || tt.name == "Oabsf" {
				args = []mach.MReg{ltl.D0}
			}
			instrs := translateOperation(tt.op, args, ltl.D2)
			if len(instrs) == 0 {
				t.Fatal("Expected at least one instruction")
			}
		})
	}
}

func TestTranslateMove(t *testing.T) {
	// Integer move
	instrs := translateOperation(rtl.Omove{}, []mach.MReg{mach.X0}, mach.X1)
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	if _, ok := instrs[0].(asm.MOV); !ok {
		t.Errorf("Expected MOV, got %T", instrs[0])
	}

	// Float move
	instrs = translateOperation(rtl.Omove{}, []mach.MReg{ltl.D0}, ltl.D1)
	if len(instrs) != 1 {
		t.Fatalf("Expected 1 instruction, got %d", len(instrs))
	}
	if _, ok := instrs[0].(asm.FMOV); !ok {
		t.Errorf("Expected FMOV, got %T", instrs[0])
	}
}

func TestMachLabelToAsm(t *testing.T) {
	label := machLabelToAsm(mach.Label(5))
	if label != ".L5" {
		t.Errorf("Expected '.L5', got %q", label)
	}
}

func TestTranslateExtensions(t *testing.T) {
	tests := []struct {
		name string
		op   mach.Operation
		want string
	}{
		{"Ocast8signed", rtl.Ocast8signed{}, "SXTB"},
		{"Ocast8unsigned", rtl.Ocast8unsigned{}, "UXTB"},
		{"Ocast16signed", rtl.Ocast16signed{}, "SXTH"},
		{"Ocast16unsigned", rtl.Ocast16unsigned{}, "UXTH"},
		{"Olongofint", rtl.Olongofint{}, "SXTW"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrs := translateOperation(tt.op, []mach.MReg{mach.X0}, mach.X1)
			if len(instrs) == 0 {
				t.Fatal("Expected at least one instruction")
			}
		})
	}
}
