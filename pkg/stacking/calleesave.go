package stacking

import (
	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
)

// ARM64 callee-saved registers:
// - X19-X28 (integer)
// - D8-D15 (floating point)

// CalleeSaveRegs lists all ARM64 callee-saved integer registers
var CalleeSaveRegs = []ltl.MReg{
	ltl.X19, ltl.X20, ltl.X21, ltl.X22, ltl.X23,
	ltl.X24, ltl.X25, ltl.X26, ltl.X27, ltl.X28,
}

// CalleeSaveFloatRegs lists all ARM64 callee-saved float registers
var CalleeSaveFloatRegs = []ltl.MReg{
	ltl.D8, ltl.D9, ltl.D10, ltl.D11, ltl.D12, ltl.D13, ltl.D14, ltl.D15,
}

// IsCalleeSaved returns true if the register is callee-saved
func IsCalleeSaved(reg ltl.MReg) bool {
	// Integer callee-save: X19-X28
	if reg >= ltl.X19 && reg <= ltl.X28 {
		return true
	}
	// Float callee-save: D8-D15
	if reg >= ltl.D8 && reg <= ltl.D15 {
		return true
	}
	return false
}

// FindUsedCalleeSaveRegs scans a Linear function and returns used callee-saved registers
func FindUsedCalleeSaveRegs(fn *linear.Function) []ltl.MReg {
	used := make(map[ltl.MReg]bool)

	for _, inst := range fn.Code {
		collectRegsFromInst(inst, used)
	}

	// Filter to callee-saved only
	var result []ltl.MReg
	for reg := range used {
		if IsCalleeSaved(reg) {
			result = append(result, reg)
		}
	}

	// Sort for deterministic output (by register number)
	sortRegs(result)
	return result
}

// collectRegsFromInst adds all registers used by an instruction to the set
func collectRegsFromInst(inst linear.Instruction, used map[ltl.MReg]bool) {
	switch i := inst.(type) {
	case linear.Lgetstack:
		used[i.Dest] = true
	case linear.Lsetstack:
		used[i.Src] = true
	case linear.Lop:
		for _, loc := range i.Args {
			if r, ok := loc.(linear.R); ok {
				used[r.Reg] = true
			}
		}
		if r, ok := i.Dest.(linear.R); ok {
			used[r.Reg] = true
		}
	case linear.Lload:
		for _, loc := range i.Args {
			if r, ok := loc.(linear.R); ok {
				used[r.Reg] = true
			}
		}
		if r, ok := i.Dest.(linear.R); ok {
			used[r.Reg] = true
		}
	case linear.Lstore:
		for _, loc := range i.Args {
			if r, ok := loc.(linear.R); ok {
				used[r.Reg] = true
			}
		}
		if r, ok := i.Src.(linear.R); ok {
			used[r.Reg] = true
		}
	case linear.Lcond:
		for _, loc := range i.Args {
			if r, ok := loc.(linear.R); ok {
				used[r.Reg] = true
			}
		}
	case linear.Ljumptable:
		if r, ok := i.Arg.(linear.R); ok {
			used[r.Reg] = true
		}
	}
}

// sortRegs sorts registers by their numeric value
func sortRegs(regs []ltl.MReg) {
	for i := 0; i < len(regs); i++ {
		for j := i + 1; j < len(regs); j++ {
			if regs[i] > regs[j] {
				regs[i], regs[j] = regs[j], regs[i]
			}
		}
	}
}

// CalleeSaveInfo holds information about callee-save register handling
type CalleeSaveInfo struct {
	Regs        []ltl.MReg // list of callee-saved regs to save
	SaveOffsets []int64    // offset from FP for each saved reg
}

// ComputeCalleeSaveInfo computes save locations for callee-saved registers
func ComputeCalleeSaveInfo(layout *FrameLayout, usedRegs []ltl.MReg) *CalleeSaveInfo {
	info := &CalleeSaveInfo{
		Regs:        usedRegs,
		SaveOffsets: make([]int64, len(usedRegs)),
	}

	// Save registers sequentially starting at CalleeSaveOffset
	// Using positive offsets from FP (which equals SP after prologue)
	offset := layout.CalleeSaveOffset
	for i := range usedRegs {
		info.SaveOffsets[i] = offset
		offset += 8 // 8 bytes per register (incrementing, not decrementing)
	}

	return info
}

// PadToEven ensures the list has even length for STP/LDP pairing
// If odd, duplicates the last register (dummy save/restore)
func PadToEven(regs []ltl.MReg) []ltl.MReg {
	if len(regs)%2 == 0 {
		return regs
	}
	// Pad with the register itself (will be a no-op save/restore)
	return append(regs, regs[len(regs)-1])
}
