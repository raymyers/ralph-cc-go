// Package rtl defines the RTL (Register Transfer Language) intermediate representation.
// RTL is the primary backend IR - a CFG-based representation with infinite pseudo-registers
// and 3-address code. Instructions have explicit destinations and branch to successor nodes.
// This mirrors CompCert's backend/RTL.v
package rtl

import "github.com/raymyers/ralph-cc/pkg/cminorsel"

// Node represents a program point in the CFG (positive integer identifier)
type Node int

// Reg represents a pseudo-register (positive integer, infinite supply)
type Reg int

// Re-export types from cminorsel that are used in RTL
type (
	Chunk          = cminorsel.Chunk
	AddressingMode = cminorsel.AddressingMode
	Sig            = cminorsel.Sig
)

// Re-export addressing mode types
type (
	Aindexed      = cminorsel.Aindexed
	Aindexed2     = cminorsel.Aindexed2
	Aglobal       = cminorsel.Aglobal
	Ainstack      = cminorsel.Ainstack
	Aindexed2shift = cminorsel.Aindexed2shift
)

// Re-export chunk constants
const (
	Mint8signed    = cminorsel.Mint8signed
	Mint8unsigned  = cminorsel.Mint8unsigned
	Mint16signed   = cminorsel.Mint16signed
	Mint16unsigned = cminorsel.Mint16unsigned
	Mint32         = cminorsel.Mint32
	Mint64         = cminorsel.Mint64
	Mfloat32       = cminorsel.Mfloat32
	Mfloat64       = cminorsel.Mfloat64
)

// --- Operation Types ---
// RTL uses target-specific operations (same as CminorSel)

// Operation represents an RTL operation (arithmetic, load address, etc.)
// Operations are encoded with an opcode and optional immediate arguments.
type Operation interface {
	implOperation()
}

// Omove copies a register value (identity operation)
type Omove struct{}

// Ointconst loads an integer constant
type Ointconst struct {
	Value int32
}

// Olongconst loads a long (int64) constant
type Olongconst struct {
	Value int64
}

// Ofloatconst loads a float64 constant
type Ofloatconst struct {
	Value float64
}

// Osingleconst loads a float32 constant
type Osingleconst struct {
	Value float32
}

// Oaddrsymbol loads address of global symbol + offset
type Oaddrsymbol struct {
	Symbol string
	Offset int64
}

// Oaddrstack loads address of stack slot
type Oaddrstack struct {
	Offset int64
}

// Integer arithmetic operations
type Oadd struct{}      // rd = rs1 + rs2
type Oaddimm struct{ N int32 } // rd = rs + n
type Oneg struct{}      // rd = -rs
type Osub struct{}      // rd = rs1 - rs2
type Omul struct{}      // rd = rs1 * rs2
type Omulimm struct{ N int32 } // rd = rs * n
type Omulhs struct{}    // rd = high(rs1 * rs2) signed
type Omulhu struct{}    // rd = high(rs1 * rs2) unsigned
type Odiv struct{}      // rd = rs1 / rs2 (signed)
type Odivu struct{}     // rd = rs1 / rs2 (unsigned)
type Omod struct{}      // rd = rs1 % rs2 (signed)
type Omodu struct{}     // rd = rs1 % rs2 (unsigned)

// Bitwise operations
type Oand struct{}      // rd = rs1 & rs2
type Oandimm struct{ N int32 } // rd = rs & n
type Oor struct{}       // rd = rs1 | rs2
type Oorimm struct{ N int32 }  // rd = rs | n
type Oxor struct{}      // rd = rs1 ^ rs2
type Oxorimm struct{ N int32 } // rd = rs ^ n
type Onot struct{}      // rd = ~rs
type Oshl struct{}      // rd = rs1 << rs2
type Oshlimm struct{ N int32 } // rd = rs << n
type Oshr struct{}      // rd = rs1 >> rs2 (signed)
type Oshrimm struct{ N int32 } // rd = rs >> n (signed)
type Oshru struct{}     // rd = rs1 >> rs2 (unsigned)
type Oshruimm struct{ N int32 }// rd = rs >> n (unsigned)

