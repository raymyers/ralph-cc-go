package regalloc

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestAllocateSimpleFunction(t *testing.T) {
	// Simple function:
	// 1: x1 = int 1
	// 2: x2 = int 2
	// 3: x3 = add(x1, x2)
	// 4: return x3
	fn := &rtl.Function{
		Name: "simple",
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Ointconst{Value: 1}, Args: nil, Dest: 1, Succ: 2},
			2: rtl.Iop{Op: rtl.Ointconst{Value: 2}, Args: nil, Dest: 2, Succ: 3},
			3: rtl.Iop{Op: rtl.Oadd{}, Args: []rtl.Reg{1, 2}, Dest: 3, Succ: 4},
			4: rtl.Ireturn{Arg: ptr(rtl.Reg(3))},
		},
		Entrypoint: 1,
	}

	result := AllocateFunction(fn)

	// All registers should be allocated (no spills for simple function)
	if len(result.SpilledRegs) != 0 {
		t.Errorf("expected no spills, got %d", len(result.SpilledRegs))
	}

	// Each register should have a location
	for _, r := range []rtl.Reg{1, 2, 3} {
		if _, ok := result.RegToLoc[r]; !ok {
			t.Errorf("register %d should have a location", r)
		}
	}

	// Interfering registers should have different locations
	loc1 := result.RegToLoc[1]
	loc2 := result.RegToLoc[2]

	r1, ok1 := loc1.(ltl.R)
	r2, ok2 := loc2.(ltl.R)

	if ok1 && ok2 && r1.Reg == r2.Reg {
		t.Error("x1 and x2 should have different registers (they interfere)")
	}
}

func TestAllocateFunctionWithMove(t *testing.T) {
	// Function with move (should be coalesced):
	// 1: x1 = int 42
	// 2: x2 = move(x1)
	// 3: return x2
	fn := &rtl.Function{
		Name: "move",
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Ointconst{Value: 42}, Args: nil, Dest: 1, Succ: 2},
			2: rtl.Iop{Op: rtl.Omove{}, Args: []rtl.Reg{1}, Dest: 2, Succ: 3},
			3: rtl.Ireturn{Arg: ptr(rtl.Reg(2))},
		},
		Entrypoint: 1,
	}

	result := AllocateFunction(fn)

	// No spills
	if len(result.SpilledRegs) != 0 {
		t.Errorf("expected no spills, got %d", len(result.SpilledRegs))
	}

	// x1 and x2 should be coalesced to the same register
	loc1 := result.RegToLoc[1]
	loc2 := result.RegToLoc[2]

	r1, ok1 := loc1.(ltl.R)
	r2, ok2 := loc2.(ltl.R)

	if ok1 && ok2 && r1.Reg != r2.Reg {
		t.Errorf("x1 and x2 should be coalesced to same register, got %s and %s", r1.Reg, r2.Reg)
	}
}

func TestAllocateFunctionManyRegisters(t *testing.T) {
	// Function that uses many registers (but still < K)
	// Create a chain of additions
	code := make(map[rtl.Node]rtl.Instruction)
	numRegs := 10 // Use 10 registers

	for i := 1; i <= numRegs; i++ {
		code[rtl.Node(i)] = rtl.Iop{
			Op:   rtl.Ointconst{Value: int32(i)},
			Args: nil,
			Dest: rtl.Reg(i),
			Succ: rtl.Node(i + 1),
		}
	}
	code[rtl.Node(numRegs+1)] = rtl.Ireturn{Arg: ptr(rtl.Reg(numRegs))}

	fn := &rtl.Function{
		Name:       "many",
		Code:       code,
		Entrypoint: 1,
	}

	result := AllocateFunction(fn)

	// Should fit without spilling
	if len(result.SpilledRegs) != 0 {
		t.Errorf("expected no spills, got %d", len(result.SpilledRegs))
	}

	// All registers should have locations
	for i := 1; i <= numRegs; i++ {
		if _, ok := result.RegToLoc[rtl.Reg(i)]; !ok {
			t.Errorf("register %d should have a location", i)
		}
	}
}

func TestAllocateWithConditional(t *testing.T) {
	// Function with conditional:
	// 1: x1 = int 1
	// 2: if x1 == 0 goto 3 else goto 4
	// 3: x2 = int 10; goto 5
	// 4: x2 = int 20; goto 5
	// 5: return x2
	fn := &rtl.Function{
		Name: "cond",
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Ointconst{Value: 1}, Args: nil, Dest: 1, Succ: 2},
			2: rtl.Icond{Cond: rtl.Ccompimm{Cond: rtl.Ceq, N: 0}, Args: []rtl.Reg{1}, IfSo: 3, IfNot: 4},
			3: rtl.Iop{Op: rtl.Ointconst{Value: 10}, Args: nil, Dest: 2, Succ: 5},
			4: rtl.Iop{Op: rtl.Ointconst{Value: 20}, Args: nil, Dest: 2, Succ: 5},
			5: rtl.Ireturn{Arg: ptr(rtl.Reg(2))},
		},
		Entrypoint: 1,
	}

	result := AllocateFunction(fn)

	// Should allocate without spills
	if len(result.SpilledRegs) != 0 {
		t.Errorf("expected no spills, got %d", len(result.SpilledRegs))
	}

	// Both registers should have locations
	if _, ok := result.RegToLoc[1]; !ok {
		t.Error("x1 should have a location")
	}
	if _, ok := result.RegToLoc[2]; !ok {
		t.Error("x2 should have a location")
	}
}

