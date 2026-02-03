// Package cminorsel defines the CminorSel intermediate representation.
// CminorSel is Cminor with target-specific operators and addressing modes.
// This is the boundary between frontend and backend (instruction selection).
// This mirrors CompCert's backend/CminorSel.v
package cminorsel

import "github.com/raymyers/ralph-cc/pkg/cminor"

// Re-export types from cminor that are identical in CminorSel
type (
	Chunk      = cminor.Chunk
	UnaryOp    = cminor.UnaryOp
	BinaryOp   = cminor.BinaryOp
	Comparison = cminor.Comparison
)

// Re-export chunk constants
const (
	Mint8signed    = cminor.Mint8signed
	Mint8unsigned  = cminor.Mint8unsigned
	Mint16signed   = cminor.Mint16signed
	Mint16unsigned = cminor.Mint16unsigned
	Mint32         = cminor.Mint32
	Mint64         = cminor.Mint64
	Mfloat32       = cminor.Mfloat32
	Mfloat64       = cminor.Mfloat64
	Many32         = cminor.Many32
	Many64         = cminor.Many64
)

// Re-export comparison constants
const (
	Ceq = cminor.Ceq
	Cne = cminor.Cne
	Clt = cminor.Clt
	Cle = cminor.Cle
	Cgt = cminor.Cgt
	Cge = cminor.Cge
)

// Re-export unary operator constants
const (
	Ocast8signed    = cminor.Ocast8signed
	Ocast8unsigned  = cminor.Ocast8unsigned
	Ocast16signed   = cminor.Ocast16signed
	Ocast16unsigned = cminor.Ocast16unsigned
	Onegint         = cminor.Onegint
	Onegf           = cminor.Onegf
	Onegl           = cminor.Onegl
	Onegs           = cminor.Onegs
	Onotint         = cminor.Onotint
	Onotl           = cminor.Onotl
	Onotbool        = cminor.Onotbool
	Osingleoffloat  = cminor.Osingleoffloat
	Ofloatofsingle  = cminor.Ofloatofsingle
	Ointoffloat     = cminor.Ointoffloat
	Ointuoffloat    = cminor.Ointuoffloat
	Ofloatofint     = cminor.Ofloatofint
	Ofloatofintu    = cminor.Ofloatofintu
	Olongoffloat    = cminor.Olongoffloat
	Olonguoffloat   = cminor.Olonguoffloat
	Ofloatoflong    = cminor.Ofloatoflong
	Ofloatoflongu   = cminor.Ofloatoflongu
	Olongofsingle   = cminor.Olongofsingle
	Olonguofsingle  = cminor.Olonguofsingle
	Osingleoflong   = cminor.Osingleoflong
	Osingleoflongu  = cminor.Osingleoflongu
	Ointoflong      = cminor.Ointoflong
	Olongofint      = cminor.Olongofint
	Olongofintu     = cminor.Olongofintu
)

// Re-export binary operator constants
const (
	Oadd   = cminor.Oadd
	Osub   = cminor.Osub
	Omul   = cminor.Omul
	Odiv   = cminor.Odiv
	Odivu  = cminor.Odivu
	Omod   = cminor.Omod
	Omodu  = cminor.Omodu
	Oaddf  = cminor.Oaddf
	Osubf  = cminor.Osubf
	Omulf  = cminor.Omulf
	Odivf  = cminor.Odivf
	Oadds  = cminor.Oadds
	Osubs  = cminor.Osubs
	Omuls  = cminor.Omuls
	Odivs  = cminor.Odivs
	Oaddl  = cminor.Oaddl
	Osubl  = cminor.Osubl
	Omull  = cminor.Omull
	Odivl  = cminor.Odivl
	Odivlu = cminor.Odivlu
	Omodl  = cminor.Omodl
	Omodlu = cminor.Omodlu
	Oand   = cminor.Oand
	Oor    = cminor.Oor
	Oxor   = cminor.Oxor
	Oshl   = cminor.Oshl
	Oshr   = cminor.Oshr
	Oshru  = cminor.Oshru
	Oandl  = cminor.Oandl
	Oorl   = cminor.Oorl
	Oxorl  = cminor.Oxorl
	Oshll  = cminor.Oshll
	Oshrl  = cminor.Oshrl
	Oshrlu = cminor.Oshrlu
	Ocmp   = cminor.Ocmp
	Ocmpu  = cminor.Ocmpu
	Ocmpf  = cminor.Ocmpf
	Ocmps  = cminor.Ocmps
	Ocmpl  = cminor.Ocmpl
	Ocmplu = cminor.Ocmplu
)

