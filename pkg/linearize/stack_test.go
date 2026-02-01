package linearize

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestStackInfoEmpty(t *testing.T) {
	fn := linear.NewFunction("empty", linear.Sig{})
	info := CollectStackInfo(fn)

	if info.LocalSize != 0 {
		t.Errorf("LocalSize = %d, want 0", info.LocalSize)
	}
	if info.TotalSize() != 0 {
		t.Errorf("TotalSize = %d, want 0", info.TotalSize())
	}
}

func TestStackInfoLocalSlots(t *testing.T) {
	fn := linear.NewFunction("locals", linear.Sig{})

	// Use local slots at offsets 0, 8, 16
	fn.Append(linear.Lgetstack{Slot: linear.SlotLocal, Ofs: 0, Ty: linear.Tlong, Dest: ltl.X0})
	fn.Append(linear.Lgetstack{Slot: linear.SlotLocal, Ofs: 8, Ty: linear.Tlong, Dest: ltl.X1})
	fn.Append(linear.Lsetstack{Src: ltl.X0, Slot: linear.SlotLocal, Ofs: 16, Ty: linear.Tlong})

	info := CollectStackInfo(fn)

	// Max offset 16 + size 8 = 24
	if info.LocalSize != 24 {
		t.Errorf("LocalSize = %d, want 24", info.LocalSize)
	}

	// Total should be aligned to 16
	if info.TotalSize() != 32 {
		t.Errorf("TotalSize = %d, want 32", info.TotalSize())
	}
}

func TestStackInfoIncomingSlots(t *testing.T) {
	fn := linear.NewFunction("incoming", linear.Sig{})

	// Incoming args at offsets 0 and 8
	fn.Append(linear.Lgetstack{Slot: linear.SlotIncoming, Ofs: 0, Ty: linear.Tlong, Dest: ltl.X0})
	fn.Append(linear.Lgetstack{Slot: linear.SlotIncoming, Ofs: 8, Ty: linear.Tlong, Dest: ltl.X1})

	info := CollectStackInfo(fn)

	// Incoming size should be 16
	if info.IncomingSize != 16 {
		t.Errorf("IncomingSize = %d, want 16", info.IncomingSize)
	}

	// But incoming doesn't add to frame size (it's in caller's frame)
	// Only locals and outgoing count
	if info.TotalSize() != 0 {
		t.Errorf("TotalSize = %d, want 0", info.TotalSize())
	}
}

func TestStackInfoOutgoingSlots(t *testing.T) {
	fn := linear.NewFunction("outgoing", linear.Sig{})

	// Outgoing args for calls
	fn.Append(linear.Lsetstack{Src: ltl.X0, Slot: linear.SlotOutgoing, Ofs: 0, Ty: linear.Tlong})
	fn.Append(linear.Lsetstack{Src: ltl.X1, Slot: linear.SlotOutgoing, Ofs: 8, Ty: linear.Tlong})

	info := CollectStackInfo(fn)

	// Outgoing size should be 16
	if info.OutgoingSize != 16 {
		t.Errorf("OutgoingSize = %d, want 16", info.OutgoingSize)
	}

	if info.TotalSize() != 16 {
		t.Errorf("TotalSize = %d, want 16", info.TotalSize())
	}
}

func TestStackInfoMixed(t *testing.T) {
	fn := linear.NewFunction("mixed", linear.Sig{})

	// Local: 24 bytes (offsets 0, 8, 16)
	fn.Append(linear.Lgetstack{Slot: linear.SlotLocal, Ofs: 0, Ty: linear.Tlong, Dest: ltl.X0})
	fn.Append(linear.Lsetstack{Src: ltl.X1, Slot: linear.SlotLocal, Ofs: 16, Ty: linear.Tlong})

	// Outgoing: 16 bytes
	fn.Append(linear.Lsetstack{Src: ltl.X0, Slot: linear.SlotOutgoing, Ofs: 8, Ty: linear.Tlong})

	info := CollectStackInfo(fn)

	// Local = 24, Outgoing = 16
	if info.LocalSize != 24 {
		t.Errorf("LocalSize = %d, want 24", info.LocalSize)
	}
	if info.OutgoingSize != 16 {
		t.Errorf("OutgoingSize = %d, want 16", info.OutgoingSize)
	}

	// Total = 24 + 16 = 40, aligned to 48
	if info.TotalSize() != 48 {
		t.Errorf("TotalSize = %d, want 48", info.TotalSize())
	}
}

func TestStackInfoInt32Slots(t *testing.T) {
	fn := linear.NewFunction("int32", linear.Sig{})

	// Int slots are 4 bytes
	fn.Append(linear.Lgetstack{Slot: linear.SlotLocal, Ofs: 0, Ty: linear.Tint, Dest: ltl.X0})
	fn.Append(linear.Lgetstack{Slot: linear.SlotLocal, Ofs: 4, Ty: linear.Tint, Dest: ltl.X1})

	info := CollectStackInfo(fn)

	// 4 + 4 = 8 bytes
	if info.LocalSize != 8 {
		t.Errorf("LocalSize = %d, want 8", info.LocalSize)
	}
}

func TestStackInfoSlotLocations(t *testing.T) {
	fn := linear.NewFunction("slotloc", linear.Sig{})

	// Ops and loads can reference stack slots via S locations
	fn.Append(linear.Lop{
		Op:   rtl.Ointconst{Value: 42},
		Args: []linear.Loc{linear.S{Slot: linear.SlotLocal, Ofs: 0, Ty: linear.Tlong}},
		Dest: linear.R{Reg: ltl.X0},
	})

	info := CollectStackInfo(fn)

	// Should detect the slot reference
	if info.LocalSize != 8 {
		t.Errorf("LocalSize = %d, want 8", info.LocalSize)
	}
}

func TestComputeStackSize(t *testing.T) {
	fn := linear.NewFunction("compute", linear.Sig{})
	fn.Append(linear.Lgetstack{Slot: linear.SlotLocal, Ofs: 0, Ty: linear.Tlong, Dest: ltl.X0})
	fn.Append(linear.Lsetstack{Src: ltl.X0, Slot: linear.SlotLocal, Ofs: 8, Ty: linear.Tlong})

	ComputeStackSize(fn)

	// 16 bytes, already aligned
	if fn.Stacksize != 16 {
		t.Errorf("Stacksize = %d, want 16", fn.Stacksize)
	}
}

func TestStackInfoAlignment(t *testing.T) {
	tests := []struct {
		localSize int64
		outSize   int64
		expected  int64
	}{
		{0, 0, 0},
		{8, 0, 16},
		{16, 0, 16},
		{17, 0, 32},
		{8, 8, 16},
		{24, 16, 48},
	}

	for _, tt := range tests {
		info := &StackInfo{LocalSize: tt.localSize, OutgoingSize: tt.outSize}
		if got := info.TotalSize(); got != tt.expected {
			t.Errorf("TotalSize(local=%d, out=%d) = %d, want %d",
				tt.localSize, tt.outSize, got, tt.expected)
		}
	}
}
