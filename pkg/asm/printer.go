package asm

import (
	"fmt"
	"io"
)

// Printer outputs ARM64 assembly in GNU as syntax
type Printer struct {
	w io.Writer
}

// NewPrinter creates a new assembly printer
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w}
}

// PrintProgram outputs an entire program
func (p *Printer) PrintProgram(prog *Program) {
	// Output global variables
	if len(prog.Globals) > 0 {
		fmt.Fprintf(p.w, "\t.data\n")
		for _, g := range prog.Globals {
			p.printGlobal(g)
		}
		fmt.Fprintf(p.w, "\n")
	}

	// Output functions
	fmt.Fprintf(p.w, "\t.text\n")
	for _, f := range prog.Functions {
		p.printFunction(f)
	}
}

func (p *Printer) printGlobal(g GlobVar) {
	fmt.Fprintf(p.w, "\t.global\t%s\n", g.Name)
	if g.Align > 1 {
		fmt.Fprintf(p.w, "\t.align\t%d\n", g.Align)
	}
	fmt.Fprintf(p.w, "%s:\n", g.Name)
	if len(g.Init) > 0 {
		for _, b := range g.Init {
			fmt.Fprintf(p.w, "\t.byte\t%d\n", b)
		}
	} else if g.Size > 0 {
		fmt.Fprintf(p.w, "\t.zero\t%d\n", g.Size)
	}
}

func (p *Printer) printFunction(f Function) {
	fmt.Fprintf(p.w, "\t.align\t2\n")
	fmt.Fprintf(p.w, "\t.global\t%s\n", f.Name)
	fmt.Fprintf(p.w, "\t.type\t%s, %%function\n", f.Name)
	fmt.Fprintf(p.w, "%s:\n", f.Name)

	for _, inst := range f.Code {
		p.printInstruction(inst)
	}

	fmt.Fprintf(p.w, "\t.size\t%s, .-%s\n", f.Name, f.Name)
	fmt.Fprintf(p.w, "\n")
}

// regName32 returns the 32-bit register name
func regName32(r MReg) string {
	if r.IsFloat() {
		return fmt.Sprintf("s%d", r-D0)
	}
	if r == X29 {
		return "w29"
	}
	if r == X30 {
		return "w30"
	}
	return fmt.Sprintf("w%d", r)
}

// regName64 returns the 64-bit register name
func regName64(r MReg) string {
	if r.IsFloat() {
		return fmt.Sprintf("d%d", r-D0)
	}
	if r == X29 {
		return "x29"
	}
	if r == X30 {
		return "x30"
	}
	return fmt.Sprintf("x%d", r)
}

// regName returns register name based on Is64 flag
func regName(r MReg, is64 bool) string {
	if is64 {
		return regName64(r)
	}
	return regName32(r)
}

// floatRegName returns the float register name
func floatRegName(r MReg, isDouble bool) string {
	idx := r - D0
	if isDouble {
		return fmt.Sprintf("d%d", idx)
	}
	return fmt.Sprintf("s%d", idx)
}

