// Package mach defines the Mach intermediate representation.
// Mach is a near-assembly representation with concrete activation record layout.
// Stack slots are replaced with concrete offsets from the frame pointer.
// This is the last IR before assembly generation.
// This mirrors CompCert's backend/Mach.v
package mach

import (
	"github.com/raymyers/ralph-cc/pkg/ltl"
	"github.com/raymyers/ralph-cc/pkg/rtl"
)

// Re-export types from ltl that are used in Mach
type (
	Chunk          = ltl.Chunk
	AddressingMode = ltl.AddressingMode
	Sig            = ltl.Sig
	Operation      = ltl.Operation
	Condition      = ltl.Condition
	ConditionCode  = ltl.ConditionCode
	MReg           = ltl.MReg
	Typ            = ltl.Typ
)

// Re-export commonly used operations from rtl for testing
type (
	Oadd  = rtl.Oadd
	Omove = rtl.Omove
)

// Re-export condition codes from rtl for testing
type (
	Ccomp = rtl.Ccomp
)

// Re-export condition constants
const (
	Ceq = rtl.Ceq
	Cne = rtl.Cne
	Clt = rtl.Clt
	Cle = rtl.Cle
	Cgt = rtl.Cgt
	Cge = rtl.Cge
)

// Re-export chunk constants
const (
	Mint8signed    = ltl.Mint8signed
	Mint8unsigned  = ltl.Mint8unsigned
	Mint16signed   = ltl.Mint16signed
	Mint16unsigned = ltl.Mint16unsigned
	Mint32         = ltl.Mint32
	Mint64         = ltl.Mint64
	Mfloat32       = ltl.Mfloat32
	Mfloat64       = ltl.Mfloat64
)

// Re-export register constants
const (
	X0  = ltl.X0
	X1  = ltl.X1
	X2  = ltl.X2
	X3  = ltl.X3
	X4  = ltl.X4
	X5  = ltl.X5
	X6  = ltl.X6
	X7  = ltl.X7
	X8  = ltl.X8
	X29 = ltl.X29 // FP
	X30 = ltl.X30 // LR
	D0  = ltl.D0
)

// Re-export typ constants
const (
	Tint    = ltl.Tint
	Tfloat  = ltl.Tfloat
	Tlong   = ltl.Tlong
	Tsingle = ltl.Tsingle
	Tany32  = ltl.Tany32
	Tany64  = ltl.Tany64
)

// Label represents a branch target in Mach code.
// Labels are positive integers, with 0 indicating no label.
type Label int

// Valid returns true if this is a valid label (positive)
func (l Label) Valid() bool {
	return l > 0
}

// --- Mach Instructions ---
// Mach instructions are similar to Linear but with concrete stack offsets.
// Stack operations reference concrete offsets from the frame pointer.

// Instruction is the interface for Mach instructions
type Instruction interface {
	implMachInstruction()
}

// Mgetstack loads from a stack slot at a concrete frame offset
type Mgetstack struct {
	Ofs  int64 // offset from frame pointer (FP)
	Ty   Typ   // type of value
	Dest MReg  // destination register
}

// Msetstack stores to a stack slot at a concrete frame offset
type Msetstack struct {
	Src MReg  // source register
	Ofs int64 // offset from frame pointer (FP)
	Ty  Typ   // type of value
}

// Mgetparam loads a parameter from the caller's frame
type Mgetparam struct {
	Ofs  int64 // offset from caller's frame pointer
	Ty   Typ   // type of value
	Dest MReg  // destination register
}

// Mop performs an operation: dest = op(args...)
type Mop struct {
	Op   Operation // the operation
	Args []MReg    // source registers
	Dest MReg      // destination register
}

// Mload loads from memory: dest = Mem[addr(args...)]
type Mload struct {
	Chunk Chunk          // memory access size/type
	Addr  AddressingMode // addressing mode
	Args  []MReg         // registers for addressing
	Dest  MReg           // destination register
}

// Mstore stores to memory: Mem[addr(args...)] = src
type Mstore struct {
	Chunk Chunk          // memory access size/type
	Addr  AddressingMode // addressing mode
	Args  []MReg         // registers for addressing
	Src   MReg           // source register (value to store)
}

// Mcall performs a function call
type Mcall struct {
	Sig Sig    // function signature
	Fn  FunRef // function to call (reg or symbol)
}

