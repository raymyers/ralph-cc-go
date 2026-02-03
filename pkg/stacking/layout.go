// Package stacking transforms Linear to Mach by laying out activation records.
// This involves computing concrete stack offsets and generating prologue/epilogue.
// This mirrors CompCert's backend/Stacking.v and backend/Bounds.v
package stacking

import "github.com/raymyers/ralph-cc/pkg/linear"

const (
	stackAlignment = 16 // ARM64 requires 16-byte stack alignment
	pointerSize    = 8  // 64-bit pointers
)

// ARM64 frame layout (called function's view):
//
//	+---------------------------+  <- old SP (before call)
//	| LR (return address)       |  +16 from new FP
//	| Old FP                    |  +8 from new FP
//	+---------------------------+  <- FP points here (after setup)
//	| Callee-saved registers    |  negative offsets from FP
//	| Local variables           |
//	| Outgoing arguments        |
//	+---------------------------+  <- SP (16-byte aligned)
//
// Incoming arguments from caller are at positive offsets from FP.

// FrameLayout describes the concrete stack frame layout
type FrameLayout struct {
	// Sizes for each section (in bytes)
	CalleeSaveSize int64 // space for callee-saved registers
	LocalSize      int64 // space for local variables
	OutgoingSize   int64 // space for outgoing call arguments

	// Computed offsets (from FP)
	CalleeSaveOffset int64 // start of callee-save area (negative)
	LocalOffset      int64 // start of locals area (negative)
	OutgoingOffset   int64 // start of outgoing area (negative)

	// Total frame size (SP decrement from old SP)
	TotalSize int64

	// Whether we use frame pointer
	UseFramePointer bool
}

// ComputeLayout computes the frame layout for a Linear function
func ComputeLayout(fn *linear.Function, calleeSaveRegs int) *FrameLayout {
	layout := &FrameLayout{
		UseFramePointer: true, // ARM64 typically uses FP
	}

	// Collect stack info from the function
	info := collectStackInfo(fn)

	// Callee-save area: 8 bytes per saved register, paired for STP/LDP
	// Round up to even number for paired stores
	numRegs := calleeSaveRegs
	if numRegs%2 != 0 {
		numRegs++ // pad to even for STP/LDP pairs
	}
	layout.CalleeSaveSize = int64(numRegs) * pointerSize

	// Local variable area
	layout.LocalSize = alignUp(info.LocalSize, 8)

	// Outgoing argument area
	layout.OutgoingSize = alignUp(info.OutgoingSize, 8)

	// Compute offsets from FP
	// After prologue: FP = SP, and FP/LR were saved at [SP] and [SP+8].
	// Frame layout from FP (low to high addresses):
	//   [FP + 0]                        : saved old FP
	//   [FP + 8]                        : saved LR
	//   [FP + 16 ... +16+CalleeSaveSize-1] : callee-saved registers
	//   [FP + 16 + CalleeSaveSize ...]  : locals
	//
	// Callee-save registers start at offset 16 from FP (after FP/LR)
	layout.CalleeSaveOffset = 16

	// Local variables come after FP/LR (16) + callee-saves
	layout.LocalOffset = 16 + layout.CalleeSaveSize

	// Outgoing arguments at the end (not typically used with this layout)
	layout.OutgoingOffset = 16 + layout.CalleeSaveSize + layout.LocalSize

	// Total frame size: includes FP/LR save area (16 bytes) plus our sections
	// This is the amount SP is decremented from old SP
	frameBody := layout.CalleeSaveSize + layout.LocalSize + layout.OutgoingSize
	frameBody = alignUp(frameBody, stackAlignment) // ensure 16-byte alignment

	// Total includes the saved FP and LR (16 bytes)
	layout.TotalSize = frameBody + 16

	return layout
}

// LocalSlotOffset returns the concrete offset from FP for a local slot
func (l *FrameLayout) LocalSlotOffset(slotOffset int64) int64 {
	return l.LocalOffset + slotOffset
}

