package rtl

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintFunction_Simple(t *testing.T) {
	// Function that returns its argument + 1
	fn := Function{
		Name:       "inc",
		Sig:        Sig{Args: []string{"int"}, Return: "int"},
		Params:     []Reg{1},
		Stacksize:  0,
		Entrypoint: 3,
		Code: map[Node]Instruction{
			3: Iop{
				Op:   Ointconst{Value: 1},
				Args: nil,
				Dest: 2,
				Succ: 2,
			},
			2: Iop{
				Op:   Oadd{},
				Args: []Reg{1, 2},
				Dest: 3,
				Succ: 1,
			},
			1: Ireturn{Arg: ptrReg(3)},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(&fn)

	output := buf.String()

	// Check function header
	if !strings.Contains(output, "inc(x1)") {
		t.Errorf("expected function header, got:\n%s", output)
	}

	// Check instructions
	if !strings.Contains(output, "int 1") {
		t.Errorf("expected int const, got:\n%s", output)
	}
	if !strings.Contains(output, "add") {
		t.Errorf("expected add op, got:\n%s", output)
	}
	if !strings.Contains(output, "return x3") {
		t.Errorf("expected return, got:\n%s", output)
	}

	// Check entry point
	if !strings.Contains(output, "entry: 3") {
		t.Errorf("expected entry point, got:\n%s", output)
	}
}

func TestPrintFunction_Load(t *testing.T) {
	fn := Function{
		Name:       "load_test",
		Params:     []Reg{1},
		Entrypoint: 2,
		Code: map[Node]Instruction{
			2: Iload{
				Chunk: Mint32,
				Addr:  Aindexed{Offset: 8},
				Args:  []Reg{1},
				Dest:  2,
				Succ:  1,
			},
			1: Ireturn{Arg: ptrReg(2)},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(&fn)

	output := buf.String()

	if !strings.Contains(output, "int32[x1 + 8]") {
		t.Errorf("expected load instruction, got:\n%s", output)
	}
}

func TestPrintFunction_Store(t *testing.T) {
	fn := Function{
		Name:       "store_test",
		Params:     []Reg{1, 2},
		Entrypoint: 2,
		Code: map[Node]Instruction{
			2: Istore{
				Chunk: Mint32,
				Addr:  Aindexed{Offset: 0},
				Args:  []Reg{1},
				Src:   2,
				Succ:  1,
			},
			1: Ireturn{Arg: nil},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(&fn)

	output := buf.String()

	if !strings.Contains(output, "int32[x1 + 0] = x2") {
		t.Errorf("expected store instruction, got:\n%s", output)
	}
}

func TestPrintFunction_Call(t *testing.T) {
	fn := Function{
		Name:       "caller",
		Params:     []Reg{1},
		Entrypoint: 2,
		Code: map[Node]Instruction{
			2: Icall{
				Sig:  Sig{Args: []string{"int"}, Return: "int"},
				Fn:   FunSymbol{Name: "callee"},
				Args: []Reg{1},
				Dest: 2,
				Succ: 1,
			},
			1: Ireturn{Arg: ptrReg(2)},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(&fn)

	output := buf.String()

	if !strings.Contains(output, `call "callee"(x1)`) {
		t.Errorf("expected call instruction, got:\n%s", output)
	}
}

func TestPrintFunction_Cond(t *testing.T) {
	fn := Function{
		Name:       "cond_test",
		Params:     []Reg{1, 2},
		Entrypoint: 3,
		Code: map[Node]Instruction{
			3: Icond{
				Cond:  Ccomp{Cond: Clt},
				Args:  []Reg{1, 2},
				IfSo:  2,
				IfNot: 1,
			},
			2: Ireturn{Arg: ptrReg(1)},
			1: Ireturn{Arg: ptrReg(2)},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(&fn)

	output := buf.String()

	if !strings.Contains(output, "if x1 < x2") {
		t.Errorf("expected condition, got:\n%s", output)
	}
	if !strings.Contains(output, "goto 2 else goto 1") {
		t.Errorf("expected branch targets, got:\n%s", output)
	}
}

func TestPrintFunction_Jumptable(t *testing.T) {
	fn := Function{
		Name:       "switch_test",
		Params:     []Reg{1},
		Entrypoint: 4,
		Code: map[Node]Instruction{
			4: Ijumptable{
				Arg:     1,
				Targets: []Node{1, 2, 3},
			},
			3: Ireturn{Arg: nil},
			2: Ireturn{Arg: nil},
			1: Ireturn{Arg: nil},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintFunction(&fn)

	output := buf.String()

	if !strings.Contains(output, "jumptable x1 [1, 2, 3]") {
		t.Errorf("expected jumptable, got:\n%s", output)
	}
}

func TestPrintProgram(t *testing.T) {
	prog := Program{
		Globals: []GlobVar{
			{Name: "counter", Size: 4},
		},
		Functions: []Function{
			{
				Name:       "main",
				Entrypoint: 1,
				Code: map[Node]Instruction{
					1: Ireturn{Arg: nil},
				},
			},
		},
	}

	var buf bytes.Buffer
	p := NewPrinter(&buf)
	p.PrintProgram(&prog)

	output := buf.String()

	if !strings.Contains(output, `var "counter"[4]`) {
		t.Errorf("expected global var, got:\n%s", output)
	}
	if !strings.Contains(output, "main()") {
		t.Errorf("expected function, got:\n%s", output)
	}
}

func TestPrintOperations(t *testing.T) {
	tests := []struct {
		op     Operation
		expect string
	}{
		{Omove{}, "move"},
		{Ointconst{Value: 42}, "int 42"},
		{Olongconst{Value: 123}, "long 123L"},
		{Oadd{}, "add"},
		{Osub{}, "sub"},
		{Omul{}, "mul"},
		{Odiv{}, "divs"},
		{Oand{}, "and"},
		{Oor{}, "or"},
		{Oxor{}, "xor"},
		{Oshl{}, "shl"},
		{Oshlimm{N: 2}, "shlimm 2"},
		{Oaddl{}, "addl"},
		{Onegl{}, "negl"},
		{Ocast8signed{}, "cast8signed"},
		{Olongofint{}, "longofint"},
		{Oaddf{}, "addf"},
		{Ocmp{Cond: Ceq}, "cmp =="},
	}

	for _, tt := range tests {
		fn := Function{
			Name:       "test",
			Entrypoint: 1,
			Code: map[Node]Instruction{
				1: Iop{Op: tt.op, Dest: 1, Succ: 1},
			},
		}

		var buf bytes.Buffer
		p := NewPrinter(&buf)
		p.PrintFunction(&fn)

		if !strings.Contains(buf.String(), tt.expect) {
			t.Errorf("op %T: expected %q in output, got:\n%s", tt.op, tt.expect, buf.String())
		}
	}
}

func TestPrintAddressingModes(t *testing.T) {
	tests := []struct {
		name   string
		addr   AddressingMode
		args   []Reg
		expect string
	}{
		{"indexed", Aindexed{Offset: 8}, []Reg{1}, "x1 + 8"},
		{"indexed2", Aindexed2{}, []Reg{1, 2}, "x1 + x2"},
		{"global", Aglobal{Symbol: "foo", Offset: 4}, nil, `"foo" + 4`},
		{"stack", Ainstack{Offset: 16}, nil, "stack(16)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := Function{
				Name:       "test",
				Entrypoint: 1,
				Code: map[Node]Instruction{
					1: Iload{
						Chunk: Mint32,
						Addr:  tt.addr,
						Args:  tt.args,
						Dest:  1,
						Succ:  1,
					},
				},
			}

			var buf bytes.Buffer
			p := NewPrinter(&buf)
			p.PrintFunction(&fn)

			if !strings.Contains(buf.String(), tt.expect) {
				t.Errorf("expected %q in output, got:\n%s", tt.expect, buf.String())
			}
		})
	}
}

// Helper to create pointer to register
func ptrReg(r Reg) *Reg {
	return &r
}
