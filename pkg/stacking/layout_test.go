package stacking

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
)

func TestAlignUp(t *testing.T) {
	tests := []struct {
		n, align, want int64
	}{
		{0, 8, 0},
		{1, 8, 8},
		{7, 8, 8},
		{8, 8, 8},
		{9, 8, 16},
		{15, 16, 16},
		{16, 16, 16},
		{17, 16, 32},
		{0, 16, 0},
	}

	for _, tt := range tests {
		got := alignUp(tt.n, tt.align)
		if got != tt.want {
			t.Errorf("alignUp(%d, %d) = %d, want %d", tt.n, tt.align, got, tt.want)
		}
	}
}

func TestComputeLayoutEmpty(t *testing.T) {
	fn := linear.NewFunction("empty", linear.Sig{})
	layout := ComputeLayout(fn, 0)

	if layout.LocalSize != 0 {
		t.Errorf("LocalSize = %d, want 0", layout.LocalSize)
	}
	if layout.OutgoingSize != 0 {
		t.Errorf("OutgoingSize = %d, want 0", layout.OutgoingSize)
	}
	if layout.CalleeSaveSize != 0 {
		t.Errorf("CalleeSaveSize = %d, want 0", layout.CalleeSaveSize)
	}
	// TotalSize includes 16 bytes for saved FP/LR
	if layout.TotalSize != 16 {
		t.Errorf("TotalSize = %d, want 16", layout.TotalSize)
	}
	if !layout.UseFramePointer {
		t.Error("UseFramePointer should be true")
	}
}

func TestComputeLayoutWithLocals(t *testing.T) {
	fn := linear.NewFunction("withLocals", linear.Sig{})
	// Add a local slot access: 8 bytes at offset 0
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotLocal,
		Ofs:  0,
		Ty:   linear.Tlong,
		Dest: ltl.X0,
	})

	layout := ComputeLayout(fn, 0)

	if layout.LocalSize != 8 {
		t.Errorf("LocalSize = %d, want 8", layout.LocalSize)
	}
	// Frame body is 8 (locals), aligned to 16 = 16
	// Total = 16 (FP/LR) + 16 (body) = 32
	if layout.TotalSize != 32 {
		t.Errorf("TotalSize = %d, want 32", layout.TotalSize)
	}
}

func TestComputeLayoutWithOutgoing(t *testing.T) {
	fn := linear.NewFunction("withOutgoing", linear.Sig{})
	// Add an outgoing slot access: 8 bytes at offset 0
	fn.Append(linear.Lsetstack{
		Src:  ltl.X0,
		Slot: linear.SlotOutgoing,
		Ofs:  0,
		Ty:   linear.Tlong,
	})

	layout := ComputeLayout(fn, 0)

	if layout.OutgoingSize != 8 {
		t.Errorf("OutgoingSize = %d, want 8", layout.OutgoingSize)
	}
}

func TestComputeLayoutWithCalleeSave(t *testing.T) {
	fn := linear.NewFunction("withCalleeSave", linear.Sig{})
	// 3 callee-saved registers -> rounds to 4 for STP/LDP pairs
	layout := ComputeLayout(fn, 3)

	// 4 regs * 8 bytes = 32 bytes
	if layout.CalleeSaveSize != 32 {
		t.Errorf("CalleeSaveSize = %d, want 32", layout.CalleeSaveSize)
	}
}

func TestComputeLayoutWithCalleeSaveEven(t *testing.T) {
	fn := linear.NewFunction("withCalleeSaveEven", linear.Sig{})
	// 4 callee-saved registers (already even)
	layout := ComputeLayout(fn, 4)

	// 4 regs * 8 bytes = 32 bytes
	if layout.CalleeSaveSize != 32 {
		t.Errorf("CalleeSaveSize = %d, want 32", layout.CalleeSaveSize)
	}
}

func TestComputeLayoutOffsets(t *testing.T) {
	fn := linear.NewFunction("offsets", linear.Sig{})
	// 16 bytes of locals
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotLocal,
		Ofs:  0,
		Ty:   linear.Tlong,
		Dest: ltl.X0,
	})
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotLocal,
		Ofs:  8,
		Ty:   linear.Tlong,
		Dest: ltl.X1,
	})

	layout := ComputeLayout(fn, 2) // 2 callee-save regs

	// 2 regs * 8 = 16 bytes callee-save
	if layout.CalleeSaveSize != 16 {
		t.Errorf("CalleeSaveSize = %d, want 16", layout.CalleeSaveSize)
	}
	// 16 bytes of locals
	if layout.LocalSize != 16 {
		t.Errorf("LocalSize = %d, want 16", layout.LocalSize)
	}
	// CalleeSave at -8 (first slot below FP)
	if layout.CalleeSaveOffset != -8 {
		t.Errorf("CalleeSaveOffset = %d, want -8", layout.CalleeSaveOffset)
	}
	// Locals at -32 (below callee-saves: -16 callee-save - 16 locals)
	if layout.LocalOffset != -32 {
		t.Errorf("LocalOffset = %d, want -32", layout.LocalOffset)
	}
}

