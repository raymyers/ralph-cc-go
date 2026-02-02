// Package cminor defines the Cminor intermediate representation.
// Cminor introduces explicit stack allocation for address-taken local variables.
// This mirrors CompCert's backend/Cminor.v
package cminor

import "github.com/raymyers/ralph-cc/pkg/csharpminor"

// Re-export types from csharpminor that are identical in Cminor
type (
	Chunk      = csharpminor.Chunk
	UnaryOp    = csharpminor.UnaryOp
	BinaryOp   = csharpminor.BinaryOp
	Comparison = csharpminor.Comparison
)

// Re-export chunk constants
const (
	Mint8signed   = csharpminor.Mint8signed
	Mint8unsigned = csharpminor.Mint8unsigned
	Mint16signed  = csharpminor.Mint16signed
	Mint16unsigned = csharpminor.Mint16unsigned
	Mint32        = csharpminor.Mint32
	Mint64        = csharpminor.Mint64
	Mfloat32      = csharpminor.Mfloat32
	Mfloat64      = csharpminor.Mfloat64
	Many32        = csharpminor.Many32
	Many64        = csharpminor.Many64
)

// Re-export comparison constants
const (
	Ceq = csharpminor.Ceq
	Cne = csharpminor.Cne
	Clt = csharpminor.Clt
	Cle = csharpminor.Cle
	Cgt = csharpminor.Cgt
	Cge = csharpminor.Cge
)

// Re-export unary operator constants
const (
	Ocast8signed   = csharpminor.Ocast8signed
	Ocast8unsigned = csharpminor.Ocast8unsigned
	Ocast16signed  = csharpminor.Ocast16signed
	Ocast16unsigned = csharpminor.Ocast16unsigned
	Onegint        = csharpminor.Onegint
	Onegf          = csharpminor.Onegf
	Onegl          = csharpminor.Onegl
	Onegs          = csharpminor.Onegs
	Onotint        = csharpminor.Onotint
	Onotl          = csharpminor.Onotl
	Onotbool       = csharpminor.Onotbool
	Osingleoffloat = csharpminor.Osingleoffloat
	Ofloatofsingle = csharpminor.Ofloatofsingle
	Ointoffloat    = csharpminor.Ointoffloat
	Ointuoffloat   = csharpminor.Ointuoffloat
	Ofloatofint    = csharpminor.Ofloatofint
	Ofloatofintu   = csharpminor.Ofloatofintu
	Olongoffloat   = csharpminor.Olongoffloat
	Olonguoffloat  = csharpminor.Olonguoffloat
	Ofloatoflong   = csharpminor.Ofloatoflong
	Ofloatoflongu  = csharpminor.Ofloatoflongu
	Olongofsingle  = csharpminor.Olongofsingle
	Olonguofsingle = csharpminor.Olonguofsingle
	Osingleoflong  = csharpminor.Osingleoflong
	Osingleoflongu = csharpminor.Osingleoflongu
	Ointoflong     = csharpminor.Ointoflong
	Olongofint     = csharpminor.Olongofint
	Olongofintu    = csharpminor.Olongofintu
)

// Re-export binary operator constants
const (
	Oadd   = csharpminor.Oadd
	Osub   = csharpminor.Osub
	Omul   = csharpminor.Omul
	Odiv   = csharpminor.Odiv
	Odivu  = csharpminor.Odivu
	Omod   = csharpminor.Omod
	Omodu  = csharpminor.Omodu
	Oaddf  = csharpminor.Oaddf
	Osubf  = csharpminor.Osubf
	Omulf  = csharpminor.Omulf
	Odivf  = csharpminor.Odivf
	Oadds  = csharpminor.Oadds
	Osubs  = csharpminor.Osubs
	Omuls  = csharpminor.Omuls
	Odivs  = csharpminor.Odivs
	Oaddl  = csharpminor.Oaddl
	Osubl  = csharpminor.Osubl
	Omull  = csharpminor.Omull
	Odivl  = csharpminor.Odivl
	Odivlu = csharpminor.Odivlu
	Omodl  = csharpminor.Omodl
	Omodlu = csharpminor.Omodlu
	Oand   = csharpminor.Oand
	Oor    = csharpminor.Oor
	Oxor   = csharpminor.Oxor
	Oshl   = csharpminor.Oshl
	Oshr   = csharpminor.Oshr
	Oshru  = csharpminor.Oshru
	Oandl  = csharpminor.Oandl
	Oorl   = csharpminor.Oorl
	Oxorl  = csharpminor.Oxorl
	Oshll  = csharpminor.Oshll
	Oshrl  = csharpminor.Oshrl
	Oshrlu = csharpminor.Oshrlu
	Ocmp   = csharpminor.Ocmp
	Ocmpu  = csharpminor.Ocmpu
	Ocmpf  = csharpminor.Ocmpf
	Ocmps  = csharpminor.Ocmps
	Ocmpl  = csharpminor.Ocmpl
	Ocmplu = csharpminor.Ocmplu
)

// Node is the base interface for all Cminor AST nodes
type Node interface {
	implCminorNode()
}

// Expr is the interface for Cminor expressions
type Expr interface {
	Node
	implCminorExpr()
}

// Stmt is the interface for Cminor statements
type Stmt interface {
	Node
	implCminorStmt()
}

// --- Constants ---
// In Cminor, constants are the same as Csharpminor

