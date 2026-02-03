// Package linear defines the Linear intermediate representation.
// Linear is linearized LTL with explicit labels and branches (no CFG).
// Instructions are arranged sequentially with labels marking branch targets.
// This mirrors CompCert's backend/Linear.v
package linear

import "github.com/raymyers/ralph-cc/pkg/ltl"

// Re-export types from ltl that are used in Linear
type (
	Chunk          = ltl.Chunk
	AddressingMode = ltl.AddressingMode
	Sig            = ltl.Sig
	Operation      = ltl.Operation
	Condition      = ltl.Condition
	ConditionCode  = ltl.ConditionCode
	Loc            = ltl.Loc
	MReg           = ltl.MReg
	SlotKind       = ltl.SlotKind
	Typ            = ltl.Typ
	R              = ltl.R
	S              = ltl.S
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

// Re-export slot kinds
const (
	SlotLocal    = ltl.SlotLocal
	SlotIncoming = ltl.SlotIncoming
	SlotOutgoing = ltl.SlotOutgoing
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

// Label represents a branch target in linearized code.
// Labels are positive integers, with 0 indicating no label.
type Label int

// Valid returns true if this is a valid label (positive)
func (l Label) Valid() bool {
	return l > 0
}

// --- Linear Instructions ---
// Linear instructions are a sequence with explicit labels and branches.
// Unlike LTL, there is no CFG structure - just a flat list of instructions.

// Instruction is the interface for Linear instructions
type Instruction interface {
	implLinearInstruction()
}

// Lgetstack loads from a stack slot to a register
type Lgetstack struct {
	Slot SlotKind // kind of stack slot
	Ofs  int64    // offset within slot kind
	Ty   Typ      // type of value
	Dest MReg     // destination register
}

// Lsetstack stores from a register to a stack slot
type Lsetstack struct {
	Src  MReg     // source register
	Slot SlotKind // kind of stack slot
	Ofs  int64    // offset within slot kind
	Ty   Typ      // type of value
}

// Lop performs an operation: dest = op(args...)
type Lop struct {
	Op   Operation // the operation
	Args []Loc     // source locations
	Dest Loc       // destination location
}

// Lload loads from memory: dest = Mem[addr(args...)]
type Lload struct {
	Chunk Chunk          // memory access size/type
	Addr  AddressingMode // addressing mode
	Args  []Loc          // locations for addressing
	Dest  Loc            // destination location
}

// Lstore stores to memory: Mem[addr(args...)] = src
type Lstore struct {
	Chunk Chunk          // memory access size/type
	Addr  AddressingMode // addressing mode
	Args  []Loc          // locations for addressing
	Src   Loc            // source location (value to store)
}

// Lcall performs a function call
type Lcall struct {
	Sig Sig    // function signature
	Fn  FunRef // function to call (reg or symbol)
}

// Ltailcall performs a tail call (no return to caller)
type Ltailcall struct {
	Sig Sig    // function signature
	Fn  FunRef // function to call
}

// Lbuiltin calls a builtin function
type Lbuiltin struct {
	Builtin string // builtin function name
	Args    []Loc  // argument locations
	Dest    *Loc   // destination location (nil if no result)
}

// Llabel marks a branch target
type Llabel struct {
	Lbl Label // the label
}

// Lgoto is an unconditional jump
type Lgoto struct {
	Target Label // jump target
}

// Lcond is a conditional branch
type Lcond struct {
	Cond  ConditionCode // condition to evaluate
	Args  []Loc         // argument locations
	IfSo  Label         // branch target if condition is true
}

// Ljumptable is an indexed jump (switch)
type Ljumptable struct {
	Arg     Loc     // location containing index
	Targets []Label // jump targets
}

// Lreturn returns from the function
type Lreturn struct{}

// Marker methods for Instruction interface
func (Lgetstack) implLinearInstruction()  {}
func (Lsetstack) implLinearInstruction()  {}
func (Lop) implLinearInstruction()        {}
func (Lload) implLinearInstruction()      {}
func (Lstore) implLinearInstruction()     {}
func (Lcall) implLinearInstruction()      {}
func (Ltailcall) implLinearInstruction()  {}
func (Lbuiltin) implLinearInstruction()   {}
func (Llabel) implLinearInstruction()     {}
func (Lgoto) implLinearInstruction()      {}
func (Lcond) implLinearInstruction()      {}
func (Ljumptable) implLinearInstruction() {}
func (Lreturn) implLinearInstruction()    {}

// --- Function Reference ---
// Re-use from LTL

// FunRef represents a function reference (either register or symbol)
type FunRef interface {
	implFunRef()
}

// FunReg is a function pointer in a register
type FunReg struct {
	Loc Loc
}

// FunSymbol is a named function symbol
type FunSymbol struct {
	Name string
}

func (FunReg) implFunRef()    {}
func (FunSymbol) implFunRef() {}

// --- Function and Program ---

// Function represents a Linear function
type Function struct {
	Name      string        // function name
	Sig       Sig           // function signature
	Params    []Loc         // parameter locations (after register allocation)
	Stacksize int64         // stack frame size
	Code      []Instruction // linear instruction sequence
}

// GlobVar represents a global variable
type GlobVar struct {
	Name     string
	Size     int64
	Init     []byte
	ReadOnly bool // true for .rodata section (e.g., string literals)
}

// Program represents a complete Linear program
type Program struct {
	Globals   []GlobVar
	Functions []Function
}

// NewFunction creates a new Linear function
func NewFunction(name string, sig Sig) *Function {
	return &Function{
		Name: name,
		Sig:  sig,
		Code: make([]Instruction, 0),
	}
}

// Append adds an instruction to the function's code
func (f *Function) Append(inst Instruction) {
	f.Code = append(f.Code, inst)
}

// Labels returns all labels used in the code
func (f *Function) Labels() []Label {
	seen := make(map[Label]bool)
	var labels []Label
	for _, inst := range f.Code {
		switch i := inst.(type) {
		case Llabel:
			if !seen[i.Lbl] {
				seen[i.Lbl] = true
				labels = append(labels, i.Lbl)
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
		case Lgoto:
			if !seen[i.Target] {
				seen[i.Target] = true
				labels = append(labels, i.Target)
			}
		case Lcond:
			if !seen[i.IfSo] {
				seen[i.IfSo] = true
				labels = append(labels, i.IfSo)
			}
		case Ljumptable:
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
