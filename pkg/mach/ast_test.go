package mach

import "testing"

func TestLabelValid(t *testing.T) {
	tests := []struct {
		label Label
		want  bool
	}{
		{Label(0), false},
		{Label(1), true},
		{Label(100), true},
		{Label(-1), false},
	}
	for _, tt := range tests {
		if got := tt.label.Valid(); got != tt.want {
			t.Errorf("Label(%d).Valid() = %v, want %v", tt.label, got, tt.want)
		}
	}
}

func TestInstructionTypes(t *testing.T) {
	// Verify all instruction types implement the Instruction interface
	var _ Instruction = Mgetstack{}
	var _ Instruction = Msetstack{}
	var _ Instruction = Mgetparam{}
	var _ Instruction = Mop{}
	var _ Instruction = Mload{}
	var _ Instruction = Mstore{}
	var _ Instruction = Mcall{}
	var _ Instruction = Mtailcall{}
	var _ Instruction = Mbuiltin{}
	var _ Instruction = Mlabel{}
	var _ Instruction = Mgoto{}
	var _ Instruction = Mcond{}
	var _ Instruction = Mjumptable{}
	var _ Instruction = Mreturn{}
}

func TestFunRefTypes(t *testing.T) {
	// Verify FunRef implementations
	var _ FunRef = FunReg{}
	var _ FunRef = FunSymbol{}
}

func TestMgetstack(t *testing.T) {
	inst := Mgetstack{
		Ofs:  16,
		Ty:   Tlong,
		Dest: X0,
	}
	if inst.Ofs != 16 {
		t.Errorf("Mgetstack.Ofs = %d, want 16", inst.Ofs)
	}
	if inst.Ty != Tlong {
		t.Errorf("Mgetstack.Ty = %v, want Tlong", inst.Ty)
	}
	if inst.Dest != X0 {
		t.Errorf("Mgetstack.Dest = %v, want X0", inst.Dest)
	}
}

func TestMsetstack(t *testing.T) {
	inst := Msetstack{
		Src: X1,
		Ofs: 24,
		Ty:  Tint,
	}
	if inst.Src != X1 {
		t.Errorf("Msetstack.Src = %v, want X1", inst.Src)
	}
	if inst.Ofs != 24 {
		t.Errorf("Msetstack.Ofs = %d, want 24", inst.Ofs)
	}
	if inst.Ty != Tint {
		t.Errorf("Msetstack.Ty = %v, want Tint", inst.Ty)
	}
}

func TestMgetparam(t *testing.T) {
	inst := Mgetparam{
		Ofs:  32,
		Ty:   Tlong,
		Dest: X2,
	}
	if inst.Ofs != 32 {
		t.Errorf("Mgetparam.Ofs = %d, want 32", inst.Ofs)
	}
}

func TestMop(t *testing.T) {
	inst := Mop{
		Op:   Oadd{},
		Args: []MReg{X0, X1},
		Dest: X2,
	}
	if _, ok := inst.Op.(Oadd); !ok {
		t.Errorf("Mop.Op is not Oadd")
	}
	if len(inst.Args) != 2 {
		t.Errorf("Mop.Args length = %d, want 2", len(inst.Args))
	}
}

func TestMload(t *testing.T) {
	inst := Mload{
		Chunk: Mint64,
		Args:  []MReg{X0},
		Dest:  X1,
	}
	if inst.Chunk != Mint64 {
		t.Errorf("Mload.Chunk = %v, want Mint64", inst.Chunk)
	}
}

func TestMstore(t *testing.T) {
	inst := Mstore{
		Chunk: Mint32,
		Args:  []MReg{X0},
		Src:   X1,
	}
	if inst.Chunk != Mint32 {
		t.Errorf("Mstore.Chunk = %v, want Mint32", inst.Chunk)
	}
}

func TestMcall(t *testing.T) {
	inst := Mcall{
		Sig: Sig{},
		Fn:  FunSymbol{Name: "foo"},
	}
	if sym, ok := inst.Fn.(FunSymbol); !ok || sym.Name != "foo" {
		t.Errorf("Mcall.Fn = %v, want FunSymbol{foo}", inst.Fn)
	}
}

func TestMcallReg(t *testing.T) {
	inst := Mcall{
		Sig: Sig{},
		Fn:  FunReg{Reg: X8},
	}
	if reg, ok := inst.Fn.(FunReg); !ok || reg.Reg != X8 {
		t.Errorf("Mcall.Fn = %v, want FunReg{X8}", inst.Fn)
	}
}

func TestMtailcall(t *testing.T) {
	inst := Mtailcall{
		Sig: Sig{},
		Fn:  FunSymbol{Name: "bar"},
	}
	if sym, ok := inst.Fn.(FunSymbol); !ok || sym.Name != "bar" {
		t.Errorf("Mtailcall.Fn = %v, want FunSymbol{bar}", inst.Fn)
	}
}

func TestMbuiltin(t *testing.T) {
	dest := X0
	inst := Mbuiltin{
		Builtin: "__builtin_memcpy",
		Args:    []MReg{X1, X2},
		Dest:    &dest,
	}
	if inst.Builtin != "__builtin_memcpy" {
		t.Errorf("Mbuiltin.Builtin = %s, want __builtin_memcpy", inst.Builtin)
	}
	if inst.Dest == nil || *inst.Dest != X0 {
		t.Errorf("Mbuiltin.Dest = %v, want X0", inst.Dest)
	}
}