// Constant represents typed constants
type Constant interface {
	Node
	implCminorConst()
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

// Oaddrsymbol represents the address of a symbol (string constant, global var)
type Oaddrsymbol struct {
	Name   string
	Offset int64
}

// --- Expressions ---
// Cminor expressions are similar to Csharpminor but use identifiers instead of temp IDs

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

// Ecmp represents a comparison operation (result is int 0 or 1)
type Ecmp struct {
	Op    BinaryOp   // Ocmp, Ocmpu, Ocmpf, Ocmps, Ocmpl, Ocmplu
	Cmp   Comparison // Ceq, Cne, etc.
	Left  Expr
	Right Expr
}

// Eload represents explicit memory load with chunk
type Eload struct {
	Chunk Chunk // memory access size/type
	Addr  Expr  // address to load from
}

// --- Statements ---

// Sskip represents an empty statement
type Sskip struct{}

// Sassign represents assignment to a local variable: var = expr
// This is different from Csharpminor's Sset which uses temp IDs
type Sassign struct {
	Name string // variable name
	RHS  Expr
}

// Sstore represents memory store: *addr = value
type Sstore struct {
	Chunk Chunk // memory access size/type
	Addr  Expr  // address to store to
	Value Expr  // value to store
}

// Scall represents a function call
type Scall struct {
	Result *string // variable name for result, nil for void
	Sig    *Sig    // function signature (optional)
	Func   Expr    // function to call
	Args   []Expr  // arguments
}

// Stailcall represents a tail call
type Stailcall struct {
	Sig  *Sig   // function signature
	Func Expr   // function to call
	Args []Expr // arguments
}

// Sbuiltin represents a call to a builtin function
type Sbuiltin struct {
	Result  *string // variable name for result, nil for void
	Builtin string  // builtin name
	Args    []Expr  // arguments
}

// Sseq represents a sequence of two statements
type Sseq struct {
	First  Stmt
	Second Stmt
}

// Sifthenelse represents an if-then-else statement
type Sifthenelse struct {
	Cond Expr
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

// Sexit represents exit from n nested blocks (n >= 1)
type Sexit struct {
	N int // number of blocks to exit
}

// Sswitch represents a switch statement
type Sswitch struct {
	IsLong  bool         // true for long switch, false for int
	Expr    Expr         // switch expression
	Cases   []SwitchCase // case branches
	Default Stmt         // default branch
}

// SwitchCase represents a case in a switch
type SwitchCase struct {
	Value int64 // case label value
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

// Function represents a function in Cminor
type Function struct {
	Name       string
	Sig        Sig
	Params     []string // parameter names
	Vars       []string // local variable names (stack allocated)
	Stackspace int64    // stack space required in bytes
	Body       Stmt
}

// GlobVar represents a global variable
type GlobVar struct {
	Name string
	Size int64 // size in bytes
	Init []byte // initial data (nil if uninitialized)
}

// Program represents a complete Cminor program
type Program struct {
	Globals   []GlobVar  // global variables
	Functions []Function // function definitions
}

// --- Interface implementations ---

// Marker methods for Node interface
func (Ointconst) implCminorNode()   {}
func (Ofloatconst) implCminorNode() {}
func (Olongconst) implCminorNode()  {}
func (Osingleconst) implCminorNode() {}
func (Oaddrsymbol) implCminorNode() {}
func (Evar) implCminorNode()        {}
func (Econst) implCminorNode()      {}
func (Eunop) implCminorNode()       {}
func (Ebinop) implCminorNode()      {}
func (Ecmp) implCminorNode()        {}
func (Eload) implCminorNode()       {}

func (Sskip) implCminorNode()       {}
func (Sassign) implCminorNode()     {}
func (Sstore) implCminorNode()      {}
func (Scall) implCminorNode()       {}
func (Stailcall) implCminorNode()   {}
func (Sbuiltin) implCminorNode()    {}
func (Sseq) implCminorNode()        {}
func (Sifthenelse) implCminorNode() {}
func (Sloop) implCminorNode()       {}
func (Sblock) implCminorNode()      {}
func (Sexit) implCminorNode()       {}
func (Sswitch) implCminorNode()     {}
func (Sreturn) implCminorNode()     {}
func (Slabel) implCminorNode()      {}
func (Sgoto) implCminorNode()       {}

// Marker methods for Constant interface
func (Ointconst) implCminorConst()   {}
func (Ofloatconst) implCminorConst() {}
func (Olongconst) implCminorConst()  {}
func (Osingleconst) implCminorConst() {}
func (Oaddrsymbol) implCminorConst() {}

// Marker methods for Expr interface
func (Evar) implCminorExpr()   {}
func (Econst) implCminorExpr() {}
func (Eunop) implCminorExpr()  {}
func (Ebinop) implCminorExpr() {}
func (Ecmp) implCminorExpr()   {}
func (Eload) implCminorExpr()  {}

// Marker methods for Stmt interface
func (Sskip) implCminorStmt()       {}
func (Sassign) implCminorStmt()     {}
func (Sstore) implCminorStmt()      {}
func (Scall) implCminorStmt()       {}
func (Stailcall) implCminorStmt()   {}
func (Sbuiltin) implCminorStmt()    {}
func (Sseq) implCminorStmt()        {}
func (Sifthenelse) implCminorStmt() {}
func (Sloop) implCminorStmt()       {}
func (Sblock) implCminorStmt()      {}
func (Sexit) implCminorStmt()       {}
func (Sswitch) implCminorStmt()     {}
func (Sreturn) implCminorStmt()     {}
func (Slabel) implCminorStmt()      {}
func (Sgoto) implCminorStmt()       {}

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
