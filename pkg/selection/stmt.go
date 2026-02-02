// Package selection - Statement selection for instruction selection pass.
// This file transforms Cminor statements to CminorSel statements,
// selecting addressing modes for stores and conditions for branches.
package selection

import (
	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/cminorsel"
)

// SelectStmt transforms a Cminor statement to a CminorSel statement.
func (ctx *SelectionContext) SelectStmt(s cminor.Stmt) cminorsel.Stmt {
	switch stmt := s.(type) {
	case cminor.Sskip:
		return cminorsel.Sskip{}

	case cminor.Sassign:
		return ctx.selectAssign(stmt)

	case cminor.Sstore:
		return ctx.selectStore(stmt)

	case cminor.Scall:
		return ctx.selectCall(stmt)

	case cminor.Stailcall:
		return ctx.selectTailcall(stmt)

	case cminor.Sbuiltin:
		return ctx.selectBuiltin(stmt)

	case cminor.Sseq:
		return ctx.selectSeq(stmt)

	case cminor.Sifthenelse:
		return ctx.selectIfthenelse(stmt)

	case cminor.Sloop:
		return ctx.selectLoop(stmt)

	case cminor.Sblock:
		return ctx.selectBlock(stmt)

	case cminor.Sexit:
		return cminorsel.Sexit{N: stmt.N}

	case cminor.Sswitch:
		return ctx.selectSwitch(stmt)

	case cminor.Sreturn:
		return ctx.selectReturn(stmt)

	case cminor.Slabel:
		return ctx.selectLabel(stmt)

	case cminor.Sgoto:
		return cminorsel.Sgoto{Label: stmt.Label}

	default:
		return cminorsel.Sskip{}
	}
}

// selectAssign handles assignment to a local variable.
func (ctx *SelectionContext) selectAssign(s cminor.Sassign) cminorsel.Stmt {
	rhs := ctx.SelectExpr(s.RHS)
	return cminorsel.Sassign{
		Name: s.Name,
		RHS:  rhs,
	}
}

// selectStore handles memory store with addressing mode selection.
func (ctx *SelectionContext) selectStore(s cminor.Sstore) cminorsel.Stmt {
	// Select addressing mode for the address expression
	addrResult := SelectAddressing(s.Addr, ctx.Globals, ctx.StackVars)

	// Convert args using full selection
	selectedArgs := make([]cminorsel.Expr, len(addrResult.Args))
	for i, arg := range addrResult.Args {
		selectedArgs[i] = ctx.reSelectExpr(arg)
	}

	// Select the value expression
	value := ctx.SelectExpr(s.Value)

	return cminorsel.Sstore{
		Chunk: cminorsel.Chunk(s.Chunk),
		Mode:  addrResult.Mode,
		Args:  selectedArgs,
		Value: value,
	}
}

// selectCall handles function calls.
func (ctx *SelectionContext) selectCall(s cminor.Scall) cminorsel.Stmt {
	// Select function expression
	fn := ctx.SelectExpr(s.Func)

	// Select argument expressions
	args := make([]cminorsel.Expr, len(s.Args))
	for i, arg := range s.Args {
		args[i] = ctx.SelectExpr(arg)
	}

	// Convert signature if present
	var sig *cminorsel.Sig
	if s.Sig != nil {
		sig = &cminorsel.Sig{
			Args:   s.Sig.Args,
			Return: s.Sig.Return,
			VarArg: s.Sig.VarArg,
		}
	}

	return cminorsel.Scall{
		Result: s.Result,
		Sig:    sig,
		Func:   fn,
		Args:   args,
	}
}

// selectTailcall handles tail calls.
func (ctx *SelectionContext) selectTailcall(s cminor.Stailcall) cminorsel.Stmt {
	// Select function expression
	fn := ctx.SelectExpr(s.Func)

	// Select argument expressions
	args := make([]cminorsel.Expr, len(s.Args))
	for i, arg := range s.Args {
		args[i] = ctx.SelectExpr(arg)
	}

	// Convert signature if present
	var sig *cminorsel.Sig
	if s.Sig != nil {
		sig = &cminorsel.Sig{
			Args:   s.Sig.Args,
			Return: s.Sig.Return,
			VarArg: s.Sig.VarArg,
		}
	}

	return cminorsel.Stailcall{
		Sig:  sig,
		Func: fn,
		Args: args,
	}
}

// selectBuiltin handles builtin function calls.
func (ctx *SelectionContext) selectBuiltin(s cminor.Sbuiltin) cminorsel.Stmt {
	args := make([]cminorsel.Expr, len(s.Args))
	for i, arg := range s.Args {
		args[i] = ctx.SelectExpr(arg)
	}

	return cminorsel.Sbuiltin{
		Result:  s.Result,
		Builtin: s.Builtin,
		Args:    args,
	}
}

// selectSeq handles sequences of statements.
func (ctx *SelectionContext) selectSeq(s cminor.Sseq) cminorsel.Stmt {
	first := ctx.SelectStmt(s.First)
	second := ctx.SelectStmt(s.Second)
	return cminorsel.Sseq{
		First:  first,
		Second: second,
	}
}

// selectIfthenelse handles conditional statements.
func (ctx *SelectionContext) selectIfthenelse(s cminor.Sifthenelse) cminorsel.Stmt {
	// Select the condition
	cond := ctx.SelectCondition(s.Cond)

	// Select the branches
	thenBranch := ctx.SelectStmt(s.Then)
	elseBranch := ctx.SelectStmt(s.Else)

	return cminorsel.Sifthenelse{
		Cond: cond,
		Then: thenBranch,
		Else: elseBranch,
	}
}

