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
	OpAddAssign // +=
	OpSubAssign // -=
	OpMulAssign // *=
	OpDivAssign // /=
	OpModAssign // %=
	OpAndAssign // &=
	OpOrAssign  // |=
	OpXorAssign // ^=
	OpShlAssign // <<=
	OpShrAssign // >>=
	OpComma     // ,
)

func (op BinaryOp) String() string {
	names := []string{"+", "-", "*", "/", "%", "<", "<=", ">", ">=", "==", "!=", "&&", "||", "&", "|", "^", "<<", ">>", "=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ","}
	if int(op) < len(names) {
		return names[op]
	}
	return "?"
}

// UnaryOp represents unary operators
type UnaryOp int

const (
	OpNeg      UnaryOp = iota // -
	OpNot                     // !
	OpBitNot                  // ~
	OpPreInc                  // ++x
	OpPreDec                  // --x
	OpPostInc                 // x++
	OpPostDec                 // x--
	OpAddrOf                  // &x
	OpDeref                   // *x
)

func (op UnaryOp) String() string {
	names := []string{"-", "!", "~", "++", "--", "++", "--", "&", "*"}
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

// Member represents member access: s.x or p->y
type Member struct {
	Expr    Expr
	Name    string
	IsArrow bool // true for ->, false for .
}

// SizeofExpr represents sizeof applied to an expression
type SizeofExpr struct {
	Expr Expr
}

// SizeofType represents sizeof applied to a type
type SizeofType struct {
	TypeName string
}

// Cast represents a type cast: (type)expr
type Cast struct {
	TypeName string
	Expr     Expr
}

// Return represents a return statement
type Return struct {
	Expr Expr // nil for bare return
}

// Computation represents an expression statement (expr;)
type Computation struct {
	Expr Expr
}

// If represents an if statement (with optional else)
type If struct {
	Cond Expr
	Then Stmt
	Else Stmt // nil if no else branch
}

// While represents a while loop
type While struct {
	Cond Expr
	Body Stmt
}

// DoWhile represents a do-while loop
type DoWhile struct {
	Body Stmt
	Cond Expr
}

// For represents a for loop
type For struct {
	Init     Expr   // may be nil (mutually exclusive with InitDecl)
	InitDecl []Decl // C99 for-loop declaration (mutually exclusive with Init)
	Cond     Expr   // may be nil (infinite loop)
	Step     Expr   // may be nil
	Body     Stmt
}

// Break represents a break statement
type Break struct{}

// Continue represents a continue statement
type Continue struct{}

// Switch represents a switch statement
type Switch struct {
	Expr  Expr
	Cases []SwitchCase
}

// SwitchCase represents a case or default in a switch
type SwitchCase struct {
	Expr  Expr  // nil for default case
	Stmts []Stmt
}

// Goto represents a goto statement
type Goto struct {
	Label string
}

// Label represents a label statement
type Label struct {
	Name string
	Stmt Stmt
}

// Block represents a compound statement (block)
type Block struct {
	Items []Stmt
}

// FunDef represents a function definition
type FunDef struct {
	ReturnType string
	Name       string
	Params     []Param
	Body       *Block
}

// Param represents a function parameter
type Param struct {
	TypeSpec string
	Name     string
}

// Decl represents a variable declaration (with optional initializer)
type Decl struct {
	TypeSpec    string
	Name        string
	ArrayDims   []Expr // array dimensions: nil for non-array, [nil] for int arr[], [expr] for int arr[n]
	Initializer Expr   // nil if no initializer
}

// DeclStmt represents a declaration statement (can have multiple declarators)
type DeclStmt struct {
	Decls []Decl
}

// StorageClass represents storage class specifiers
type StorageClass int

const (
	StorageNone StorageClass = iota
	StorageStatic
	StorageExtern
	StorageAuto
	StorageRegister
)

// TypeQualifier represents type qualifiers
type TypeQualifier int

const (
	QualNone TypeQualifier = iota
	QualConst
	QualVolatile
	QualRestrict
)

// TypedefDef represents a typedef declaration
type TypedefDef struct {
	TypeSpec string
	Name     string
}

// StructField represents a field in a struct definition
type StructField struct {
	TypeSpec string
	Name     string
}

// StructDef represents a struct type definition
type StructDef struct {
	Name   string // empty for anonymous structs
	Fields []StructField
}

// UnionDef represents a union type definition
type UnionDef struct {
	Name   string
	Fields []StructField
}

// EnumVal represents a single enumerator
type EnumVal struct {
	Name  string
	Value Expr // nil for auto-assigned values
}

// EnumDef represents an enum type definition
type EnumDef struct {
	Name   string
	Values []EnumVal
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

func (Member) implCabsNode() {}
func (Member) implCabsExpr() {}

func (SizeofExpr) implCabsNode() {}
func (SizeofExpr) implCabsExpr() {}

func (SizeofType) implCabsNode() {}
func (SizeofType) implCabsExpr() {}

func (Cast) implCabsNode() {}
func (Cast) implCabsExpr() {}

func (Return) implCabsNode() {}
func (Return) implCabsStmt() {}

func (Computation) implCabsNode() {}
func (Computation) implCabsStmt() {}

func (If) implCabsNode() {}
func (If) implCabsStmt() {}

func (While) implCabsNode() {}
func (While) implCabsStmt() {}

func (DoWhile) implCabsNode() {}
func (DoWhile) implCabsStmt() {}

func (For) implCabsNode() {}
func (For) implCabsStmt() {}

func (Break) implCabsNode() {}
func (Break) implCabsStmt() {}

func (Continue) implCabsNode() {}
func (Continue) implCabsStmt() {}

func (Switch) implCabsNode() {}
func (Switch) implCabsStmt() {}

func (Goto) implCabsNode() {}
func (Goto) implCabsStmt() {}

func (Label) implCabsNode() {}
func (Label) implCabsStmt() {}

func (Block) implCabsNode() {}
func (Block) implCabsStmt() {}

func (FunDef) implCabsNode()    {}
func (FunDef) implDefinition() {}

func (DeclStmt) implCabsNode() {}
func (DeclStmt) implCabsStmt() {}

func (TypedefDef) implCabsNode()    {}
func (TypedefDef) implDefinition() {}

func (StructDef) implCabsNode()    {}
func (StructDef) implDefinition() {}

func (UnionDef) implCabsNode()    {}
func (UnionDef) implDefinition() {}

func (EnumDef) implCabsNode()    {}
func (EnumDef) implDefinition() {}

// Program represents a complete translation unit (file)
type Program struct {
	Definitions []Definition
}

func (Program) implCabsNode() {}
