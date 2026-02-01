// Package cabs defines the abstract syntax tree for C, mirroring CompCert's Cabs.v
package cabs

// Node is the base interface for all AST nodes
type Node interface {
	implCabsNode()
}

// Expr is the interface for all expression nodes
type Expr interface {
	Node
	implCabsExpr()
}

// Stmt is the interface for all statement nodes
type Stmt interface {
	Node
	implCabsStmt()
}

// Definition is the interface for top-level definitions
type Definition interface {
	Node
	implDefinition()
}

// BinaryOp represents binary operators
type BinaryOp int

const (
	OpAdd BinaryOp = iota
	OpSub
	OpMul
	OpDiv
	OpMod
	OpLt
	OpLe
	OpGt
	OpGe
	OpEq
	OpNe
	OpAnd // &&
	OpOr  // ||
	OpBitAnd
	OpBitOr
	OpBitXor
	OpShl // <<
	OpShr // >>
	OpAssign
)

func (op BinaryOp) String() string {
	names := []string{"+", "-", "*", "/", "%", "<", "<=", ">", ">=", "==", "!=", "&&", "||", "&", "|", "^", "<<", ">>", "="}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

// UnaryOp represents unary operators
type UnaryOp int

const (
	OpNeg    UnaryOp = iota // -
	OpNot                   // !
	OpBitNot                // ~
)

func (op UnaryOp) String() string {
	names := []string{"-", "!", "~"}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

// Constant represents an integer constant
type Constant struct {
	Value int64
}

// Variable represents an identifier expression
type Variable struct {
	Name string
}

// Unary represents a unary expression
type Unary struct {
	Op   UnaryOp
	Expr Expr
}

// Binary represents a binary expression
type Binary struct {
	Op    BinaryOp
	Left  Expr
	Right Expr
}

// Paren represents a parenthesized expression
type Paren struct {
	Expr Expr
}

// Conditional represents the ternary operator: cond ? then : else
type Conditional struct {
	Cond Expr
	Then Expr
	Else Expr
}

// Call represents a function call
type Call struct {
	Func Expr
	Args []Expr
}

// Index represents array subscript access: arr[idx]
type Index struct {
	Array Expr
	Index Expr
}

// Return represents a return statement
type Return struct {
	Expr Expr // nil for bare return
}

// Block represents a compound statement (block)
type Block struct {
	Items []Stmt
}

// FunDef represents a function definition
type FunDef struct {
	ReturnType string
	Name       string
	Body       *Block
}

// Marker methods for interface implementation
func (Constant) implCabsNode() {}
func (Constant) implCabsExpr() {}

func (Variable) implCabsNode() {}
func (Variable) implCabsExpr() {}

func (Unary) implCabsNode() {}
func (Unary) implCabsExpr() {}

func (Binary) implCabsNode() {}
func (Binary) implCabsExpr() {}

func (Paren) implCabsNode() {}
func (Paren) implCabsExpr() {}

func (Conditional) implCabsNode() {}
func (Conditional) implCabsExpr() {}

func (Call) implCabsNode() {}
func (Call) implCabsExpr() {}

func (Index) implCabsNode() {}
func (Index) implCabsExpr() {}

func (Return) implCabsNode() {}
func (Return) implCabsStmt() {}

func (Block) implCabsNode() {}
func (Block) implCabsStmt() {}

func (FunDef) implCabsNode()    {}
func (FunDef) implDefinition() {}