// --- Addressing Modes ---
// Target-specific addressing modes for memory operations.
// These are based on ARM64 (aarch64) which is our primary target.

// AddressingMode is the interface for addressing modes
type AddressingMode interface {
	implAddressingMode()
}

// Aindexed represents base + offset addressing: [base + offset]
type Aindexed struct {
	Offset int64 // constant offset
}

// Aindexed2 represents base + index addressing: [base + index]
type Aindexed2 struct{}

// Aindexed2shift represents base + (index << shift) addressing: [base + index*scale]
// Used on ARM64 for scaled array access
type Aindexed2shift struct {
	Shift int // shift amount (0-3 for scale 1,2,4,8)
}

// Aindexed2ext represents base + extended index addressing
// Used on ARM64 for sign/zero-extended indices
type Aindexed2ext struct {
	Extend ExtendOp // extension operation
	Shift  int      // shift amount
}

// Aglobal represents global symbol + offset addressing: [global + offset]
type Aglobal struct {
	Symbol string
	Offset int64
}

// Ainstack represents stack slot addressing: [sp + offset]
type Ainstack struct {
	Offset int64
}

// ExtendOp represents ARM64 register extension operations
type ExtendOp int

const (
	Xsgn32 ExtendOp = iota // sign-extend 32-bit to 64-bit
	Xuns32                 // zero-extend 32-bit to 64-bit
)

func (e ExtendOp) String() string {
	names := []string{"sxtw", "uxtw"}
	if int(e) < len(names) {
		return names[e]
	}
	return "?"
}

// Marker methods for AddressingMode interface
func (Aindexed) implAddressingMode()       {}
func (Aindexed2) implAddressingMode()      {}
func (Aindexed2shift) implAddressingMode() {}
func (Aindexed2ext) implAddressingMode()   {}
func (Aglobal) implAddressingMode()        {}
func (Ainstack) implAddressingMode()       {}

// --- Condition Codes ---
// Conditions for conditional branches and selects

// Condition is the interface for branch conditions
type Condition interface {
	implCondition()
}

// CondTrue represents always true condition
type CondTrue struct{}

// CondFalse represents always false condition
type CondFalse struct{}

// CondCmp represents a comparison: cmp(arg1, arg2)
type CondCmp struct {
	Cmp   Comparison // Ceq, Cne, etc.
	Left  Expr
	Right Expr
}

// CondCmpu represents unsigned comparison
type CondCmpu struct {
	Cmp   Comparison
	Left  Expr
	Right Expr
}

// CondCmpf represents float64 comparison
type CondCmpf struct {
	Cmp   Comparison
	Left  Expr
	Right Expr
}

// CondCmps represents float32 comparison
type CondCmps struct {
	Cmp   Comparison
	Left  Expr
	Right Expr
}

// CondCmpl represents long (int64) comparison
type CondCmpl struct {
	Cmp   Comparison
	Left  Expr
	Right Expr
}

// CondCmplu represents unsigned long comparison
type CondCmplu struct {
	Cmp   Comparison
	Left  Expr
	Right Expr
}

// CondNot represents negation of a condition
type CondNot struct {
	Cond Condition
}

// CondAnd represents conjunction of conditions (short-circuit)
type CondAnd struct {
	Left  Condition
	Right Condition
}

// CondOr represents disjunction of conditions (short-circuit)
type CondOr struct {
	Left  Condition
	Right Condition
}

// Marker methods for Condition interface
func (CondTrue) implCondition()   {}
func (CondFalse) implCondition()  {}
func (CondCmp) implCondition()    {}
func (CondCmpu) implCondition()   {}
func (CondCmpf) implCondition()   {}
func (CondCmps) implCondition()   {}
func (CondCmpl) implCondition()   {}
func (CondCmplu) implCondition()  {}
func (CondNot) implCondition()    {}
func (CondAnd) implCondition()    {}
func (CondOr) implCondition()     {}

// --- Node Interface ---

// Node is the base interface for all CminorSel AST nodes
type Node interface {
	implCminorSelNode()
}

// Expr is the interface for CminorSel expressions
type Expr interface {
	Node
	implCminorSelExpr()
}

// Stmt is the interface for CminorSel statements
type Stmt interface {
	Node
	implCminorSelStmt()
}

// --- Constants ---
// Re-use constant types from cminor

// Constant represents typed constants
type Constant interface {
	Node
	implCminorSelConst()
}

// Ointconst represents an integer constant
type Ointconst struct {
	Value int32
}

// Ofloatconst represents a float64 constant
type Ofloatconst struct {
	Value float64
}

// Olongconst represents a long (int64) constant
type Olongconst struct {
	Value int64
}