// OutgoingSlotOffset returns the concrete offset from SP for an outgoing arg slot
func (l *FrameLayout) OutgoingSlotOffset(slotOffset int64) int64 {
	// Outgoing args are relative to SP, at the bottom of frame
	return slotOffset
}

// IncomingSlotOffset returns the concrete offset from FP for an incoming arg
// Incoming args are in caller's frame, above our FP/LR save area
func (l *FrameLayout) IncomingSlotOffset(slotOffset int64) int64 {
	// FP points at saved FP, saved LR is at FP+8
	// Incoming args start at FP+16
	return 16 + slotOffset
}

// stackInfo holds collected info about stack usage
type stackInfo struct {
	LocalSize    int64
	IncomingSize int64
	OutgoingSize int64
}

// collectStackInfo scans a Linear function for stack slot usage
func collectStackInfo(fn *linear.Function) *stackInfo {
	info := &stackInfo{}

	maxLocal := int64(0)
	maxIncoming := int64(0)
	maxOutgoing := int64(0)

	for _, inst := range fn.Code {
		switch i := inst.(type) {
		case linear.Lgetstack:
			size := slotSize(i.Ty)
			switch i.Slot {
			case linear.SlotLocal:
				if end := i.Ofs + size; end > maxLocal {
					maxLocal = end
				}
			case linear.SlotIncoming:
				if end := i.Ofs + size; end > maxIncoming {
					maxIncoming = end
				}
			case linear.SlotOutgoing:
				if end := i.Ofs + size; end > maxOutgoing {
					maxOutgoing = end
				}
			}

		case linear.Lsetstack:
			size := slotSize(i.Ty)
			switch i.Slot {
			case linear.SlotLocal:
				if end := i.Ofs + size; end > maxLocal {
					maxLocal = end
				}
			case linear.SlotIncoming:
				if end := i.Ofs + size; end > maxIncoming {
					maxIncoming = end
				}
			case linear.SlotOutgoing:
				if end := i.Ofs + size; end > maxOutgoing {
					maxOutgoing = end
				}
			}

		case linear.Lop:
			for _, loc := range i.Args {
				checkSlotLoc(loc, &maxLocal, &maxIncoming, &maxOutgoing)
			}
			checkSlotLoc(i.Dest, &maxLocal, &maxIncoming, &maxOutgoing)

		case linear.Lload:
			for _, loc := range i.Args {
				checkSlotLoc(loc, &maxLocal, &maxIncoming, &maxOutgoing)
			}
			checkSlotLoc(i.Dest, &maxLocal, &maxIncoming, &maxOutgoing)

		case linear.Lstore:
			for _, loc := range i.Args {
				checkSlotLoc(loc, &maxLocal, &maxIncoming, &maxOutgoing)
			}
			checkSlotLoc(i.Src, &maxLocal, &maxIncoming, &maxOutgoing)
		}
	}

	info.LocalSize = maxLocal
	info.IncomingSize = maxIncoming
	info.OutgoingSize = maxOutgoing

	return info
}

// checkSlotLoc updates max sizes if loc is a stack slot
func checkSlotLoc(loc linear.Loc, maxLocal, maxIncoming, maxOutgoing *int64) {
	if s, ok := loc.(linear.S); ok {
		size := slotSize(s.Ty)
		switch s.Slot {
		case linear.SlotLocal:
			if end := s.Ofs + size; end > *maxLocal {
				*maxLocal = end
			}
		case linear.SlotIncoming:
			if end := s.Ofs + size; end > *maxIncoming {
				*maxIncoming = end
			}
		case linear.SlotOutgoing:
			if end := s.Ofs + size; end > *maxOutgoing {
				*maxOutgoing = end
			}
		}
	}
}

// slotSize returns the size in bytes for a type
func slotSize(ty linear.Typ) int64 {
	switch ty {
	case linear.Tint, linear.Tsingle, linear.Tany32:
		return 4
	case linear.Tfloat, linear.Tlong, linear.Tany64:
		return 8
	default:
		return 8
	}
}

// alignUp rounds n up to the nearest multiple of align
func alignUp(n, align int64) int64 {
	if align == 0 {
		return n
	}
	return ((n + align - 1) / align) * align
}