// Long (int64) operations
type Oaddl struct{}     // rd = rs1 + rs2 (long)
type Oaddlimm struct{ N int64 } // rd = rs + n (long)
type Onegl struct{}     // rd = -rs (long)
type Osubl struct{}     // rd = rs1 - rs2 (long)
type Omull struct{}     // rd = rs1 * rs2 (long)
type Omullimm struct{ N int64 } // rd = rs * n (long)
type Omullhs struct{}   // high signed (long)
type Omullhu struct{}   // high unsigned (long)
type Odivl struct{}     // signed div (long)
type Odivlu struct{}    // unsigned div (long)
type Omodl struct{}     // signed mod (long)
type Omodlu struct{}    // unsigned mod (long)
type Oandl struct{}     // rd = rs1 & rs2 (long)
type Oandlimm struct{ N int64 }// rd = rs & n (long)
type Oorl struct{}      // rd = rs1 | rs2 (long)
type Oorlimm struct{ N int64 } // rd = rs | n (long)
type Oxorl struct{}     // rd = rs1 ^ rs2 (long)
type Oxorlimm struct{ N int64 }// rd = rs ^ n (long)
type Onotl struct{}     // rd = ~rs (long)
type Oshll struct{}     // rd = rs1 << rs2 (long)
type Oshllimm struct{ N int32 }// rd = rs << n (long)
type Oshrl struct{}     // rd = rs1 >> rs2 (long, signed)
type Oshrlimm struct{ N int32 }// rd = rs >> n (long, signed)
type Oshrlu struct{}    // rd = rs1 >> rs2 (long, unsigned)
type Oshrluimm struct{ N int32 }// rd = rs >> n (long, unsigned)

// Conversions
type Ocast8signed struct{}   // sign-extend 8->32
type Ocast8unsigned struct{} // zero-extend 8->32
type Ocast16signed struct{}  // sign-extend 16->32
type Ocast16unsigned struct{}// zero-extend 16->32
type Olongofint struct{}     // int -> long (signed)
type Olongofintu struct{}    // int -> long (unsigned)
type Ointoflong struct{}     // long -> int (truncate)

// Float64 operations
type Onegf struct{}     // rd = -rs (float64)
type Oabsf struct{}     // rd = |rs| (float64)
type Oaddf struct{}     // rd = rs1 + rs2 (float64)
type Osubf struct{}     // rd = rs1 - rs2 (float64)
type Omulf struct{}     // rd = rs1 * rs2 (float64)
type Odivf struct{}     // rd = rs1 / rs2 (float64)

// Float32 operations
type Onegs struct{}     // rd = -rs (float32)
type Oabss struct{}     // rd = |rs| (float32)
type Oadds struct{}     // rd = rs1 + rs2 (float32)
type Osubs struct{}     // rd = rs1 - rs2 (float32)
type Omuls struct{}     // rd = rs1 * rs2 (float32)
type Odivs struct{}     // rd = rs1 / rs2 (float32)

// Float conversions
type Osingleoffloat struct{} // float64 -> float32
type Ofloatofsingle struct{} // float32 -> float64
type Ointoffloat struct{}    // float64 -> int (signed)
type Ointuoffloat struct{}   // float64 -> int (unsigned)
type Ofloatofint struct{}    // int -> float64 (signed)
type Ofloatofintu struct{}   // int -> float64 (unsigned)
type Olongoffloat struct{}   // float64 -> long (signed)
type Olonguoffloat struct{}  // float64 -> long (unsigned)
type Ofloatoflong struct{}   // long -> float64 (signed)
type Ofloatoflongu struct{}  // long -> float64 (unsigned)

