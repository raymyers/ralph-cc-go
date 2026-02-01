// Package simpllocals transforms Clight to optimize local variable handling.
// This mirrors CompCert's SimplLocals.v transformation.
//
// The key optimization is turning non-addressable scalar locals into temporaries.
// A local variable can be turned into a temporary if:
// 1. It is a scalar type (not struct/union/array)
// 2. Its address is never taken (never used with &)
package simpllocals

import (
	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

// Transformer converts Clight locals to temporaries where possible.
type Transformer struct {
	addressTaken map[string]bool   // set of variables whose address is taken
	varToTemp    map[string]int    // mapping from var name to temp ID
	nextTempID   int               // counter for new temps
	temps        []ctypes.Type     // types of generated temps
}

// New creates a new SimplLocals transformer.
func New() *Transformer {
	return &Transformer{
		addressTaken: make(map[string]bool),
		varToTemp:    make(map[string]int),
		nextTempID:   1,
		temps:        nil,
	}
}

// Reset resets the transformer state for a new function.
func (t *Transformer) Reset() {
	t.addressTaken = make(map[string]bool)
	t.varToTemp = make(map[string]int)
	t.nextTempID = 1
	t.temps = nil
}

// SetNextTempID sets the starting temp ID (to continue from SimplExpr temps).
func (t *Transformer) SetNextTempID(id int) {
	t.nextTempID = id
}

// TempTypes returns the types of all temporaries generated during transformation.
func (t *Transformer) TempTypes() []ctypes.Type {
	return t.temps
}

// VarToTemp returns the mapping from variable names to temp IDs.
func (t *Transformer) VarToTemp() map[string]int {
	return t.varToTemp
}

// IsAddressTaken checks if a variable's address is taken.
func (t *Transformer) IsAddressTaken(name string) bool {
	return t.addressTaken[name]
}

// AnalyzeAddressTaken scans a Cabs expression to find address-taken variables.
func (t *Transformer) AnalyzeAddressTaken(e cabs.Expr) {
	switch expr := e.(type) {
	case cabs.Unary:
		if expr.Op == cabs.OpAddrOf {
			// Mark the variable as address-taken
			if v, ok := expr.Expr.(cabs.Variable); ok {
				t.addressTaken[v.Name] = true
			}
		}
		t.AnalyzeAddressTaken(expr.Expr)

	case cabs.Binary:
		t.AnalyzeAddressTaken(expr.Left)
		t.AnalyzeAddressTaken(expr.Right)

	case cabs.Paren:
		t.AnalyzeAddressTaken(expr.Expr)

	case cabs.Conditional:
		t.AnalyzeAddressTaken(expr.Cond)
		t.AnalyzeAddressTaken(expr.Then)
		t.AnalyzeAddressTaken(expr.Else)

	case cabs.Call:
		t.AnalyzeAddressTaken(expr.Func)
		for _, arg := range expr.Args {
			t.AnalyzeAddressTaken(arg)
		}

	case cabs.Index:
		t.AnalyzeAddressTaken(expr.Array)
		t.AnalyzeAddressTaken(expr.Index)

	case cabs.Member:
		t.AnalyzeAddressTaken(expr.Expr)

	case cabs.Cast:
		t.AnalyzeAddressTaken(expr.Expr)

	case cabs.SizeofExpr:
		// sizeof doesn't evaluate, but we still scan for consistency
		t.AnalyzeAddressTaken(expr.Expr)
	}
}

// AnalyzeStmt scans a Cabs statement for address-taken variables.
func (t *Transformer) AnalyzeStmt(s cabs.Stmt) {
	switch stmt := s.(type) {
	case cabs.Return:
		if stmt.Expr != nil {
			t.AnalyzeAddressTaken(stmt.Expr)
		}

	case cabs.Computation:
		t.AnalyzeAddressTaken(stmt.Expr)

	case cabs.If:
		t.AnalyzeAddressTaken(stmt.Cond)
		t.AnalyzeStmt(stmt.Then)
		if stmt.Else != nil {
			t.AnalyzeStmt(stmt.Else)
		}

	case cabs.While:
		t.AnalyzeAddressTaken(stmt.Cond)
		t.AnalyzeStmt(stmt.Body)

	case cabs.DoWhile:
		t.AnalyzeStmt(stmt.Body)
		t.AnalyzeAddressTaken(stmt.Cond)

	case cabs.For:
		if stmt.Init != nil {
			t.AnalyzeAddressTaken(stmt.Init)
		}
		if stmt.Cond != nil {
			t.AnalyzeAddressTaken(stmt.Cond)
		}
		if stmt.Step != nil {
			t.AnalyzeAddressTaken(stmt.Step)
		}
		t.AnalyzeStmt(stmt.Body)

	case cabs.Switch:
		t.AnalyzeAddressTaken(stmt.Expr)
		for _, c := range stmt.Cases {
			for _, s := range c.Stmts {
				t.AnalyzeStmt(s)
			}
		}

	case cabs.Label:
		t.AnalyzeStmt(stmt.Stmt)

	case cabs.Block:
		for _, item := range stmt.Items {
			t.AnalyzeStmt(item)
		}

	case cabs.DeclStmt:
		for _, decl := range stmt.Decls {
			if decl.Initializer != nil {
				t.AnalyzeAddressTaken(decl.Initializer)
			}
		}
	}
}

// AnalyzeFunction scans an entire function for address-taken variables.
func (t *Transformer) AnalyzeFunction(fn *cabs.FunDef) {
	if fn.Body != nil {
		t.AnalyzeStmt(fn.Body)
	}
}

// IsScalarType checks if a type is scalar (can be held in a temp).
func IsScalarType(typ ctypes.Type) bool {
	switch typ.(type) {
	case ctypes.Tint, ctypes.Tlong, ctypes.Tfloat, ctypes.Tpointer:
		return true
	default:
		return false
	}
}

// CanPromoteToTemp checks if a local variable can be promoted to a temporary.
func (t *Transformer) CanPromoteToTemp(name string, typ ctypes.Type) bool {
	// Cannot promote if address is taken
	if t.addressTaken[name] {
		return false
	}
	// Can only promote scalar types
	return IsScalarType(typ)
}

// PromoteLocal promotes a local variable to a temporary.
// Returns the temp ID assigned to this variable.
func (t *Transformer) PromoteLocal(name string, typ ctypes.Type) int {
	if !t.CanPromoteToTemp(name, typ) {
		return -1 // cannot promote
	}

	// Check if already promoted
	if id, ok := t.varToTemp[name]; ok {
		return id
	}

	// Allocate a new temp
	id := t.nextTempID
	t.nextTempID++
	t.temps = append(t.temps, typ)
	t.varToTemp[name] = id
	return id
}

// TransformExpr transforms a Clight expression, replacing promoted locals with temps.
func (t *Transformer) TransformExpr(e clight.Expr) clight.Expr {
	switch expr := e.(type) {
	case clight.Evar:
		// Check if this variable was promoted to a temp
		if tempID, ok := t.varToTemp[expr.Name]; ok {
			return clight.Etempvar{ID: tempID, Typ: expr.Typ}
		}
		return expr

	case clight.Ederef:
		return clight.Ederef{
			Ptr: t.TransformExpr(expr.Ptr),
			Typ: expr.Typ,
		}

	case clight.Eaddrof:
		return clight.Eaddrof{
			Arg: t.TransformExpr(expr.Arg),
			Typ: expr.Typ,
		}

	case clight.Eunop:
		return clight.Eunop{
			Op:  expr.Op,
			Arg: t.TransformExpr(expr.Arg),
			Typ: expr.Typ,
		}

	case clight.Ebinop:
		return clight.Ebinop{
			Op:    expr.Op,
			Left:  t.TransformExpr(expr.Left),
			Right: t.TransformExpr(expr.Right),
			Typ:   expr.Typ,
		}

	case clight.Ecast:
		return clight.Ecast{
			Arg: t.TransformExpr(expr.Arg),
			Typ: expr.Typ,
		}

	case clight.Efield:
		return clight.Efield{
			Arg:       t.TransformExpr(expr.Arg),
			FieldName: expr.FieldName,
			Typ:       expr.Typ,
		}

	default:
		return e
	}
}

// TransformStmt transforms a Clight statement, replacing promoted locals.
func (t *Transformer) TransformStmt(s clight.Stmt) clight.Stmt {
	switch stmt := s.(type) {
	case clight.Sassign:
		lhs := t.TransformExpr(stmt.LHS)
		rhs := t.TransformExpr(stmt.RHS)

		// If LHS is now a tempvar, convert to Sset
		if tv, ok := lhs.(clight.Etempvar); ok {
			return clight.Sset{TempID: tv.ID, RHS: rhs}
		}
		return clight.Sassign{LHS: lhs, RHS: rhs}

	case clight.Sset:
		return clight.Sset{
			TempID: stmt.TempID,
			RHS:    t.TransformExpr(stmt.RHS),
		}

	case clight.Scall:
		newArgs := make([]clight.Expr, len(stmt.Args))
		for i, arg := range stmt.Args {
			newArgs[i] = t.TransformExpr(arg)
		}
		return clight.Scall{
			Result: stmt.Result,
			Func:   t.TransformExpr(stmt.Func),
			Args:   newArgs,
		}

	case clight.Sbuiltin:
		newArgs := make([]clight.Expr, len(stmt.Args))
		for i, arg := range stmt.Args {
			newArgs[i] = t.TransformExpr(arg)
		}
		return clight.Sbuiltin{
			Result:  stmt.Result,
			Builtin: stmt.Builtin,
			Args:    newArgs,
		}

	case clight.Ssequence:
		return clight.Ssequence{
			First:  t.TransformStmt(stmt.First),
			Second: t.TransformStmt(stmt.Second),
		}

	case clight.Sifthenelse:
		return clight.Sifthenelse{
			Cond: t.TransformExpr(stmt.Cond),
			Then: t.TransformStmt(stmt.Then),
			Else: t.TransformStmt(stmt.Else),
		}

	case clight.Sloop:
		return clight.Sloop{
			Body:     t.TransformStmt(stmt.Body),
			Continue: t.TransformStmt(stmt.Continue),
		}

	case clight.Sreturn:
		if stmt.Value != nil {
			return clight.Sreturn{Value: t.TransformExpr(stmt.Value)}
		}
		return stmt

	case clight.Sswitch:
		newCases := make([]clight.SwitchCase, len(stmt.Cases))
		for i, c := range stmt.Cases {
			newCases[i] = clight.SwitchCase{
				Value: c.Value,
				Body:  t.TransformStmt(c.Body),
			}
		}
		return clight.Sswitch{
			Expr:     t.TransformExpr(stmt.Expr),
			Cases:    newCases,
			Default:  t.TransformStmt(stmt.Default),
			HasBreak: stmt.HasBreak,
		}

	case clight.Slabel:
		return clight.Slabel{
			Label: stmt.Label,
			Stmt:  t.TransformStmt(stmt.Stmt),
		}

	default:
		return s
	}
}

// LocalInfo holds information about a local variable during analysis.
type LocalInfo struct {
	Name         string
	Type         ctypes.Type
	AddressTaken bool
	Promoted     bool
	TempID       int
}

// AnalyzeLocals analyzes a list of local declarations and determines
// which can be promoted to temporaries.
func (t *Transformer) AnalyzeLocals(decls []clight.VarDecl) []LocalInfo {
	result := make([]LocalInfo, len(decls))
	for i, decl := range decls {
		info := LocalInfo{
			Name:         decl.Name,
			Type:         decl.Type,
			AddressTaken: t.addressTaken[decl.Name],
			Promoted:     false,
			TempID:       -1,
		}

		if t.CanPromoteToTemp(decl.Name, decl.Type) {
			info.Promoted = true
			info.TempID = t.PromoteLocal(decl.Name, decl.Type)
		}

		result[i] = info
	}
	return result
}

// FilterUnpromotedLocals returns only the locals that were not promoted.
func FilterUnpromotedLocals(infos []LocalInfo) []clight.VarDecl {
	var result []clight.VarDecl
	for _, info := range infos {
		if !info.Promoted {
			result = append(result, clight.VarDecl{
				Name: info.Name,
				Type: info.Type,
			})
		}
	}
	return result
}
