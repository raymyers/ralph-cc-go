package linearize

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestLinearizeEmpty(t *testing.T) {
	fn := ltl.NewFunction("empty", ltl.Sig{})
	result := Linearize(fn)

	if result.Name != "empty" {
		t.Errorf("Name = %s, want empty", result.Name)
	}
	if len(result.Code) != 0 {
		t.Errorf("Code length = %d, want 0", len(result.Code))
	}
}

func TestLinearizeSingleBlock(t *testing.T) {
	fn := ltl.NewFunction("single", ltl.Sig{Return: "int"})
	fn.Entrypoint = 1
	fn.Code[1] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lop{Op: rtl.Ointconst{Value: 42}, Dest: ltl.R{Reg: ltl.X0}},
			ltl.Lreturn{},
		},
	}

	result := Linearize(fn)

	// Should have: label, op, return
	if len(result.Code) != 3 {
		t.Errorf("Code length = %d, want 3", len(result.Code))
	}

	// First instruction should be a label
	if _, ok := result.Code[0].(linear.Llabel); !ok {
		t.Errorf("First instruction should be Llabel, got %T", result.Code[0])
	}

	// Second should be the op
	if op, ok := result.Code[1].(linear.Lop); !ok {
		t.Errorf("Second instruction should be Lop, got %T", result.Code[1])
	} else {
		if c, ok := op.Op.(rtl.Ointconst); !ok || c.Value != 42 {
			t.Errorf("Expected Ointconst{42}, got %v", op.Op)
		}
	}

	// Third should be return
	if _, ok := result.Code[2].(linear.Lreturn); !ok {
		t.Errorf("Third instruction should be Lreturn, got %T", result.Code[2])
	}
}

func TestLinearizeTwoBlocksFallThrough(t *testing.T) {
	fn := ltl.NewFunction("fallthrough", ltl.Sig{Return: "int"})
	fn.Entrypoint = 1

	// Block 1: branch to block 2
	fn.Code[1] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lop{Op: rtl.Ointconst{Value: 1}, Dest: ltl.R{Reg: ltl.X0}},
			ltl.Lbranch{Succ: 2},
		},
	}

	// Block 2: return
	fn.Code[2] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lop{Op: rtl.Ointconst{Value: 2}, Dest: ltl.R{Reg: ltl.X1}},
			ltl.Lreturn{},
		},
	}

	result := Linearize(fn)

	// With fall-through optimization, should NOT have a goto between blocks
	// Expected: label1, op1, label2, op2, return (5 instructions)
	// The goto should be omitted because block 2 follows block 1

	hasGoto := false
	for _, inst := range result.Code {
		if _, ok := inst.(linear.Lgoto); ok {
			hasGoto = true
			break
		}
	}

	if hasGoto {
		t.Errorf("Expected fall-through optimization to eliminate goto")
	}

	// Count labels - should be 2
	labelCount := 0
	for _, inst := range result.Code {
		if _, ok := inst.(linear.Llabel); ok {
			labelCount++
		}
	}
	if labelCount != 2 {
		t.Errorf("Label count = %d, want 2", labelCount)
	}
}

func TestLinearizeWithBranch(t *testing.T) {
	fn := ltl.NewFunction("branching", ltl.Sig{Return: "int"})
	fn.Entrypoint = 1

	// Block 1: branch to block 3 (skipping block 2)
	fn.Code[1] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lbranch{Succ: 3},
		},
	}

	// Block 2: unreachable
	fn.Code[2] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lreturn{},
		},
	}

	// Block 3: return
	fn.Code[3] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lreturn{},
		},
	}

	result := Linearize(fn)

	// Block 1 jumps to block 3, which may not be next in order
	// depending on traversal. We should have a goto if not fall-through.

	// Verify we have at least a label and return in the output
	hasReturn := false
	for _, inst := range result.Code {
		if _, ok := inst.(linear.Lreturn); ok {
			hasReturn = true
			break
		}
	}

	if !hasReturn {
		t.Error("Expected at least one return instruction")
	}
}

func TestLinearizeConditional(t *testing.T) {
	fn := ltl.NewFunction("conditional", ltl.Sig{Return: "int"})
	fn.Entrypoint = 1

	cond := rtl.Ccompimm{Cond: rtl.Ceq, N: 0}

	// Block 1: conditional branch
	fn.Code[1] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lcond{
				Cond:  cond,
				Args:  []ltl.Loc{ltl.R{Reg: ltl.X0}},
				IfSo:  2,
				IfNot: 3,
			},
		},
	}

	// Block 2: then branch
	fn.Code[2] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lop{Op: rtl.Ointconst{Value: 1}, Dest: ltl.R{Reg: ltl.X0}},
			ltl.Lreturn{},
		},
	}

	// Block 3: else branch
	fn.Code[3] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lop{Op: rtl.Ointconst{Value: 0}, Dest: ltl.R{Reg: ltl.X0}},
			ltl.Lreturn{},
		},
	}

	result := Linearize(fn)

	// Should have a conditional instruction
	hasCond := false
	for _, inst := range result.Code {
		if _, ok := inst.(linear.Lcond); ok {
			hasCond = true
			break
		}
	}

	if !hasCond {
		t.Error("Expected a conditional instruction")
	}

	// Should have multiple labels
	labelCount := 0
	for _, inst := range result.Code {
		if _, ok := inst.(linear.Llabel); ok {
			labelCount++
		}
	}
	if labelCount < 2 {
		t.Errorf("Label count = %d, want >= 2", labelCount)
	}
}