// Mtailcall performs a tail call (no return to caller)
type Mtailcall struct {
	Sig Sig    // function signature
	Fn  FunRef // function to call
}

// Mbuiltin calls a builtin function
type Mbuiltin struct {
	Builtin string // builtin function name
	Args    []MReg // argument registers
	Dest    *MReg  // destination register (nil if no result)
}

// Mlabel marks a branch target
type Mlabel struct {
	Lbl Label // the label
}

// Mgoto is an unconditional jump
type Mgoto struct {
	Target Label // jump target
}

// Mcond is a conditional branch
type Mcond struct {
	Cond ConditionCode // condition to evaluate
	Args []MReg        // argument registers
	IfSo Label         // branch target if condition is true
}

// Mjumptable is an indexed jump (switch)
type Mjumptable struct {
	Arg     MReg    // register containing index
	Targets []Label // jump targets
}

// Mreturn returns from the function
type Mreturn struct{}

// Marker methods for Instruction interface
func (Mgetstack) implMachInstruction()  {}
func (Msetstack) implMachInstruction()  {}
func (Mgetparam) implMachInstruction()  {}
func (Mop) implMachInstruction()        {}
func (Mload) implMachInstruction()      {}
func (Mstore) implMachInstruction()     {}
func (Mcall) implMachInstruction()      {}
func (Mtailcall) implMachInstruction()  {}
func (Mbuiltin) implMachInstruction()   {}
func (Mlabel) implMachInstruction()     {}
func (Mgoto) implMachInstruction()      {}
func (Mcond) implMachInstruction()      {}
func (Mjumptable) implMachInstruction() {}
func (Mreturn) implMachInstruction()    {}

// --- Function Reference ---

// FunRef represents a function reference (either register or symbol)
type FunRef interface {
	implFunRef()
}

// FunReg is a function pointer in a register
type FunReg struct {
	Reg MReg
}

// FunSymbol is a named function symbol
type FunSymbol struct {
	Name string
}

func (FunReg) implFunRef()    {}
func (FunSymbol) implFunRef() {}

// --- Function and Program ---

// Function represents a Mach function with concrete stack layout
type Function struct {
	Name            string        // function name
	Sig             Sig           // function signature
	Code            []Instruction // mach instruction sequence
	Stacksize       int64         // total stack frame size
	CalleeSaveRegs  []MReg        // callee-saved registers used
	UsesFramePtr    bool          // whether function uses frame pointer
}

// GlobVar represents a global variable
type GlobVar struct {
	Name string
	Size int64
	Init []byte
}

// Program represents a complete Mach program
type Program struct {
	Globals   []GlobVar
	Functions []Function
}

// NewFunction creates a new Mach function
func NewFunction(name string, sig Sig) *Function {
	return &Function{
		Name:           name,
		Sig:            sig,
		Code:           make([]Instruction, 0),
		CalleeSaveRegs: make([]MReg, 0),
		UsesFramePtr:   true, // ARM64 typically uses frame pointer
	}
}

// Append adds an instruction to the function's code
func (f *Function) Append(inst Instruction) {
	f.Code = append(f.Code, inst)
}

// Labels returns all labels defined in the code
func (f *Function) Labels() []Label {
	seen := make(map[Label]bool)
	var labels []Label
	for _, inst := range f.Code {
		if lbl, ok := inst.(Mlabel); ok {
			if !seen[lbl.Lbl] {
				seen[lbl.Lbl] = true
				labels = append(labels, lbl.Lbl)
			}
		}
	}
	return labels
}

// ReferencedLabels returns all labels that are targets of jumps
func (f *Function) ReferencedLabels() []Label {
	seen := make(map[Label]bool)
	var labels []Label
	for _, inst := range f.Code {
		switch i := inst.(type) {
		case Mgoto:
			if !seen[i.Target] {
				seen[i.Target] = true
				labels = append(labels, i.Target)
			}
		case Mcond:
			if !seen[i.IfSo] {
				seen[i.IfSo] = true
				labels = append(labels, i.IfSo)
			}
		case Mjumptable:
			for _, lbl := range i.Targets {
				if !seen[lbl] {
					seen[lbl] = true
					labels = append(labels, lbl)
				}
			}
		}
	}
	return labels
}
