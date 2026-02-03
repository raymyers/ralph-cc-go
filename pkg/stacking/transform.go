package stacking

import (
	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/mach"
)

// Transform converts a Linear function to Mach code
// This is the main stacking transformation
func Transform(fn *linear.Function) *mach.Function {
	t := &transformer{
		linearFn: fn,
	}
	return t.transform()
}

// TransformProgram transforms a complete Linear program to Mach
func TransformProgram(prog *linear.Program) *mach.Program {
	machProg := &mach.Program{
		Globals: make([]mach.GlobVar, len(prog.Globals)),
	}

	// Copy globals
	for i, g := range prog.Globals {
		machProg.Globals[i] = mach.GlobVar{
			Name:     g.Name,
			Size:     g.Size,
			Init:     g.Init,
			ReadOnly: g.ReadOnly,
		}
	}

	// Transform each function
	for _, fn := range prog.Functions {
		machFn := Transform(&fn)
		machProg.Functions = append(machProg.Functions, *machFn)
	}

	return machProg
}

// transformer holds state during Linear -> Mach transformation
type transformer struct {
	linearFn   *linear.Function
	layout     *FrameLayout
	calleeSave *CalleeSaveInfo
	slotTrans  *SlotTranslator
}

func (t *transformer) transform() *mach.Function {
	// 1. Find callee-saved registers used in the function
	usedCalleeSave := FindUsedCalleeSaveRegs(t.linearFn)
	usedCalleeSave = PadToEven(usedCalleeSave) // Pad for STP/LDP

	// 2. Compute stack frame layout
	t.layout = ComputeLayout(t.linearFn, len(usedCalleeSave))

	// 3. Compute callee-save info
	t.calleeSave = ComputeCalleeSaveInfo(t.layout, usedCalleeSave)

	// 4. Create slot translator
	t.slotTrans = NewSlotTranslator(t.layout)

	// 5. Create Mach function
	machFn := mach.NewFunction(t.linearFn.Name, t.linearFn.Sig)
	machFn.Stacksize = t.layout.TotalSize
	machFn.CalleeSaveRegs = usedCalleeSave
	machFn.UsesFramePtr = t.layout.UseFramePointer

	// 6. Generate prologue
	prologue := GeneratePrologue(t.layout, t.calleeSave)
	for _, inst := range prologue {
		machFn.Append(inst)
	}

	// 6b. Generate parameter copies (move from incoming regs to allocated locations)
	paramCopies := GenerateParamCopies(t.linearFn.Params)
	for _, inst := range paramCopies {
		machFn.Append(inst)
	}

	// 7. Transform body instructions
	for _, inst := range t.linearFn.Code {
		machInsts := t.transformInst(inst)
		for _, mi := range machInsts {
			machFn.Append(mi)
		}
	}

	return machFn
}

// tempRegs are scratch registers for spilling operations during stacking
// Using X16/X17 (IP0/IP1) which are reserved for linker veneers but safe to use here
var stackingTempRegs = []ltl.MReg{ltl.X16, ltl.X17}

// transformInst transforms a single Linear instruction to Mach
// Returns a slice because some instructions expand to multiple
func (t *transformer) transformInst(inst linear.Instruction) []mach.Instruction {
	switch i := inst.(type) {
	case linear.Lgetstack:
		return []mach.Instruction{t.slotTrans.TranslateGetstack(i)}

	case linear.Lsetstack:
		return []mach.Instruction{t.slotTrans.TranslateSetstack(i)}

	case linear.Lop:
		return t.transformLop(i)

	case linear.Lload:
		return []mach.Instruction{mach.Mload{
			Chunk: i.Chunk,
			Addr:  i.Addr,
			Args:  t.locsToRegs(i.Args),
			Dest:  t.locToReg(i.Dest),
		}}

	case linear.Lstore:
		return []mach.Instruction{mach.Mstore{
			Chunk: i.Chunk,
			Addr:  i.Addr,
			Args:  t.locsToRegs(i.Args),
			Src:   t.locToReg(i.Src),
		}}

	case linear.Lcall:
		return []mach.Instruction{mach.Mcall{
			Sig: i.Sig,
			Fn:  t.transformFunRef(i.Fn),
		}}

	case linear.Ltailcall:
		// Tail call: generate epilogue before the call
		epilogue := GenerateTailEpilogue(t.layout, t.calleeSave)
		result := make([]mach.Instruction, 0, len(epilogue)+1)
		result = append(result, epilogue...)
		result = append(result, mach.Mtailcall{
			Sig: i.Sig,
			Fn:  t.transformFunRef(i.Fn),
		})
		return result

	case linear.Lbuiltin:
		var dest *ltl.MReg
		if i.Dest != nil {
			r := t.locToReg(*i.Dest)
			dest = &r
		}
		return []mach.Instruction{mach.Mbuiltin{
			Builtin: i.Builtin,
			Args:    t.locsToRegs(i.Args),
			Dest:    dest,
		}}

	case linear.Llabel:
		return []mach.Instruction{mach.Mlabel{Lbl: mach.Label(i.Lbl)}}

	case linear.Lgoto:
		return []mach.Instruction{mach.Mgoto{Target: mach.Label(i.Target)}}

	case linear.Lcond:
		return []mach.Instruction{mach.Mcond{
			Cond: i.Cond,
			Args: t.locsToRegs(i.Args),
			IfSo: mach.Label(i.IfSo),
		}}

	case linear.Ljumptable:
		targets := make([]mach.Label, len(i.Targets))
		for j, lbl := range i.Targets {
			targets[j] = mach.Label(lbl)
		}
		return []mach.Instruction{mach.Mjumptable{
			Arg:     t.locToReg(i.Arg),
			Targets: targets,
		}}

	case linear.Lreturn:
		// Return: generate epilogue (which includes Mreturn)
		return GenerateEpilogue(t.layout, t.calleeSave)

	default:
		// Unknown instruction - should not happen
		return nil
	}
}

