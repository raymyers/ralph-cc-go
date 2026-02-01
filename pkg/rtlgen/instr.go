// Instruction generation for RTLgen.
// Translates CminorSel operations to RTL instructions.

package rtlgen

import (
	"github.com/raymyers/ralph-cc/pkg/cminorsel"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// TranslateUnaryOp converts a CminorSel unary operator to RTL operation.
func TranslateUnaryOp(op cminorsel.UnaryOp) rtl.Operation {
	switch op {
	case cminorsel.Ocast8signed:
		return rtl.Ocast8signed{}
	case cminorsel.Ocast8unsigned:
		return rtl.Ocast8unsigned{}
	case cminorsel.Ocast16signed:
		return rtl.Ocast16signed{}
	case cminorsel.Ocast16unsigned:
		return rtl.Ocast16unsigned{}
	case cminorsel.Onegint:
		return rtl.Oneg{}
	case cminorsel.Onegl:
		return rtl.Onegl{}
	case cminorsel.Onegf:
		return rtl.Onegf{}
	case cminorsel.Onegs:
		return rtl.Onegs{}
	case cminorsel.Onotint:
		return rtl.Onot{}
	case cminorsel.Onotl:
		return rtl.Onotl{}
	case cminorsel.Osingleoffloat:
		return rtl.Osingleoffloat{}
	case cminorsel.Ofloatofsingle:
		return rtl.Ofloatofsingle{}
	case cminorsel.Ointoffloat:
		return rtl.Ointoffloat{}
	case cminorsel.Ointuoffloat:
		return rtl.Ointuoffloat{}
	case cminorsel.Ofloatofint:
		return rtl.Ofloatofint{}
	case cminorsel.Ofloatofintu:
		return rtl.Ofloatofintu{}
	case cminorsel.Olongoffloat:
		return rtl.Olongoffloat{}
	case cminorsel.Olonguoffloat:
		return rtl.Olonguoffloat{}
	case cminorsel.Ofloatoflong:
		return rtl.Ofloatoflong{}
	case cminorsel.Ofloatoflongu:
		return rtl.Ofloatoflongu{}
	case cminorsel.Ointoflong:
		return rtl.Ointoflong{}
	case cminorsel.Olongofint:
		return rtl.Olongofint{}
	case cminorsel.Olongofintu:
		return rtl.Olongofintu{}
	default:
		return rtl.Omove{}
	}
}

// TranslateBinaryOp converts a CminorSel binary operator to RTL operation.
func TranslateBinaryOp(op cminorsel.BinaryOp) rtl.Operation {
	switch op {
	case cminorsel.Oadd:
		return rtl.Oadd{}
	case cminorsel.Osub:
		return rtl.Osub{}
	case cminorsel.Omul:
		return rtl.Omul{}
	case cminorsel.Odiv:
		return rtl.Odiv{}
	case cminorsel.Odivu:
		return rtl.Odivu{}
	case cminorsel.Omod:
		return rtl.Omod{}
	case cminorsel.Omodu:
		return rtl.Omodu{}
	case cminorsel.Oand:
		return rtl.Oand{}
	case cminorsel.Oor:
		return rtl.Oor{}
	case cminorsel.Oxor:
		return rtl.Oxor{}
	case cminorsel.Oshl:
		return rtl.Oshl{}
	case cminorsel.Oshr:
		return rtl.Oshr{}
	case cminorsel.Oshru:
		return rtl.Oshru{}
	case cminorsel.Oaddf:
		return rtl.Oaddf{}
	case cminorsel.Osubf:
		return rtl.Osubf{}
	case cminorsel.Omulf:
		return rtl.Omulf{}
	case cminorsel.Odivf:
		return rtl.Odivf{}
	case cminorsel.Oadds:
		return rtl.Oadds{}
	case cminorsel.Osubs:
		return rtl.Osubs{}
	case cminorsel.Omuls:
		return rtl.Omuls{}
	case cminorsel.Odivs:
		return rtl.Odivs{}
	case cminorsel.Oaddl:
		return rtl.Oaddl{}
	case cminorsel.Osubl:
		return rtl.Osubl{}
	case cminorsel.Omull:
		return rtl.Omull{}
	case cminorsel.Odivl:
		return rtl.Odivl{}
	case cminorsel.Odivlu:
		return rtl.Odivlu{}
	case cminorsel.Omodl:
		return rtl.Omodl{}
	case cminorsel.Omodlu:
		return rtl.Omodlu{}
	case cminorsel.Oandl:
		return rtl.Oandl{}
	case cminorsel.Oorl:
		return rtl.Oorl{}
	case cminorsel.Oxorl:
		return rtl.Oxorl{}
	case cminorsel.Oshll:
		return rtl.Oshll{}
	case cminorsel.Oshrl:
		return rtl.Oshrl{}
	case cminorsel.Oshrlu:
		return rtl.Oshrlu{}
	default:
		return rtl.Oadd{}
	}
}

// TranslateConstant converts a CminorSel constant to an RTL operation.
func TranslateConstant(c cminorsel.Constant) rtl.Operation {
	switch v := c.(type) {
	case cminorsel.Ointconst:
		return rtl.Ointconst{Value: v.Value}
	case cminorsel.Olongconst:
		return rtl.Olongconst{Value: v.Value}
	case cminorsel.Ofloatconst:
		return rtl.Ofloatconst{Value: v.Value}
	case cminorsel.Osingleconst:
		return rtl.Osingleconst{Value: v.Value}
	case cminorsel.Oaddrsymbol:
		return rtl.Oaddrsymbol{Symbol: v.Symbol, Offset: v.Offset}
	case cminorsel.Oaddrstack:
		return rtl.Oaddrstack{Offset: v.Offset}
	default:
		return rtl.Ointconst{Value: 0}
	}
}

// TranslateAddressingMode converts a CminorSel addressing mode.
// Returns the RTL addressing mode (same type structure).
func TranslateAddressingMode(addr cminorsel.AddressingMode) rtl.AddressingMode {
	switch v := addr.(type) {
	case cminorsel.Aindexed:
		return rtl.Aindexed{Offset: v.Offset}
	case cminorsel.Aindexed2:
		return rtl.Aindexed2{}
	case cminorsel.Aindexed2shift:
		return rtl.Aindexed2shift{Shift: v.Shift}
	case cminorsel.Aglobal:
		return rtl.Aglobal{Symbol: v.Symbol, Offset: v.Offset}
	case cminorsel.Ainstack:
		return rtl.Ainstack{Offset: v.Offset}
	default:
		return rtl.Aindexed{Offset: 0}
	}
}

// TranslateChunk converts a CminorSel memory chunk.
// RTL uses the same chunk type as CminorSel.
func TranslateChunk(c cminorsel.Chunk) rtl.Chunk {
	return c
}

// InstrBuilder helps build sequences of RTL instructions.
// It integrates CFGBuilder and RegAllocator for streamlined generation.
type InstrBuilder struct {
	cfg  *CFGBuilder
	regs *RegAllocator
}

// NewInstrBuilder creates an instruction builder from cfg and reg allocator.
func NewInstrBuilder(cfg *CFGBuilder, regs *RegAllocator) *InstrBuilder {
	return &InstrBuilder{cfg: cfg, regs: regs}
}

// EmitOp emits an Iop instruction: dest = op(args...), goto succ
// Returns the node containing this instruction.
func (b *InstrBuilder) EmitOp(op rtl.Operation, args []rtl.Reg, dest rtl.Reg, succ rtl.Node) rtl.Node {
	return b.cfg.EmitInstr(rtl.Iop{
		Op:   op,
		Args: args,
		Dest: dest,
		Succ: succ,
	})
}

// EmitMove emits a move instruction: dest = src, goto succ
func (b *InstrBuilder) EmitMove(src, dest rtl.Reg, succ rtl.Node) rtl.Node {
	return b.EmitOp(rtl.Omove{}, []rtl.Reg{src}, dest, succ)
}

// EmitConst emits a constant load: dest = const, goto succ
func (b *InstrBuilder) EmitConst(c cminorsel.Constant, dest rtl.Reg, succ rtl.Node) rtl.Node {
	return b.EmitOp(TranslateConstant(c), nil, dest, succ)
}

// EmitLoad emits a memory load: dest = Mem[addr(args...)], goto succ
func (b *InstrBuilder) EmitLoad(chunk rtl.Chunk, addr rtl.AddressingMode, args []rtl.Reg, dest rtl.Reg, succ rtl.Node) rtl.Node {
	return b.cfg.EmitInstr(rtl.Iload{
		Chunk: chunk,
		Addr:  addr,
		Args:  args,
		Dest:  dest,
		Succ:  succ,
	})
}

// EmitStore emits a memory store: Mem[addr(args...)] = src, goto succ
func (b *InstrBuilder) EmitStore(chunk rtl.Chunk, addr rtl.AddressingMode, args []rtl.Reg, src rtl.Reg, succ rtl.Node) rtl.Node {
	return b.cfg.EmitInstr(rtl.Istore{
		Chunk: chunk,
		Addr:  addr,
		Args:  args,
		Src:   src,
		Succ:  succ,
	})
}

// EmitCall emits a function call: dest = fn(args...), goto succ
func (b *InstrBuilder) EmitCall(sig rtl.Sig, fn rtl.FunRef, args []rtl.Reg, dest rtl.Reg, succ rtl.Node) rtl.Node {
	return b.cfg.EmitInstr(rtl.Icall{
		Sig:  sig,
		Fn:   fn,
		Args: args,
		Dest: dest,
		Succ: succ,
	})
}

// EmitTailcall emits a tail call (no successor).
func (b *InstrBuilder) EmitTailcall(sig rtl.Sig, fn rtl.FunRef, args []rtl.Reg) rtl.Node {
	return b.cfg.EmitInstr(rtl.Itailcall{
		Sig:  sig,
		Fn:   fn,
		Args: args,
	})
}

// EmitCond emits a conditional branch: if cond(args) goto ifso else ifnot
func (b *InstrBuilder) EmitCond(cond rtl.ConditionCode, args []rtl.Reg, ifso, ifnot rtl.Node) rtl.Node {
	return b.cfg.EmitInstr(rtl.Icond{
		Cond:  cond,
		Args:  args,
		IfSo:  ifso,
		IfNot: ifnot,
	})
}

// EmitJumptable emits a jump table: jump to targets[arg]
func (b *InstrBuilder) EmitJumptable(arg rtl.Reg, targets []rtl.Node) rtl.Node {
	return b.cfg.EmitInstr(rtl.Ijumptable{
		Arg:     arg,
		Targets: targets,
	})
}

// EmitReturn emits a function return.
func (b *InstrBuilder) EmitReturn(arg *rtl.Reg) rtl.Node {
	return b.cfg.EmitInstr(rtl.Ireturn{Arg: arg})
}

// EmitNop emits a no-op instruction: goto succ
func (b *InstrBuilder) EmitNop(succ rtl.Node) rtl.Node {
	return b.cfg.EmitInstr(rtl.Inop{Succ: succ})
}

// AllocNode allocates a fresh CFG node.
func (b *InstrBuilder) AllocNode() rtl.Node {
	return b.cfg.AllocNode()
}

// Fresh allocates a fresh register.
func (b *InstrBuilder) Fresh() rtl.Reg {
	return b.regs.Fresh()
}
