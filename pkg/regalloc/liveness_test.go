package regalloc

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestRegSetOperations(t *testing.T) {
	t.Run("Add and Contains", func(t *testing.T) {
		s := NewRegSet()
		s.Add(1)
		s.Add(2)

		if !s.Contains(1) {
			t.Error("set should contain 1")
		}
		if !s.Contains(2) {
			t.Error("set should contain 2")
		}
		if s.Contains(3) {
			t.Error("set should not contain 3")
		}
	})

	t.Run("Union", func(t *testing.T) {
		s1 := NewRegSet()
		s1.Add(1)
		s1.Add(2)

		s2 := NewRegSet()
		s2.Add(2)
		s2.Add(3)

		u := s1.Union(s2)
		if !u.Contains(1) || !u.Contains(2) || !u.Contains(3) {
			t.Error("union should contain 1, 2, and 3")
		}
	})

	t.Run("Minus", func(t *testing.T) {
		s1 := NewRegSet()
		s1.Add(1)
		s1.Add(2)
		s1.Add(3)

		s2 := NewRegSet()
		s2.Add(2)

		diff := s1.Minus(s2)
		if !diff.Contains(1) || !diff.Contains(3) {
			t.Error("difference should contain 1 and 3")
		}
		if diff.Contains(2) {
			t.Error("difference should not contain 2")
		}
	})

	t.Run("Equal", func(t *testing.T) {
		s1 := NewRegSet()
		s1.Add(1)
		s1.Add(2)

		s2 := NewRegSet()
		s2.Add(1)
		s2.Add(2)

		s3 := NewRegSet()
		s3.Add(1)

		if !s1.Equal(s2) {
			t.Error("s1 and s2 should be equal")
		}
		if s1.Equal(s3) {
			t.Error("s1 and s3 should not be equal")
		}
	})

	t.Run("Copy", func(t *testing.T) {
		s := NewRegSet()
		s.Add(1)
		s.Add(2)

		c := s.Copy()
		s.Add(3)

		if c.Contains(3) {
			t.Error("copy should not be affected by modifications to original")
		}
	})
}

func TestComputeDefUse(t *testing.T) {
	// Create a simple function:
	// 1: x1 = int 42       ; def: x1
	// 2: x2 = add(x1, x1)  ; use: x1; def: x2
	// 3: return x2         ; use: x2
	fn := &rtl.Function{
		Name: "test",
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Ointconst{Value: 42}, Args: nil, Dest: 1, Succ: 2},
			2: rtl.Iop{Op: rtl.Oadd{}, Args: []rtl.Reg{1, 1}, Dest: 2, Succ: 3},
			3: rtl.Ireturn{Arg: ptr(rtl.Reg(2))},
		},
		Entrypoint: 1,
	}

	def, use := ComputeDefUse(fn)

	// Node 1: def={1}, use={}
	if !def[1].Contains(1) || len(def[1]) != 1 {
		t.Errorf("node 1 def = %v, want {1}", def[1].Slice())
	}
	if len(use[1]) != 0 {
		t.Errorf("node 1 use = %v, want {}", use[1].Slice())
	}

	// Node 2: def={2}, use={1}
	if !def[2].Contains(2) || len(def[2]) != 1 {
		t.Errorf("node 2 def = %v, want {2}", def[2].Slice())
	}
	if !use[2].Contains(1) || len(use[2]) != 1 {
		t.Errorf("node 2 use = %v, want {1}", use[2].Slice())
	}

	// Node 3: def={}, use={2}
	if len(def[3]) != 0 {
		t.Errorf("node 3 def = %v, want {}", def[3].Slice())
	}
	if !use[3].Contains(2) || len(use[3]) != 1 {
		t.Errorf("node 3 use = %v, want {2}", use[3].Slice())
	}
}