// selectLoop handles loop statements.
func (ctx *SelectionContext) selectLoop(s cminor.Sloop) cminorsel.Stmt {
	body := ctx.SelectStmt(s.Body)
	return cminorsel.Sloop{
		Body: body,
	}
}

// selectBlock handles block statements.
func (ctx *SelectionContext) selectBlock(s cminor.Sblock) cminorsel.Stmt {
	body := ctx.SelectStmt(s.Body)
	return cminorsel.Sblock{
		Body: body,
	}
}

// selectSwitch handles switch statements.
func (ctx *SelectionContext) selectSwitch(s cminor.Sswitch) cminorsel.Stmt {
	// Select the switch expression
	expr := ctx.SelectExpr(s.Expr)

	// Select case bodies
	cases := make([]cminorsel.SwitchCase, len(s.Cases))
	for i, c := range s.Cases {
		cases[i] = cminorsel.SwitchCase{
			Value: c.Value,
			Body:  ctx.SelectStmt(c.Body),
		}
	}

	// Select default body
	def := ctx.SelectStmt(s.Default)

	return cminorsel.Sswitch{
		IsLong:  s.IsLong,
		Expr:    expr,
		Cases:   cases,
		Default: def,
	}
}

// selectReturn handles return statements.
func (ctx *SelectionContext) selectReturn(s cminor.Sreturn) cminorsel.Stmt {
	var value cminorsel.Expr
	if s.Value != nil {
		value = ctx.SelectExpr(s.Value)
	}
	return cminorsel.Sreturn{
		Value: value,
	}
}

// selectLabel handles labeled statements.
func (ctx *SelectionContext) selectLabel(s cminor.Slabel) cminorsel.Stmt {
	body := ctx.SelectStmt(s.Body)
	return cminorsel.Slabel{
		Label: s.Label,
		Body:  body,
	}
}

// SelectFunction transforms a Cminor function to a CminorSel function.
func (ctx *SelectionContext) SelectFunction(f cminor.Function) cminorsel.Function {
	// Select the function body
	body := ctx.SelectStmt(f.Body)

	// Convert signature
	sig := cminorsel.Sig{
		Args:   f.Sig.Args,
		Return: f.Sig.Return,
		VarArg: f.Sig.VarArg,
	}

	return cminorsel.Function{
		Name:       f.Name,
		Sig:        sig,
		Params:     f.Params,
		Vars:       f.Vars,
		Stackspace: f.Stackspace,
		Body:       body,
	}
}

// SelectProgram transforms a Cminor program to a CminorSel program.
func (ctx *SelectionContext) SelectProgram(p cminor.Program) cminorsel.Program {
	// Build globals set from program (includes global variables and function names)
	globals := make(map[string]bool)
	for _, g := range p.Globals {
		globals[g.Name] = true
	}
	// Function names are also global symbols - this allows direct calls
	// to use FunSymbol instead of indirect calls via FunReg
	for _, f := range p.Functions {
		globals[f.Name] = true
	}

	// Collect external function references (functions called but not defined)
	// This is needed for direct calls to external functions like printf
	externals := collectExternalFunctions(p, globals)
	for name := range externals {
		globals[name] = true
	}

	ctx.Globals = globals

	// Transform global variables
	globVars := make([]cminorsel.GlobVar, len(p.Globals))
	for i, g := range p.Globals {
		globVars[i] = cminorsel.GlobVar{
			Name:     g.Name,
			Size:     g.Size,
			Init:     g.Init,
			ReadOnly: g.ReadOnly,
		}
	}

	// Transform functions
	funcs := make([]cminorsel.Function, len(p.Functions))
	for i, f := range p.Functions {
		funcs[i] = ctx.SelectFunction(f)
	}

	return cminorsel.Program{
		Globals:   globVars,
		Functions: funcs,
	}
}

// collectExternalFunctions scans the program for function calls to names that
// are not defined in the program. These are external functions (like printf).
func collectExternalFunctions(p cminor.Program, defined map[string]bool) map[string]bool {
	externals := make(map[string]bool)

	for _, f := range p.Functions {
		collectExternalFunctionsInStmt(f.Body, defined, externals)
	}

	return externals
}

// collectExternalFunctionsInStmt recursively scans a statement for external function calls.
func collectExternalFunctionsInStmt(s cminor.Stmt, defined map[string]bool, externals map[string]bool) {
	switch stmt := s.(type) {
	case cminor.Scall:
		// Check if the function is an Evar reference to an undefined name
		if evar, ok := stmt.Func.(cminor.Evar); ok {
			if !defined[evar.Name] {
				externals[evar.Name] = true
			}
		}
	case cminor.Stailcall:
		if evar, ok := stmt.Func.(cminor.Evar); ok {
			if !defined[evar.Name] {
				externals[evar.Name] = true
			}
		}
	case cminor.Sseq:
		collectExternalFunctionsInStmt(stmt.First, defined, externals)
		collectExternalFunctionsInStmt(stmt.Second, defined, externals)
	case cminor.Sifthenelse:
		collectExternalFunctionsInStmt(stmt.Then, defined, externals)
		collectExternalFunctionsInStmt(stmt.Else, defined, externals)
	case cminor.Sloop:
		collectExternalFunctionsInStmt(stmt.Body, defined, externals)
	case cminor.Sblock:
		collectExternalFunctionsInStmt(stmt.Body, defined, externals)
	case cminor.Sswitch:
		for _, c := range stmt.Cases {
			collectExternalFunctionsInStmt(c.Body, defined, externals)
		}
		collectExternalFunctionsInStmt(stmt.Default, defined, externals)
	}
}
