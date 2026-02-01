package rtlgen

import (
	"testing"
)

func TestRegAllocatorFresh(t *testing.T) {
	a := NewRegAllocator()

	r1 := a.Fresh()
	r2 := a.Fresh()
	r3 := a.Fresh()

	if r1 != 1 {
		t.Errorf("first reg = %d, want 1", r1)
	}
	if r2 != 2 {
		t.Errorf("second reg = %d, want 2", r2)
	}
	if r3 != 3 {
		t.Errorf("third reg = %d, want 3", r3)
	}
}

func TestRegAllocatorFreshN(t *testing.T) {
	a := NewRegAllocator()

	regs := a.FreshN(3)

	if len(regs) != 3 {
		t.Fatalf("got %d regs, want 3", len(regs))
	}
	if regs[0] != 1 || regs[1] != 2 || regs[2] != 3 {
		t.Errorf("regs = %v, want [1 2 3]", regs)
	}

	// Next alloc should continue from 4
	next := a.Fresh()
	if next != 4 {
		t.Errorf("next reg = %d, want 4", next)
	}
}

func TestRegAllocatorMapVar(t *testing.T) {
	a := NewRegAllocator()

	r1 := a.MapVar("x")
	r2 := a.MapVar("y")
	r3 := a.MapVar("x") // should return same as r1

	if r1 != r3 {
		t.Errorf("x mapped to %d and %d, should be same", r1, r3)
	}
	if r1 == r2 {
		t.Errorf("x and y mapped to same register %d", r1)
	}
}

func TestRegAllocatorLookupVar(t *testing.T) {
	a := NewRegAllocator()

	_, ok := a.LookupVar("x")
	if ok {
		t.Error("LookupVar should return false for unmapped var")
	}

	expected := a.MapVar("x")
	got, ok := a.LookupVar("x")
	if !ok {
		t.Error("LookupVar should return true for mapped var")
	}
	if got != expected {
		t.Errorf("LookupVar = %d, want %d", got, expected)
	}
}

func TestRegAllocatorSetVar(t *testing.T) {
	a := NewRegAllocator()

	a.SetVar("x", 100)

	r, ok := a.LookupVar("x")
	if !ok {
		t.Error("LookupVar should return true after SetVar")
	}
	if r != 100 {
		t.Errorf("x = %d, want 100", r)
	}

	// SetVar should override
	a.SetVar("x", 200)
	r, _ = a.LookupVar("x")
	if r != 200 {
		t.Errorf("x = %d, want 200", r)
	}
}

func TestRegAllocatorMapParams(t *testing.T) {
	a := NewRegAllocator()

	params := []string{"a", "b", "c"}
	regs := a.MapParams(params)

	if len(regs) != 3 {
		t.Fatalf("got %d param regs, want 3", len(regs))
	}

	// Parameters should get sequential registers starting at 1
	if regs[0] != 1 || regs[1] != 2 || regs[2] != 3 {
		t.Errorf("param regs = %v, want [1 2 3]", regs)
	}

	// Parameters should be mapped to their registers
	for i, name := range params {
		r, ok := a.LookupVar(name)
		if !ok {
			t.Errorf("param %s not mapped", name)
		}
		if r != regs[i] {
			t.Errorf("param %s mapped to %d, want %d", name, r, regs[i])
		}
	}

	// GetParamRegs should return same list
	gotRegs := a.GetParamRegs()
	if len(gotRegs) != len(regs) {
		t.Errorf("GetParamRegs returned %d regs, want %d", len(gotRegs), len(regs))
	}
	for i, r := range regs {
		if gotRegs[i] != r {
			t.Errorf("GetParamRegs[%d] = %d, want %d", i, gotRegs[i], r)
		}
	}
}

