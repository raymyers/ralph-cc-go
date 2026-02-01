package rtl

import (
	"testing"
)

func TestRegisterTypes(t *testing.T) {
	// Test register creation and comparison
	r1 := Reg(1)
	r2 := Reg(2)
	r1dup := Reg(1)

	if r1 == r2 {
		t.Error("Different registers should not be equal")
	}
	if r1 != r1dup {
		t.Error("Same register value should be equal")
	}
}

func TestNodeTypes(t *testing.T) {
	n1 := Node(1)
	n2 := Node(2)
	n1dup := Node(1)

	if n1 == n2 {
		t.Error("Different nodes should not be equal")
	}
	if n1 != n1dup {
		t.Error("Same node value should be equal")
	}
}

func TestConditionNegate(t *testing.T) {
	tests := []struct {
		cond     Condition
		expected Condition
	}{
		{Ceq, Cne},
		{Cne, Ceq},
		{Clt, Cge},
		{Cle, Cgt},
		{Cgt, Cle},
		{Cge, Clt},
	}

	for _, tc := range tests {
		got := tc.cond.Negate()
		if got != tc.expected {
			t.Errorf("Negate(%v) = %v, want %v", tc.cond, got, tc.expected)
		}
	}
}

func TestConditionString(t *testing.T) {
	tests := []struct {
		cond     Condition
		expected string
	}{
		{Ceq, "=="},
		{Cne, "!="},
		{Clt, "<"},
		{Cle, "<="},
		{Cgt, ">"},
		{Cge, ">="},
	}

	for _, tc := range tests {
		got := tc.cond.String()
		if got != tc.expected {
			t.Errorf("String(%v) = %v, want %v", tc.cond, got, tc.expected)
		}
	}
}

func TestInstructionSuccessors(t *testing.T) {
	tests := []struct {
		name     string
		instr    Instruction
		expected []Node
	}{
		{
			name:     "Inop",
			instr:    Inop{Succ: Node(5)},
			expected: []Node{5},
		},
		{
			name:     "Iop",
			instr:    Iop{Op: Oadd{}, Args: []Reg{1, 2}, Dest: 3, Succ: Node(10)},
			expected: []Node{10},
		},
		{
			name:     "Iload",
			instr:    Iload{Chunk: Mint32, Args: []Reg{1}, Dest: 2, Succ: Node(7)},
			expected: []Node{7},
		},
		{
			name:     "Istore",
			instr:    Istore{Chunk: Mint32, Args: []Reg{1}, Src: 2, Succ: Node(8)},
			expected: []Node{8},
		},
		{
			name:     "Icall",
			instr:    Icall{Fn: FunSymbol{Name: "foo"}, Dest: 1, Succ: Node(9)},
			expected: []Node{9},
		},
		{
			name:     "Itailcall",
			instr:    Itailcall{Fn: FunSymbol{Name: "bar"}},
			expected: nil,
		},
		{
			name:     "Ibuiltin",
			instr:    Ibuiltin{Builtin: "memcpy", Succ: Node(11)},
			expected: []Node{11},
		},
		{
			name:     "Icond",
			instr:    Icond{Cond: Ccomp{Cond: Ceq}, Args: []Reg{1, 2}, IfSo: Node(20), IfNot: Node(30)},
			expected: []Node{20, 30},
		},
		{
			name:     "Ijumptable",
			instr:    Ijumptable{Arg: 1, Targets: []Node{10, 20, 30}},
			expected: []Node{10, 20, 30},
		},
		{
			name:     "Ireturn",
			instr:    Ireturn{Arg: nil},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.instr.Successors()
			if len(got) != len(tc.expected) {
				t.Errorf("Successors() length = %d, want %d", len(got), len(tc.expected))
				return
			}
			for i, n := range got {
				if n != tc.expected[i] {
					t.Errorf("Successors()[%d] = %d, want %d", i, n, tc.expected[i])
				}
			}
		})
	}
}

func TestFunctionConstruction(t *testing.T) {
	sig := Sig{Args: []string{"int"}, Return: "int"}
	fn := NewFunction("test", sig)

	if fn.Name != "test" {
		t.Errorf("Name = %s, want test", fn.Name)
	}
	if fn.Code == nil {
		t.Error("Code map should be initialized")
	}

	// Add some instructions
	fn.Entrypoint = Node(1)
	fn.Params = []Reg{1}
	fn.Code[Node(1)] = Iop{
		Op:   Oaddimm{N: 1},
		Args: []Reg{1},
		Dest: 2,
		Succ: Node(2),
	}
	fn.Code[Node(2)] = Ireturn{Arg: regPtr(2)}

	if len(fn.Code) != 2 {
		t.Errorf("Code should have 2 instructions, got %d", len(fn.Code))
	}
}