func (p *Printer) printInstruction(inst Instruction) {
	switch i := inst.(type) {
	// Labels
	case LabelDef:
		fmt.Fprintf(p.w, "%s:\n", i.Name)
		return

	// Data processing
	case ADD:
		fmt.Fprintf(p.w, "\tadd\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case ADDi:
		fmt.Fprintf(p.w, "\tadd\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Imm)
	case SUB:
		fmt.Fprintf(p.w, "\tsub\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case SUBi:
		fmt.Fprintf(p.w, "\tsub\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Imm)
	case MUL:
		fmt.Fprintf(p.w, "\tmul\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case MADD:
		fmt.Fprintf(p.w, "\tmadd\t%s, %s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64), regName(i.Ra, i.Is64))
	case SMULL:
		fmt.Fprintf(p.w, "\tsmull\t%s, %s, %s\n", regName64(i.Rd), regName32(i.Rn), regName32(i.Rm))
	case UMULL:
		fmt.Fprintf(p.w, "\tumull\t%s, %s, %s\n", regName64(i.Rd), regName32(i.Rn), regName32(i.Rm))
	case SDIV:
		fmt.Fprintf(p.w, "\tsdiv\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case UDIV:
		fmt.Fprintf(p.w, "\tudiv\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case AND:
		fmt.Fprintf(p.w, "\tand\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case ANDi:
		fmt.Fprintf(p.w, "\tand\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Imm)
	case ORR:
		fmt.Fprintf(p.w, "\torr\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case ORRi:
		fmt.Fprintf(p.w, "\torr\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Imm)
	case EOR:
		fmt.Fprintf(p.w, "\teor\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case EORi:
		fmt.Fprintf(p.w, "\teor\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Imm)
	case MVN:
		fmt.Fprintf(p.w, "\tmvn\t%s, %s\n", regName(i.Rd, i.Is64), regName(i.Rm, i.Is64))
	case NEG:
		fmt.Fprintf(p.w, "\tneg\t%s, %s\n", regName(i.Rd, i.Is64), regName(i.Rm, i.Is64))

	// Shifts
	case LSL:
		fmt.Fprintf(p.w, "\tlsl\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case LSLi:
		fmt.Fprintf(p.w, "\tlsl\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Shift)
	case LSR:
		fmt.Fprintf(p.w, "\tlsr\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case LSRi:
		fmt.Fprintf(p.w, "\tlsr\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Shift)
	case ASR:
		fmt.Fprintf(p.w, "\tasr\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case ASRi:
		fmt.Fprintf(p.w, "\tasr\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Shift)
	case ROR:
		fmt.Fprintf(p.w, "\tror\t%s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case RORi:
		fmt.Fprintf(p.w, "\tror\t%s, %s, #%d\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), i.Shift)

	// Load/store integer
	case LDR:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldr\t%s, [%s]\n", regName(i.Rt, i.Is64), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldr\t%s, [%s, #%d]\n", regName(i.Rt, i.Is64), regName64(i.Rn), i.Ofs)
		}
	case LDRr:
		fmt.Fprintf(p.w, "\tldr\t%s, [%s, %s]\n", regName(i.Rt, i.Is64), regName64(i.Rn), regName64(i.Rm))
	case LDRB:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldrb\t%s, [%s]\n", regName32(i.Rt), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldrb\t%s, [%s, #%d]\n", regName32(i.Rt), regName64(i.Rn), i.Ofs)
		}
	case LDRH:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldrh\t%s, [%s]\n", regName32(i.Rt), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldrh\t%s, [%s, #%d]\n", regName32(i.Rt), regName64(i.Rn), i.Ofs)
		}
	case LDRSB:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldrsb\t%s, [%s]\n", regName(i.Rt, i.Is64), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldrsb\t%s, [%s, #%d]\n", regName(i.Rt, i.Is64), regName64(i.Rn), i.Ofs)
		}
	case LDRSH:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldrsh\t%s, [%s]\n", regName(i.Rt, i.Is64), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldrsh\t%s, [%s, #%d]\n", regName(i.Rt, i.Is64), regName64(i.Rn), i.Ofs)
		}
	case LDRSW:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldrsw\t%s, [%s]\n", regName64(i.Rt), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldrsw\t%s, [%s, #%d]\n", regName64(i.Rt), regName64(i.Rn), i.Ofs)
		}
	case STR:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tstr\t%s, [%s]\n", regName(i.Rt, i.Is64), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tstr\t%s, [%s, #%d]\n", regName(i.Rt, i.Is64), regName64(i.Rn), i.Ofs)
		}
	case STRr:
		fmt.Fprintf(p.w, "\tstr\t%s, [%s, %s]\n", regName(i.Rt, i.Is64), regName64(i.Rn), regName64(i.Rm))
	case STRB:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tstrb\t%s, [%s]\n", regName32(i.Rt), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tstrb\t%s, [%s, #%d]\n", regName32(i.Rt), regName64(i.Rn), i.Ofs)
		}
	case STRH:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tstrh\t%s, [%s]\n", regName32(i.Rt), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tstrh\t%s, [%s, #%d]\n", regName32(i.Rt), regName64(i.Rn), i.Ofs)
		}
	case LDP:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldp\t%s, %s, [%s]\n", regName(i.Rt1, i.Is64), regName(i.Rt2, i.Is64), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldp\t%s, %s, [%s, #%d]\n", regName(i.Rt1, i.Is64), regName(i.Rt2, i.Is64), regName64(i.Rn), i.Ofs)
		}
	case STP:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tstp\t%s, %s, [%s]\n", regName(i.Rt1, i.Is64), regName(i.Rt2, i.Is64), regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tstp\t%s, %s, [%s, #%d]\n", regName(i.Rt1, i.Is64), regName(i.Rt2, i.Is64), regName64(i.Rn), i.Ofs)
		}

	// Load/store float
	case FLDRs:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldr\ts%d, [%s]\n", i.Ft-D0, regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldr\ts%d, [%s, #%d]\n", i.Ft-D0, regName64(i.Rn), i.Ofs)
		}
	case FLDRd:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tldr\td%d, [%s]\n", i.Ft-D0, regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tldr\td%d, [%s, #%d]\n", i.Ft-D0, regName64(i.Rn), i.Ofs)
		}
	case FSTRs:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tstr\ts%d, [%s]\n", i.Ft-D0, regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tstr\ts%d, [%s, #%d]\n", i.Ft-D0, regName64(i.Rn), i.Ofs)
		}
	case FSTRd:
		if i.Ofs == 0 {
			fmt.Fprintf(p.w, "\tstr\td%d, [%s]\n", i.Ft-D0, regName64(i.Rn))
		} else {
			fmt.Fprintf(p.w, "\tstr\td%d, [%s, #%d]\n", i.Ft-D0, regName64(i.Rn), i.Ofs)
		}

	// Branches
	case B:
		fmt.Fprintf(p.w, "\tb\t%s\n", i.Target)
	case BL:
		fmt.Fprintf(p.w, "\tbl\t%s\n", i.Target)
	case BR:
		fmt.Fprintf(p.w, "\tbr\t%s\n", regName64(i.Rn))
	case BLR:
		fmt.Fprintf(p.w, "\tblr\t%s\n", regName64(i.Rn))
	case RET:
		fmt.Fprintf(p.w, "\tret\n")
	case Bcond:
		fmt.Fprintf(p.w, "\tb.%s\t%s\n", i.Cond.String(), i.Target)

	// Compares
	case CMP:
		fmt.Fprintf(p.w, "\tcmp\t%s, %s\n", regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case CMPi:
		fmt.Fprintf(p.w, "\tcmp\t%s, #%d\n", regName(i.Rn, i.Is64), i.Imm)
	case CMN:
		fmt.Fprintf(p.w, "\tcmn\t%s, %s\n", regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case CMNi:
		fmt.Fprintf(p.w, "\tcmn\t%s, #%d\n", regName(i.Rn, i.Is64), i.Imm)
	case TST:
		fmt.Fprintf(p.w, "\ttst\t%s, %s\n", regName(i.Rn, i.Is64), regName(i.Rm, i.Is64))
	case TSTi:
		fmt.Fprintf(p.w, "\ttst\t%s, #%d\n", regName(i.Rn, i.Is64), i.Imm)

	// Conditional select
	case CSEL:
		fmt.Fprintf(p.w, "\tcsel\t%s, %s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64), i.Cond.String())
	case CSET:
		fmt.Fprintf(p.w, "\tcset\t%s, %s\n", regName(i.Rd, i.Is64), i.Cond.String())
	case CSINC:
		fmt.Fprintf(p.w, "\tcsinc\t%s, %s, %s, %s\n", regName(i.Rd, i.Is64), regName(i.Rn, i.Is64), regName(i.Rm, i.Is64), i.Cond.String())

	// Moves
	case MOV:
		fmt.Fprintf(p.w, "\tmov\t%s, %s\n", regName(i.Rd, i.Is64), regName(i.Rm, i.Is64))
	case MOVi:
		fmt.Fprintf(p.w, "\tmov\t%s, #%d\n", regName(i.Rd, i.Is64), i.Imm)
	case MOVZ:
		if i.Shift == 0 {
			fmt.Fprintf(p.w, "\tmovz\t%s, #%d\n", regName(i.Rd, i.Is64), i.Imm)
		} else {
			fmt.Fprintf(p.w, "\tmovz\t%s, #%d, lsl #%d\n", regName(i.Rd, i.Is64), i.Imm, i.Shift)
		}
	case MOVK:
		if i.Shift == 0 {
			fmt.Fprintf(p.w, "\tmovk\t%s, #%d\n", regName(i.Rd, i.Is64), i.Imm)
		} else {
			fmt.Fprintf(p.w, "\tmovk\t%s, #%d, lsl #%d\n", regName(i.Rd, i.Is64), i.Imm, i.Shift)
		}
	case MOVN:
		if i.Shift == 0 {
			fmt.Fprintf(p.w, "\tmovn\t%s, #%d\n", regName(i.Rd, i.Is64), i.Imm)
		} else {
			fmt.Fprintf(p.w, "\tmovn\t%s, #%d, lsl #%d\n", regName(i.Rd, i.Is64), i.Imm, i.Shift)
		}

	// Address computation
	case ADR:
		fmt.Fprintf(p.w, "\tadr\t%s, %s\n", regName64(i.Rd), i.Target)
	case ADRP:
		fmt.Fprintf(p.w, "\tadrp\t%s, %s\n", regName64(i.Rd), i.Target)

	// Floating point operations
	case FADD:
		fmt.Fprintf(p.w, "\tfadd\t%s, %s, %s\n", floatRegName(i.Fd, i.IsDouble), floatRegName(i.Fn, i.IsDouble), floatRegName(i.Fm, i.IsDouble))
	case FSUB:
		fmt.Fprintf(p.w, "\tfsub\t%s, %s, %s\n", floatRegName(i.Fd, i.IsDouble), floatRegName(i.Fn, i.IsDouble), floatRegName(i.Fm, i.IsDouble))
	case FMUL:
		fmt.Fprintf(p.w, "\tfmul\t%s, %s, %s\n", floatRegName(i.Fd, i.IsDouble), floatRegName(i.Fn, i.IsDouble), floatRegName(i.Fm, i.IsDouble))
	case FDIV:
		fmt.Fprintf(p.w, "\tfdiv\t%s, %s, %s\n", floatRegName(i.Fd, i.IsDouble), floatRegName(i.Fn, i.IsDouble), floatRegName(i.Fm, i.IsDouble))
	case FNEG:
		fmt.Fprintf(p.w, "\tfneg\t%s, %s\n", floatRegName(i.Fd, i.IsDouble), floatRegName(i.Fn, i.IsDouble))
	case FABS:
		fmt.Fprintf(p.w, "\tfabs\t%s, %s\n", floatRegName(i.Fd, i.IsDouble), floatRegName(i.Fn, i.IsDouble))
	case FSQRT:
		fmt.Fprintf(p.w, "\tfsqrt\t%s, %s\n", floatRegName(i.Fd, i.IsDouble), floatRegName(i.Fn, i.IsDouble))
	case FMOV:
		fmt.Fprintf(p.w, "\tfmov\t%s, %s\n", floatRegName(i.Fd, i.IsDouble), floatRegName(i.Fn, i.IsDouble))
	case FMOVi:
		fmt.Fprintf(p.w, "\tfmov\t%s, #%g\n", floatRegName(i.Fd, i.IsDouble), i.Imm)

	// Float conversions
	case SCVTF:
		fmt.Fprintf(p.w, "\tscvtf\t%s, %s\n", floatRegName(i.Fd, i.IsDouble), regName(i.Rn, i.Is64Src))
	case UCVTF:
		fmt.Fprintf(p.w, "\tucvtf\t%s, %s\n", floatRegName(i.Fd, i.IsDouble), regName(i.Rn, i.Is64Src))
	case FCVTZS:
		fmt.Fprintf(p.w, "\tfcvtzs\t%s, %s\n", regName(i.Rd, i.Is64Dst), floatRegName(i.Fn, i.IsDouble))
	case FCVTZU:
		fmt.Fprintf(p.w, "\tfcvtzu\t%s, %s\n", regName(i.Rd, i.Is64Dst), floatRegName(i.Fn, i.IsDouble))
	case FCVT:
		if i.DstDouble {
			fmt.Fprintf(p.w, "\tfcvt\t%s, %s\n", floatRegName(i.Fd, true), floatRegName(i.Fn, false))
		} else {
			fmt.Fprintf(p.w, "\tfcvt\t%s, %s\n", floatRegName(i.Fd, false), floatRegName(i.Fn, true))
		}

	// Float compare
	case FCMP:
		fmt.Fprintf(p.w, "\tfcmp\t%s, %s\n", floatRegName(i.Fn, i.IsDouble), floatRegName(i.Fm, i.IsDouble))
	case FCMPz:
		fmt.Fprintf(p.w, "\tfcmp\t%s, #0.0\n", floatRegName(i.Fn, i.IsDouble))

	// Sign/zero extension
	case SXTB:
		fmt.Fprintf(p.w, "\tsxtb\t%s, %s\n", regName(i.Rd, i.Is64), regName32(i.Rn))
	case SXTH:
		fmt.Fprintf(p.w, "\tsxth\t%s, %s\n", regName(i.Rd, i.Is64), regName32(i.Rn))
	case SXTW:
		fmt.Fprintf(p.w, "\tsxtw\t%s, %s\n", regName64(i.Rd), regName32(i.Rn))
	case UXTB:
		fmt.Fprintf(p.w, "\tuxtb\t%s, %s\n", regName32(i.Rd), regName32(i.Rn))
	case UXTH:
		fmt.Fprintf(p.w, "\tuxth\t%s, %s\n", regName32(i.Rd), regName32(i.Rn))

	default:
		fmt.Fprintf(p.w, "\t// unknown instruction: %T\n", inst)
	}
}