// Osingleconst represents a float32 constant
type Osingleconst struct {
	Value float32
}

// Oaddrsymbol represents address of a global symbol + offset
type Oaddrsymbol struct {
	Symbol string
	Offset int64
}

// Oaddrstack represents address of a stack slot
type Oaddrstack struct {
	Offset int64
}

// --- Expressions ---
// CminorSel expressions include machine-level operations

// Evar represents a reference to a variable (local or global)
type Evar struct {
	Name string
}

// Econst represents a constant value
type Econst struct {
	Const Constant
}

// Eunop represents a typed unary operation
type Eunop struct {
	Op  UnaryOp
	Arg Expr
}

// Ebinop represents a typed binary operation
type Ebinop struct {
	Op    BinaryOp
	Left  Expr
	Right Expr
}

// Eload represents memory load with addressing mode
type Eload struct {
	Chunk Chunk          // memory access size/type
	Mode  AddressingMode // addressing mode
	Args  []Expr         // arguments for addressing mode
}

// Econdition represents conditional expression (ternary): cond ? then : else
type Econdition struct {
	Cond Condition
	Then Expr
	Else Expr
}

// Elet represents let-binding: let temp = e1 in e2
// This introduces a new temporary for common subexpressions
type Elet struct {
	Bind Expr // expression to bind
	Body Expr // body using the bound value (via Eletvar)
}

// Eletvar represents reference to the let-bound variable
// Uses de Bruijn indices: 0 = innermost let, 1 = next outer, etc.
type Eletvar struct {
	Index int
}

// --- ARM64-Specific Combined Operations ---
// These fuse multiple operations for efficiency

// Eaddshift represents add with shifted operand: a + (b << shift)
type Eaddshift struct {
	Op    ShiftOp // shift operation
	Shift int     // shift amount
	Left  Expr
	Right Expr
}

// Esubshift represents sub with shifted operand: a - (b << shift)
type Esubshift struct {
	Op    ShiftOp
	Shift int
	Left  Expr
	Right Expr
}

// Ecmp represents a comparison expression that produces int 0 or 1
type Ecmp struct {
	Op    BinaryOp   // Ocmp, Ocmpu, Ocmpf, Ocmps, Ocmpl, Ocmplu
	Cmp   Comparison // Ceq, Cne, Clt, Cle, Cgt, Cge
	Left  Expr
	Right Expr
}

// ShiftOp represents shift operations for combined ops
type ShiftOp int

const (
	Slsl ShiftOp = iota // logical shift left
	Slsr                // logical shift right
	Sasr                // arithmetic shift right
)

func (s ShiftOp) String() string {
	names := []string{"lsl", "lsr", "asr"}
	if int(s) < len(names) {
		return names[s]
	}
	return "?"
}

// --- Statements ---
// CminorSel statements are similar to Cminor

// Sskip represents an empty statement
type Sskip struct{}

// Sassign represents assignment to a local variable: var = expr
type Sassign struct {
	Name string
	RHS  Expr
}

// Sstore represents memory store with addressing mode
type Sstore struct {
	Chunk Chunk
	Mode  AddressingMode
	Args  []Expr // address arguments
	Value Expr   // value to store
}

// Scall represents a function call
type Scall struct {
	Result *string // variable name for result, nil for void
	Sig    *Sig    // function signature
	Func   Expr    // function to call
	Args   []Expr  // arguments
}

// Stailcall represents a tail call
type Stailcall struct {
	Sig  *Sig
	Func Expr
	Args []Expr
}

// Sbuiltin represents a call to a builtin function
type Sbuiltin struct {
	Result  *string
	Builtin string
	Args    []Expr
}

// Sseq represents a sequence of two statements
type Sseq struct {
	First  Stmt
	Second Stmt
}

// Sifthenelse represents conditional with condition type
type Sifthenelse struct {
	Cond Condition
	Then Stmt
	Else Stmt
}

// Sloop represents infinite loop
type Sloop struct {
	Body Stmt
}

// Sblock represents a block (target for Sexit)
type Sblock struct {
	Body Stmt
}

// Sexit represents exit from n nested blocks
type Sexit struct {
	N int
}

// Sswitch represents a switch statement
type Sswitch struct {
	IsLong  bool
	Expr    Expr
	Cases   []SwitchCase
	Default Stmt
}

// SwitchCase represents a case in a switch
type SwitchCase struct {
	Value int64
	Body  Stmt
}

// Sreturn represents return from function
type Sreturn struct {
	Value Expr // nil for void return
}

// Slabel represents a labeled statement
type Slabel struct {
	Label string
	Body  Stmt
}