func TestComputeDefUseInstructions(t *testing.T) {
	tests := []struct {
		name    string
		instr   rtl.Instruction
		wantDef []rtl.Reg
		wantUse []rtl.Reg
	}{
		{
			name:    "Inop",
			instr:   rtl.Inop{Succ: 1},
			wantDef: nil,
			wantUse: nil,
		},
		{
			name:    "Iop",
			instr:   rtl.Iop{Op: rtl.Oadd{}, Args: []rtl.Reg{1, 2}, Dest: 3, Succ: 1},
			wantDef: []rtl.Reg{3},
			wantUse: []rtl.Reg{1, 2},
		},
		{
			name:    "Iload",
			instr:   rtl.Iload{Chunk: rtl.Mint64, Args: []rtl.Reg{1}, Dest: 2, Succ: 1},
			wantDef: []rtl.Reg{2},
			wantUse: []rtl.Reg{1},
		},
		{
			name:    "Istore",
			instr:   rtl.Istore{Chunk: rtl.Mint64, Args: []rtl.Reg{1}, Src: 2, Succ: 1},
			wantDef: nil,
			wantUse: []rtl.Reg{1, 2},
		},
		{
			name:    "Icall with symbol",
			instr:   rtl.Icall{Fn: rtl.FunSymbol{Name: "foo"}, Args: []rtl.Reg{1, 2}, Dest: 3, Succ: 1},
			wantDef: []rtl.Reg{3},
			wantUse: []rtl.Reg{1, 2},
		},
		{
			name:    "Icall with reg",
			instr:   rtl.Icall{Fn: rtl.FunReg{Reg: 4}, Args: []rtl.Reg{1, 2}, Dest: 3, Succ: 1},
			wantDef: []rtl.Reg{3},
			wantUse: []rtl.Reg{1, 2, 4},
		},
		{
			name:    "Icond",
			instr:   rtl.Icond{Args: []rtl.Reg{1, 2}, IfSo: 1, IfNot: 2},
			wantDef: nil,
			wantUse: []rtl.Reg{1, 2},
		},
		{
			name:    "Ireturn",
			instr:   rtl.Ireturn{Arg: ptr(rtl.Reg(1))},
			wantDef: nil,
			wantUse: []rtl.Reg{1},
		},
		{
			name:    "Ireturn void",
			instr:   rtl.Ireturn{Arg: nil},
			wantDef: nil,
			wantUse: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := &rtl.Function{
				Code: map[rtl.Node]rtl.Instruction{1: tt.instr},
			}
			def, use := ComputeDefUse(fn)

			// Check def
			for _, r := range tt.wantDef {
				if !def[1].Contains(r) {
					t.Errorf("def should contain %d", r)
				}
			}
			if len(def[1]) != len(tt.wantDef) {
				t.Errorf("def has %d regs, want %d", len(def[1]), len(tt.wantDef))
			}

			// Check use
			for _, r := range tt.wantUse {
				if !use[1].Contains(r) {
					t.Errorf("use should contain %d", r)
				}
			}
			if len(use[1]) != len(tt.wantUse) {
				t.Errorf("use has %d regs, want %d", len(use[1]), len(tt.wantUse))
			}
		})
	}
}

func TestAnalyzeLivenessSimple(t *testing.T) {
	// Simple linear function:
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

	info := AnalyzeLiveness(fn)

	// At node 4 (return x3): live_in = {3}, live_out = {}
	if !info.LiveIn[4].Contains(3) {
		t.Error("x3 should be live at entry to node 4")
	}
	if len(info.LiveOut[4]) != 0 {
		t.Error("nothing should be live at exit of node 4")
	}

	// At node 3 (add): live_in = {1, 2}, live_out = {3}
	if !info.LiveIn[3].Contains(1) || !info.LiveIn[3].Contains(2) {
		t.Error("x1 and x2 should be live at entry to node 3")
	}
	if !info.LiveOut[3].Contains(3) {
		t.Error("x3 should be live at exit of node 3")
	}

	// At node 2: live_in = {1}, live_out = {1, 2}
	if !info.LiveIn[2].Contains(1) {
		t.Error("x1 should be live at entry to node 2")
	}
	if !info.LiveOut[2].Contains(1) || !info.LiveOut[2].Contains(2) {
		t.Error("x1 and x2 should be live at exit of node 2")
	}

	// At node 1: live_out = {1}
	if !info.LiveOut[1].Contains(1) {
		t.Error("x1 should be live at exit of node 1")
	}
}

