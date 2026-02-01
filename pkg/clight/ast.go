// Package clight defines the Clight intermediate representation.
// Clight is C without side-effects in expressions - all side effects become statements.
// This mirrors CompCert's Clight.v
package clight

import "github.com/raymyers/ralph-cc/pkg/ctypes"

// Node is the base interface for all Clight AST nodes
type Node interface {
	implClightNode()
}

// Expr is the interface for Clight expressions (side-effect free)
type Expr interface {
	Node
	implClightExpr()
	ExprType() ctypes.Type // returns the type of the expression
}

// Stmt is the interface for Clight statements
type Stmt interface {
	Node
	implClightStmt()
}

// UnaryOp represents unary operators in Clight
type UnaryOp int

const (
	Onotbool  UnaryOp = iota // boolean negation (!)
	Onotint                  // integer complement (~)
	Oneg                     // integer negation (-)
	Oabsfloat                // float absolute value
)

func (op UnaryOp) String() string {
	names := []string{"!", "~", "-", "abs"}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

// BinaryOp represents binary operators in Clight
type BinaryOp int

const (
	// Arithmetic
	Oadd BinaryOp = iota // addition
	Osub                 // subtraction
	Omul                 // multiplication
	Odiv                 // division
	Omod                 // modulo

	// Comparison
	Oeq // equal
	One // not equal
	Olt // less than
	Ogt // greater than
	Ole // less or equal
	Oge // greater or equal

	// Bitwise
	Oand // bitwise and
	Oor  // bitwise or
	Oxor // bitwise xor
	Oshl // shift left
	Oshr // shift right
)

func (op BinaryOp) String() string {
	names := []string{"+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=", "&", "|", "^", "<<", ">>"}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

// --- Expressions ---

// Econst_int represents an integer constant
type Econst_int struct {
	Value int64
	Typ   ctypes.Type
}

// Econst_float represents a floating-point constant
type Econst_float struct {
	Value float64
	Typ   ctypes.Type
}

// Econst_long represents a long integer constant
type Econst_long struct {
	Value int64
	Typ   ctypes.Type
}

// Econst_single represents a single-precision float constant
type Econst_single struct {
	Value float32
	Typ   ctypes.Type
}

// Evar represents a reference to a global or local variable (in memory)
type Evar struct {
	Name string
	Typ  ctypes.Type
}

// Etempvar represents a reference to a temporary variable (in register)
// Temporaries are introduced by SimplExpr transformation
type Etempvar struct {
	ID  int // unique identifier
	Typ ctypes.Type
}

// Ederef represents pointer dereference (*p)
type Ederef struct {
	Ptr Expr
	Typ ctypes.Type
}

// Eaddrof represents address-of operator (&x)
type Eaddrof struct {
	Arg Expr
	Typ ctypes.Type
}

// Eunop represents a unary operation
type Eunop struct {
	Op  UnaryOp
	Arg Expr
	Typ ctypes.Type
}

// Ebinop represents a binary operation
type Ebinop struct {
	Op    BinaryOp
	Left  Expr
	Right Expr
	Typ   ctypes.Type
}

// Ecast represents a type cast
type Ecast struct {
	Arg Expr
	Typ ctypes.Type
}

// Efield represents struct/union field access (s.f or p->f after desugaring)
type Efield struct {
	Arg       Expr
	FieldName string
	Typ       ctypes.Type
}

// Esizeof represents sizeof(type)
type Esizeof struct {
	ArgType ctypes.Type
	Typ     ctypes.Type // result type (usually unsigned int)
}

// Ealignof represents alignof(type)
type Ealignof struct {
	ArgType ctypes.Type
	Typ     ctypes.Type
}

// --- Statements ---

// Sskip represents an empty statement
type Sskip struct{}

// Sassign represents an assignment to a memory location: *lhs = rhs
type Sassign struct {
	LHS Expr // must be an l-value (Evar, Ederef, or Efield)
	RHS Expr
}

// Sset represents an assignment to a temporary: temp = expr
type Sset struct {
	TempID int
	RHS    Expr
}

// Scall represents a function call as a statement
// The result (if any) goes into a temporary
type Scall struct {
	Result *int // temporary ID for result, nil for void calls
	Func   Expr // function to call
	Args   []Expr
}

// Sbuiltin represents a call to a builtin function
type Sbuiltin struct {
	Result  *int // temporary ID for result, nil for void
	Builtin string
	Args    []Expr
}

// Ssequence represents a sequence of two statements
type Ssequence struct {
	First  Stmt
	Second Stmt
}

// Sifthenelse represents an if-then-else statement
type Sifthenelse struct {
	Cond Expr
	Then Stmt
	Else Stmt // Sskip for no else
}

// Sloop represents infinite loop: loop { body; continue; }
// break and continue target this loop
type Sloop struct {
	Body     Stmt
	Continue Stmt // executed before each iteration
}

// Sbreak represents breaking out of a loop
type Sbreak struct{}

// Scontinue represents continuing to next iteration
type Scontinue struct{}

// Sreturn represents returning from a function
type Sreturn struct {
	Value Expr // nil for void return
}

// Sswitch represents a switch statement
type Sswitch struct {
	Expr     Expr
	Cases    []SwitchCase
	Default  Stmt // default case body
	HasBreak bool // if false, fall-through semantics
}

// SwitchCase represents a case in a switch statement
type SwitchCase struct {
	Value int64 // case label value
	Body  Stmt
}

// Slabel represents a labeled statement
type Slabel struct {
	Label string
	Stmt  Stmt
}

// Sgoto represents a goto statement
type Sgoto struct {
	Label string
}

// --- Functions and Programs ---

// VarDecl represents a variable declaration
type VarDecl struct {
	Name string
	Type ctypes.Type
}

// Function represents a function definition in Clight
type Function struct {
	Name     string
	Return   ctypes.Type
	Params   []VarDecl // function parameters
	Locals   []VarDecl // local variables (in memory)
	Temps    []ctypes.Type // temporary variables (in registers)
	Body     Stmt
}

// Program represents a complete Clight program
type Program struct {
	Structs   []ctypes.Tstruct // struct type definitions
	Unions    []ctypes.Tunion  // union type definitions
	Globals   []VarDecl        // global variables
	Functions []Function
}

// --- Interface implementations ---

// Marker methods for Node interface
func (Econst_int) implClightNode()    {}
func (Econst_float) implClightNode()  {}
func (Econst_long) implClightNode()   {}
func (Econst_single) implClightNode() {}
func (Evar) implClightNode()          {}
func (Etempvar) implClightNode()      {}
func (Ederef) implClightNode()        {}
func (Eaddrof) implClightNode()       {}
func (Eunop) implClightNode()         {}
func (Ebinop) implClightNode()        {}
func (Ecast) implClightNode()         {}
func (Efield) implClightNode()        {}
func (Esizeof) implClightNode()       {}
func (Ealignof) implClightNode()      {}

func (Sskip) implClightNode()       {}
func (Sassign) implClightNode()     {}
func (Sset) implClightNode()        {}
func (Scall) implClightNode()       {}
func (Sbuiltin) implClightNode()    {}
func (Ssequence) implClightNode()   {}
func (Sifthenelse) implClightNode() {}
func (Sloop) implClightNode()       {}
func (Sbreak) implClightNode()      {}
func (Scontinue) implClightNode()   {}
func (Sreturn) implClightNode()     {}
func (Sswitch) implClightNode()     {}
func (Slabel) implClightNode()      {}
func (Sgoto) implClightNode()       {}

// Marker methods for Expr interface
func (Econst_int) implClightExpr()    {}
func (Econst_float) implClightExpr()  {}
func (Econst_long) implClightExpr()   {}
func (Econst_single) implClightExpr() {}
func (Evar) implClightExpr()          {}
func (Etempvar) implClightExpr()      {}
func (Ederef) implClightExpr()        {}
func (Eaddrof) implClightExpr()       {}
func (Eunop) implClightExpr()         {}
func (Ebinop) implClightExpr()        {}
func (Ecast) implClightExpr()         {}
func (Efield) implClightExpr()        {}
func (Esizeof) implClightExpr()       {}
func (Ealignof) implClightExpr()      {}

// Marker methods for Stmt interface
func (Sskip) implClightStmt()       {}
func (Sassign) implClightStmt()     {}
func (Sset) implClightStmt()        {}
func (Scall) implClightStmt()       {}
func (Sbuiltin) implClightStmt()    {}
func (Ssequence) implClightStmt()   {}
func (Sifthenelse) implClightStmt() {}
func (Sloop) implClightStmt()       {}
func (Sbreak) implClightStmt()      {}
func (Scontinue) implClightStmt()   {}
func (Sreturn) implClightStmt()     {}
func (Sswitch) implClightStmt()     {}
func (Slabel) implClightStmt()      {}
func (Sgoto) implClightStmt()       {}

// ExprType implementations - return the type of each expression
func (e Econst_int) ExprType() ctypes.Type    { return e.Typ }
func (e Econst_float) ExprType() ctypes.Type  { return e.Typ }
func (e Econst_long) ExprType() ctypes.Type   { return e.Typ }
func (e Econst_single) ExprType() ctypes.Type { return e.Typ }
func (e Evar) ExprType() ctypes.Type          { return e.Typ }
func (e Etempvar) ExprType() ctypes.Type      { return e.Typ }
func (e Ederef) ExprType() ctypes.Type        { return e.Typ }
func (e Eaddrof) ExprType() ctypes.Type       { return e.Typ }
func (e Eunop) ExprType() ctypes.Type         { return e.Typ }
func (e Ebinop) ExprType() ctypes.Type        { return e.Typ }
func (e Ecast) ExprType() ctypes.Type         { return e.Typ }
func (e Efield) ExprType() ctypes.Type        { return e.Typ }
func (e Esizeof) ExprType() ctypes.Type       { return e.Typ }
func (e Ealignof) ExprType() ctypes.Type      { return e.Typ }

// Seq creates a sequence of statements, flattening Sskip
func Seq(stmts ...Stmt) Stmt {
	var result Stmt = Sskip{}
	for _, s := range stmts {
		if _, ok := result.(Sskip); ok {
			result = s
		} else if _, ok := s.(Sskip); !ok {
			result = Ssequence{First: result, Second: s}
		}
	}
	return result
}