func TestLocalSlotOffset(t *testing.T) {
	fn := linear.NewFunction("test", linear.Sig{})
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotLocal,
		Ofs:  0,
		Ty:   linear.Tlong,
		Dest: ltl.X0,
	})
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotLocal,
		Ofs:  8,
		Ty:   linear.Tlong,
		Dest: ltl.X1,
	})

	layout := ComputeLayout(fn, 0)

	// With no callee-saves, locals start at -16 (after alignment)
	// localOffset should be -16
	offset0 := layout.LocalSlotOffset(0)
	offset8 := layout.LocalSlotOffset(8)

	// First local at LocalOffset + 0
	if offset0 != layout.LocalOffset {
		t.Errorf("LocalSlotOffset(0) = %d, want %d", offset0, layout.LocalOffset)
	}
	// Second local at LocalOffset + 8
	if offset8 != layout.LocalOffset+8 {
		t.Errorf("LocalSlotOffset(8) = %d, want %d", offset8, layout.LocalOffset+8)
	}
}

func TestIncomingSlotOffset(t *testing.T) {
	fn := linear.NewFunction("test", linear.Sig{})
	layout := ComputeLayout(fn, 0)

	// Incoming args are above FP/LR save area
	// FP+0 = saved FP, FP+8 = saved LR, FP+16 = first incoming arg
	offset0 := layout.IncomingSlotOffset(0)
	offset8 := layout.IncomingSlotOffset(8)

	if offset0 != 16 {
		t.Errorf("IncomingSlotOffset(0) = %d, want 16", offset0)
	}
	if offset8 != 24 {
		t.Errorf("IncomingSlotOffset(8) = %d, want 24", offset8)
	}
}

func TestOutgoingSlotOffset(t *testing.T) {
	fn := linear.NewFunction("test", linear.Sig{})
	layout := ComputeLayout(fn, 0)

	// Outgoing args are relative to SP (at bottom of frame)
	offset0 := layout.OutgoingSlotOffset(0)
	offset8 := layout.OutgoingSlotOffset(8)

	if offset0 != 0 {
		t.Errorf("OutgoingSlotOffset(0) = %d, want 0", offset0)
	}
	if offset8 != 8 {
		t.Errorf("OutgoingSlotOffset(8) = %d, want 8", offset8)
	}
}

func TestComputeLayoutAlignment(t *testing.T) {
	// Test that total frame size is 16-byte aligned
	fn := linear.NewFunction("alignment", linear.Sig{})
	// 4 bytes local (int)
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotLocal,
		Ofs:  0,
		Ty:   linear.Tint,
		Dest: ltl.X0,
	})

	layout := ComputeLayout(fn, 1) // 1 callee-save reg -> padded to 2 = 16 bytes

	// Check 16-byte alignment of total
	if layout.TotalSize%16 != 0 {
		t.Errorf("TotalSize %d is not 16-byte aligned", layout.TotalSize)
	}
}

func TestSlotSize(t *testing.T) {
	tests := []struct {
		ty   linear.Typ
		want int64
	}{
		{linear.Tint, 4},
		{linear.Tsingle, 4},
		{linear.Tany32, 4},
		{linear.Tfloat, 8},
		{linear.Tlong, 8},
		{linear.Tany64, 8},
	}

	for _, tt := range tests {
		got := slotSize(tt.ty)
		if got != tt.want {
			t.Errorf("slotSize(%v) = %d, want %d", tt.ty, got, tt.want)
		}
	}
}

func TestCollectStackInfoMultipleSlots(t *testing.T) {
	fn := linear.NewFunction("multi", linear.Sig{})
	// Local at offset 0 (8 bytes)
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotLocal,
		Ofs:  0,
		Ty:   linear.Tlong,
		Dest: ltl.X0,
	})
	// Local at offset 8 (8 bytes) -> total local = 16
	fn.Append(linear.Lsetstack{
		Src:  ltl.X0,
		Slot: linear.SlotLocal,
		Ofs:  8,
		Ty:   linear.Tlong,
	})
	// Incoming at offset 0 (8 bytes)
	fn.Append(linear.Lgetstack{
		Slot: linear.SlotIncoming,
		Ofs:  0,
		Ty:   linear.Tlong,
		Dest: ltl.X1,
	})
	// Outgoing at offset 0 (4 bytes)
	fn.Append(linear.Lsetstack{
		Src:  ltl.X0,
		Slot: linear.SlotOutgoing,
		Ofs:  0,
		Ty:   linear.Tint,
	})

	info := collectStackInfo(fn)

	if info.LocalSize != 16 {
		t.Errorf("LocalSize = %d, want 16", info.LocalSize)
	}
	if info.IncomingSize != 8 {
		t.Errorf("IncomingSize = %d, want 8", info.IncomingSize)
	}
	if info.OutgoingSize != 4 {
		t.Errorf("OutgoingSize = %d, want 4", info.OutgoingSize)
	}
}