// Sgoto represents a goto statement
type Sgoto struct {
	Label string
}

// --- Functions and Programs ---

// Sig represents a function signature
type Sig struct {
	Args   []string // argument type descriptors
	Return string   // return type descriptor
	VarArg bool
}

// Function represents a function in CminorSel
type Function struct {
	Name       string
	Sig        Sig
	Params     []string
	Vars       []string
	Stackspace int64
	Body       Stmt
}

// GlobVar represents a global variable
type GlobVar struct {
	Name     string
	Size     int64
	Init     []byte
	ReadOnly bool // true for .rodata section (e.g., string literals)
}

// Program represents a complete CminorSel program
type Program struct {
	Globals   []GlobVar
	Functions []Function
}

// --- Interface Implementations ---

// Marker methods for Node interface
func (Ointconst) implCminorSelNode()    {}
func (Ofloatconst) implCminorSelNode()  {}
func (Olongconst) implCminorSelNode()   {}
func (Osingleconst) implCminorSelNode() {}
func (Oaddrsymbol) implCminorSelNode()  {}
func (Oaddrstack) implCminorSelNode()   {}
func (Evar) implCminorSelNode()         {}
func (Econst) implCminorSelNode()       {}
func (Eunop) implCminorSelNode()        {}
func (Ebinop) implCminorSelNode()       {}
func (Eload) implCminorSelNode()        {}
func (Econdition) implCminorSelNode()   {}
func (Elet) implCminorSelNode()         {}
func (Eletvar) implCminorSelNode()      {}
func (Eaddshift) implCminorSelNode()    {}
func (Esubshift) implCminorSelNode()    {}
func (Ecmp) implCminorSelNode()         {}

func (Sskip) implCminorSelNode()       {}
func (Sassign) implCminorSelNode()     {}
func (Sstore) implCminorSelNode()      {}
func (Scall) implCminorSelNode()       {}
func (Stailcall) implCminorSelNode()   {}
func (Sbuiltin) implCminorSelNode()    {}
func (Sseq) implCminorSelNode()        {}
func (Sifthenelse) implCminorSelNode() {}
func (Sloop) implCminorSelNode()       {}
func (Sblock) implCminorSelNode()      {}
func (Sexit) implCminorSelNode()       {}
func (Sswitch) implCminorSelNode()     {}
func (Sreturn) implCminorSelNode()     {}
func (Slabel) implCminorSelNode()      {}
func (Sgoto) implCminorSelNode()       {}

// Marker methods for Constant interface
func (Ointconst) implCminorSelConst()    {}
func (Ofloatconst) implCminorSelConst()  {}
func (Olongconst) implCminorSelConst()   {}
func (Osingleconst) implCminorSelConst() {}
func (Oaddrsymbol) implCminorSelConst()  {}
func (Oaddrstack) implCminorSelConst()   {}

// Marker methods for Expr interface
func (Evar) implCminorSelExpr()       {}
func (Econst) implCminorSelExpr()     {}
func (Eunop) implCminorSelExpr()      {}
func (Ebinop) implCminorSelExpr()     {}
func (Eload) implCminorSelExpr()      {}
func (Econdition) implCminorSelExpr() {}
func (Elet) implCminorSelExpr()       {}
func (Eletvar) implCminorSelExpr()    {}
func (Eaddshift) implCminorSelExpr()  {}
func (Esubshift) implCminorSelExpr()  {}
func (Ecmp) implCminorSelExpr()       {}

// Marker methods for Stmt interface
func (Sskip) implCminorSelStmt()       {}
func (Sassign) implCminorSelStmt()     {}
func (Sstore) implCminorSelStmt()      {}
func (Scall) implCminorSelStmt()       {}
func (Stailcall) implCminorSelStmt()   {}
func (Sbuiltin) implCminorSelStmt()    {}
func (Sseq) implCminorSelStmt()        {}
func (Sifthenelse) implCminorSelStmt() {}
func (Sloop) implCminorSelStmt()       {}
func (Sblock) implCminorSelStmt()      {}
func (Sexit) implCminorSelStmt()       {}
func (Sswitch) implCminorSelStmt()     {}
func (Sreturn) implCminorSelStmt()     {}
func (Slabel) implCminorSelStmt()      {}
func (Sgoto) implCminorSelStmt()       {}

// Seq creates a sequence of statements, flattening Sskip
func Seq(stmts ...Stmt) Stmt {
	var result Stmt = Sskip{}
	for _, s := range stmts {
		if _, ok := result.(Sskip); ok {
			result = s
		} else if _, ok := s.(Sskip); !ok {
			result = Sseq{First: result, Second: s}
		}
	}
	return result
}
