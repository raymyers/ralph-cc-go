// Stack slot assignment for Linear code.
// This pass analyzes stack slot usage and computes frame size requirements.
package linearize

import "github.com/raymyers/ralph-cc/pkg/linear"

const stackAlignment = 16 // ARM64 requires 16-byte stack alignment

// ComputeStackSize analyzes a Linear function and updates its Stacksize field.
// It ensures proper alignment and accounts for all slot types.
func ComputeStackSize(fn *linear.Function) {
	info := CollectStackInfo(fn)
	fn.Stacksize = info.TotalSize()
}

// StackInfo holds information about stack slot usage
type StackInfo struct {
	LocalSize    int64 // total size of local slots
	IncomingSize int64 // size of incoming arguments (for callee)
	OutgoingSize int64 // max size of outgoing arguments (for calls)
}

// TotalSize returns the aligned total stack frame size
func (s *StackInfo) TotalSize() int64 {
	// Total = locals + outgoing args
	// (Incoming args are in caller's frame)
	total := s.LocalSize + s.OutgoingSize

	// Align to 16 bytes
	if total%stackAlignment != 0 {
		total = ((total / stackAlignment) + 1) * stackAlignment
	}

	return total
}

// CollectStackInfo scans a Linear function and collects stack slot information
func CollectStackInfo(fn *linear.Function) *StackInfo {
	info := &StackInfo{}

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

		case linear.Lcall:
			// Outgoing args depend on call signature
			// For simplicity, we track slot references above
			// A more accurate implementation would look at the signature

		case linear.Lop:
			// Check for stack slot locations in operation args
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

// checkSlotLoc updates max sizes based on a location if it's a stack slot
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

// slotSize returns the size in bytes for a given type
func slotSize(ty linear.Typ) int64 {
	switch ty {
	case linear.Tint, linear.Tsingle, linear.Tany32:
		return 4
	case linear.Tfloat, linear.Tlong, linear.Tany64:
		return 8
	default:
		return 8 // default to 8 bytes
	}
}
