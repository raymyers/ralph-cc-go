package cminorgen

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
)

func TestChunkForSize(t *testing.T) {
	tests := []struct {
		size int64
		want cminor.Chunk
	}{
		{1, cminor.Mint8signed},
		{2, cminor.Mint16signed},
		{4, cminor.Mint32},
		{8, cminor.Mint64},
		{16, cminor.Many64}, // array/struct
	}
	for _, tt := range tests {
		got := chunkForSize(tt.size)
		if got != tt.want {
			t.Errorf("chunkForSize(%d) = %v, want %v", tt.size, got, tt.want)
		}
	}
}

func TestClassifyVariablesEmpty(t *testing.T) {
	env := ClassifyVariables(nil, csharpminor.Sskip{})
	if len(env.Vars) != 0 {
		t.Errorf("expected 0 vars, got %d", len(env.Vars))
	}
	if env.StackSize != 0 {
		t.Errorf("expected stack size 0, got %d", env.StackSize)
	}
}

func TestClassifyVariablesAllRegister(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "x", Size: 4},
		{Name: "y", Size: 8},
	}
	// No address-taken operations
	body := csharpminor.Sset{
		TempID: 0,
		RHS:    csharpminor.Etempvar{ID: 1},
	}

	env := ClassifyVariables(locals, body)

	if !env.IsRegister("x") {
		t.Error("x should be register")
	}
	if !env.IsRegister("y") {
		t.Error("y should be register")
	}
	if env.StackSize != 0 {
		t.Errorf("stack size should be 0, got %d", env.StackSize)
	}
}

func TestClassifyVariablesAddressTaken(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "x", Size: 4}, // address taken
		{Name: "y", Size: 8}, // not address taken
	}
	// Take address of x
	body := csharpminor.Sset{
		TempID: 0,
		RHS:    csharpminor.Eaddrof{Name: "x"},
	}

	env := ClassifyVariables(locals, body)

	if env.IsRegister("x") {
		t.Error("x should be stack (address taken)")
	}
	if !env.IsStack("x") {
		t.Error("x should be stack")
	}
	if !env.IsRegister("y") {
		t.Error("y should be register")
	}
}

func TestClassifyVariablesStackOffset(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "a", Size: 4}, // address taken, offset 0
		{Name: "b", Size: 8}, // address taken, offset 8 (aligned)
		{Name: "c", Size: 4}, // not address taken
	}
	body := csharpminor.Sseq{
		First:  csharpminor.Sset{TempID: 0, RHS: csharpminor.Eaddrof{Name: "a"}},
		Second: csharpminor.Sset{TempID: 1, RHS: csharpminor.Eaddrof{Name: "b"}},
	}

	env := ClassifyVariables(locals, body)

	if env.GetStackOffset("a") != 0 {
		t.Errorf("a offset = %d, want 0", env.GetStackOffset("a"))
	}
	if env.GetStackOffset("b") != 8 {
		t.Errorf("b offset = %d, want 8", env.GetStackOffset("b"))
	}
	if env.GetStackOffset("c") != -1 {
		t.Errorf("c should not have stack offset, got %d", env.GetStackOffset("c"))
	}
	// Stack size: a(4) + padding(4) + b(8) = 16
	if env.StackSize != 16 {
		t.Errorf("stack size = %d, want 16", env.StackSize)
	}
}

func TestVarEnvRegisterAndStackVars(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "stack_var", Size: 4},
		{Name: "reg_var", Size: 4},
	}
	body := csharpminor.Sset{TempID: 0, RHS: csharpminor.Eaddrof{Name: "stack_var"}}

	env := ClassifyVariables(locals, body)

	regVars := env.RegisterVars()
	if len(regVars) != 1 || regVars[0] != "reg_var" {
		t.Errorf("RegisterVars() = %v, want [reg_var]", regVars)
	}

	stackVars := env.StackVars()
	if len(stackVars) != 1 || stackVars[0] != "stack_var" {
		t.Errorf("StackVars() = %v, want [stack_var]", stackVars)
	}
}