func TestAnalyzeLivenessWithBranch(t *testing.T) {
	// Function with conditional branch:
	// 1: x1 = int 1
	// 2: if x1 == 0 goto 3 else goto 4
	// 3: x2 = int 10
	// 4: x2 = int 20
	// 5: return x2
	fn := &rtl.Function{
		Name: "branch",
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Ointconst{Value: 1}, Args: nil, Dest: 1, Succ: 2},
			2: rtl.Icond{Cond: rtl.Ccompimm{Cond: rtl.Ceq, N: 0}, Args: []rtl.Reg{1}, IfSo: 3, IfNot: 4},
			3: rtl.Iop{Op: rtl.Ointconst{Value: 10}, Args: nil, Dest: 2, Succ: 5},
			4: rtl.Iop{Op: rtl.Ointconst{Value: 20}, Args: nil, Dest: 2, Succ: 5},
			5: rtl.Ireturn{Arg: ptr(rtl.Reg(2))},
		},
		Entrypoint: 1,
	}

	info := AnalyzeLiveness(fn)

	// At node 2 (branch): x1 should be live (used in condition)
	if !info.LiveIn[2].Contains(1) {
		t.Error("x1 should be live at entry to node 2")
	}

	// x2 should NOT be live at node 2 (defined after, on both paths)
	if info.LiveIn[2].Contains(2) || info.LiveOut[2].Contains(2) {
		t.Error("x2 should not be live at node 2")
	}
}

func TestAnalyzeLivenessWithLoop(t *testing.T) {
	// Simple loop:
	// 1: x1 = int 10
	// 2: x2 = int 0
	// 3: if x1 == 0 goto 5 else goto 4
	// 4: x1 = sub(x1, 1); x2 = add(x2, 1); goto 3
	// 5: return x2
	fn := &rtl.Function{
		Name: "loop",
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Iop{Op: rtl.Ointconst{Value: 10}, Args: nil, Dest: 1, Succ: 2},
			2: rtl.Iop{Op: rtl.Ointconst{Value: 0}, Args: nil, Dest: 2, Succ: 3},
			3: rtl.Icond{Cond: rtl.Ccompimm{Cond: rtl.Ceq, N: 0}, Args: []rtl.Reg{1}, IfSo: 5, IfNot: 4},
			// Simplified: just decrement x1
			4: rtl.Iop{Op: rtl.Oaddimm{N: -1}, Args: []rtl.Reg{1}, Dest: 1, Succ: 3},
			5: rtl.Ireturn{Arg: ptr(rtl.Reg(2))},
		},
		Entrypoint: 1,
	}

	info := AnalyzeLiveness(fn)

	// At loop header (node 3): x1 and x2 should be live
	if !info.LiveIn[3].Contains(1) {
		t.Error("x1 should be live at loop header")
	}
	if !info.LiveIn[3].Contains(2) {
		t.Error("x2 should be live at loop header")
	}

	// At node 4 (loop body): x1 should be used
	if !info.Use[4].Contains(1) {
		t.Error("x1 should be used at node 4")
	}
}

func TestAnalyzeLivenessAcrossCall(t *testing.T) {
	// Function where a value is live across a call:
	// factorial(n):
	// 1: if n <= 1 goto 5 else goto 2
	// 2: x2 = sub(n, 1)
	// 3: x3 = call factorial(x2)
	// 4: x4 = mul(n, x3) <- n is used here, after the call
	// 5: return 1
	// 6: return x4
	n := rtl.Reg(1) // param
	fn := &rtl.Function{
		Name:   "factorial",
		Params: []rtl.Reg{n},
		Code: map[rtl.Node]rtl.Instruction{
			1: rtl.Icond{Cond: rtl.Ccompimm{Cond: rtl.Cle, N: 1}, Args: []rtl.Reg{n}, IfSo: 5, IfNot: 2},
			2: rtl.Iop{Op: rtl.Oaddimm{N: -1}, Args: []rtl.Reg{n}, Dest: 2, Succ: 3},
			3: rtl.Icall{Fn: rtl.FunSymbol{Name: "factorial"}, Args: []rtl.Reg{2}, Dest: 3, Succ: 4},
			4: rtl.Iop{Op: rtl.Omul{}, Args: []rtl.Reg{n, 3}, Dest: 4, Succ: 6},
			5: rtl.Ireturn{Arg: ptr(rtl.Reg(1))}, // Actually would return 1, but use n for simplicity
			6: rtl.Ireturn{Arg: ptr(rtl.Reg(4))},
		},
		Entrypoint: 1,
	}

	info := AnalyzeLiveness(fn)

	// Critical check: at node 3 (the call), n (register 1) should be live out
	// because it's used at node 4
	if !info.LiveOut[3].Contains(n) {
		t.Error("n should be live out at call node 3 (it's used at node 4)")
	}

	// n should also be live-in at node 4
	if !info.LiveIn[4].Contains(n) {
		t.Error("n should be live in at node 4")
	}
}

func ptr[T any](v T) *T {
	return &v
}