// Comparison operations (produce int 0 or 1)
type Ocmp struct{ Cond Condition }  // compare signed
type Ocmpu struct{ Cond Condition } // compare unsigned
type Ocmpf struct{ Cond Condition } // compare float64
type Ocmps struct{ Cond Condition } // compare float32
type Ocmpl struct{ Cond Condition } // compare long signed
type Ocmplu struct{ Cond Condition }// compare long unsigned
type Ocmpimm struct{ Cond Condition; N int32 }  // compare imm signed
type Ocmpuimm struct{ Cond Condition; N int32 } // compare imm unsigned
type Ocmplimm struct{ Cond Condition; N int64 } // compare imm long signed
type Ocmpluimm struct{ Cond Condition; N int64 }// compare imm long unsigned

// Marker methods for Operation interface
func (Omove) implOperation()           {}
func (Ointconst) implOperation()       {}
func (Olongconst) implOperation()      {}
func (Ofloatconst) implOperation()     {}
func (Osingleconst) implOperation()    {}
func (Oaddrsymbol) implOperation()     {}
func (Oaddrstack) implOperation()      {}
func (Oadd) implOperation()            {}
func (Oaddimm) implOperation()         {}
func (Oneg) implOperation()            {}
func (Osub) implOperation()            {}
func (Omul) implOperation()            {}
func (Omulimm) implOperation()         {}
func (Omulhs) implOperation()          {}
func (Omulhu) implOperation()          {}
func (Odiv) implOperation()            {}
func (Odivu) implOperation()           {}
func (Omod) implOperation()            {}
func (Omodu) implOperation()           {}
func (Oand) implOperation()            {}
func (Oandimm) implOperation()         {}
func (Oor) implOperation()             {}
func (Oorimm) implOperation()          {}
func (Oxor) implOperation()            {}
func (Oxorimm) implOperation()         {}
func (Onot) implOperation()            {}
func (Oshl) implOperation()            {}
func (Oshlimm) implOperation()         {}
func (Oshr) implOperation()            {}
func (Oshrimm) implOperation()         {}
func (Oshru) implOperation()           {}
func (Oshruimm) implOperation()        {}
func (Oaddl) implOperation()           {}
func (Oaddlimm) implOperation()        {}
func (Onegl) implOperation()           {}
func (Osubl) implOperation()           {}
func (Omull) implOperation()           {}
func (Omullimm) implOperation()        {}
func (Omullhs) implOperation()         {}
func (Omullhu) implOperation()         {}
func (Odivl) implOperation()           {}
func (Odivlu) implOperation()          {}
func (Omodl) implOperation()           {}
func (Omodlu) implOperation()          {}
func (Oandl) implOperation()           {}
func (Oandlimm) implOperation()        {}
func (Oorl) implOperation()            {}
func (Oorlimm) implOperation()         {}
func (Oxorl) implOperation()           {}
func (Oxorlimm) implOperation()        {}
func (Onotl) implOperation()           {}
func (Oshll) implOperation()           {}
func (Oshllimm) implOperation()        {}
func (Oshrl) implOperation()           {}
func (Oshrlimm) implOperation()        {}
func (Oshrlu) implOperation()          {}
func (Oshrluimm) implOperation()       {}
func (Ocast8signed) implOperation()    {}
func (Ocast8unsigned) implOperation()  {}
func (Ocast16signed) implOperation()   {}
func (Ocast16unsigned) implOperation() {}
func (Olongofint) implOperation()      {}
func (Olongofintu) implOperation()     {}
func (Ointoflong) implOperation()      {}
func (Onegf) implOperation()           {}
func (Oabsf) implOperation()           {}
func (Oaddf) implOperation()           {}
func (Osubf) implOperation()           {}
func (Omulf) implOperation()           {}
func (Odivf) implOperation()           {}
func (Onegs) implOperation()           {}
func (Oabss) implOperation()           {}
func (Oadds) implOperation()           {}
func (Osubs) implOperation()           {}
func (Omuls) implOperation()           {}
func (Odivs) implOperation()           {}
func (Osingleoffloat) implOperation()  {}
func (Ofloatofsingle) implOperation()  {}
func (Ointoffloat) implOperation()     {}
func (Ointuoffloat) implOperation()    {}
func (Ofloatofint) implOperation()     {}
func (Ofloatofintu) implOperation()    {}
func (Olongoffloat) implOperation()    {}
func (Olonguoffloat) implOperation()   {}
func (Ofloatoflong) implOperation()    {}
func (Ofloatoflongu) implOperation()   {}
func (Ocmp) implOperation()            {}
func (Ocmpu) implOperation()           {}
func (Ocmpf) implOperation()           {}
func (Ocmps) implOperation()           {}
func (Ocmpl) implOperation()           {}
func (Ocmplu) implOperation()          {}
func (Ocmpimm) implOperation()         {}
func (Ocmpuimm) implOperation()        {}
func (Ocmplimm) implOperation()        {}
func (Ocmpluimm) implOperation()       {}

