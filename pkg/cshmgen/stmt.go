// Package cshmgen implements the Cshmgen pass: Clight → Csharpminor
// This file handles statement translation.
package cshmgen

import (
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

// StmtTranslator translates Clight statements to Csharpminor statements.
// It tracks loop nesting to properly translate break/continue as Sexit with depth.
type StmtTranslator struct {
	exprTr     *ExprTranslator
	loopDepth  int // current loop nesting depth
	blockDepth int // current block nesting depth (for break/continue)
	params     map[string]bool // function parameter names
	paramTemps map[string]int  // parameter name -> temp ID for modified params
	nextTempID int             // next available temp ID for param copies
}

// NewStmtTranslator creates a new statement translator.
func NewStmtTranslator(exprTr *ExprTranslator) *StmtTranslator {
	return &StmtTranslator{
		exprTr:     exprTr,
		loopDepth:  0,
		blockDepth: 0,
		params:     make(map[string]bool),
		paramTemps: make(map[string]int),
		nextTempID: 0,
	}
}

// SetParams sets the function parameter names so parameter assignments can be handled correctly.
func (t *StmtTranslator) SetParams(params []string) {
	for _, p := range params {
		t.params[p] = true
	}
}

// SetNextTempID sets the next available temp ID (should be after any temps from simplexpr).
func (t *StmtTranslator) SetNextTempID(id int) {
	t.nextTempID = id
}

// GetParamTemps returns the mapping of modified parameters to their temp IDs.
func (t *StmtTranslator) GetParamTemps() map[string]int {
	return t.paramTemps
}

// isParam returns true if the given name is a function parameter.
func (t *StmtTranslator) isParam(name string) bool {
	return t.params[name]
}

// getOrCreateParamTemp returns the temp ID for a parameter, creating one if needed.
func (t *StmtTranslator) getOrCreateParamTemp(name string) int {
	if id, ok := t.paramTemps[name]; ok {
		return id
	}
	id := t.nextTempID
	t.nextTempID++
	t.paramTemps[name] = id
	return id
}

// TranslateStmt translates a Clight statement to a Csharpminor statement.
func (t *StmtTranslator) TranslateStmt(s clight.Stmt) csharpminor.Stmt {
	switch stmt := s.(type) {
	case clight.Sskip:
		return csharpminor.Sskip{}

	case clight.Sassign:
		return t.translateAssign(stmt)

	case clight.Sset:
		return t.translateSet(stmt)

	case clight.Scall:
		return t.translateCall(stmt)

	case clight.Sbuiltin:
		return t.translateBuiltin(stmt)

	case clight.Ssequence:
		return t.translateSequence(stmt)

	case clight.Sifthenelse:
		return t.translateIf(stmt)

	case clight.Sloop:
		return t.translateLoop(stmt)

	case clight.Sbreak:
		return t.translateBreak()

	case clight.Scontinue:
		return t.translateContinue()

	case clight.Sreturn:
		return t.translateReturn(stmt)

	case clight.Sswitch:
		return t.translateSwitch(stmt)

	case clight.Slabel:
		return t.translateLabel(stmt)

	case clight.Sgoto:
		return t.translateGoto(stmt)
	}
	panic("unhandled statement type")
}

// translateAssign translates an assignment to a memory location.
// Clight: lhs = rhs (where lhs is Evar, Ederef, or Efield)
// Csharpminor: Sstore(chunk, addr, value) for locals, Sset for parameters
func (t *StmtTranslator) translateAssign(s clight.Sassign) csharpminor.Stmt {
	value := t.exprTr.TranslateExpr(s.RHS)
	
	// Check if this is an assignment to a function parameter
	if evar, ok := s.LHS.(clight.Evar); ok && t.isParam(evar.Name) {
		// Parameters don't have addresses in the same way locals do.
		// We assign to a temp that shadows the parameter.
		tempID := t.getOrCreateParamTemp(evar.Name)
		return csharpminor.Sset{
			TempID: tempID,
			RHS:    value,
		}
	}
	
	addr, chunk := t.translateLvalue(s.LHS)
	return csharpminor.Sstore{
		Chunk: chunk,
		Addr:  addr,
		Value: value,
	}
}

// translateLvalue translates an l-value to its address and memory chunk.
func (t *StmtTranslator) translateLvalue(e clight.Expr) (addr csharpminor.Expr, chunk csharpminor.Chunk) {
	switch lv := e.(type) {
	case clight.Evar:
		addr = csharpminor.Eaddrof{Name: lv.Name}
		chunk = csharpminor.ChunkForType(lv.Typ)
	case clight.Ederef:
		addr = t.exprTr.TranslateExpr(lv.Ptr)
		chunk = csharpminor.ChunkForType(lv.Typ)
	case clight.Efield:
		addr = t.exprTr.TranslateFieldAddr(lv)
		chunk = csharpminor.ChunkForType(lv.Typ)
	default:
		panic("not an l-value")
	}
	return
}

// translateSet translates assignment to a temporary.
// Clight: temp = expr
// Csharpminor: Sset(temp, expr)
func (t *StmtTranslator) translateSet(s clight.Sset) csharpminor.Stmt {
	rhs := t.exprTr.TranslateExpr(s.RHS)
	return csharpminor.Sset{
		TempID: s.TempID,
		RHS:    rhs,
	}
}

// translateCall translates a function call.
func (t *StmtTranslator) translateCall(s clight.Scall) csharpminor.Stmt {
	funcExpr := t.exprTr.TranslateExpr(s.Func)
	args := make([]csharpminor.Expr, len(s.Args))
	for i, arg := range s.Args {
		args[i] = t.exprTr.TranslateExpr(arg)
	}
	return csharpminor.Scall{
		Result: s.Result,
		Func:   funcExpr,
		Args:   args,
	}
}

// translateBuiltin translates a builtin call.
func (t *StmtTranslator) translateBuiltin(s clight.Sbuiltin) csharpminor.Stmt {
	args := make([]csharpminor.Expr, len(s.Args))
	for i, arg := range s.Args {
		args[i] = t.exprTr.TranslateExpr(arg)
	}
	return csharpminor.Sbuiltin{
		Result:  s.Result,
		Builtin: s.Builtin,
		Args:    args,
	}
}

// translateSequence translates a statement sequence.
func (t *StmtTranslator) translateSequence(s clight.Ssequence) csharpminor.Stmt {
	first := t.TranslateStmt(s.First)
	second := t.TranslateStmt(s.Second)
	return csharpminor.Sseq{
		First:  first,
		Second: second,
	}
}

// translateIf translates an if-then-else statement.
func (t *StmtTranslator) translateIf(s clight.Sifthenelse) csharpminor.Stmt {
	cond := t.exprTr.TranslateExpr(s.Cond)
	thenStmt := t.TranslateStmt(s.Then)
	elseStmt := t.TranslateStmt(s.Else)
	return csharpminor.Sifthenelse{
		Cond: cond,
		Then: thenStmt,
		Else: elseStmt,
	}
}

// translateLoop translates a Clight loop.
// Clight Sloop has body and continue parts.
// In Csharpminor, we use Sblock + Sloop + Sexit pattern.
//
// Clight: loop { body; continue_stmt }
// Csharpminor:
//
//	block {                    <- break target (exit 1)
//	  loop {
//	    block {                <- continue target (exit 1)
//	      body
//	    }
//	    continue_stmt
//	  }
//	}
//
// When translating break: Sexit(2) - exit loop + outer block
// When translating continue: Sexit(1) - exit inner block only
func (t *StmtTranslator) translateLoop(s clight.Sloop) csharpminor.Stmt {
	// Enter loop context
	t.loopDepth++
	savedBlockDepth := t.blockDepth
	t.blockDepth = 0 // reset block depth for new loop

	// Translate body inside inner block (for continue)
	t.blockDepth = 1 // inside the continue block
	body := t.TranslateStmt(s.Body)
	continueStmt := t.TranslateStmt(s.Continue)
	t.blockDepth = 0

	// Restore context
	t.loopDepth--
	t.blockDepth = savedBlockDepth

	// Inner block for continue target
	innerBlock := csharpminor.Sblock{Body: body}

	// Loop body is inner block followed by continue statement
	loopBody := csharpminor.Seq(innerBlock, continueStmt)

	// The infinite loop
	loop := csharpminor.Sloop{Body: loopBody}

	// Outer block for break target
	return csharpminor.Sblock{Body: loop}
}

// translateBreak translates a break statement.
// In Csharpminor, break becomes Sexit(2) to exit both the continue block and the loop.
func (t *StmtTranslator) translateBreak() csharpminor.Stmt {
	// Exit: continue block (1) + loop itself (doesn't count as block) + outer break block (1)
	// But the way we structured it: we're inside the continue block, so:
	// - Sexit(1) exits the continue block (back to loop iteration)
	// - Sexit(2) exits the continue block + exits the outer break block (out of loop)
	return csharpminor.Sexit{N: 2}
}

// translateContinue translates a continue statement.
// In Csharpminor, continue becomes Sexit(1) to exit the continue block
// (the loop will then execute continue_stmt and restart).
func (t *StmtTranslator) translateContinue() csharpminor.Stmt {
	return csharpminor.Sexit{N: 1}
}

// translateReturn translates a return statement.
func (t *StmtTranslator) translateReturn(s clight.Sreturn) csharpminor.Stmt {
	var value csharpminor.Expr
	if s.Value != nil {
		value = t.exprTr.TranslateExpr(s.Value)
	}
	return csharpminor.Sreturn{Value: value}
}

// translateSwitch translates a switch statement.
// Note: CompCert's switch semantics differ from C - no fall-through.
func (t *StmtTranslator) translateSwitch(s clight.Sswitch) csharpminor.Stmt {
	expr := t.exprTr.TranslateExpr(s.Expr)

	// Determine if the switch expression is long
	isLong := false
	if _, ok := s.Expr.ExprType().(ctypes.Tlong); ok {
		isLong = true
	}

	cases := make([]csharpminor.SwitchCase, len(s.Cases))
	for i, c := range s.Cases {
		cases[i] = csharpminor.SwitchCase{
			Value: c.Value,
			Body:  t.TranslateStmt(c.Body),
		}
	}

	defaultStmt := t.TranslateStmt(s.Default)

	return csharpminor.Sswitch{
		IsLong:  isLong,
		Expr:    expr,
		Cases:   cases,
		Default: defaultStmt,
	}
}

// translateLabel translates a labeled statement.
func (t *StmtTranslator) translateLabel(s clight.Slabel) csharpminor.Stmt {
	body := t.TranslateStmt(s.Stmt)
	return csharpminor.Slabel{
		Label: s.Label,
		Body:  body,
	}
}

// translateGoto translates a goto statement.
func (t *StmtTranslator) translateGoto(s clight.Sgoto) csharpminor.Stmt {
	return csharpminor.Sgoto{Label: s.Label}
}
