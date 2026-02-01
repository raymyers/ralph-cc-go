// Package asmgen transforms Mach code to ARM64 assembly.
// This is the final compilation phase, producing assembly code
// that can be assembled by a standard assembler (as/gas).
package asmgen

import (
	"fmt"

	"github.com/raymyers/ralph-cc/pkg/asm"
	"github.com/raymyers/ralph-cc/pkg/mach"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// TransformProgram transforms a Mach program to assembly
func TransformProgram(prog *mach.Program) *asm.Program {
	result := &asm.Program{
		Globals:   make([]asm.GlobVar, len(prog.Globals)),
		Functions: make([]asm.Function, len(prog.Functions)),
	}

	// Transform globals
	for i, g := range prog.Globals {
		result.Globals[i] = asm.GlobVar{
			Name:  g.Name,
			Size:  g.Size,
			Init:  g.Init,
			Align: 8, // Default alignment for 64-bit
		}
	}

	// Transform functions
	for i, f := range prog.Functions {
		result.Functions[i] = transformFunction(&f)
	}

	return result
}

// transformFunction transforms a single Mach function to assembly
func transformFunction(f *mach.Function) asm.Function {
	ctx := &genContext{
		fn:         f,
		labelCount: 0,
	}

	result := asm.Function{
		Name: f.Name,
		Code: make([]asm.Instruction, 0),
	}

	// Transform each instruction
	for _, inst := range f.Code {
		instrs := ctx.translateInstruction(inst)
		result.Code = append(result.Code, instrs...)
	}

	return result
}

// genContext holds state during code generation
type genContext struct {
	fn         *mach.Function
	labelCount int
}

// newLabel generates a unique label
func (ctx *genContext) newLabel() asm.Label {
	ctx.labelCount++
	return asm.Label(fmt.Sprintf(".L%d", ctx.labelCount))
}

// machLabelToAsm converts a Mach label to an assembly label
func machLabelToAsm(lbl mach.Label) asm.Label {
	return asm.Label(fmt.Sprintf(".L%d", lbl))
}

// translateInstruction translates a Mach instruction to assembly
func (ctx *genContext) translateInstruction(inst mach.Instruction) []asm.Instruction {
	switch i := inst.(type) {
	case mach.Mgetstack:
		return ctx.translateGetstack(i)
	case mach.Msetstack:
		return ctx.translateSetstack(i)
	case mach.Mgetparam:
		return ctx.translateGetparam(i)
	case mach.Mop:
		return ctx.translateOp(i)
	case mach.Mload:
		return ctx.translateLoad(i)
	case mach.Mstore:
		return ctx.translateStore(i)
	case mach.Mcall:
		return ctx.translateCall(i)
	case mach.Mtailcall:
		return ctx.translateTailcall(i)
	case mach.Mbuiltin:
		return ctx.translateBuiltin(i)
	case mach.Mlabel:
		return []asm.Instruction{asm.LabelDef{Name: machLabelToAsm(i.Lbl)}}
	case mach.Mgoto:
		return []asm.Instruction{asm.B{Target: machLabelToAsm(i.Target)}}
	case mach.Mcond:
		return ctx.translateCond(i)
	case mach.Mjumptable:
		return ctx.translateJumptable(i)
	case mach.Mreturn:
		return []asm.Instruction{asm.RET{}}
	default:
		// Unknown instruction - generate a comment
		return nil
	}
}

// translateGetstack generates a load from stack slot
func (ctx *genContext) translateGetstack(i mach.Mgetstack) []asm.Instruction {
	// Load from [FP + offset]
	is64 := is64BitType(i.Ty)
	if i.Dest.IsFloat() {
		if is64 {
			return []asm.Instruction{asm.FLDRd{Ft: i.Dest, Rn: asm.X29, Ofs: i.Ofs}}
		}
		return []asm.Instruction{asm.FLDRs{Ft: i.Dest, Rn: asm.X29, Ofs: i.Ofs}}
	}
	return []asm.Instruction{asm.LDR{Rt: i.Dest, Rn: asm.X29, Ofs: i.Ofs, Is64: is64}}
}

// translateSetstack generates a store to stack slot
func (ctx *genContext) translateSetstack(i mach.Msetstack) []asm.Instruction {
	// Store to [FP + offset]
	is64 := is64BitType(i.Ty)
	if i.Src.IsFloat() {
		if is64 {
			return []asm.Instruction{asm.FSTRd{Ft: i.Src, Rn: asm.X29, Ofs: i.Ofs}}
		}
		return []asm.Instruction{asm.FSTRs{Ft: i.Src, Rn: asm.X29, Ofs: i.Ofs}}
	}
	return []asm.Instruction{asm.STR{Rt: i.Src, Rn: asm.X29, Ofs: i.Ofs, Is64: is64}}
}

// translateGetparam generates a load of incoming parameter
func (ctx *genContext) translateGetparam(i mach.Mgetparam) []asm.Instruction {
	// Load from [FP + offset] (parameters are above FP)
	is64 := is64BitType(i.Ty)
	if i.Dest.IsFloat() {
		if is64 {
			return []asm.Instruction{asm.FLDRd{Ft: i.Dest, Rn: asm.X29, Ofs: i.Ofs}}
		}
		return []asm.Instruction{asm.FLDRs{Ft: i.Dest, Rn: asm.X29, Ofs: i.Ofs}}
	}
	return []asm.Instruction{asm.LDR{Rt: i.Dest, Rn: asm.X29, Ofs: i.Ofs, Is64: is64}}
}

// translateOp translates an operation
func (ctx *genContext) translateOp(i mach.Mop) []asm.Instruction {
	return translateOperation(i.Op, i.Args, i.Dest)
}

// translateOperation generates instructions for an operation
func translateOperation(op mach.Operation, args []mach.MReg, dest mach.MReg) []asm.Instruction {
	switch o := op.(type) {
	// Move operations
	case rtl.Omove:
		if dest.IsFloat() {
			return []asm.Instruction{asm.FMOV{Fd: dest, Fn: args[0], IsDouble: true}}
		}
		return []asm.Instruction{asm.MOV{Rd: dest, Rm: args[0], Is64: true}}

	// Integer constants
	case rtl.Ointconst:
		return loadIntConstant(dest, int64(o.Value), false)
	case rtl.Olongconst:
		return loadIntConstant(dest, o.Value, true)

	// Float constants
	case rtl.Ofloatconst:
		return loadFloatConstant(dest, o.Value, true)
	case rtl.Osingleconst:
		return loadFloatConstant(dest, float64(o.Value), false)

	// Address operations
	case rtl.Oaddrsymbol:
		// Load address of symbol
		return []asm.Instruction{
			asm.ADRP{Rd: dest, Target: asm.Label(o.Symbol)},
			asm.ADDi{Rd: dest, Rn: dest, Imm: o.Offset, Is64: true}, // ADD for low bits
		}
	case rtl.Oaddrstack:
		// Compute stack address
		return []asm.Instruction{
			asm.ADDi{Rd: dest, Rn: asm.X29, Imm: o.Offset, Is64: true},
		}

	// Integer arithmetic (32-bit)
	case rtl.Oadd:
		return []asm.Instruction{asm.ADD{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Oaddimm:
		return []asm.Instruction{asm.ADDi{Rd: dest, Rn: args[0], Imm: int64(o.N), Is64: false}}
	case rtl.Osub:
		return []asm.Instruction{asm.SUB{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Oneg:
		return []asm.Instruction{asm.NEG{Rd: dest, Rm: args[0], Is64: false}}
	case rtl.Omul:
		return []asm.Instruction{asm.MUL{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Omulimm:
		// MUL doesn't have immediate form, load constant first
		return []asm.Instruction{
			asm.MOVi{Rd: asm.X8, Imm: int64(o.N), Is64: false},
			asm.MUL{Rd: dest, Rn: args[0], Rm: asm.X8, Is64: false},
		}
	case rtl.Odiv:
		return []asm.Instruction{asm.SDIV{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Odivu:
		return []asm.Instruction{asm.UDIV{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Omod:
		// rd = rn - (rn/rm)*rm
		return []asm.Instruction{
			asm.SDIV{Rd: asm.X8, Rn: args[0], Rm: args[1], Is64: false},
			asm.MUL{Rd: asm.X8, Rn: asm.X8, Rm: args[1], Is64: false},
			asm.SUB{Rd: dest, Rn: args[0], Rm: asm.X8, Is64: false},
		}
	case rtl.Omodu:
		return []asm.Instruction{
			asm.UDIV{Rd: asm.X8, Rn: args[0], Rm: args[1], Is64: false},
			asm.MUL{Rd: asm.X8, Rn: asm.X8, Rm: args[1], Is64: false},
			asm.SUB{Rd: dest, Rn: args[0], Rm: asm.X8, Is64: false},
		}

	// Bitwise operations (32-bit)
	case rtl.Oand:
		return []asm.Instruction{asm.AND{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Oandimm:
		return []asm.Instruction{asm.ANDi{Rd: dest, Rn: args[0], Imm: int64(o.N), Is64: false}}
	case rtl.Oor:
		return []asm.Instruction{asm.ORR{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Oorimm:
		return []asm.Instruction{asm.ORRi{Rd: dest, Rn: args[0], Imm: int64(o.N), Is64: false}}
	case rtl.Oxor:
		return []asm.Instruction{asm.EOR{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Oxorimm:
		return []asm.Instruction{asm.EORi{Rd: dest, Rn: args[0], Imm: int64(o.N), Is64: false}}
	case rtl.Onot:
		return []asm.Instruction{asm.MVN{Rd: dest, Rm: args[0], Is64: false}}

	// Shift operations (32-bit)
	case rtl.Oshl:
		return []asm.Instruction{asm.LSL{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Oshlimm:
		return []asm.Instruction{asm.LSLi{Rd: dest, Rn: args[0], Shift: int(o.N), Is64: false}}
	case rtl.Oshr:
		return []asm.Instruction{asm.ASR{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Oshrimm:
		return []asm.Instruction{asm.ASRi{Rd: dest, Rn: args[0], Shift: int(o.N), Is64: false}}
	case rtl.Oshru:
		return []asm.Instruction{asm.LSR{Rd: dest, Rn: args[0], Rm: args[1], Is64: false}}
	case rtl.Oshruimm:
		return []asm.Instruction{asm.LSRi{Rd: dest, Rn: args[0], Shift: int(o.N), Is64: false}}

	// 64-bit integer arithmetic
	case rtl.Oaddl:
		return []asm.Instruction{asm.ADD{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Oaddlimm:
		// If args is empty, SP is the implicit source (used for stack adjustments)
		srcReg := asm.X29 // default to FP
		if len(args) > 0 {
			srcReg = args[0]
		}
		return []asm.Instruction{asm.ADDi{Rd: dest, Rn: srcReg, Imm: o.N, Is64: true}}
	case rtl.Osubl:
		return []asm.Instruction{asm.SUB{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Onegl:
		return []asm.Instruction{asm.NEG{Rd: dest, Rm: args[0], Is64: true}}
	case rtl.Omull:
		return []asm.Instruction{asm.MUL{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Odivl:
		return []asm.Instruction{asm.SDIV{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Odivlu:
		return []asm.Instruction{asm.UDIV{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Omodl:
		return []asm.Instruction{
			asm.SDIV{Rd: asm.X8, Rn: args[0], Rm: args[1], Is64: true},
			asm.MUL{Rd: asm.X8, Rn: asm.X8, Rm: args[1], Is64: true},
			asm.SUB{Rd: dest, Rn: args[0], Rm: asm.X8, Is64: true},
		}
	case rtl.Omodlu:
		return []asm.Instruction{
			asm.UDIV{Rd: asm.X8, Rn: args[0], Rm: args[1], Is64: true},
			asm.MUL{Rd: asm.X8, Rn: asm.X8, Rm: args[1], Is64: true},
			asm.SUB{Rd: dest, Rn: args[0], Rm: asm.X8, Is64: true},
		}

	// 64-bit bitwise operations
	case rtl.Oandl:
		return []asm.Instruction{asm.AND{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Oandlimm:
		return []asm.Instruction{asm.ANDi{Rd: dest, Rn: args[0], Imm: o.N, Is64: true}}
	case rtl.Oorl:
		return []asm.Instruction{asm.ORR{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Oorlimm:
		return []asm.Instruction{asm.ORRi{Rd: dest, Rn: args[0], Imm: o.N, Is64: true}}
	case rtl.Oxorl:
		return []asm.Instruction{asm.EOR{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Oxorlimm:
		return []asm.Instruction{asm.EORi{Rd: dest, Rn: args[0], Imm: o.N, Is64: true}}
	case rtl.Onotl:
		return []asm.Instruction{asm.MVN{Rd: dest, Rm: args[0], Is64: true}}

	// 64-bit shifts
	case rtl.Oshll:
		return []asm.Instruction{asm.LSL{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Oshllimm:
		return []asm.Instruction{asm.LSLi{Rd: dest, Rn: args[0], Shift: int(o.N), Is64: true}}
	case rtl.Oshrl:
		return []asm.Instruction{asm.ASR{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Oshrlimm:
		return []asm.Instruction{asm.ASRi{Rd: dest, Rn: args[0], Shift: int(o.N), Is64: true}}
	case rtl.Oshrlu:
		return []asm.Instruction{asm.LSR{Rd: dest, Rn: args[0], Rm: args[1], Is64: true}}
	case rtl.Oshrluimm:
		return []asm.Instruction{asm.LSRi{Rd: dest, Rn: args[0], Shift: int(o.N), Is64: true}}

	// Extension operations
	case rtl.Ocast8signed:
		return []asm.Instruction{asm.SXTB{Rd: dest, Rn: args[0], Is64: false}}
	case rtl.Ocast8unsigned:
		return []asm.Instruction{asm.UXTB{Rd: dest, Rn: args[0]}}
	case rtl.Ocast16signed:
		return []asm.Instruction{asm.SXTH{Rd: dest, Rn: args[0], Is64: false}}
	case rtl.Ocast16unsigned:
		return []asm.Instruction{asm.UXTH{Rd: dest, Rn: args[0]}}
	case rtl.Olongofint:
		return []asm.Instruction{asm.SXTW{Rd: dest, Rn: args[0]}}
	case rtl.Olongofintu:
		// Zero extension is implicit when writing to W register
		return []asm.Instruction{asm.MOV{Rd: dest, Rm: args[0], Is64: false}}
	case rtl.Ointoflong:
		// Truncation is implicit when reading from W register
		return []asm.Instruction{asm.MOV{Rd: dest, Rm: args[0], Is64: false}}

	// Floating-point operations (double)
	case rtl.Onegf:
		return []asm.Instruction{asm.FNEG{Fd: dest, Fn: args[0], IsDouble: true}}
	case rtl.Oabsf:
		return []asm.Instruction{asm.FABS{Fd: dest, Fn: args[0], IsDouble: true}}
	case rtl.Oaddf:
		return []asm.Instruction{asm.FADD{Fd: dest, Fn: args[0], Fm: args[1], IsDouble: true}}
	case rtl.Osubf:
		return []asm.Instruction{asm.FSUB{Fd: dest, Fn: args[0], Fm: args[1], IsDouble: true}}
	case rtl.Omulf:
		return []asm.Instruction{asm.FMUL{Fd: dest, Fn: args[0], Fm: args[1], IsDouble: true}}
	case rtl.Odivf:
		return []asm.Instruction{asm.FDIV{Fd: dest, Fn: args[0], Fm: args[1], IsDouble: true}}

	// Floating-point operations (single)
	case rtl.Onegs:
		return []asm.Instruction{asm.FNEG{Fd: dest, Fn: args[0], IsDouble: false}}
	case rtl.Oabss:
		return []asm.Instruction{asm.FABS{Fd: dest, Fn: args[0], IsDouble: false}}
	case rtl.Oadds:
		return []asm.Instruction{asm.FADD{Fd: dest, Fn: args[0], Fm: args[1], IsDouble: false}}
	case rtl.Osubs:
		return []asm.Instruction{asm.FSUB{Fd: dest, Fn: args[0], Fm: args[1], IsDouble: false}}
	case rtl.Omuls:
		return []asm.Instruction{asm.FMUL{Fd: dest, Fn: args[0], Fm: args[1], IsDouble: false}}
	case rtl.Odivs:
		return []asm.Instruction{asm.FDIV{Fd: dest, Fn: args[0], Fm: args[1], IsDouble: false}}

	// Float conversions
	case rtl.Osingleoffloat:
		return []asm.Instruction{asm.FCVT{Fd: dest, Fn: args[0], DstDouble: false}}
	case rtl.Ofloatofsingle:
		return []asm.Instruction{asm.FCVT{Fd: dest, Fn: args[0], DstDouble: true}}
	case rtl.Ointoffloat:
		return []asm.Instruction{asm.FCVTZS{Rd: dest, Fn: args[0], IsDouble: true, Is64Dst: false}}
	case rtl.Ointuoffloat:
		return []asm.Instruction{asm.FCVTZU{Rd: dest, Fn: args[0], IsDouble: true, Is64Dst: false}}
	case rtl.Ofloatofint:
		return []asm.Instruction{asm.SCVTF{Fd: dest, Rn: args[0], IsDouble: true, Is64Src: false}}
	case rtl.Ofloatofintu:
		return []asm.Instruction{asm.UCVTF{Fd: dest, Rn: args[0], IsDouble: true, Is64Src: false}}
	case rtl.Olongoffloat:
		return []asm.Instruction{asm.FCVTZS{Rd: dest, Fn: args[0], IsDouble: true, Is64Dst: true}}
	case rtl.Olonguoffloat:
		return []asm.Instruction{asm.FCVTZU{Rd: dest, Fn: args[0], IsDouble: true, Is64Dst: true}}
	case rtl.Ofloatoflong:
		return []asm.Instruction{asm.SCVTF{Fd: dest, Rn: args[0], IsDouble: true, Is64Src: true}}
	case rtl.Ofloatoflongu:
		return []asm.Instruction{asm.UCVTF{Fd: dest, Rn: args[0], IsDouble: true, Is64Src: true}}

	// Comparisons
	case rtl.Ocmp:
		return translateCompare(args, dest, o.Cond, false, false)
	case rtl.Ocmpu:
		return translateCompare(args, dest, o.Cond, true, false)
	case rtl.Ocmpl:
		return translateCompare(args, dest, o.Cond, false, true)
	case rtl.Ocmplu:
		return translateCompare(args, dest, o.Cond, true, true)
	case rtl.Ocmpimm:
		return translateCompareImm(args[0], dest, int64(o.N), o.Cond, false, false)
	case rtl.Ocmpuimm:
		return translateCompareImm(args[0], dest, int64(o.N), o.Cond, true, false)
	case rtl.Ocmplimm:
		return translateCompareImm(args[0], dest, o.N, o.Cond, false, true)
	case rtl.Ocmpluimm:
		return translateCompareImm(args[0], dest, o.N, o.Cond, true, true)

	default:
		// Unknown operation - return empty
		return nil
	}
}

// translateCompare generates compare and conditional set
func translateCompare(args []mach.MReg, dest mach.MReg, cond rtl.Condition, unsigned, is64 bool) []asm.Instruction {
	cc := conditionToCondCode(cond, unsigned)
	return []asm.Instruction{
		asm.CMP{Rn: args[0], Rm: args[1], Is64: is64},
		asm.CSET{Rd: dest, Cond: cc, Is64: is64},
	}
}

// translateCompareImm generates compare immediate and conditional set
func translateCompareImm(arg mach.MReg, dest mach.MReg, imm int64, cond rtl.Condition, unsigned, is64 bool) []asm.Instruction {
	cc := conditionToCondCode(cond, unsigned)
	return []asm.Instruction{
		asm.CMPi{Rn: arg, Imm: imm, Is64: is64},
		asm.CSET{Rd: dest, Cond: cc, Is64: is64},
	}
}

// conditionToCondCode converts RTL condition to ARM64 condition code
func conditionToCondCode(cond rtl.Condition, unsigned bool) asm.CondCode {
	switch cond {
	case rtl.Ceq:
		return asm.CondEQ
	case rtl.Cne:
		return asm.CondNE
	case rtl.Clt:
		if unsigned {
			return asm.CondCC // Carry clear = unsigned less than
		}
		return asm.CondLT
	case rtl.Cle:
		if unsigned {
			return asm.CondLS // Unsigned lower or same
		}
		return asm.CondLE
	case rtl.Cgt:
		if unsigned {
			return asm.CondHI // Unsigned higher
		}
		return asm.CondGT
	case rtl.Cge:
		if unsigned {
			return asm.CondCS // Carry set = unsigned greater or equal
		}
		return asm.CondGE
	default:
		return asm.CondAL
	}
}

// translateLoad generates load instructions
func (ctx *genContext) translateLoad(i mach.Mload) []asm.Instruction {
	base := i.Args[0]
	ofs := int64(0)

	// Extract offset from addressing mode
	switch addr := i.Addr.(type) {
	case rtl.Aindexed:
		ofs = addr.Offset
	case rtl.Ainstack:
		base = asm.X29 // FP
		ofs = addr.Offset
	}

	// Generate appropriate load based on chunk type
	switch i.Chunk {
	case mach.Mint8signed:
		return []asm.Instruction{asm.LDRSB{Rt: i.Dest, Rn: base, Ofs: ofs, Is64: false}}
	case mach.Mint8unsigned:
		return []asm.Instruction{asm.LDRB{Rt: i.Dest, Rn: base, Ofs: ofs}}
	case mach.Mint16signed:
		return []asm.Instruction{asm.LDRSH{Rt: i.Dest, Rn: base, Ofs: ofs, Is64: false}}
	case mach.Mint16unsigned:
		return []asm.Instruction{asm.LDRH{Rt: i.Dest, Rn: base, Ofs: ofs}}
	case mach.Mint32:
		return []asm.Instruction{asm.LDR{Rt: i.Dest, Rn: base, Ofs: ofs, Is64: false}}
	case mach.Mint64:
		return []asm.Instruction{asm.LDR{Rt: i.Dest, Rn: base, Ofs: ofs, Is64: true}}
	case mach.Mfloat32:
		return []asm.Instruction{asm.FLDRs{Ft: i.Dest, Rn: base, Ofs: ofs}}
	case mach.Mfloat64:
		return []asm.Instruction{asm.FLDRd{Ft: i.Dest, Rn: base, Ofs: ofs}}
	default:
		return []asm.Instruction{asm.LDR{Rt: i.Dest, Rn: base, Ofs: ofs, Is64: true}}
	}
}

// translateStore generates store instructions
func (ctx *genContext) translateStore(i mach.Mstore) []asm.Instruction {
	base := i.Args[0]
	ofs := int64(0)

	// Extract offset from addressing mode
	switch addr := i.Addr.(type) {
	case rtl.Aindexed:
		ofs = addr.Offset
	case rtl.Ainstack:
		base = asm.X29 // FP
		ofs = addr.Offset
	}

	// Generate appropriate store based on chunk type
	switch i.Chunk {
	case mach.Mint8signed, mach.Mint8unsigned:
		return []asm.Instruction{asm.STRB{Rt: i.Src, Rn: base, Ofs: ofs}}
	case mach.Mint16signed, mach.Mint16unsigned:
		return []asm.Instruction{asm.STRH{Rt: i.Src, Rn: base, Ofs: ofs}}
	case mach.Mint32:
		return []asm.Instruction{asm.STR{Rt: i.Src, Rn: base, Ofs: ofs, Is64: false}}
	case mach.Mint64:
		return []asm.Instruction{asm.STR{Rt: i.Src, Rn: base, Ofs: ofs, Is64: true}}
	case mach.Mfloat32:
		return []asm.Instruction{asm.FSTRs{Ft: i.Src, Rn: base, Ofs: ofs}}
	case mach.Mfloat64:
		return []asm.Instruction{asm.FSTRd{Ft: i.Src, Rn: base, Ofs: ofs}}
	default:
		return []asm.Instruction{asm.STR{Rt: i.Src, Rn: base, Ofs: ofs, Is64: true}}
	}
}

// translateCall generates function call instructions
func (ctx *genContext) translateCall(i mach.Mcall) []asm.Instruction {
	switch fn := i.Fn.(type) {
	case mach.FunSymbol:
		return []asm.Instruction{asm.BL{Target: asm.Label(fn.Name)}}
	case mach.FunReg:
		return []asm.Instruction{asm.BLR{Rn: fn.Reg}}
	default:
		return nil
	}
}

// translateTailcall generates tail call instructions
func (ctx *genContext) translateTailcall(i mach.Mtailcall) []asm.Instruction {
	switch fn := i.Fn.(type) {
	case mach.FunSymbol:
		return []asm.Instruction{asm.B{Target: asm.Label(fn.Name)}}
	case mach.FunReg:
		return []asm.Instruction{asm.BR{Rn: fn.Reg}}
	default:
		return nil
	}
}

// translateBuiltin generates builtin function calls
func (ctx *genContext) translateBuiltin(i mach.Mbuiltin) []asm.Instruction {
	// For now, just generate a BL to the builtin name
	return []asm.Instruction{asm.BL{Target: asm.Label(i.Builtin)}}
}

// translateCond generates conditional branch
func (ctx *genContext) translateCond(i mach.Mcond) []asm.Instruction {
	// Generate compare instruction based on condition code type
	cc := condCodeToAsmCond(i.Cond, i.Args)
	return []asm.Instruction{
		asm.Bcond{Cond: cc, Target: machLabelToAsm(i.IfSo)},
	}
}

// condCodeToAsmCond converts a Mach condition code to an ARM64 condition
func condCodeToAsmCond(cond mach.ConditionCode, args []mach.MReg) asm.CondCode {
	switch c := cond.(type) {
	case rtl.Ccomp:
		return conditionToCondCode(c.Cond, false)
	case rtl.Ccompu:
		return conditionToCondCode(c.Cond, true)
	case rtl.Ccompl:
		return conditionToCondCode(c.Cond, false)
	case rtl.Ccomplu:
		return conditionToCondCode(c.Cond, true)
	default:
		return asm.CondAL
	}
}

// translateJumptable generates a switch/jump table
func (ctx *genContext) translateJumptable(i mach.Mjumptable) []asm.Instruction {
	// Simplified: generate a series of compare and branch
	// A real implementation would use an actual jump table
	result := make([]asm.Instruction, 0)
	for idx, target := range i.Targets {
		result = append(result,
			asm.CMPi{Rn: i.Arg, Imm: int64(idx), Is64: true},
			asm.Bcond{Cond: asm.CondEQ, Target: machLabelToAsm(target)},
		)
	}
	return result
}

// loadIntConstant generates instructions to load an integer constant
func loadIntConstant(dest mach.MReg, val int64, is64 bool) []asm.Instruction {
	// For small constants, use MOVi
	if val >= 0 && val <= 65535 {
		return []asm.Instruction{asm.MOVi{Rd: dest, Imm: val, Is64: is64}}
	}
	// For negative small constants, use MOVN
	if val < 0 && val >= -65536 {
		return []asm.Instruction{asm.MOVN{Rd: dest, Imm: uint16(^val), Shift: 0, Is64: is64}}
	}
	// For larger constants, use MOVZ + MOVK sequence
	result := make([]asm.Instruction, 0)
	result = append(result, asm.MOVZ{Rd: dest, Imm: uint16(val & 0xFFFF), Shift: 0, Is64: is64})
	if (val>>16)&0xFFFF != 0 {
		result = append(result, asm.MOVK{Rd: dest, Imm: uint16((val >> 16) & 0xFFFF), Shift: 16, Is64: is64})
	}
	if is64 {
		if (val>>32)&0xFFFF != 0 {
			result = append(result, asm.MOVK{Rd: dest, Imm: uint16((val >> 32) & 0xFFFF), Shift: 32, Is64: true})
		}
		if (val>>48)&0xFFFF != 0 {
			result = append(result, asm.MOVK{Rd: dest, Imm: uint16((val >> 48) & 0xFFFF), Shift: 48, Is64: true})
		}
	}
	return result
}

// loadFloatConstant generates instructions to load a float constant
func loadFloatConstant(dest mach.MReg, val float64, isDouble bool) []asm.Instruction {
	// Try to use FMOV immediate if possible (limited range)
	// For now, always use literal load via ADRP+LDR
	// A real implementation would check if the constant is representable
	return []asm.Instruction{asm.FMOVi{Fd: dest, Imm: val, IsDouble: isDouble}}
}

// is64BitType returns true if the type is 64-bit
func is64BitType(ty mach.Typ) bool {
	switch ty {
	case mach.Tlong, mach.Tfloat, mach.Tany64:
		return true
	default:
		return false
	}
}