// --- Condition Codes ---
// Conditions for Icond instruction

// Condition represents a comparison condition
type Condition int

const (
	Ceq Condition = iota // equal
	Cne                  // not equal
	Clt                  // less than
	Cle                  // less than or equal
	Cgt                  // greater than
	Cge                  // greater than or equal
)

func (c Condition) String() string {
	names := []string{"==", "!=", "<", "<=", ">", ">="}
	if int(c) < len(names) {
		return names[c]
	}
	return "?"
}

// Negate returns the negated condition
func (c Condition) Negate() Condition {
	switch c {
	case Ceq:
		return Cne
	case Cne:
		return Ceq
	case Clt:
		return Cge
	case Cle:
		return Cgt
	case Cgt:
		return Cle
	case Cge:
		return Clt
	}
	return c
}

// --- Instruction Types ---
// Each instruction operates on pseudo-registers and branches to successor(s)

// Instruction is the interface for RTL instructions
type Instruction interface {
	implInstruction()
	Successors() []Node
}

// Inop is a no-operation that just branches to the successor
type Inop struct {
	Succ Node
}

// Iop performs an operation: dest = op(args...)
type Iop struct {
	Op   Operation // the operation
	Args []Reg     // source registers
	Dest Reg       // destination register
	Succ Node      // successor node
}

// Iload loads from memory: dest = Mem[addr(args...)]
type Iload struct {
	Chunk Chunk          // memory access size/type
	Addr  AddressingMode // addressing mode
	Args  []Reg          // registers for addressing
	Dest  Reg            // destination register
	Succ  Node           // successor node
}

// Istore stores to memory: Mem[addr(args...)] = src
type Istore struct {
	Chunk Chunk          // memory access size/type
	Addr  AddressingMode // addressing mode
	Args  []Reg          // registers for addressing
	Src   Reg            // source register (value to store)
	Succ  Node           // successor node
}

// Icall performs a function call
type Icall struct {
	Sig    Sig         // function signature
	Fn     FunRef      // function to call (reg or symbol)
	Args   []Reg       // argument registers
	Dest   Reg         // destination for return value
	Succ   Node        // successor node
}

// Itailcall performs a tail call (no return to caller)
type Itailcall struct {
	Sig  Sig    // function signature
	Fn   FunRef // function to call
	Args []Reg  // argument registers
}

// Ibuiltin calls a builtin function
type Ibuiltin struct {
	Builtin string // builtin function name
	Args    []Reg  // argument registers
	Dest    *Reg   // destination register (nil if no result)
	Succ    Node   // successor node
}

// Icond is a conditional branch
type Icond struct {
	Cond  ConditionCode // condition to evaluate
	Args  []Reg         // argument registers
	IfSo  Node          // branch target if condition is true
	IfNot Node          // branch target if condition is false
}

// Ijumptable is an indexed jump (switch)
type Ijumptable struct {
	Arg    Reg    // register containing index
	Targets []Node // jump targets
}

// Ireturn returns from the function
type Ireturn struct {
	Arg *Reg // return value register (nil for void)
}

// Marker methods for Instruction interface
func (Inop) implInstruction()       {}
func (Iop) implInstruction()        {}
func (Iload) implInstruction()      {}
func (Istore) implInstruction()     {}
func (Icall) implInstruction()      {}
func (Itailcall) implInstruction()  {}
func (Ibuiltin) implInstruction()   {}
func (Icond) implInstruction()      {}
func (Ijumptable) implInstruction() {}
func (Ireturn) implInstruction()    {}