func TestMbuiltinNoResult(t *testing.T) {
	inst := Mbuiltin{
		Builtin: "__builtin_trap",
		Args:    []MReg{},
		Dest:    nil,
	}
	if inst.Dest != nil {
		t.Errorf("Mbuiltin.Dest = %v, want nil", inst.Dest)
	}
}

func TestMlabel(t *testing.T) {
	inst := Mlabel{Lbl: Label(5)}
	if inst.Lbl != Label(5) {
		t.Errorf("Mlabel.Lbl = %v, want 5", inst.Lbl)
	}
}

func TestMgoto(t *testing.T) {
	inst := Mgoto{Target: Label(10)}
	if inst.Target != Label(10) {
		t.Errorf("Mgoto.Target = %v, want 10", inst.Target)
	}
}

func TestMcond(t *testing.T) {
	inst := Mcond{
		Cond: Ccomp{Cond: Ceq},
		Args: []MReg{X0, X1},
		IfSo: Label(3),
	}
	if cc, ok := inst.Cond.(Ccomp); !ok || cc.Cond != Ceq {
		t.Errorf("Mcond.Cond is not Ccomp{Ceq}")
	}
	if inst.IfSo != Label(3) {
		t.Errorf("Mcond.IfSo = %v, want 3", inst.IfSo)
	}
}

func TestMjumptable(t *testing.T) {
	inst := Mjumptable{
		Arg:     X0,
		Targets: []Label{Label(1), Label(2), Label(3)},
	}
	if len(inst.Targets) != 3 {
		t.Errorf("Mjumptable.Targets length = %d, want 3", len(inst.Targets))
	}
}

func TestMreturn(t *testing.T) {
	inst := Mreturn{}
	_ = inst // Just verify it can be created
}

func TestNewFunction(t *testing.T) {
	fn := NewFunction("test_func", Sig{})
	if fn.Name != "test_func" {
		t.Errorf("Function.Name = %s, want test_func", fn.Name)
	}
	if fn.Code == nil {
		t.Error("Function.Code is nil")
	}
	if fn.CalleeSaveRegs == nil {
		t.Error("Function.CalleeSaveRegs is nil")
	}
	if !fn.UsesFramePtr {
		t.Error("Function.UsesFramePtr should default to true")
	}
}

func TestFunctionAppend(t *testing.T) {
	fn := NewFunction("test", Sig{})
	fn.Append(Mlabel{Lbl: Label(1)})
	fn.Append(Mop{Op: Oadd{}, Args: []MReg{X0, X1}, Dest: X2})
	fn.Append(Mreturn{})

	if len(fn.Code) != 3 {
		t.Errorf("len(fn.Code) = %d, want 3", len(fn.Code))
	}
}

func TestFunctionLabels(t *testing.T) {
	fn := NewFunction("test", Sig{})
	fn.Append(Mlabel{Lbl: Label(1)})
	fn.Append(Mop{Op: Omove{}, Args: nil, Dest: X0})
	fn.Append(Mlabel{Lbl: Label(2)})
	fn.Append(Mlabel{Lbl: Label(1)}) // duplicate
	fn.Append(Mreturn{})

	labels := fn.Labels()
	if len(labels) != 2 {
		t.Errorf("len(labels) = %d, want 2", len(labels))
	}
}

func TestFunctionReferencedLabels(t *testing.T) {
	fn := NewFunction("test", Sig{})
	fn.Append(Mlabel{Lbl: Label(1)})
	fn.Append(Mcond{Cond: Ccomp{Cond: Ceq}, Args: []MReg{X0}, IfSo: Label(2)})
	fn.Append(Mlabel{Lbl: Label(2)})
	fn.Append(Mgoto{Target: Label(1)})
	fn.Append(Mjumptable{Arg: X0, Targets: []Label{Label(1), Label(3)}})
	fn.Append(Mreturn{})

	refs := fn.ReferencedLabels()
	// Should have 2, 1, 3 (deduped: 3 unique)
	if len(refs) != 3 {
		t.Errorf("len(refs) = %d, want 3", len(refs))
	}
}

func TestFunctionWithCalleeSaveRegs(t *testing.T) {
	fn := NewFunction("test", Sig{})
	fn.CalleeSaveRegs = []MReg{X29, X30}
	fn.Stacksize = 32

	if len(fn.CalleeSaveRegs) != 2 {
		t.Errorf("len(CalleeSaveRegs) = %d, want 2", len(fn.CalleeSaveRegs))
	}
	if fn.Stacksize != 32 {
		t.Errorf("Stacksize = %d, want 32", fn.Stacksize)
	}
}

func TestProgram(t *testing.T) {
	prog := Program{
		Globals: []GlobVar{
			{Name: "global_x", Size: 8, Init: nil},
		},
		Functions: []Function{
			*NewFunction("main", Sig{}),
		},
	}

	if len(prog.Globals) != 1 {
		t.Errorf("len(Globals) = %d, want 1", len(prog.Globals))
	}
	if len(prog.Functions) != 1 {
		t.Errorf("len(Functions) = %d, want 1", len(prog.Functions))
	}
}

func TestGlobVar(t *testing.T) {
	gv := GlobVar{
		Name: "test_global",
		Size: 16,
		Init: []byte{1, 2, 3, 4},
	}
	if gv.Name != "test_global" {
		t.Errorf("GlobVar.Name = %s, want test_global", gv.Name)
	}
	if gv.Size != 16 {
		t.Errorf("GlobVar.Size = %d, want 16", gv.Size)
	}
	if len(gv.Init) != 4 {
		t.Errorf("len(GlobVar.Init) = %d, want 4", len(gv.Init))
	}
}
