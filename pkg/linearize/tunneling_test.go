package linearize

import (
	"testing"

	"github.com/raymyers/ralph-cc/pkg/linear"
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

func TestTunnelEmpty(t *testing.T) {
	fn := linear.NewFunction("empty", linear.Sig{})
	Tunnel(fn)
	// Should not panic
	if len(fn.Code) != 0 {
		t.Errorf("Expected empty code")
	}
}

func TestTunnelSimpleChain(t *testing.T) {
	// L1: goto L2
	// L2: goto L3
	// L3: return
	// => goto L1 should become goto L3
	fn := linear.NewFunction("chain", linear.Sig{})

	fn.Append(linear.Lgoto{Target: 1}) // Entry jumps to L1
	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lgoto{Target: 2})
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lgoto{Target: 3})
	fn.Append(linear.Llabel{Lbl: 3})
	fn.Append(linear.Lreturn{})

	Tunnel(fn)

	// The first goto should now point to L3
	if gt, ok := fn.Code[0].(linear.Lgoto); ok {
		if gt.Target != 3 {
			t.Errorf("Expected goto target = 3, got %d", gt.Target)
		}
	} else {
		t.Errorf("First instruction should be Lgoto")
	}
}

func TestTunnelConditional(t *testing.T) {
	// if cond then L1 else (fall-through)
	// L1: goto L2
	// L2: return
	// => "if then L1" should become "if then L2"
	fn := linear.NewFunction("cond", linear.Sig{})
	cond := rtl.Ccompimm{Cond: rtl.Ceq, N: 0}

	fn.Append(linear.Lcond{Cond: cond, Args: []linear.Loc{linear.R{Reg: ltl.X0}}, IfSo: 1})
	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lgoto{Target: 2})
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lreturn{})

	Tunnel(fn)

	// The conditional should now branch to L2
	if c, ok := fn.Code[0].(linear.Lcond); ok {
		if c.IfSo != 2 {
			t.Errorf("Expected conditional IfSo = 2, got %d", c.IfSo)
		}
	} else {
		t.Errorf("First instruction should be Lcond")
	}
}

func TestTunnelJumptable(t *testing.T) {
	// jumptable [L1, L2, L3]
	// L1: goto L4
	// L2: return
	// L3: goto L4
	// L4: return
	// => jumptable should become [L4, L2, L4]
	fn := linear.NewFunction("jt", linear.Sig{})

	fn.Append(linear.Ljumptable{Arg: linear.R{Reg: ltl.X0}, Targets: []linear.Label{1, 2, 3}})
	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lgoto{Target: 4})
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lreturn{})
	fn.Append(linear.Llabel{Lbl: 3})
	fn.Append(linear.Lgoto{Target: 4})
	fn.Append(linear.Llabel{Lbl: 4})
	fn.Append(linear.Lreturn{})

	Tunnel(fn)

	// The jumptable should have targets [L4, L2, L4]
	if jt, ok := fn.Code[0].(linear.Ljumptable); ok {
		expected := []linear.Label{4, 2, 4}
		for i, target := range jt.Targets {
			if target != expected[i] {
				t.Errorf("Jumptable target[%d] = %d, want %d", i, target, expected[i])
			}
		}
	} else {
		t.Errorf("First instruction should be Ljumptable")
	}
}

func TestTunnelCycle(t *testing.T) {
	// L1: goto L2
	// L2: goto L1  (cycle!)
	// Should not infinite loop
	fn := linear.NewFunction("cycle", linear.Sig{})

	fn.Append(linear.Lgoto{Target: 1})
	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lgoto{Target: 2})
	fn.Append(linear.Llabel{Lbl: 2})
	fn.Append(linear.Lgoto{Target: 1})

	Tunnel(fn) // Should not hang

	// The cycle should be detected and handled
	// (exact behavior is implementation-defined, just verify no hang)
}

func TestTunnelNoChange(t *testing.T) {
	// L1: op; return (not a goto)
	// => goto L1 should stay goto L1
	fn := linear.NewFunction("nochange", linear.Sig{})

	fn.Append(linear.Lgoto{Target: 1})
	fn.Append(linear.Llabel{Lbl: 1})
	fn.Append(linear.Lop{Op: rtl.Ointconst{Value: 42}, Dest: linear.R{Reg: ltl.X0}})
	fn.Append(linear.Lreturn{})

	Tunnel(fn)

	// First goto should still point to L1
	if gt, ok := fn.Code[0].(linear.Lgoto); ok {
		if gt.Target != 1 {
			t.Errorf("Expected goto target = 1, got %d", gt.Target)
		}
	}
}

func TestTunnelLongChain(t *testing.T) {
	// L1 -> L2 -> L3 -> L4 -> L5
	fn := linear.NewFunction("longchain", linear.Sig{})

	fn.Append(linear.Lgoto{Target: 1})
	for i := linear.Label(1); i <= 4; i++ {
		fn.Append(linear.Llabel{Lbl: i})
		fn.Append(linear.Lgoto{Target: i + 1})
	}
	fn.Append(linear.Llabel{Lbl: 5})
	fn.Append(linear.Lreturn{})

	Tunnel(fn)

	// The first goto should now point to L5
	if gt, ok := fn.Code[0].(linear.Lgoto); ok {
		if gt.Target != 5 {
			t.Errorf("Expected goto target = 5, got %d", gt.Target)
		}
	}
}
