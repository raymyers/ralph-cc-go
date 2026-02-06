package stacking

import (
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/mach"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// ARM64 special registers
const (
	FP = ltl.X29 // Frame pointer
	LR = ltl.X30 // Link register (return address)
)

// GeneratePrologue generates the function prologue instructions
// ARM64 prologue:
//  1. Save LR and FP (using STP)
//  2. Set up new FP
//  3. Allocate stack frame
//  4. Save callee-saved registers
func GeneratePrologue(layout *FrameLayout, calleeSave *CalleeSaveInfo) []mach.Instruction {
	var prologue []mach.Instruction

	// The prologue performs:
	// sub sp, sp, #framesize    -- allocate frame
	// stp fp, lr, [sp, #offset] -- save FP and LR
	// add fp, sp, #offset       -- set up frame pointer

	// For simplicity, we represent these as Mop instructions with special ops.
	// In actual assembly generation, these would map to specific ARM64 instructions.

	// 1. Allocate stack frame: sub sp, sp, #TotalSize
	// Represented as addlimm with negative value (sp = sp + (-TotalSize))
	if layout.TotalSize > 0 {
		prologue = append(prologue, mach.Mop{
			Op:   rtl.Oaddlimm{N: -layout.TotalSize},
			Args: nil, // SP is implicit
			Dest: FP,  // Result conceptually goes to SP, represented via FP
		})
	}

	// 2. Save FP and LR
	// stp fp, lr, [sp, #FPoffset]
	// We represent this as two Msetstack operations
	fpSaveOffset := layout.TotalSize - 16 // FP/LR saved at top of frame
	prologue = append(prologue, mach.Msetstack{
		Src: FP,
		Ofs: fpSaveOffset,
		Ty:  ltl.Tlong,
	})
	prologue = append(prologue, mach.Msetstack{
		Src: LR,
		Ofs: fpSaveOffset + 8,
		Ty:  ltl.Tlong,
	})

	// 3. Set up FP: add fp, sp, #offset
	prologue = append(prologue, mach.Mop{
		Op:   rtl.Oaddlimm{N: fpSaveOffset},
		Args: nil, // SP is implicit source
		Dest: FP,
	})

	// 4. Save callee-saved registers
	// For paired saves (STP), we save two at a time
	regs := calleeSave.Regs
	for i := 0; i+1 < len(regs); i += 2 {
		// Save pair at their pre-computed offsets from FP
		prologue = append(prologue, mach.Msetstack{
			Src: regs[i],
			Ofs: calleeSave.SaveOffsets[i],
			Ty:  ltl.Tlong,
		})
		prologue = append(prologue, mach.Msetstack{
			Src: regs[i+1],
			Ofs: calleeSave.SaveOffsets[i+1],
			Ty:  ltl.Tlong,
		})
	}

	return prologue
}

// GenerateEpilogue generates the function epilogue instructions
// ARM64 epilogue:
//  1. Restore callee-saved registers
//  2. Restore FP and LR
//  3. Deallocate stack frame
//  4. Return
func GenerateEpilogue(layout *FrameLayout, calleeSave *CalleeSaveInfo) []mach.Instruction {
	var epilogue []mach.Instruction

	// 1. Restore callee-saved registers (in reverse order)
	regs := calleeSave.Regs
	for i := len(regs) - 2; i >= 0; i -= 2 {
		epilogue = append(epilogue, mach.Mgetstack{
			Ofs:  calleeSave.SaveOffsets[i],
			Ty:   ltl.Tlong,
			Dest: regs[i],
		})
		if i+1 < len(regs) {
			epilogue = append(epilogue, mach.Mgetstack{
				Ofs:  calleeSave.SaveOffsets[i+1],
				Ty:   ltl.Tlong,
				Dest: regs[i+1],
			})
		}
	}

	// 2. Restore FP and LR
	fpSaveOffset := layout.TotalSize - 16
	epilogue = append(epilogue, mach.Mgetstack{
		Ofs:  fpSaveOffset,
		Ty:   ltl.Tlong,
		Dest: FP,
	})
	epilogue = append(epilogue, mach.Mgetstack{
		Ofs:  fpSaveOffset + 8,
		Ty:   ltl.Tlong,
		Dest: LR,
	})

	// 3. Deallocate stack frame: add sp, sp, #TotalSize
	if layout.TotalSize > 0 {
		epilogue = append(epilogue, mach.Mop{
			Op:   rtl.Oaddlimm{N: layout.TotalSize},
			Args: nil,
			Dest: FP, // Conceptually SP
		})
	}

	// 4. Return
	epilogue = append(epilogue, mach.Mreturn{})

	return epilogue
}

// GenerateTailEpilogue generates epilogue for tail calls (without return)
func GenerateTailEpilogue(layout *FrameLayout, calleeSave *CalleeSaveInfo) []mach.Instruction {
	epilogue := GenerateEpilogue(layout, calleeSave)
	// Remove the Mreturn at the end - tail call will replace it
	if len(epilogue) > 0 {
		epilogue = epilogue[:len(epilogue)-1]
	}
	return epilogue
}

// IsLeafFunction returns true if the function doesn't call other functions
// Leaf functions may be able to omit some prologue/epilogue operations
func IsLeafFunction(code []mach.Instruction) bool {
	for _, inst := range code {
		switch inst.(type) {
		case mach.Mcall, mach.Mtailcall:
			return false
		}
	}
	return true
}

// ARM64 argument registers (X0-X7 for integers)
var intArgRegs = []ltl.MReg{ltl.X0, ltl.X1, ltl.X2, ltl.X3, ltl.X4, ltl.X5, ltl.X6, ltl.X7}

// X8 is a good temp register - it's caller-saved and not used for argument passing
const paramCopyTempReg = ltl.X8

// GenerateParamCopies generates move instructions to copy incoming parameters
// from their ABI-specified locations (X0, X1, etc.) to their allocated locations.
// This must be emitted after the prologue, before the function body.
//
// Handles the parallel move problem: when parameters are allocated to registers
// that conflict with incoming argument registers, we need to be careful about
// the order of moves (or use a temporary register to break cycles).
func GenerateParamCopies(params []ltl.Loc) []mach.Instruction {
	// Build a map of moves needed: dest -> src (incoming reg)
	moves := make(map[ltl.MReg]ltl.MReg)
	var stackMoves []mach.Instruction

	for i, paramLoc := range params {
		if i >= len(intArgRegs) {
			break
		}

		incomingReg := intArgRegs[i]

		switch loc := paramLoc.(type) {
		case ltl.R:
			if loc.Reg != incomingReg {
				moves[loc.Reg] = incomingReg
			}

		case ltl.S:
			// Stack moves can be done immediately - no conflict possible
			stackMoves = append(stackMoves, mach.Msetstack{
				Src: incomingReg,
				Ofs: loc.Ofs,
				Ty:  loc.Ty,
			})
		}
	}

	// Now resolve the parallel moves using the algorithm from linearize/convertCall
	// This handles cycles by using a temporary register
	var result []mach.Instruction
	result = append(result, stackMoves...)

	done := make(map[ltl.MReg]bool)

	// isSourceOfPendingMove returns true if reg is the source of a move that hasn't been done
	isSourceOfPendingMove := func(reg ltl.MReg) bool {
		for dest, src := range moves {
			if done[dest] {
				continue
			}
			if src == reg {
				return true
			}
		}
		return false
	}

	for len(done) < len(moves) {
		madeProgress := false

		// Try to find a move where the destination is not needed as a source
		for dest, src := range moves {
			if done[dest] {
				continue
			}

			// Check if this destination register is used as source by another pending move
			if isSourceOfPendingMove(dest) {
				continue
			}

			// Safe to do this move
			result = append(result, mach.Mop{
				Op:   rtl.Omove{},
				Args: []ltl.MReg{src},
				Dest: dest,
			})
			done[dest] = true
			madeProgress = true
		}

		// If no progress, we have a cycle - break it with temp register
		if !madeProgress {
			// Find any pending move and save its source to temp
			for dest, src := range moves {
				if done[dest] {
					continue
				}

				// Save the source value in temp
				result = append(result, mach.Mop{
					Op:   rtl.Omove{},
					Args: []ltl.MReg{src},
					Dest: paramCopyTempReg,
				})

				// Update moves map: any move that uses src as source now uses temp
				for dest2, src2 := range moves {
					if src2 == src {
						moves[dest2] = paramCopyTempReg
					}
				}
				break
			}
		}
	}

	return result
}