// Successors returns the list of successor nodes
func (i Inop) Successors() []Node       { return []Node{i.Succ} }
func (i Iop) Successors() []Node        { return []Node{i.Succ} }
func (i Iload) Successors() []Node      { return []Node{i.Succ} }
func (i Istore) Successors() []Node     { return []Node{i.Succ} }
func (i Icall) Successors() []Node      { return []Node{i.Succ} }
func (i Itailcall) Successors() []Node  { return nil }
func (i Ibuiltin) Successors() []Node   { return []Node{i.Succ} }
func (i Icond) Successors() []Node      { return []Node{i.IfSo, i.IfNot} }
func (i Ijumptable) Successors() []Node { return i.Targets }
func (i Ireturn) Successors() []Node    { return nil }

// FunRef represents a function reference (either register or symbol)
type FunRef interface {
	implFunRef()
}

// FunReg is a function pointer in a register
type FunReg struct {
	Reg Reg
}

// FunSymbol is a named function symbol
type FunSymbol struct {
	Name string
}

func (FunReg) implFunRef()    {}
func (FunSymbol) implFunRef() {}

// ConditionCode represents a comparison condition for Icond
type ConditionCode interface {
	implConditionCode()
}

// Ccomp is an integer comparison
type Ccomp struct {
	Cond Condition
}

// Ccompu is an unsigned integer comparison
type Ccompu struct {
	Cond Condition
}

// Ccompimm is an integer comparison with immediate
type Ccompimm struct {
	Cond Condition
	N    int32
}

// Ccompuimm is an unsigned integer comparison with immediate
type Ccompuimm struct {
	Cond Condition
	N    int32
}

// Ccompl is a long comparison
type Ccompl struct {
	Cond Condition
}

// Ccomplu is an unsigned long comparison
type Ccomplu struct {
	Cond Condition
}

// Ccomplimm is a long comparison with immediate
type Ccomplimm struct {
	Cond Condition
	N    int64
}

// Ccompluimm is an unsigned long comparison with immediate
type Ccompluimm struct {
	Cond Condition
	N    int64
}

// Ccompf is a float64 comparison
type Ccompf struct {
	Cond Condition
}

// Cnotcompf is a negated float64 comparison
type Cnotcompf struct {
	Cond Condition
}

// Ccomps is a float32 comparison
type Ccomps struct {
	Cond Condition
}

// Cnotcomps is a negated float32 comparison
type Cnotcomps struct {
	Cond Condition
}

func (Ccomp) implConditionCode()     {}
func (Ccompu) implConditionCode()    {}
func (Ccompimm) implConditionCode()  {}
func (Ccompuimm) implConditionCode() {}
func (Ccompl) implConditionCode()    {}
func (Ccomplu) implConditionCode()   {}
func (Ccomplimm) implConditionCode() {}
func (Ccompluimm) implConditionCode(){}
func (Ccompf) implConditionCode()    {}
func (Cnotcompf) implConditionCode() {}
func (Ccomps) implConditionCode()    {}
func (Cnotcomps) implConditionCode() {}

// --- Function and Program ---

// Function represents an RTL function
type Function struct {
	Name       string             // function name
	Sig        Sig                // function signature
	Params     []Reg              // parameter registers
	Stacksize  int64              // stack frame size
	Code       map[Node]Instruction // CFG: node -> instruction
	Entrypoint Node               // entry node
}

// GlobVar represents a global variable
type GlobVar struct {
	Name string
	Size int64
	Init []byte
}

// Program represents a complete RTL program
type Program struct {
	Globals   []GlobVar
	Functions []Function
}

// NewFunction creates a new RTL function with initialized code map
func NewFunction(name string, sig Sig) *Function {
	return &Function{
		Name: name,
		Sig:  sig,
		Code: make(map[Node]Instruction),
	}
}
