// Package cminorgen implements the Cminorgen pass: Csharpminor → Cminor
// This file handles the main transformation logic.
package cminorgen

import (
	"fmt"

	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
)

// Transformer handles the Csharpminor → Cminor transformation.
type Transformer struct {
	varEnv   *VarEnv
	tempMap  map[int]string // Csharpminor temp ID → Cminor variable name
	globals  map[string]bool
	nextTemp int // Counter for generating unique temp names
}

// NewTransformer creates a new transformer with the given variable environment.
func NewTransformer(env *VarEnv, globals map[string]bool) *Transformer {
	return &Transformer{
		varEnv:   env,
		tempMap:  make(map[int]string),
		globals:  globals,
		nextTemp: 0,
	}
}

// getTempName returns the Cminor variable name for a Csharpminor temp ID.
// Creates a new name if needed.
func (t *Transformer) getTempName(id int) string {
	if name, ok := t.tempMap[id]; ok {
		return name
	}
	name := fmt.Sprintf("_t%d", id)
	t.tempMap[id] = name
	return name
}

// TransformExpr translates a Csharpminor expression to a Cminor expression.
func (t *Transformer) TransformExpr(e csharpminor.Expr) cminor.Expr {
	switch expr := e.(type) {
	case csharpminor.Evar:
		// Global variable reference
		return cminor.Evar{Name: expr.Name}

	case csharpminor.Etempvar:
		// Temporary → named variable in Cminor
		name := t.getTempName(expr.ID)
		return t.varEnv.TransformVarRead(name)

	case csharpminor.Eaddrof:
		// Address-of a global or local
		if t.varEnv.IsStack(expr.Name) {
			return t.varEnv.TransformAddrOf(expr.Name)
		}
		// For globals, keep as-is (handled differently in Cminor)
		return cminor.Evar{Name: expr.Name}

	case csharpminor.Econst:
		return t.transformConst(expr.Const)

	case csharpminor.Eunop:
		arg := t.TransformExpr(expr.Arg)
		return cminor.Eunop{Op: cminor.UnaryOp(expr.Op), Arg: arg}

	case csharpminor.Ebinop:
		left := t.TransformExpr(expr.Left)
		right := t.TransformExpr(expr.Right)
		return cminor.Ebinop{Op: cminor.BinaryOp(expr.Op), Left: left, Right: right}

	case csharpminor.Ecmp:
		left := t.TransformExpr(expr.Left)
		right := t.TransformExpr(expr.Right)
		return cminor.Ecmp{
			Op:    cminor.BinaryOp(expr.Op),
			Cmp:   cminor.Comparison(expr.Cmp),
			Left:  left,
			Right: right,
		}

	case csharpminor.Eload:
		addr := t.TransformExpr(expr.Addr)
		return cminor.Eload{Chunk: cminor.Chunk(expr.Chunk), Addr: addr}
	}
	panic(fmt.Sprintf("unhandled expression type: %T", e))
}

// transformConst translates a Csharpminor constant to a Cminor expression.
func (t *Transformer) transformConst(c csharpminor.Constant) cminor.Expr {
	switch cnst := c.(type) {
	case csharpminor.Ointconst:
		return cminor.Econst{Const: cminor.Ointconst{Value: cnst.Value}}
	case csharpminor.Ofloatconst:
		return cminor.Econst{Const: cminor.Ofloatconst{Value: cnst.Value}}
	case csharpminor.Olongconst:
		return cminor.Econst{Const: cminor.Olongconst{Value: cnst.Value}}
	case csharpminor.Osingleconst:
		return cminor.Econst{Const: cminor.Osingleconst{Value: cnst.Value}}
	}
	panic(fmt.Sprintf("unhandled constant type: %T", c))
}

// TransformStmt translates a Csharpminor statement to a Cminor statement.
func (t *Transformer) TransformStmt(s csharpminor.Stmt) cminor.Stmt {
	switch stmt := s.(type) {
	case csharpminor.Sskip:
		return cminor.Sskip{}

	case csharpminor.Sset:
		// Temp assignment → variable assignment in Cminor
		name := t.getTempName(stmt.TempID)
		rhs := t.TransformExpr(stmt.RHS)
		return t.varEnv.TransformVarWrite(name, rhs)

	case csharpminor.Sstore:
		addr := t.TransformExpr(stmt.Addr)
		value := t.TransformExpr(stmt.Value)
		return cminor.Sstore{
			Chunk: cminor.Chunk(stmt.Chunk),
			Addr:  addr,
			Value: value,
		}

	case csharpminor.Scall:
		return t.transformCall(stmt)

	case csharpminor.Stailcall:
		return t.transformTailcall(stmt)

	case csharpminor.Sbuiltin:
		return t.transformBuiltin(stmt)

	case csharpminor.Sseq:
		first := t.TransformStmt(stmt.First)
		second := t.TransformStmt(stmt.Second)
		return cminor.Sseq{First: first, Second: second}

	case csharpminor.Sifthenelse:
		cond := t.TransformExpr(stmt.Cond)
		thenStmt := t.TransformStmt(stmt.Then)
		elseStmt := t.TransformStmt(stmt.Else)
		return cminor.Sifthenelse{Cond: cond, Then: thenStmt, Else: elseStmt}

	case csharpminor.Sloop:
		body := t.TransformStmt(stmt.Body)
		return cminor.Sloop{Body: body}

	case csharpminor.Sblock:
		body := t.TransformStmt(stmt.Body)
		return cminor.Sblock{Body: body}

	case csharpminor.Sexit:
		return cminor.Sexit{N: stmt.N}

	case csharpminor.Sswitch:
		return t.transformSwitch(stmt)

	case csharpminor.Sreturn:
		var value cminor.Expr
		if stmt.Value != nil {
			value = t.TransformExpr(stmt.Value)
		}
		return cminor.Sreturn{Value: value}

	case csharpminor.Slabel:
		body := t.TransformStmt(stmt.Body)
		return cminor.Slabel{Label: stmt.Label, Body: body}

	case csharpminor.Sgoto:
		return cminor.Sgoto{Label: stmt.Label}
	}
	panic(fmt.Sprintf("unhandled statement type: %T", s))
}

