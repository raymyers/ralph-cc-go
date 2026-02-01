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

// Constant represents an integer constant
type Constant struct {
	Value int64
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

func (Return) implCabsNode() {}
func (Return) implCabsStmt() {}

func (Block) implCabsNode() {}
func (Block) implCabsStmt() {}

func (FunDef) implCabsNode()    {}
func (FunDef) implDefinition() {}