func TestRegAllocatorMapVars(t *testing.T) {
	a := NewRegAllocator()

	vars := []string{"x", "y", "z"}
	regs := a.MapVars(vars)

	if len(regs) != 3 {
		t.Fatalf("got %d var regs, want 3", len(regs))
	}

	// Variables should get sequential registers
	for i, name := range vars {
		r, ok := a.LookupVar(name)
		if !ok {
			t.Errorf("var %s not mapped", name)
		}
		if r != regs[i] {
			t.Errorf("var %s mapped to %d, want %d", name, r, regs[i])
		}
	}
}

func TestRegAllocatorParamsThenVars(t *testing.T) {
	a := NewRegAllocator()

	// First map params
	params := []string{"a", "b"}
	paramRegs := a.MapParams(params)

	// Then map vars
	vars := []string{"x", "y"}
	varRegs := a.MapVars(vars)

	// Params should be 1, 2
	if paramRegs[0] != 1 || paramRegs[1] != 2 {
		t.Errorf("param regs = %v, want [1 2]", paramRegs)
	}

	// Vars should continue: 3, 4
	if varRegs[0] != 3 || varRegs[1] != 4 {
		t.Errorf("var regs = %v, want [3 4]", varRegs)
	}
}

func TestRegAllocatorResultReg(t *testing.T) {
	a := NewRegAllocator()

	// Initially no result reg
	if a.GetResultReg() != 0 {
		t.Errorf("initial result reg = %d, want 0", a.GetResultReg())
	}

	// Set explicitly
	a.SetResultReg(10)
	if a.GetResultReg() != 10 {
		t.Errorf("result reg = %d, want 10", a.GetResultReg())
	}

	// Alloc fresh
	a2 := NewRegAllocator()
	a2.Fresh() // consume 1
	a2.Fresh() // consume 2
	r := a2.AllocResultReg()
	if r != 3 {
		t.Errorf("allocated result reg = %d, want 3", r)
	}
	if a2.GetResultReg() != 3 {
		t.Errorf("GetResultReg = %d, want 3", a2.GetResultReg())
	}
}

func TestRegAllocatorNextRegID(t *testing.T) {
	a := NewRegAllocator()

	if a.NextRegID() != 1 {
		t.Errorf("initial NextRegID = %d, want 1", a.NextRegID())
	}

	a.Fresh()
	a.Fresh()

	if a.NextRegID() != 3 {
		t.Errorf("NextRegID = %d, want 3", a.NextRegID())
	}
}

func TestRegAllocatorClone(t *testing.T) {
	a := NewRegAllocator()

	a.MapParams([]string{"a"})
	a.MapVar("x")
	a.SetResultReg(10)

	// Clone
	b := a.Clone()

	// Modify original
	a.Fresh()
	a.MapVar("y")
	a.SetResultReg(20)

	// Clone should be unaffected
	if b.NextRegID() != 3 {
		t.Errorf("clone NextRegID = %d, want 3", b.NextRegID())
	}
	if _, ok := b.LookupVar("y"); ok {
		t.Error("clone should not have 'y' mapping")
	}
	if b.GetResultReg() != 10 {
		t.Errorf("clone result reg = %d, want 10", b.GetResultReg())
	}

	// But original should have changes
	// After MapParams(1) + MapVar(1) + Fresh(1) + MapVar(1) = 4 allocated, next is 5
	if a.NextRegID() != 5 {
		t.Errorf("original NextRegID = %d, want 5", a.NextRegID())
	}
	if _, ok := a.LookupVar("y"); !ok {
		t.Error("original should have 'y' mapping")
	}
	if a.GetResultReg() != 20 {
		t.Errorf("original result reg = %d, want 20", a.GetResultReg())
	}
}

func TestRegAllocatorEmpty(t *testing.T) {
	a := NewRegAllocator()

	// Empty params and vars should work
	regs := a.MapParams(nil)
	if len(regs) != 0 {
		t.Errorf("MapParams(nil) = %v, want []", regs)
	}

	regs = a.MapVars(nil)
	if len(regs) != 0 {
		t.Errorf("MapVars(nil) = %v, want []", regs)
	}

	regs = a.MapParams([]string{})
	if len(regs) != 0 {
		t.Errorf("MapParams([]) = %v, want []", regs)
	}
}
