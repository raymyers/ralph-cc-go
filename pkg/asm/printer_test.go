package asm

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintArithmeticInstructions(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"ADD 64-bit", ADD{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\tadd\tx0, x1, x2\n"},
		{"ADD 32-bit", ADD{Rd: X0, Rn: X1, Rm: X2, Is64: false}, "\tadd\tw0, w1, w2\n"},
		{"ADDi 64-bit", ADDi{Rd: X0, Rn: X1, Imm: 16, Is64: true}, "\tadd\tx0, x1, #16\n"},
		{"SUB", SUB{Rd: X3, Rn: X4, Rm: X5, Is64: true}, "\tsub\tx3, x4, x5\n"},
		{"SUBi", SUBi{Rd: X3, Rn: X4, Imm: 32, Is64: true}, "\tsub\tx3, x4, #32\n"},
		{"MUL", MUL{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\tmul\tx0, x1, x2\n"},
		{"SDIV", SDIV{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\tsdiv\tx0, x1, x2\n"},
		{"UDIV", UDIV{Rd: X0, Rn: X1, Rm: X2, Is64: false}, "\tudiv\tw0, w1, w2\n"},
		{"AND", AND{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\tand\tx0, x1, x2\n"},
		{"ANDi", ANDi{Rd: X0, Rn: X1, Imm: 0xff, Is64: true}, "\tand\tx0, x1, #255\n"},
		{"ORR", ORR{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\torr\tx0, x1, x2\n"},
		{"EOR", EOR{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\teor\tx0, x1, x2\n"},
		{"MVN", MVN{Rd: X0, Rm: X1, Is64: true}, "\tmvn\tx0, x1\n"},
		{"NEG", NEG{Rd: X0, Rm: X1, Is64: true}, "\tneg\tx0, x1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.printInstruction(tt.inst)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintShiftInstructions(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"LSL reg", LSL{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\tlsl\tx0, x1, x2\n"},
		{"LSL imm", LSLi{Rd: X0, Rn: X1, Shift: 4, Is64: true}, "\tlsl\tx0, x1, #4\n"},
		{"LSR reg", LSR{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\tlsr\tx0, x1, x2\n"},
		{"LSR imm", LSRi{Rd: X0, Rn: X1, Shift: 8, Is64: true}, "\tlsr\tx0, x1, #8\n"},
		{"ASR reg", ASR{Rd: X0, Rn: X1, Rm: X2, Is64: true}, "\tasr\tx0, x1, x2\n"},
		{"ASR imm", ASRi{Rd: X0, Rn: X1, Shift: 2, Is64: true}, "\tasr\tx0, x1, #2\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.printInstruction(tt.inst)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintLoadStoreInstructions(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"LDR no offset", LDR{Rt: X0, Rn: X1, Ofs: 0, Is64: true}, "\tldr\tx0, [x1]\n"},
		{"LDR with offset", LDR{Rt: X0, Rn: X1, Ofs: 16, Is64: true}, "\tldr\tx0, [x1, #16]\n"},
		{"LDR 32-bit", LDR{Rt: X0, Rn: X1, Ofs: 8, Is64: false}, "\tldr\tw0, [x1, #8]\n"},
		{"LDRB", LDRB{Rt: X0, Rn: X1, Ofs: 4}, "\tldrb\tw0, [x1, #4]\n"},
		{"LDRH", LDRH{Rt: X0, Rn: X1, Ofs: 2}, "\tldrh\tw0, [x1, #2]\n"},
		{"LDRSB", LDRSB{Rt: X0, Rn: X1, Ofs: 0, Is64: true}, "\tldrsb\tx0, [x1]\n"},
		{"LDRSW", LDRSW{Rt: X0, Rn: X1, Ofs: 4}, "\tldrsw\tx0, [x1, #4]\n"},
		{"STR no offset", STR{Rt: X0, Rn: X1, Ofs: 0, Is64: true}, "\tstr\tx0, [x1]\n"},
		{"STR with offset", STR{Rt: X0, Rn: X1, Ofs: 24, Is64: true}, "\tstr\tx0, [x1, #24]\n"},
		{"STRB", STRB{Rt: X0, Rn: X1, Ofs: 1}, "\tstrb\tw0, [x1, #1]\n"},
		{"STRH", STRH{Rt: X0, Rn: X1, Ofs: 2}, "\tstrh\tw0, [x1, #2]\n"},
		{"LDP", LDP{Rt1: X29, Rt2: X30, Rn: X0, Ofs: 16, Is64: true}, "\tldp\tx29, x30, [x0, #16]\n"},
		{"STP", STP{Rt1: X29, Rt2: X30, Rn: X0, Ofs: 16, Is64: true}, "\tstp\tx29, x30, [x0, #16]\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.printInstruction(tt.inst)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintBranchInstructions(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"B", B{Target: ".L1"}, "\tb\t.L1\n"},
		{"BL", BL{Target: "printf"}, "\tbl\tprintf\n"},
		{"BR", BR{Rn: X0}, "\tbr\tx0\n"},
		{"BLR", BLR{Rn: X1}, "\tblr\tx1\n"},
		{"RET", RET{}, "\tret\n"},
		{"B.EQ", Bcond{Cond: CondEQ, Target: ".L2"}, "\tb.eq\t.L2\n"},
		{"B.NE", Bcond{Cond: CondNE, Target: ".L3"}, "\tb.ne\t.L3\n"},
		{"B.LT", Bcond{Cond: CondLT, Target: ".L4"}, "\tb.lt\t.L4\n"},
		{"B.GE", Bcond{Cond: CondGE, Target: ".L5"}, "\tb.ge\t.L5\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.printInstruction(tt.inst)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintCompareInstructions(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"CMP reg", CMP{Rn: X0, Rm: X1, Is64: true}, "\tcmp\tx0, x1\n"},
		{"CMP imm", CMPi{Rn: X0, Imm: 10, Is64: true}, "\tcmp\tx0, #10\n"},
		{"TST", TST{Rn: X0, Rm: X1, Is64: true}, "\ttst\tx0, x1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.printInstruction(tt.inst)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintMoveInstructions(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"MOV", MOV{Rd: X0, Rm: X1, Is64: true}, "\tmov\tx0, x1\n"},
		{"MOVi", MOVi{Rd: X0, Imm: 42, Is64: true}, "\tmov\tx0, #42\n"},
		{"MOVZ no shift", MOVZ{Rd: X0, Imm: 0x1234, Shift: 0, Is64: true}, "\tmovz\tx0, #4660\n"},
		{"MOVZ with shift", MOVZ{Rd: X0, Imm: 0x5678, Shift: 16, Is64: true}, "\tmovz\tx0, #22136, lsl #16\n"},
		{"MOVK", MOVK{Rd: X0, Imm: 0xabcd, Shift: 32, Is64: true}, "\tmovk\tx0, #43981, lsl #32\n"},
		{"CSET", CSET{Rd: X0, Cond: CondEQ, Is64: true}, "\tcset\tx0, eq\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.printInstruction(tt.inst)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintFloatInstructions(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"FADD double", FADD{Fd: D0, Fn: D1, Fm: D2, IsDouble: true}, "\tfadd\td0, d1, d2\n"},
		{"FADD single", FADD{Fd: D0, Fn: D1, Fm: D2, IsDouble: false}, "\tfadd\ts0, s1, s2\n"},
		{"FSUB", FSUB{Fd: D0, Fn: D1, Fm: D2, IsDouble: true}, "\tfsub\td0, d1, d2\n"},
		{"FMUL", FMUL{Fd: D0, Fn: D1, Fm: D2, IsDouble: true}, "\tfmul\td0, d1, d2\n"},
		{"FDIV", FDIV{Fd: D0, Fn: D1, Fm: D2, IsDouble: true}, "\tfdiv\td0, d1, d2\n"},
		{"FNEG", FNEG{Fd: D0, Fn: D1, IsDouble: true}, "\tfneg\td0, d1\n"},
		{"FABS", FABS{Fd: D0, Fn: D1, IsDouble: true}, "\tfabs\td0, d1\n"},
		{"FCMP", FCMP{Fn: D0, Fm: D1, IsDouble: true}, "\tfcmp\td0, d1\n"},
		{"SCVTF", SCVTF{Fd: D0, Rn: X0, IsDouble: true, Is64Src: true}, "\tscvtf\td0, x0\n"},
		{"FCVTZS", FCVTZS{Rd: X0, Fn: D0, IsDouble: true, Is64Dst: true}, "\tfcvtzs\tx0, d0\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.printInstruction(tt.inst)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintLabelDef(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.printInstruction(LabelDef{Name: ".L1"})
	if got := buf.String(); got != ".L1:\n" {
		t.Errorf("got %q, want %q", got, ".L1:\n")
	}
}

func TestPrintFunction(t *testing.T) {
	f := Function{
		Name: "add_one",
		Code: []Instruction{
			ADDi{Rd: X0, Rn: X0, Imm: 1, Is64: true},
			RET{},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.printFunction(f)

	output := buf.String()

	if !strings.Contains(output, ".global\tadd_one") {
		t.Error("Missing .global directive")
	}
	if !strings.Contains(output, "add_one:") {
		t.Error("Missing function label")
	}
	if !strings.Contains(output, "add\tx0, x0, #1") {
		t.Error("Missing ADD instruction")
	}
	if !strings.Contains(output, "ret") {
		t.Error("Missing RET instruction")
	}
	if !strings.Contains(output, ".size\tadd_one") {
		t.Error("Missing .size directive")
	}
}

func TestPrintProgram(t *testing.T) {
	prog := &Program{
		Globals: []GlobVar{
			{Name: "global_var", Size: 8, Align: 8},
		},
		Functions: []Function{
			{
				Name: "main",
				Code: []Instruction{
					MOVi{Rd: X0, Imm: 0, Is64: true},
					RET{},
				},
			},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintProgram(prog)

	output := buf.String()

	if !strings.Contains(output, ".data") {
		t.Error("Missing .data section")
	}
	if !strings.Contains(output, ".global\tglobal_var") {
		t.Error("Missing global variable directive")
	}
	if !strings.Contains(output, ".text") {
		t.Error("Missing .text section")
	}
	if !strings.Contains(output, ".global\tmain") {
		t.Error("Missing main function directive")
	}
}

func TestPrintExtensionInstructions(t *testing.T) {
	tests := []struct {
		name string
		inst Instruction
		want string
	}{
		{"SXTB 32", SXTB{Rd: X0, Rn: X1, Is64: false}, "\tsxtb\tw0, w1\n"},
		{"SXTB 64", SXTB{Rd: X0, Rn: X1, Is64: true}, "\tsxtb\tx0, w1\n"},
		{"SXTH", SXTH{Rd: X0, Rn: X1, Is64: true}, "\tsxth\tx0, w1\n"},
		{"SXTW", SXTW{Rd: X0, Rn: X1}, "\tsxtw\tx0, w1\n"},
		{"UXTB", UXTB{Rd: X0, Rn: X1}, "\tuxtb\tw0, w1\n"},
		{"UXTH", UXTH{Rd: X0, Rn: X1}, "\tuxth\tw0, w1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.printInstruction(tt.inst)
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