func TestGetChunk(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "i", Size: 4},
		{Name: "l", Size: 8},
		{Name: "c", Size: 1},
	}
	env := ClassifyVariables(locals, csharpminor.Sskip{})

	if env.GetChunk("i") != cminor.Mint32 {
		t.Errorf("GetChunk(i) = %v, want Mint32", env.GetChunk("i"))
	}
	if env.GetChunk("l") != cminor.Mint64 {
		t.Errorf("GetChunk(l) = %v, want Mint64", env.GetChunk("l"))
	}
	if env.GetChunk("c") != cminor.Mint8signed {
		t.Errorf("GetChunk(c) = %v, want Mint8signed", env.GetChunk("c"))
	}
	// Unknown variable
	if env.GetChunk("unknown") != cminor.Many32 {
		t.Errorf("GetChunk(unknown) = %v, want Many32", env.GetChunk("unknown"))
	}
}

func TestTransformAddrOf(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "x", Size: 4}, // offset 0
		{Name: "y", Size: 8}, // offset 8
	}
	body := csharpminor.Sseq{
		First:  csharpminor.Sset{TempID: 0, RHS: csharpminor.Eaddrof{Name: "x"}},
		Second: csharpminor.Sset{TempID: 1, RHS: csharpminor.Eaddrof{Name: "y"}},
	}
	env := ClassifyVariables(locals, body)

	// &x at offset 0
	addrX := env.TransformAddrOf("x")
	if ec, ok := addrX.(cminor.Econst); ok {
		if lc, ok := ec.Const.(cminor.Olongconst); ok {
			if lc.Value != 0 {
				t.Errorf("&x offset = %d, want 0", lc.Value)
			}
		} else {
			t.Errorf("expected Olongconst, got %T", ec.Const)
		}
	} else {
		t.Errorf("expected Econst, got %T", addrX)
	}

	// &y at offset 8
	addrY := env.TransformAddrOf("y")
	if ec, ok := addrY.(cminor.Econst); ok {
		if lc, ok := ec.Const.(cminor.Olongconst); ok {
			if lc.Value != 8 {
				t.Errorf("&y offset = %d, want 8", lc.Value)
			}
		} else {
			t.Errorf("expected Olongconst, got %T", ec.Const)
		}
	} else {
		t.Errorf("expected Econst, got %T", addrY)
	}
}

func TestTransformVarReadRegister(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "x", Size: 4},
	}
	env := ClassifyVariables(locals, csharpminor.Sskip{})

	expr := env.TransformVarRead("x")
	if ev, ok := expr.(cminor.Evar); ok {
		if ev.Name != "x" {
			t.Errorf("Evar name = %q, want %q", ev.Name, "x")
		}
	} else {
		t.Errorf("expected Evar, got %T", expr)
	}
}

func TestTransformVarReadStack(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "x", Size: 4},
	}
	body := csharpminor.Sset{TempID: 0, RHS: csharpminor.Eaddrof{Name: "x"}}
	env := ClassifyVariables(locals, body)

	expr := env.TransformVarRead("x")
	if el, ok := expr.(cminor.Eload); ok {
		if el.Chunk != cminor.Mint32 {
			t.Errorf("Eload chunk = %v, want Mint32", el.Chunk)
		}
		// Address should be constant 0 (offset of x)
		if ec, ok := el.Addr.(cminor.Econst); ok {
			if lc, ok := ec.Const.(cminor.Olongconst); ok {
				if lc.Value != 0 {
					t.Errorf("Eload addr offset = %d, want 0", lc.Value)
				}
			} else {
				t.Errorf("expected Olongconst, got %T", ec.Const)
			}
		} else {
			t.Errorf("expected Econst addr, got %T", el.Addr)
		}
	} else {
		t.Errorf("expected Eload, got %T", expr)
	}
}

func TestTransformVarReadGlobal(t *testing.T) {
	// Global variable not in locals list
	env := ClassifyVariables(nil, csharpminor.Sskip{})

	expr := env.TransformVarRead("global_var")
	if ev, ok := expr.(cminor.Evar); ok {
		if ev.Name != "global_var" {
			t.Errorf("Evar name = %q, want %q", ev.Name, "global_var")
		}
	} else {
		t.Errorf("expected Evar for global, got %T", expr)
	}
}