// transformCall translates a function call.
func (t *Transformer) transformCall(s csharpminor.Scall) cminor.Stmt {
	fn := t.TransformExpr(s.Func)
	args := make([]cminor.Expr, len(s.Args))
	for i, arg := range s.Args {
		args[i] = t.TransformExpr(arg)
	}

	var result *string
	if s.Result != nil {
		name := t.getTempName(*s.Result)
		result = &name
	}

	var sig *cminor.Sig
	if s.Sig != nil {
		sig = t.transformSig(s.Sig)
	}

	return cminor.Scall{
		Result: result,
		Sig:    sig,
		Func:   fn,
		Args:   args,
	}
}

// transformTailcall translates a tail call.
func (t *Transformer) transformTailcall(s csharpminor.Stailcall) cminor.Stmt {
	fn := t.TransformExpr(s.Func)
	args := make([]cminor.Expr, len(s.Args))
	for i, arg := range s.Args {
		args[i] = t.TransformExpr(arg)
	}

	var sig *cminor.Sig
	if s.Sig != nil {
		sig = t.transformSig(s.Sig)
	}

	return cminor.Stailcall{
		Sig:  sig,
		Func: fn,
		Args: args,
	}
}

// transformBuiltin translates a builtin call.
func (t *Transformer) transformBuiltin(s csharpminor.Sbuiltin) cminor.Stmt {
	args := make([]cminor.Expr, len(s.Args))
	for i, arg := range s.Args {
		args[i] = t.TransformExpr(arg)
	}

	var result *string
	if s.Result != nil {
		name := t.getTempName(*s.Result)
		result = &name
	}

	return cminor.Sbuiltin{
		Result:  result,
		Builtin: s.Builtin,
		Args:    args,
	}
}

// transformSwitch translates a switch statement.
// This is where switch simplification from switch.go would be applied if needed.
func (t *Transformer) transformSwitch(s csharpminor.Sswitch) cminor.Stmt {
	expr := t.TransformExpr(s.Expr)

	cases := make([]cminor.SwitchCase, len(s.Cases))
	for i, c := range s.Cases {
		cases[i] = cminor.SwitchCase{
			Value: c.Value,
			Body:  t.TransformStmt(c.Body),
		}
	}

	defaultStmt := t.TransformStmt(s.Default)

	// Create the basic Cminor switch
	sw := cminor.Sswitch{
		IsLong:  s.IsLong,
		Expr:    expr,
		Cases:   cases,
		Default: defaultStmt,
	}

	// For now, return the switch directly. Switch simplification
	// (converting to if-cascades or jump tables) would happen here
	// if desired, using the functions in switch.go.
	return sw
}

// transformSig translates a function signature.
func (t *Transformer) transformSig(s *csharpminor.Sig) *cminor.Sig {
	sig := &cminor.Sig{
		VarArg: s.VarArg,
	}
	// Convert types to string descriptors
	for _, arg := range s.Args {
		sig.Args = append(sig.Args, typeDescriptor(arg))
	}
	sig.Return = typeDescriptor(s.Return)
	return sig
}

// typeDescriptor returns a string descriptor for a type.
// This is a simplified version - a full implementation would handle all types.
func typeDescriptor(t interface{}) string {
	if t == nil {
		return "void"
	}
	return fmt.Sprintf("%T", t)
}

// TransformFunction translates a Csharpminor function to a Cminor function.
func TransformFunction(fn *csharpminor.Function, globals map[string]bool) cminor.Function {
	// Classify variables and compute stack layout
	env := ClassifyVariables(fn.Locals, fn.Body)

	// Create transformer
	tr := NewTransformer(env, globals)

	// Pre-populate temp map with temps from the function
	// Temps in Csharpminor become local variables in Cminor
	for i := range fn.Temps {
		tr.getTempName(i)
	}

	// Transform body
	body := tr.TransformStmt(fn.Body)

	// Build signature
	sig := cminor.Sig{
		VarArg: fn.Sig.VarArg,
	}
	for _, arg := range fn.Sig.Args {
		sig.Args = append(sig.Args, typeDescriptor(arg))
	}
	sig.Return = typeDescriptor(fn.Sig.Return)

	// Collect all variable names (temps + register locals)
	var vars []string

	// Add temps
	for _, name := range tr.tempMap {
		vars = append(vars, name)
	}

	// Add register-allocated locals
	vars = append(vars, env.RegisterVars()...)

	return cminor.Function{
		Name:       fn.Name,
		Sig:        sig,
		Params:     fn.Params,
		Vars:       vars,
		Stackspace: env.StackSize,
		Body:       body,
	}
}

// TransformProgram translates a complete Csharpminor program to Cminor.
func TransformProgram(prog *csharpminor.Program) *cminor.Program {
	result := &cminor.Program{}

	// Build global variable set
	globals := make(map[string]bool)
	for _, g := range prog.Globals {
		globals[g.Name] = true
	}

	// Translate global variables
	for _, g := range prog.Globals {
		result.Globals = append(result.Globals, cminor.GlobVar{
			Name: g.Name,
			Size: g.Size,
		})
	}

	// Translate functions
	for _, fn := range prog.Functions {
		cminorFn := TransformFunction(&fn, globals)
		result.Functions = append(result.Functions, cminorFn)
	}

	return result
}