func TestLinearizeJumptable(t *testing.T) {
	fn := ltl.NewFunction("jumptable", ltl.Sig{Return: "int"})
	fn.Entrypoint = 1

	// Block 1: jumptable
	fn.Code[1] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Ljumptable{
				Arg:     ltl.R{Reg: ltl.X0},
				Targets: []ltl.Node{2, 3, 4},
			},
		},
	}

	// Blocks 2, 3, 4: return
	for n := ltl.Node(2); n <= 4; n++ {
		fn.Code[n] = &ltl.BBlock{
			Body: []ltl.Instruction{
				ltl.Lreturn{},
			},
		}
	}

	result := Linearize(fn)

	// Should have a jumptable instruction
	hasJumptable := false
	for _, inst := range result.Code {
		if jt, ok := inst.(linear.Ljumptable); ok {
			hasJumptable = true
			if len(jt.Targets) != 3 {
				t.Errorf("Jumptable targets = %d, want 3", len(jt.Targets))
			}
			break
		}
	}

	if !hasJumptable {
		t.Error("Expected a jumptable instruction")
	}
}

func TestLinearizeFunctionCall(t *testing.T) {
	fn := ltl.NewFunction("caller", ltl.Sig{Return: "int"})
	fn.Entrypoint = 1

	// Block 1: call then return
	fn.Code[1] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lcall{Sig: ltl.Sig{Return: "int"}, Fn: ltl.FunSymbol{Name: "callee"}},
			ltl.Lreturn{},
		},
	}

	result := Linearize(fn)

	// Should have a call instruction
	hasCall := false
	for _, inst := range result.Code {
		if call, ok := inst.(linear.Lcall); ok {
			hasCall = true
			if sym, ok := call.Fn.(linear.FunSymbol); !ok || sym.Name != "callee" {
				t.Errorf("Expected FunSymbol{callee}, got %v", call.Fn)
			}
			break
		}
	}

	if !hasCall {
		t.Error("Expected a call instruction")
	}
}

func TestLinearizeTailcall(t *testing.T) {
	fn := ltl.NewFunction("tailcaller", ltl.Sig{Return: "int"})
	fn.Entrypoint = 1

	// Block 1: tail call
	fn.Code[1] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Ltailcall{Sig: ltl.Sig{Return: "int"}, Fn: ltl.FunSymbol{Name: "target"}},
		},
	}

	result := Linearize(fn)

	// Should have a tailcall instruction
	hasTailcall := false
	for _, inst := range result.Code {
		if tc, ok := inst.(linear.Ltailcall); ok {
			hasTailcall = true
			if sym, ok := tc.Fn.(linear.FunSymbol); !ok || sym.Name != "target" {
				t.Errorf("Expected FunSymbol{target}, got %v", tc.Fn)
			}
			break
		}
	}

	if !hasTailcall {
		t.Error("Expected a tailcall instruction")
	}
}

func TestLinearizeLoadStore(t *testing.T) {
	fn := ltl.NewFunction("loadstore", ltl.Sig{})
	fn.Entrypoint = 1

	// Block 1: load, store, return
	fn.Code[1] = &ltl.BBlock{
		Body: []ltl.Instruction{
			ltl.Lload{
				Chunk: ltl.Mint64,
				Addr:  rtl.Aindexed{Offset: 0},
				Args:  []ltl.Loc{ltl.R{Reg: ltl.X0}},
				Dest:  ltl.R{Reg: ltl.X1},
			},
			ltl.Lstore{
				Chunk: ltl.Mint64,
				Addr:  rtl.Aindexed{Offset: 8},
				Args:  []ltl.Loc{ltl.R{Reg: ltl.X0}},
				Src:   ltl.R{Reg: ltl.X1},
			},
			ltl.Lreturn{},
		},
	}

	result := Linearize(fn)

	// Should have load and store instructions
	hasLoad := false
	hasStore := false
	for _, inst := range result.Code {
		if _, ok := inst.(linear.Lload); ok {
			hasLoad = true
		}
		if _, ok := inst.(linear.Lstore); ok {
			hasStore = true
		}
	}

	if !hasLoad {
		t.Error("Expected a load instruction")
	}
	if !hasStore {
		t.Error("Expected a store instruction")
	}
}