func TestOperationTypes(t *testing.T) {
	// Test that all operations implement the Operation interface
	ops := []Operation{
		Omove{},
		Ointconst{Value: 42},
		Olongconst{Value: 100},
		Ofloatconst{Value: 3.14},
		Osingleconst{Value: 2.5},
		Oaddrsymbol{Symbol: "x", Offset: 0},
		Oaddrstack{Offset: 8},
		Oadd{},
		Oaddimm{N: 1},
		Oneg{},
		Osub{},
		Omul{},
		Omulimm{N: 2},
		Odiv{},
		Odivu{},
		Omod{},
		Omodu{},
		Oand{},
		Oandimm{N: 0xff},
		Oor{},
		Oorimm{N: 0x10},
		Oxor{},
		Oxorimm{N: 0x1},
		Onot{},
		Oshl{},
		Oshlimm{N: 2},
		Oshr{},
		Oshrimm{N: 1},
		Oshru{},
		Oshruimm{N: 3},
		Oaddl{},
		Onegl{},
		Osubl{},
		Omull{},
		Odivl{},
		Odivlu{},
		Omodl{},
		Omodlu{},
		Oandl{},
		Oorl{},
		Oxorl{},
		Onotl{},
		Oshll{},
		Oshrl{},
		Oshrlu{},
		Ocast8signed{},
		Ocast8unsigned{},
		Ocast16signed{},
		Ocast16unsigned{},
		Olongofint{},
		Olongofintu{},
		Ointoflong{},
		Onegf{},
		Oabsf{},
		Oaddf{},
		Osubf{},
		Omulf{},
		Odivf{},
		Onegs{},
		Oabss{},
		Oadds{},
		Osubs{},
		Omuls{},
		Odivs{},
		Osingleoffloat{},
		Ofloatofsingle{},
		Ointoffloat{},
		Ointuoffloat{},
		Ofloatofint{},
		Ofloatofintu{},
		Olongoffloat{},
		Olonguoffloat{},
		Ofloatoflong{},
		Ofloatoflongu{},
		Ocmp{Cond: Ceq},
		Ocmpu{Cond: Cne},
		Ocmpf{Cond: Clt},
		Ocmps{Cond: Cle},
		Ocmpl{Cond: Cgt},
		Ocmplu{Cond: Cge},
	}

	for _, op := range ops {
		// Just verify they implement the interface (compile-time check)
		_ = op
	}
}

func TestFunRefTypes(t *testing.T) {
	// Test function reference types
	var fr FunRef

	fr = FunReg{Reg: 5}
	if reg, ok := fr.(FunReg); !ok {
		t.Error("FunReg should implement FunRef")
	} else if reg.Reg != 5 {
		t.Errorf("FunReg.Reg = %d, want 5", reg.Reg)
	}

	fr = FunSymbol{Name: "printf"}
	if sym, ok := fr.(FunSymbol); !ok {
		t.Error("FunSymbol should implement FunRef")
	} else if sym.Name != "printf" {
		t.Errorf("FunSymbol.Name = %s, want printf", sym.Name)
	}
}

func TestConditionCodeTypes(t *testing.T) {
	// Test all condition code types
	conds := []ConditionCode{
		Ccomp{Cond: Ceq},
		Ccompu{Cond: Cne},
		Ccompimm{Cond: Clt, N: 0},
		Ccompuimm{Cond: Cle, N: 10},
		Ccompl{Cond: Cgt},
		Ccomplu{Cond: Cge},
		Ccomplimm{Cond: Ceq, N: 100},
		Ccompluimm{Cond: Cne, N: 200},
		Ccompf{Cond: Clt},
		Cnotcompf{Cond: Ceq},
		Ccomps{Cond: Cle},
		Cnotcomps{Cond: Cne},
	}

	for _, cc := range conds {
		// Just verify they implement the interface
		_ = cc
	}
}

func TestProgramConstruction(t *testing.T) {
	prog := Program{
		Globals: []GlobVar{
			{Name: "x", Size: 4},
		},
		Functions: []Function{
			*NewFunction("main", Sig{Return: "int"}),
		},
	}

	if len(prog.Globals) != 1 {
		t.Errorf("Globals length = %d, want 1", len(prog.Globals))
	}
	if len(prog.Functions) != 1 {
		t.Errorf("Functions length = %d, want 1", len(prog.Functions))
	}
}

// Helper function
func regPtr(r Reg) *Reg {
	return &r
}
