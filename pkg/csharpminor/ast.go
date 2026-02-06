// Package csharpminor defines the Csharpminor intermediate representation.
// Csharpminor is a low-level structured language where type-dependent operations
// are made explicit. This mirrors CompCert's Csharpminor.v
package csharpminor

import "github.com/raymyers/ralph-cc/pkg/ctypes"

// Node is the base interface for all Csharpminor AST nodes
type Node interface {
	implCsharpminorNode()
}

// Expr is the interface for Csharpminor expressions
type Expr interface {
	Node
	implCsharpminorExpr()
}

// Stmt is the interface for Csharpminor statements
type Stmt interface {
	Node
	implCsharpminorStmt()
}

// --- Memory Chunks ---

// Chunk represents memory access size/type (memory_chunk in Csharpminor.v)
type Chunk int

const (
	Mint8signed Chunk = iota
	Mint8unsigned
	Mint16signed
	Mint16unsigned
	Mint32
	Mint64
	Mfloat32
	Mfloat64
	Many32 // any 32-bit value
	Many64 // any 64-bit value
)

func (c Chunk) String() string {
	names := []string{
		"int8s", "int8u", "int16s", "int16u",
		"int32", "int64", "float32", "float64",
		"any32", "any64",
	}
	if int(c) < len(names) {
		return names[c]
	}
	return "?"
}

// --- Typed Unary Operators ---

// UnaryOp represents typed unary operators in Csharpminor
type UnaryOp int

const (
	// Cast operations
	Ocast8signed UnaryOp = iota
	Ocast8unsigned
	Ocast16signed
	Ocast16unsigned

	// Negation
	Onegint // integer negation
	Onegf   // float64 negation
	Onegl   // long negation
	Onegs   // float32 negation

	// Bitwise not
	Onotint // integer bitwise not
	Onotl   // long bitwise not

	// Boolean not (actually int)
	Onotbool

	// Float conversions
	Osingleoffloat // float64 -> float32
	Ofloatofsingle // float32 -> float64

	// Int/float conversions
	Ointoffloat  // float64 -> int (signed)
	Ointuoffloat // float64 -> int (unsigned)
	Ofloatofint  // int -> float64 (signed)
	Ofloatofintu // int -> float64 (unsigned)

	// Long/float conversions
	Olongoffloat  // float64 -> long (signed)
	Olonguoffloat // float64 -> long (unsigned)
	Ofloatoflong  // long -> float64 (signed)
	Ofloatoflongu // long -> float64 (unsigned)

	// Long/single conversions
	Olongofsingle  // float32 -> long (signed)
	Olonguofsingle // float32 -> long (unsigned)
	Osingleoflong  // long -> float32 (signed)
	Osingleoflongu // long -> float32 (unsigned)

	// Int/long conversions
	Ointoflong  // long -> int
	Olongofint  // int -> long (signed)
	Olongofintu // int -> long (unsigned)
)