func TestTransformVarWriteRegister(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "x", Size: 4},
	}
	env := ClassifyVariables(locals, csharpminor.Sskip{})

	value := cminor.Econst{Const: cminor.Ointconst{Value: 42}}
	stmt := env.TransformVarWrite("x", value)

	if sa, ok := stmt.(cminor.Sassign); ok {
		if sa.Name != "x" {
			t.Errorf("Sassign name = %q, want %q", sa.Name, "x")
		}
	} else {
		t.Errorf("expected Sassign, got %T", stmt)
	}
}

func TestTransformVarWriteStack(t *testing.T) {
	locals := []csharpminor.VarDecl{
		{Name: "x", Size: 4},
	}
	body := csharpminor.Sset{TempID: 0, RHS: csharpminor.Eaddrof{Name: "x"}}
	env := ClassifyVariables(locals, body)

	value := cminor.Econst{Const: cminor.Ointconst{Value: 42}}
	stmt := env.TransformVarWrite("x", value)

	if ss, ok := stmt.(cminor.Sstore); ok {
		if ss.Chunk != cminor.Mint32 {
			t.Errorf("Sstore chunk = %v, want Mint32", ss.Chunk)
		}
		// Address should be constant 0 (offset of x)
		if ec, ok := ss.Addr.(cminor.Econst); ok {
			if lc, ok := ec.Const.(cminor.Olongconst); ok {
				if lc.Value != 0 {
					t.Errorf("Sstore addr offset = %d, want 0", lc.Value)
				}
			}
		}
	} else {
		t.Errorf("expected Sstore, got %T", stmt)
	}
}

func TestTransformVarWriteUnknown(t *testing.T) {
	// Unknown variable (could be param) -> treat as register
	env := ClassifyVariables(nil, csharpminor.Sskip{})

	value := cminor.Econst{Const: cminor.Ointconst{Value: 42}}
	stmt := env.TransformVarWrite("param", value)

	if sa, ok := stmt.(cminor.Sassign); ok {
		if sa.Name != "param" {
			t.Errorf("Sassign name = %q, want %q", sa.Name, "param")
		}
	} else {
		t.Errorf("expected Sassign, got %T", stmt)
	}
}

func TestComplexStackLayout(t *testing.T) {
	// Multiple address-taken variables with different sizes
	locals := []csharpminor.VarDecl{
		{Name: "c", Size: 1},  // char, address taken
		{Name: "i", Size: 4},  // int, address taken
		{Name: "l", Size: 8},  // long, address taken
		{Name: "r", Size: 4},  // int, register
	}
	body := csharpminor.Sseq{
		First: csharpminor.Sset{TempID: 0, RHS: csharpminor.Eaddrof{Name: "c"}},
		Second: csharpminor.Sseq{
			First:  csharpminor.Sset{TempID: 1, RHS: csharpminor.Eaddrof{Name: "i"}},
			Second: csharpminor.Sset{TempID: 2, RHS: csharpminor.Eaddrof{Name: "l"}},
		},
	}
	env := ClassifyVariables(locals, body)

	// c: offset 0, size 1
	if env.GetStackOffset("c") != 0 {
		t.Errorf("c offset = %d, want 0", env.GetStackOffset("c"))
	}

	// i: offset 4 (aligned to 4), size 4
	if env.GetStackOffset("i") != 4 {
		t.Errorf("i offset = %d, want 4", env.GetStackOffset("i"))
	}

	// l: offset 8 (aligned to 8), size 8
	if env.GetStackOffset("l") != 8 {
		t.Errorf("l offset = %d, want 8", env.GetStackOffset("l"))
	}

	// r is register, no stack offset
	if env.GetStackOffset("r") != -1 {
		t.Errorf("r should not have stack offset")
	}

	// Total: c(1) + pad(3) + i(4) + l(8) = 16
	if env.StackSize != 16 {
		t.Errorf("stack size = %d, want 16", env.StackSize)
	}
}