// transformLop handles Lop instructions, generating loads for stack slot args
// and stores for stack slot destinations
func (t *transformer) transformLop(i linear.Lop) []mach.Instruction {
	var result []mach.Instruction
	tempIdx := 0

	// Process arguments - load stack slots into temp registers
	args := make([]ltl.MReg, len(i.Args))
	for j, arg := range i.Args {
		switch loc := arg.(type) {
		case linear.R:
			args[j] = loc.Reg
		case linear.S:
			// Load stack slot into temp register
			if tempIdx >= len(stackingTempRegs) {
				panic("too many stack slot arguments in Lop")
			}
			tempReg := stackingTempRegs[tempIdx]
			tempIdx++
			result = append(result, t.slotTrans.TranslateGetstack(linear.Lgetstack{
				Slot: loc.Slot,
				Ofs:  loc.Ofs,
				Ty:   loc.Ty,
				Dest: tempReg,
			}))
			args[j] = tempReg
		default:
			panic("unknown location type in Lop arg")
		}
	}

	// Process destination
	var destReg ltl.MReg
	var destSlot *linear.S
	if i.Dest == nil {
		// Some operations (like Onop) may have no destination
		// In this case, emit the op without a destination (it should be ignored)
		result = append(result, mach.Mop{
			Op:   i.Op,
			Args: args,
			Dest: 0, // No destination
		})
		return result
	}
	switch loc := i.Dest.(type) {
	case linear.R:
		destReg = loc.Reg
	case linear.S:
		// Use temp register for result, store afterwards
		destReg = stackingTempRegs[0] // Use first temp for dest
		destSlot = &loc
	default:
		panic("unknown location type in Lop dest")
	}

	// Emit the operation
	result = append(result, mach.Mop{
		Op:   i.Op,
		Args: args,
		Dest: destReg,
	})

	// If dest was a stack slot, store the result
	if destSlot != nil {
		result = append(result, t.slotTrans.TranslateSetstack(linear.Lsetstack{
			Src:  destReg,
			Slot: destSlot.Slot,
			Ofs:  destSlot.Ofs,
			Ty:   destSlot.Ty,
		}))
	}

	return result
}

// locsToRegs converts a slice of locations to machine registers
// Panics on stack slots - caller should use transformLop for Lop instructions
func (t *transformer) locsToRegs(locs []linear.Loc) []ltl.MReg {
	regs := make([]ltl.MReg, len(locs))
	for i, loc := range locs {
		regs[i] = t.locToReg(loc)
	}
	return regs
}

// locToReg converts a location to a machine register
// Panics if the location is a stack slot (should have been handled by regalloc)
func (t *transformer) locToReg(loc linear.Loc) ltl.MReg {
	switch l := loc.(type) {
	case linear.R:
		return l.Reg
	case linear.S:
		// Stack slots in operation args should not happen after proper regalloc
		// For now, we panic. A full implementation would need to generate load/store.
		panic("stack slot in register position - regalloc incomplete")
	default:
		panic("unknown location type")
	}
}

// transformFunRef converts a Linear function reference to Mach
func (t *transformer) transformFunRef(fn linear.FunRef) mach.FunRef {
	switch f := fn.(type) {
	case linear.FunReg:
		return mach.FunReg{Reg: t.locToReg(f.Loc)}
	case linear.FunSymbol:
		return mach.FunSymbol{Name: f.Name}
	default:
		panic("unknown function reference type")
	}
}