func TestAllocateWithLoop(t *testing.T) {
	// Function with loop:
	// 1: x1 = int 10
	// 2: x2 = int 0
	// 3: if x1 == 0 goto 5 else goto 4
	// 4: x1 = sub(x1, 1); goto 3
	// 5: return x2
	fn := &rtl.Function{
		Name: "loop",
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Ointconst{Value: 10}, Args: nil, Dest: 1, Succ: 2},
			2: rtl.Iop{Op: rtl.Ointconst{Value: 0}, Args: nil, Dest: 2, Succ: 3},
			3: rtl.Icond{Cond: rtl.Ccompimm{Cond: rtl.Ceq, N: 0}, Args: []rtl.Reg{1}, IfSo: 5, IfNot: 4},
			4: rtl.Iop{Op: rtl.Oaddimm{N: -1}, Args: []rtl.Reg{1}, Dest: 1, Succ: 3},
			5: rtl.Ireturn{Arg: ptr(rtl.Reg(2))},
		},
		Entrypoint: 1,
	}

	result := AllocateFunction(fn)

	// Should allocate without spills
	if len(result.SpilledRegs) != 0 {
		t.Errorf("expected no spills, got %d", len(result.SpilledRegs))
	}

	// Registers should have different physical registers (they interfere)
	loc1 := result.RegToLoc[1]
	loc2 := result.RegToLoc[2]

	r1, ok1 := loc1.(ltl.R)
	r2, ok2 := loc2.(ltl.R)

	if ok1 && ok2 && r1.Reg == r2.Reg {
		t.Error("x1 and x2 should have different registers (both live in loop)")
	}
}

func TestGetAllRegisters(t *testing.T) {
	fn := &rtl.Function{
		Name:   "test",
		Params: []rtl.Reg{1},
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Oadd{}, Args: []rtl.Reg{1, 2}, Dest: 3, Succ: 2},
			2: rtl.Iload{Chunk: rtl.Mint64, Args: []rtl.Reg{3}, Dest: 4, Succ: 3},
			3: rtl.Istore{Chunk: rtl.Mint64, Args: []rtl.Reg{3}, Src: 4, Succ: 4},
			4: rtl.Ireturn{Arg: ptr(rtl.Reg(4))},
		},
		Entrypoint: 1,
	}

	regs := GetAllRegisters(fn)

	for _, r := range []rtl.Reg{1, 2, 3, 4} {
		if !regs.Contains(r) {
			t.Errorf("should contain register %d", r)
		}
	}
}

func TestSortedRegSlice(t *testing.T) {
	s := NewRegSet()
	s.Add(5)
	s.Add(1)
	s.Add(3)

	sorted := SortedRegSlice(s)

	if len(sorted) != 3 {
		t.Errorf("sorted slice has %d elements, want 3", len(sorted))
	}
	if sorted[0] != 1 || sorted[1] != 3 || sorted[2] != 5 {
		t.Errorf("sorted = %v, want [1, 3, 5]", sorted)
	}
}

func TestLocationIsPhysicalRegister(t *testing.T) {
	fn := &rtl.Function{
		Name: "test",
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Ointconst{Value: 1}, Args: nil, Dest: 1, Succ: 2},
			2: rtl.Ireturn{Arg: ptr(rtl.Reg(1))},
		},
		Entrypoint: 1,
	}

	result := AllocateFunction(fn)

	loc := result.RegToLoc[1]
	r, ok := loc.(ltl.R)
	if !ok {
		t.Fatal("location should be a register")
	}

	// Should be a valid ARM64 register
	if !r.Reg.IsInteger() {
		t.Error("should be an integer register")
	}
}

func TestRegisterLiveAcrossCallUsesCalleeSaved(t *testing.T) {
	// factorial(n):
	// 1: if n <= 1 goto 5 else goto 2
	// 2: x2 = sub(n, 1)
	// 3: x3 = call factorial(x2)
	// 4: x4 = mul(n, x3) <- n is used here, after the call
	// 5: return 1
	// 6: return x4
	//
	// n (register 1) is live across the call at node 3.
	// The allocator must assign it to a callee-saved register.
	n := rtl.Reg(1) // param
	fn := &rtl.Function{
		Name:   "factorial",
		Params: []rtl.Reg{n},
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Icond{Cond: rtl.Ccompimm{Cond: rtl.Cle, N: 1}, Args: []rtl.Reg{n}, IfSo: 5, IfNot: 2},
			2: rtl.Iop{Op: rtl.Oaddimm{N: -1}, Args: []rtl.Reg{n}, Dest: 2, Succ: 3},
			3: rtl.Icall{Fn: rtl.FunSymbol{Name: "factorial"}, Args: []rtl.Reg{2}, Dest: 3, Succ: 4},
			4: rtl.Iop{Op: rtl.Omul{}, Args: []rtl.Reg{n, 3}, Dest: 4, Succ: 6},
			5: rtl.Ireturn{Arg: ptr(rtl.Reg(1))},
			6: rtl.Ireturn{Arg: ptr(rtl.Reg(4))},
		},
		Entrypoint: 1,
	}

	result := AllocateFunction(fn)

	// n (reg 1) should be assigned to a callee-saved register
	loc := result.RegToLoc[n]
	r, ok := loc.(ltl.R)
	if !ok {
		// Also OK if spilled to stack, but prefer callee-saved
		_, isStack := loc.(ltl.S)
		if isStack {
			t.Log("n was spilled to stack (acceptable)")
			return
		}
		t.Fatalf("n should be in a register or stack, got %T", loc)
	}

	if !IsCalleeSaved(r.Reg) {
		t.Errorf("n is live across call and should be in callee-saved register, got %s (caller-saved)", r.Reg)
	}
}
