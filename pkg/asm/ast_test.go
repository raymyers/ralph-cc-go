package asm

import "testing"

func TestCondCodeString(t *testing.T) {
	tests := []struct {
		cond CondCode
		want string
	}{
		{CondEQ, "eq"},
		{CondNE, "ne"},
		{CondCS, "cs"},
		{CondCC, "cc"},
		{CondMI, "mi"},
		{CondPL, "pl"},
		{CondVS, "vs"},
		{CondVC, "vc"},
		{CondHI, "hi"},
		{CondLS, "ls"},
		{CondGE, "ge"},
		{CondLT, "lt"},
		{CondGT, "gt"},
		{CondLE, "le"},
		{CondAL, "al"},
		{CondCode(100), "?"}, // invalid
	}
	for _, tt := range tests {
		if got := tt.cond.String(); got != tt.want {
			t.Errorf("CondCode(%d).String() = %q, want %q", tt.cond, got, tt.want)
		}
	}
}

func TestInstructionInterface(t *testing.T) {
	// Verify all instruction types implement the Instruction interface
	var _ Instruction = ADD{}
	var _ Instruction = ADDi{}
	var _ Instruction = SUB{}
	var _ Instruction = SUBi{}
	var _ Instruction = MUL{}
	var _ Instruction = MADD{}
	var _ Instruction = SMULL{}
	var _ Instruction = UMULL{}
	var _ Instruction = SDIV{}
	var _ Instruction = UDIV{}
	var _ Instruction = AND{}
	var _ Instruction = ANDi{}
	var _ Instruction = ORR{}
	var _ Instruction = ORRi{}
	var _ Instruction = EOR{}
	var _ Instruction = EORi{}
	var _ Instruction = MVN{}
	var _ Instruction = NEG{}
	var _ Instruction = LSL{}
	var _ Instruction = LSLi{}
	var _ Instruction = LSR{}
	var _ Instruction = LSRi{}
	var _ Instruction = ASR{}
	var _ Instruction = ASRi{}
	var _ Instruction = ROR{}
	var _ Instruction = RORi{}
	var _ Instruction = LDR{}
	var _ Instruction = LDRr{}
	var _ Instruction = LDRB{}
	var _ Instruction = LDRH{}
	var _ Instruction = LDRSB{}
	var _ Instruction = LDRSH{}
	var _ Instruction = LDRSW{}
	var _ Instruction = STR{}
	var _ Instruction = STRr{}
	var _ Instruction = STRB{}
	var _ Instruction = STRH{}
	var _ Instruction = LDP{}
	var _ Instruction = STP{}
	var _ Instruction = FLDRs{}
	var _ Instruction = FLDRd{}
	var _ Instruction = FSTRs{}
	var _ Instruction = FSTRd{}
	var _ Instruction = B{}
	var _ Instruction = BL{}
	var _ Instruction = BR{}
	var _ Instruction = BLR{}
	var _ Instruction = RET{}
	var _ Instruction = Bcond{}
	var _ Instruction = CMP{}
	var _ Instruction = CMPi{}
	var _ Instruction = CMN{}
	var _ Instruction = CMNi{}
	var _ Instruction = TST{}
	var _ Instruction = TSTi{}
	var _ Instruction = CSEL{}
	var _ Instruction = CSET{}
	var _ Instruction = CSINC{}
	var _ Instruction = MOV{}
	var _ Instruction = MOVi{}
	var _ Instruction = MOVZ{}
	var _ Instruction = MOVK{}
	var _ Instruction = MOVN{}
	var _ Instruction = ADR{}
	var _ Instruction = ADRP{}
	var _ Instruction = FADD{}
	var _ Instruction = FSUB{}
	var _ Instruction = FMUL{}
	var _ Instruction = FDIV{}
	var _ Instruction = FNEG{}
	var _ Instruction = FABS{}
	var _ Instruction = FSQRT{}
	var _ Instruction = FMOV{}
	var _ Instruction = FMOVi{}
	var _ Instruction = SCVTF{}
	var _ Instruction = UCVTF{}
	var _ Instruction = FCVTZS{}
	var _ Instruction = FCVTZU{}
	var _ Instruction = FCVT{}
	var _ Instruction = FCMP{}
	var _ Instruction = FCMPz{}
	var _ Instruction = SXTB{}
	var _ Instruction = SXTH{}
	var _ Instruction = SXTW{}
	var _ Instruction = UXTB{}
	var _ Instruction = UXTH{}
	var _ Instruction = LabelDef{}
}

func TestNewFunction(t *testing.T) {
	f := NewFunction("test_func")
	if f.Name != "test_func" {
		t.Errorf("Name = %q, want %q", f.Name, "test_func")
	}
	if len(f.Code) != 0 {
		t.Errorf("Code length = %d, want 0", len(f.Code))
	}
}

func TestFunctionAppend(t *testing.T) {
	f := NewFunction("test")
	f.Append(ADD{Rd: X0, Rn: X1, Rm: X2, Is64: true})
	f.Append(SUBi{Rd: X3, Rn: X4, Imm: 16, Is64: true})
	f.Append(RET{})

	if len(f.Code) != 3 {
		t.Errorf("Code length = %d, want 3", len(f.Code))
	}
}

func TestFunctionAppendLabel(t *testing.T) {
	f := NewFunction("test")
	f.AppendLabel(".L1")
	f.Append(RET{})

	if len(f.Code) != 2 {
		t.Errorf("Code length = %d, want 2", len(f.Code))
	}

	lbl, ok := f.Code[0].(LabelDef)
	if !ok {
		t.Fatal("First instruction is not LabelDef")
	}
	if lbl.Name != ".L1" {
		t.Errorf("Label name = %q, want %q", lbl.Name, ".L1")
	}
}

func TestRegisterConstants(t *testing.T) {
	// Verify register constants are correctly exported
	if X0 != 0 {
		t.Errorf("X0 = %d, want 0", X0)
	}
	if X29 != 29 {
		t.Errorf("X29 = %d, want 29", X29)
	}
	if X30 != 30 {
		t.Errorf("X30 = %d, want 30", X30)
	}
}

func TestInstructionFields(t *testing.T) {
	// Test ADD instruction
	add := ADD{Rd: X0, Rn: X1, Rm: X2, Is64: true}
	if add.Rd != X0 || add.Rn != X1 || add.Rm != X2 || !add.Is64 {
		t.Error("ADD fields not set correctly")
	}

	// Test ADDi instruction
	addi := ADDi{Rd: X3, Rn: X4, Imm: 100, Is64: false}
	if addi.Rd != X3 || addi.Rn != X4 || addi.Imm != 100 || addi.Is64 {
		t.Error("ADDi fields not set correctly")
	}

	// Test LDR instruction
	ldr := LDR{Rt: X5, Rn: X29, Ofs: -16, Is64: true}
	if ldr.Rt != X5 || ldr.Rn != X29 || ldr.Ofs != -16 || !ldr.Is64 {
		t.Error("LDR fields not set correctly")
	}

	// Test branch instruction
	b := B{Target: ".L1"}
	if b.Target != ".L1" {
		t.Errorf("B.Target = %q, want %q", b.Target, ".L1")
	}

	// Test conditional branch
	bcond := Bcond{Cond: CondEQ, Target: ".L2"}
	if bcond.Cond != CondEQ || bcond.Target != ".L2" {
		t.Error("Bcond fields not set correctly")
	}
}