func (op UnaryOp) String() string {
	names := []string{
		"cast8signed", "cast8unsigned", "cast16signed", "cast16unsigned",
		"negint", "negf", "negl", "negs",
		"notint", "notl", "notbool",
		"singleoffloat", "floatofsingle",
		"intoffloat", "intuoffloat", "floatofint", "floatofintu",
		"longoffloat", "longuoffloat", "floatoflong", "floatoflongu",
		"longofsingle", "longuofsingle", "singleoflong", "singleoflongu",
		"intoflong", "longofint", "longofintu",
	}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

// --- Typed Binary Operators ---

// BinaryOp represents typed binary operators in Csharpminor
type BinaryOp int

const (
	// Integer arithmetic
	Oadd  BinaryOp = iota // int addition
	Osub                  // int subtraction
	Omul                  // int multiplication
	Odiv                  // int signed division
	Odivu                 // int unsigned division
	Omod                  // int signed modulo
	Omodu                 // int unsigned modulo

	// Float64 arithmetic
	Oaddf // float64 addition
	Osubf // float64 subtraction
	Omulf // float64 multiplication
	Odivf // float64 division

	// Float32 arithmetic
	Oadds // float32 addition
	Osubs // float32 subtraction
	Omuls // float32 multiplication
	Odivs // float32 division

	// Long arithmetic
	Oaddl  // long addition
	Osubl  // long subtraction
	Omull  // long multiplication
	Odivl  // long signed division
	Odivlu // long unsigned division
	Omodl  // long signed modulo
	Omodlu // long unsigned modulo

	// Integer bitwise
	Oand  // int bitwise and
	Oor   // int bitwise or
	Oxor  // int bitwise xor
	Oshl  // int shift left
	Oshr  // int shift right (signed)
	Oshru // int shift right (unsigned)

	// Long bitwise
	Oandl  // long bitwise and
	Oorl   // long bitwise or
	Oxorl  // long bitwise xor
	Oshll  // long shift left
	Oshrl  // long shift right (signed)
	Oshrlu // long shift right (unsigned)

	// Integer comparisons
	Ocmp  // int signed comparison
	Ocmpu // int unsigned comparison

	// Float comparisons
	Ocmpf // float64 comparison
	Ocmps // float32 comparison

	// Long comparisons
	Ocmpl  // long signed comparison
	Ocmplu // long unsigned comparison
)

func (op BinaryOp) String() string {
	names := []string{
		"add", "sub", "mul", "div", "divu", "mod", "modu",
		"addf", "subf", "mulf", "divf",
		"adds", "subs", "muls", "divs",
		"addl", "subl", "mull", "divl", "divlu", "modl", "modlu",
		"and", "or", "xor", "shl", "shr", "shru",
		"andl", "orl", "xorl", "shll", "shrl", "shrlu",
		"cmp", "cmpu", "cmpf", "cmps", "cmpl", "cmplu",
	}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

// Comparison represents comparison operators for Ocmp, Ocmpu, etc.
type Comparison int

const (
	Ceq Comparison = iota // equal
	Cne                   // not equal
	Clt                   // less than
	Cle                   // less or equal
	Cgt                   // greater than
	Cge                   // greater or equal
)

func (c Comparison) String() string {
	names := []string{"==", "!=", "<", "<=", ">", ">="}
	if int(c) < len(names) {
		return names[c]
	}
	return "?"
}

// --- Constants ---

// Constant represents typed constants
type Constant interface {
	Node
	implCsharpminorConst()
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
	Offset int64 // offset from symbol start
}

// --- Expressions ---

// Evar represents a reference to a global variable
type Evar struct {
	Name string
}

// Etempvar represents a reference to a local temporary
type Etempvar struct {
	ID int
}

// Eaddrof represents address of a global variable
type Eaddrof struct {
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

// Sset represents assignment to a temporary: temp = expr
type Sset struct {
	TempID int
	RHS    Expr
}

// Sstore represents memory store: *addr = value
type Sstore struct {
	Chunk Chunk // memory access size/type
	Addr  Expr  // address to store to
	Value Expr  // value to store
}

// Scall represents a function call
type Scall struct {
	Result *int   // temporary ID for result, nil for void
	Sig    *Sig   // function signature (optional)
	Func   Expr   // function to call (typically Eaddrof)
	Args   []Expr // arguments
}

// Stailcall represents a tail call
type Stailcall struct {
	Sig  *Sig   // function signature
	Func Expr   // function to call
	Args []Expr // arguments
}

// Sbuiltin represents a call to a builtin function
type Sbuiltin struct {
	Result  *int   // temporary ID for result, nil for void
	Builtin string // builtin name
	Args    []Expr // arguments
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

// VarDecl represents a variable declaration
type VarDecl struct {
	Name     string
	Size     int64  // size in bytes
	Init     []byte // initial data (nil if uninitialized)
	ReadOnly bool   // true for read-only data (e.g., string literals)
	Signed   bool   // true for signed types (int8_t), false for unsigned (uint8_t)
}

// Sig represents a function signature
type Sig struct {
	Args   []ctypes.Type
	Return ctypes.Type
	VarArg bool
}

// Function represents a function in Csharpminor
type Function struct {
	Name   string
	Sig    Sig
	Params []string      // parameter names
	Locals []VarDecl     // local variables (stack allocated)
	Temps  []ctypes.Type // temporary types
	Body   Stmt
}

// Program represents a complete Csharpminor program
type Program struct {
	Globals   []VarDecl  // global variables
	Functions []Function // function definitions
}

// --- Interface implementations ---

// Marker methods for Node interface
func (Ointconst) implCsharpminorNode()   {}
func (Ofloatconst) implCsharpminorNode() {}
func (Olongconst) implCsharpminorNode()  {}
func (Osingleconst) implCsharpminorNode() {}
func (Oaddrsymbol) implCsharpminorNode() {}
func (Evar) implCsharpminorNode()        {}
func (Etempvar) implCsharpminorNode()    {}
func (Eaddrof) implCsharpminorNode()     {}
func (Econst) implCsharpminorNode()      {}
func (Eunop) implCsharpminorNode()       {}
func (Ebinop) implCsharpminorNode()      {}
func (Ecmp) implCsharpminorNode()        {}
func (Eload) implCsharpminorNode()       {}

func (Sskip) implCsharpminorNode()       {}
func (Sset) implCsharpminorNode()        {}
func (Sstore) implCsharpminorNode()      {}
func (Scall) implCsharpminorNode()       {}
func (Stailcall) implCsharpminorNode()   {}
func (Sbuiltin) implCsharpminorNode()    {}
func (Sseq) implCsharpminorNode()        {}
func (Sifthenelse) implCsharpminorNode() {}
func (Sloop) implCsharpminorNode()       {}
func (Sblock) implCsharpminorNode()      {}
func (Sexit) implCsharpminorNode()       {}
func (Sswitch) implCsharpminorNode()     {}
func (Sreturn) implCsharpminorNode()     {}
func (Slabel) implCsharpminorNode()      {}
func (Sgoto) implCsharpminorNode()       {}

// Marker methods for Constant interface
func (Ointconst) implCsharpminorConst()   {}
func (Ofloatconst) implCsharpminorConst() {}
func (Olongconst) implCsharpminorConst()  {}
func (Osingleconst) implCsharpminorConst() {}
func (Oaddrsymbol) implCsharpminorConst() {}

// Marker methods for Expr interface
func (Evar) implCsharpminorExpr()     {}
func (Etempvar) implCsharpminorExpr() {}
func (Eaddrof) implCsharpminorExpr()  {}
func (Econst) implCsharpminorExpr()   {}
func (Eunop) implCsharpminorExpr()    {}
func (Ebinop) implCsharpminorExpr()   {}
func (Ecmp) implCsharpminorExpr()     {}
func (Eload) implCsharpminorExpr()    {}

// Marker methods for Stmt interface
func (Sskip) implCsharpminorStmt()       {}
func (Sset) implCsharpminorStmt()        {}
func (Sstore) implCsharpminorStmt()      {}
func (Scall) implCsharpminorStmt()       {}
func (Stailcall) implCsharpminorStmt()   {}
func (Sbuiltin) implCsharpminorStmt()    {}
func (Sseq) implCsharpminorStmt()        {}
func (Sifthenelse) implCsharpminorStmt() {}
func (Sloop) implCsharpminorStmt()       {}
func (Sblock) implCsharpminorStmt()      {}
func (Sexit) implCsharpminorStmt()       {}
func (Sswitch) implCsharpminorStmt()     {}
func (Sreturn) implCsharpminorStmt()     {}
func (Slabel) implCsharpminorStmt()      {}
func (Sgoto) implCsharpminorStmt()       {}

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

// ChunkForType returns the appropriate memory chunk for a type
func ChunkForType(t ctypes.Type) Chunk {
	switch typ := t.(type) {
	case ctypes.Tint:
		switch typ.Size {
		case ctypes.I8:
			if typ.Sign == ctypes.Signed {
				return Mint8signed
			}
			return Mint8unsigned
		case ctypes.I16:
			if typ.Sign == ctypes.Signed {
				return Mint16signed
			}
			return Mint16unsigned
		case ctypes.I32, ctypes.IBool:
			return Mint32
		}
	case ctypes.Tlong:
		return Mint64
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return Mfloat32
		}
		return Mfloat64
	case ctypes.Tpointer:
		return Mint64 // pointers are 64-bit on aarch64
	}
	return Many32 // default
}
