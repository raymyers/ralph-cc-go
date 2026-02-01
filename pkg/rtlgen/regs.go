// Register allocation for RTLgen.
// Manages the assignment of pseudo-registers to CminorSel variables,
// temporaries, function parameters, and expression results.

package rtlgen

import "github.com/raymyers/ralph-cc/pkg/rtl"

// RegAllocator manages pseudo-register allocation for RTL generation.
// It tracks variable-to-register mappings and generates fresh registers
// for expression temporaries.
type RegAllocator struct {
	nextReg    rtl.Reg           // next available register ID
	varToReg   map[string]rtl.Reg // CminorSel variable -> register mapping
	paramRegs  []rtl.Reg         // registers for function parameters (in order)
	resultReg  rtl.Reg           // register for function return value (0 = none)
}

// NewRegAllocator creates a new register allocator.
func NewRegAllocator() *RegAllocator {
	return &RegAllocator{
		nextReg:  1, // Register IDs start at 1
		varToReg: make(map[string]rtl.Reg),
	}
}

// Fresh allocates a fresh pseudo-register.
func (a *RegAllocator) Fresh() rtl.Reg {
	r := a.nextReg
	a.nextReg++
	return r
}

// FreshN allocates n fresh pseudo-registers.
func (a *RegAllocator) FreshN(n int) []rtl.Reg {
	regs := make([]rtl.Reg, n)
	for i := 0; i < n; i++ {
		regs[i] = a.Fresh()
	}
	return regs
}

// MapVar maps a variable name to a register.
// If already mapped, returns the existing register.
// Otherwise, allocates a fresh register and maps it.
func (a *RegAllocator) MapVar(name string) rtl.Reg {
	if r, ok := a.varToReg[name]; ok {
		return r
	}
	r := a.Fresh()
	a.varToReg[name] = r
	return r
}

// LookupVar returns the register for a variable, or 0 if not mapped.
func (a *RegAllocator) LookupVar(name string) (rtl.Reg, bool) {
	r, ok := a.varToReg[name]
	return r, ok
}

// SetVar explicitly sets the register for a variable.
// Used when parameter registers are pre-determined.
func (a *RegAllocator) SetVar(name string, r rtl.Reg) {
	a.varToReg[name] = r
}

// MapParams maps function parameters to registers.
// Returns the list of parameter registers (in order).
func (a *RegAllocator) MapParams(params []string) []rtl.Reg {
	a.paramRegs = make([]rtl.Reg, len(params))
	for i, name := range params {
		r := a.Fresh()
		a.paramRegs[i] = r
		a.varToReg[name] = r
	}
	return a.paramRegs
}

// GetParamRegs returns the parameter registers.
func (a *RegAllocator) GetParamRegs() []rtl.Reg {
	return a.paramRegs
}

// MapVars maps local variables to registers.
// Returns the list of variable registers (in order).
func (a *RegAllocator) MapVars(vars []string) []rtl.Reg {
	regs := make([]rtl.Reg, len(vars))
	for i, name := range vars {
		regs[i] = a.MapVar(name)
	}
	return regs
}

// SetResultReg sets the register for the function return value.
func (a *RegAllocator) SetResultReg(r rtl.Reg) {
	a.resultReg = r
}

// GetResultReg returns the register for the function return value.
// Returns 0 if no result register has been set.
func (a *RegAllocator) GetResultReg() rtl.Reg {
	return a.resultReg
}

// AllocResultReg allocates a fresh register for the function return value.
func (a *RegAllocator) AllocResultReg() rtl.Reg {
	a.resultReg = a.Fresh()
	return a.resultReg
}

// NextRegID returns the next register ID that will be allocated.
// Useful for determining register counts.
func (a *RegAllocator) NextRegID() rtl.Reg {
	return a.nextReg
}

// Clone creates a copy of the allocator state.
// Useful for speculative allocation in expression translation.
func (a *RegAllocator) Clone() *RegAllocator {
	varToReg := make(map[string]rtl.Reg, len(a.varToReg))
	for k, v := range a.varToReg {
		varToReg[k] = v
	}
	paramRegs := make([]rtl.Reg, len(a.paramRegs))
	copy(paramRegs, a.paramRegs)
	
	return &RegAllocator{
		nextReg:   a.nextReg,
		varToReg:  varToReg,
		paramRegs: paramRegs,
		resultReg: a.resultReg,
	}
}
