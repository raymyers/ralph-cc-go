// Package simplexpr transforms Cabs expressions to Clight, extracting side-effects.
// This mirrors CompCert's SimplExpr.v transformation.
package simplexpr

import (
	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

// Transformer converts Cabs AST to Clight AST by extracting side-effects from expressions.
type Transformer struct {
	nextTempID int                       // counter for generating unique temp IDs
	tempTypes  []ctypes.Type             // types of generated temporaries
	typeEnv    map[string]ctypes.Type    // variable name -> type
	structDefs map[string]ctypes.Tstruct // struct name -> full definition
}

// New creates a new SimplExpr transformer.
func New() *Transformer {
	return &Transformer{
		nextTempID: 1,
		tempTypes:  nil,
		typeEnv:    make(map[string]ctypes.Type),
		structDefs: make(map[string]ctypes.Tstruct),
	}
}

// Reset resets the transformer state for a new function.
func (t *Transformer) Reset() {
	t.nextTempID = 1
	t.tempTypes = nil
}

// SetNextTempID sets the starting temp ID (to continue from other passes).
func (t *Transformer) SetNextTempID(id int) {
	t.nextTempID = id
}

// TempTypes returns the types of all temporaries generated during transformation.
func (t *Transformer) TempTypes() []ctypes.Type {
	return t.tempTypes
}

// newTemp allocates a new temporary variable with the given type.
func (t *Transformer) newTemp(typ ctypes.Type) int {
	id := t.nextTempID
	t.nextTempID++
	t.tempTypes = append(t.tempTypes, typ)
	return id
}

// SetType records the type of a variable in the environment.
func (t *Transformer) SetType(name string, typ ctypes.Type) {
	t.typeEnv[name] = typ
}

// SetStructDef registers a struct definition.
func (t *Transformer) SetStructDef(s ctypes.Tstruct) {
	t.structDefs[s.Name] = s
}

// ResolveStruct looks up a struct definition by name and returns it with fields.
// If not found, returns the input unchanged.
func (t *Transformer) ResolveStruct(s ctypes.Tstruct) ctypes.Tstruct {
	if def, ok := t.structDefs[s.Name]; ok {
		return def
	}
	return s
}

// GetType looks up the type of a variable.
func (t *Transformer) GetType(name string) ctypes.Type {
	if typ, ok := t.typeEnv[name]; ok {
		return typ
	}
	return ctypes.Int() // default to int for unknown variables
}

// TransformResult holds the result of transforming an expression.
// An expression in Cabs may produce:
// - A pure Clight expression
// - A list of side-effect statements that must execute before
// - Both
type TransformResult struct {
	Expr  clight.Expr // the side-effect-free result expression
	Stmts []clight.Stmt // side-effect statements to execute first
}

// HasSideEffects checks if a Cabs expression has side-effects.
func HasSideEffects(e cabs.Expr) bool {
	switch expr := e.(type) {
	case cabs.Constant:
		return false
	case cabs.StringLiteral:
		return false
	case cabs.CharLiteral:
		return false
	case cabs.Variable:
		return false
	case cabs.Paren:
		return HasSideEffects(expr.Expr)
	case cabs.Unary:
		switch expr.Op {
		case cabs.OpPreInc, cabs.OpPreDec, cabs.OpPostInc, cabs.OpPostDec:
			return true
		default:
			return HasSideEffects(expr.Expr)
		}
	case cabs.Binary:
		// Assignment operators have side-effects
		switch expr.Op {
		case cabs.OpAssign, cabs.OpAddAssign, cabs.OpSubAssign, cabs.OpMulAssign,
			cabs.OpDivAssign, cabs.OpModAssign, cabs.OpAndAssign, cabs.OpOrAssign,
			cabs.OpXorAssign, cabs.OpShlAssign, cabs.OpShrAssign, cabs.OpComma:
			return true
		default:
			return HasSideEffects(expr.Left) || HasSideEffects(expr.Right)
		}
	case cabs.Conditional:
		return HasSideEffects(expr.Cond) || HasSideEffects(expr.Then) || HasSideEffects(expr.Else)
	case cabs.Call:
		// Function calls always have potential side-effects
		return true
	case cabs.Index:
		return HasSideEffects(expr.Array) || HasSideEffects(expr.Index)
	case cabs.Member:
		return HasSideEffects(expr.Expr)
	case cabs.SizeofExpr:
		return false // sizeof is evaluated at compile time
	case cabs.SizeofType:
		return false
	case cabs.Cast:
		return HasSideEffects(expr.Expr)
	}
	return false
}

// TransformExpr transforms a Cabs expression to a Clight expression,
// extracting any side-effects into statements.
func (t *Transformer) TransformExpr(e cabs.Expr) TransformResult {
	switch expr := e.(type) {
	case cabs.Constant:
		// Determine type based on value range following C99/C11 rules for
		// decimal integer literals without suffix. The sequence is:
		// 1. int
		// 2. long int  
		// 3. long long int
		// Note: decimal literals without 'u' suffix are NEVER unsigned
		const intMax = 2147483647   // 2^31 - 1
		const intMin = -2147483648  // -2^31
		const longMax = 9223372036854775807 // 2^63 - 1 (assuming LP64)
		var typ ctypes.Type
		if expr.Value >= intMin && expr.Value <= intMax {
			typ = ctypes.Int()
		} else if expr.Value >= intMin && expr.Value <= longMax {
			// Value doesn't fit in int but fits in long long (signed)
			typ = ctypes.Long()
		} else {
			// Value too large - would be undefined behavior, use unsigned long long
			typ = ctypes.Tlong{Sign: ctypes.Unsigned}
		}
		return TransformResult{
			Expr: clight.Econst_int{Value: expr.Value, Typ: typ},
		}

	case cabs.StringLiteral:
		// String literals become pointers to constant char arrays
		// Process escape sequences in the string value
		return TransformResult{
			Expr: clight.Estring{Value: processEscapeSequences(expr.Value), Typ: ctypes.Pointer(ctypes.Char())},
		}

	case cabs.CharLiteral:
		// Character literals become integer constants (ASCII value)
		value := int64(0)
		if len(expr.Value) > 0 {
			if expr.Value[0] == '\\' && len(expr.Value) > 1 {
				// Handle escape sequences
				switch expr.Value[1] {
				case 'n':
					value = 10 // newline
				case 't':
					value = 9 // tab
				case 'r':
					value = 13 // carriage return
				case '0':
					value = 0 // null
				case '\\':
					value = 92 // backslash
				case '\'':
					value = 39 // single quote
				case '"':
					value = 34 // double quote
				default:
					value = int64(expr.Value[1])
				}
			} else {
				value = int64(expr.Value[0])
			}
		}
		return TransformResult{
			Expr: clight.Econst_int{Value: value, Typ: ctypes.Int()},
		}

	case cabs.Variable:
		typ := t.GetType(expr.Name)
		// Resolve struct types to include field information
		if st, ok := typ.(ctypes.Tstruct); ok {
			typ = t.ResolveStruct(st)
		}
		return TransformResult{
			Expr: clight.Evar{Name: expr.Name, Typ: typ},
		}

	case cabs.Paren:
		return t.TransformExpr(expr.Expr)

	case cabs.Unary:
		return t.transformUnary(expr)

	case cabs.Binary:
		return t.transformBinary(expr)

	case cabs.Conditional:
		return t.transformConditional(expr)

	case cabs.Call:
		return t.transformCall(expr)

	case cabs.Index:
		return t.transformIndex(expr)

	case cabs.Member:
		return t.transformMember(expr)

	case cabs.SizeofType:
		return TransformResult{
			Expr: clight.Esizeof{
				ArgType: t.typeFromString(expr.TypeName),
				Typ:     ctypes.UInt(),
			},
		}

	case cabs.SizeofExpr:
		// For sizeof(expr), we need the type of the expression but don't evaluate it
		inner := t.TransformExpr(expr.Expr)
		return TransformResult{
			Expr: clight.Esizeof{
				ArgType: inner.Expr.ExprType(),
				Typ:     ctypes.UInt(),
			},
		}

	case cabs.Cast:
		inner := t.TransformExpr(expr.Expr)
		return TransformResult{
			Stmts: inner.Stmts,
			Expr: clight.Ecast{
				Arg: inner.Expr,
				Typ: t.typeFromString(expr.TypeName),
			},
		}
	}

	// Unknown expression type - return a placeholder
	return TransformResult{
		Expr: clight.Econst_int{Value: 0, Typ: ctypes.Int()},
	}
}

func (t *Transformer) transformUnary(expr cabs.Unary) TransformResult {
	switch expr.Op {
	case cabs.OpPlus:
		// Unary plus is a no-op - just return the inner expression
		return t.TransformExpr(expr.Expr)

	case cabs.OpNeg:
		inner := t.TransformExpr(expr.Expr)
		return TransformResult{
			Stmts: inner.Stmts,
			Expr:  clight.Eunop{Op: clight.Oneg, Arg: inner.Expr, Typ: inner.Expr.ExprType()},
		}

	case cabs.OpNot:
		inner := t.TransformExpr(expr.Expr)
		return TransformResult{
			Stmts: inner.Stmts,
			Expr:  clight.Eunop{Op: clight.Onotbool, Arg: inner.Expr, Typ: ctypes.Int()},
		}

	case cabs.OpBitNot:
		inner := t.TransformExpr(expr.Expr)
		return TransformResult{
			Stmts: inner.Stmts,
			Expr:  clight.Eunop{Op: clight.Onotint, Arg: inner.Expr, Typ: inner.Expr.ExprType()},
		}

	case cabs.OpAddrOf:
		inner := t.TransformExpr(expr.Expr)
		return TransformResult{
			Stmts: inner.Stmts,
			Expr:  clight.Eaddrof{Arg: inner.Expr, Typ: ctypes.Pointer(inner.Expr.ExprType())},
		}

	case cabs.OpDeref:
		inner := t.TransformExpr(expr.Expr)
		// Get the pointed-to type
		ptrTyp := inner.Expr.ExprType()
		elemTyp := ctypes.Int() // default
		if ptr, ok := ptrTyp.(ctypes.Tpointer); ok {
			elemTyp = ptr.Elem
		}
		return TransformResult{
			Stmts: inner.Stmts,
			Expr:  clight.Ederef{Ptr: inner.Expr, Typ: elemTyp},
		}

	case cabs.OpPreInc:
		// ++x becomes: tmp = x + 1; x = tmp; result is tmp
		return t.transformIncDec(expr.Expr, clight.Oadd, true)

	case cabs.OpPreDec:
		// --x becomes: tmp = x - 1; x = tmp; result is tmp
		return t.transformIncDec(expr.Expr, clight.Osub, true)

	case cabs.OpPostInc:
		// x++ becomes: tmp = x; x = x + 1; result is tmp
		return t.transformIncDec(expr.Expr, clight.Oadd, false)

	case cabs.OpPostDec:
		// x-- becomes: tmp = x; x = x - 1; result is tmp
		return t.transformIncDec(expr.Expr, clight.Osub, false)
	}

	// Fallback
	inner := t.TransformExpr(expr.Expr)
	return TransformResult{
		Stmts: inner.Stmts,
		Expr:  inner.Expr,
	}
}

func (t *Transformer) transformIncDec(operand cabs.Expr, op clight.BinaryOp, isPre bool) TransformResult {
	inner := t.TransformExpr(operand)
	typ := inner.Expr.ExprType()
	one := clight.Econst_int{Value: 1, Typ: typ}

	// Create the computed value: x + 1 or x - 1
	computed := clight.Ebinop{Op: op, Left: inner.Expr, Right: one, Typ: typ}

	var stmts []clight.Stmt
	stmts = append(stmts, inner.Stmts...)

	if isPre {
		// ++x: compute x+1, assign to x, result is x+1
		// We need a temp to hold the result
		tempID := t.newTemp(typ)
		stmts = append(stmts, clight.Sset{TempID: tempID, RHS: computed})
		stmts = append(stmts, clight.Sassign{LHS: inner.Expr, RHS: clight.Etempvar{ID: tempID, Typ: typ}})
		return TransformResult{
			Stmts: stmts,
			Expr:  clight.Etempvar{ID: tempID, Typ: typ},
		}
	} else {
		// x++: save x to temp, compute x+1, assign to x, result is saved temp
		tempID := t.newTemp(typ)
		stmts = append(stmts, clight.Sset{TempID: tempID, RHS: inner.Expr})
		stmts = append(stmts, clight.Sassign{LHS: inner.Expr, RHS: computed})
		return TransformResult{
			Stmts: stmts,
			Expr:  clight.Etempvar{ID: tempID, Typ: typ},
		}
	}
}

func (t *Transformer) transformBinary(expr cabs.Binary) TransformResult {
	switch expr.Op {
	case cabs.OpAssign:
		return t.transformAssign(expr.Left, expr.Right)

	case cabs.OpAddAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Oadd)
	case cabs.OpSubAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Osub)
	case cabs.OpMulAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Omul)
	case cabs.OpDivAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Odiv)
	case cabs.OpModAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Omod)
	case cabs.OpAndAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Oand)
	case cabs.OpOrAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Oor)
	case cabs.OpXorAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Oxor)
	case cabs.OpShlAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Oshl)
	case cabs.OpShrAssign:
		return t.transformCompoundAssign(expr.Left, expr.Right, clight.Oshr)

	case cabs.OpComma:
		return t.transformComma(expr.Left, expr.Right)

	case cabs.OpAnd:
		// Logical && with short-circuit: a && b => a ? (b ? 1 : 0) : 0
		return t.transformLogicalAnd(expr.Left, expr.Right)

	case cabs.OpOr:
		// Logical || with short-circuit: a || b => a ? 1 : (b ? 1 : 0)
		return t.transformLogicalOr(expr.Left, expr.Right)

	default:
		// Pure binary operators
		left := t.TransformExpr(expr.Left)
		right := t.TransformExpr(expr.Right)

		var stmts []clight.Stmt
		stmts = append(stmts, left.Stmts...)
		stmts = append(stmts, right.Stmts...)

		clightOp := t.cabsToBinaryOp(expr.Op)
		// Apply C's usual arithmetic conversions for result type
		typ := usualArithmeticConversion(left.Expr.ExprType(), right.Expr.ExprType())

		// Comparison operators return int
		if clightOp >= clight.Oeq && clightOp <= clight.Oge {
			typ = ctypes.Int()
		}

		return TransformResult{
			Stmts: stmts,
			Expr:  clight.Ebinop{Op: clightOp, Left: left.Expr, Right: right.Expr, Typ: typ},
		}
	}
}

func (t *Transformer) transformAssign(lhs, rhs cabs.Expr) TransformResult {
	// Transform both sides
	left := t.TransformExpr(lhs)
	right := t.TransformExpr(rhs)

	var stmts []clight.Stmt
	stmts = append(stmts, left.Stmts...)
	stmts = append(stmts, right.Stmts...)

	// Assignment: lhs = rhs
	// In C, the value of an assignment expression is the assigned value
	// We use a temp to capture this
	typ := left.Expr.ExprType()
	tempID := t.newTemp(typ)

	stmts = append(stmts, clight.Sset{TempID: tempID, RHS: right.Expr})
	stmts = append(stmts, clight.Sassign{LHS: left.Expr, RHS: clight.Etempvar{ID: tempID, Typ: typ}})

	return TransformResult{
		Stmts: stmts,
		Expr:  clight.Etempvar{ID: tempID, Typ: typ},
	}
}

func (t *Transformer) transformCompoundAssign(lhs, rhs cabs.Expr, op clight.BinaryOp) TransformResult {
	// x += e becomes: tmp = x + e; x = tmp; result is tmp
	left := t.TransformExpr(lhs)
	right := t.TransformExpr(rhs)

	var stmts []clight.Stmt
	stmts = append(stmts, left.Stmts...)
	stmts = append(stmts, right.Stmts...)

	typ := left.Expr.ExprType()
	computed := clight.Ebinop{Op: op, Left: left.Expr, Right: right.Expr, Typ: typ}

	tempID := t.newTemp(typ)
	stmts = append(stmts, clight.Sset{TempID: tempID, RHS: computed})
	stmts = append(stmts, clight.Sassign{LHS: left.Expr, RHS: clight.Etempvar{ID: tempID, Typ: typ}})

	return TransformResult{
		Stmts: stmts,
		Expr:  clight.Etempvar{ID: tempID, Typ: typ},
	}
}

func (t *Transformer) transformComma(left, right cabs.Expr) TransformResult {
	// e1, e2: evaluate e1 for side effects, result is e2
	leftResult := t.TransformExpr(left)
	rightResult := t.TransformExpr(right)

	var stmts []clight.Stmt
	stmts = append(stmts, leftResult.Stmts...)
	// The left expression's value is discarded, but we still need to evaluate it
	// If it's pure, we can skip it; if it has side effects, they're already in stmts
	if !HasSideEffects(left) {
		// Pure expression used in comma - its value is discarded anyway
		// but we should still compute it in case it has visible side effects
		// (which it doesn't, since HasSideEffects returned false)
	}
	stmts = append(stmts, rightResult.Stmts...)

	return TransformResult{
		Stmts: stmts,
		Expr:  rightResult.Expr,
	}
}

func (t *Transformer) transformConditional(expr cabs.Conditional) TransformResult {
	cond := t.TransformExpr(expr.Cond)

	// If condition has side effects, they must be evaluated first
	var stmts []clight.Stmt
	stmts = append(stmts, cond.Stmts...)

	// Check if then/else have side effects
	thenHasSE := HasSideEffects(expr.Then)
	elseHasSE := HasSideEffects(expr.Else)

	if !thenHasSE && !elseHasSE {
		// Pure conditional: can remain as an expression
		thenResult := t.TransformExpr(expr.Then)
		elseResult := t.TransformExpr(expr.Else)
		// Clight doesn't have a conditional expression, so we must use if-then-else
		// and a temporary
		typ := thenResult.Expr.ExprType()
		tempID := t.newTemp(typ)

		thenStmt := clight.Sset{TempID: tempID, RHS: thenResult.Expr}
		elseStmt := clight.Sset{TempID: tempID, RHS: elseResult.Expr}

		stmts = append(stmts, clight.Sifthenelse{
			Cond: cond.Expr,
			Then: thenStmt,
			Else: elseStmt,
		})

		return TransformResult{
			Stmts: stmts,
			Expr:  clight.Etempvar{ID: tempID, Typ: typ},
		}
	}

	// Conditional with side-effects: must use if-then-else statement
	thenResult := t.TransformExpr(expr.Then)
	elseResult := t.TransformExpr(expr.Else)

	typ := thenResult.Expr.ExprType()
	tempID := t.newTemp(typ)

	// Build then branch: execute side effects, then set temp
	thenStmts := append(thenResult.Stmts, clight.Sset{TempID: tempID, RHS: thenResult.Expr})
	// Build else branch: execute side effects, then set temp
	elseStmts := append(elseResult.Stmts, clight.Sset{TempID: tempID, RHS: elseResult.Expr})

	stmts = append(stmts, clight.Sifthenelse{
		Cond: cond.Expr,
		Then: clight.Seq(thenStmts...),
		Else: clight.Seq(elseStmts...),
	})

	return TransformResult{
		Stmts: stmts,
		Expr:  clight.Etempvar{ID: tempID, Typ: typ},
	}
}

func (t *Transformer) transformCall(expr cabs.Call) TransformResult {
	// Transform the function expression
	funcResult := t.TransformExpr(expr.Func)

	var stmts []clight.Stmt
	stmts = append(stmts, funcResult.Stmts...)

	// Transform all arguments (left-to-right evaluation)
	var args []clight.Expr
	for _, arg := range expr.Args {
		argResult := t.TransformExpr(arg)
		stmts = append(stmts, argResult.Stmts...)
		args = append(args, argResult.Expr)
	}

	// Determine return type (simplified - assume int if unknown)
	retType := ctypes.Int()
	if fn, ok := funcResult.Expr.ExprType().(ctypes.Tfunction); ok {
		retType = fn.Return
	}

	// Function call becomes a statement; result goes into a temporary
	tempID := t.newTemp(retType)
	stmts = append(stmts, clight.Scall{
		Result: &tempID,
		Func:   funcResult.Expr,
		Args:   args,
	})

	return TransformResult{
		Stmts: stmts,
		Expr:  clight.Etempvar{ID: tempID, Typ: retType},
	}
}

func (t *Transformer) transformIndex(expr cabs.Index) TransformResult {
	// a[i] is equivalent to *(a + i)
	array := t.TransformExpr(expr.Array)
	index := t.TransformExpr(expr.Index)

	var stmts []clight.Stmt
	stmts = append(stmts, array.Stmts...)
	stmts = append(stmts, index.Stmts...)

	// Get element type and apply array-to-pointer decay if needed
	elemTyp := ctypes.Int() // default
	arrayExpr := array.Expr
	switch at := array.Expr.ExprType().(type) {
	case ctypes.Tarray:
		elemTyp = at.Elem
		// Array decay: array variable becomes pointer to first element
		// a[i] where a is int[3] becomes *(&a + i)
		arrayExpr = clight.Eaddrof{Arg: array.Expr, Typ: ctypes.Pointer(elemTyp)}
	case ctypes.Tpointer:
		elemTyp = at.Elem
	}

	// Compute address: a + i (where a is now a pointer after decay)
	ptrAdd := clight.Ebinop{
		Op:    clight.Oadd,
		Left:  arrayExpr,
		Right: index.Expr,
		Typ:   ctypes.Pointer(elemTyp),
	}

	return TransformResult{
		Stmts: stmts,
		Expr:  clight.Ederef{Ptr: ptrAdd, Typ: elemTyp},
	}
}

func (t *Transformer) transformMember(expr cabs.Member) TransformResult {
	inner := t.TransformExpr(expr.Expr)

	var stmts []clight.Stmt
	stmts = append(stmts, inner.Stmts...)

	// For s.f, inner is the struct
	// For p->f, inner is a pointer (transformed as (*p).f)
	base := inner.Expr
	baseTyp := inner.Expr.ExprType()
	if expr.IsArrow {
		// Dereference the pointer first
		elemTyp := ctypes.Int()
		if ptr, ok := baseTyp.(ctypes.Tpointer); ok {
			elemTyp = ptr.Elem
			// Resolve struct type if the pointed-to type is a struct
			if st, ok := elemTyp.(ctypes.Tstruct); ok {
				elemTyp = t.ResolveStruct(st)
			}
		}
		base = clight.Ederef{Ptr: inner.Expr, Typ: elemTyp}
		baseTyp = elemTyp
	}

	// Resolve struct type to get field information
	if st, ok := baseTyp.(ctypes.Tstruct); ok {
		baseTyp = t.ResolveStruct(st)
	}

	// Look up field type from resolved struct
	fieldTyp := ctypes.Int()
	if st, ok := baseTyp.(ctypes.Tstruct); ok {
		for _, f := range st.Fields {
			if f.Name == expr.Name {
				fieldTyp = f.Type
				break
			}
		}
	}

	return TransformResult{
		Stmts: stmts,
		Expr:  clight.Efield{Arg: base, FieldName: expr.Name, Typ: fieldTyp},
	}
}

func (t *Transformer) cabsToBinaryOp(op cabs.BinaryOp) clight.BinaryOp {
	switch op {
	case cabs.OpAdd:
		return clight.Oadd
	case cabs.OpSub:
		return clight.Osub
	case cabs.OpMul:
		return clight.Omul
	case cabs.OpDiv:
		return clight.Odiv
	case cabs.OpMod:
		return clight.Omod
	case cabs.OpEq:
		return clight.Oeq
	case cabs.OpNe:
		return clight.One
	case cabs.OpLt:
		return clight.Olt
	case cabs.OpGt:
		return clight.Ogt
	case cabs.OpLe:
		return clight.Ole
	case cabs.OpGe:
		return clight.Oge
	case cabs.OpBitAnd:
		return clight.Oand
	case cabs.OpBitOr:
		return clight.Oor
	case cabs.OpBitXor:
		return clight.Oxor
	case cabs.OpShl:
		return clight.Oshl
	case cabs.OpShr:
		return clight.Oshr
	case cabs.OpAnd:
		// Logical && should be handled in transformBinary, not here
		panic("OpAnd should be handled in transformBinary, not cabsToBinaryOp")
	case cabs.OpOr:
		// Logical || should be handled in transformBinary, not here
		panic("OpOr should be handled in transformBinary, not cabsToBinaryOp")
	}
	return clight.Oadd // fallback
}

func (t *Transformer) typeFromString(typeName string) ctypes.Type {
	switch typeName {
	case "void":
		return ctypes.Void()
	case "char", "signed char":
		return ctypes.Char()
	case "unsigned char":
		return ctypes.UChar()
	case "short", "signed short", "short int", "signed short int":
		return ctypes.Short()
	case "unsigned short", "unsigned short int":
		return ctypes.Tint{Size: ctypes.I16, Sign: ctypes.Unsigned}
	case "int", "signed", "signed int":
		return ctypes.Int()
	case "unsigned int", "unsigned":
		return ctypes.UInt()
	case "long", "long long", "signed long long":
		return ctypes.Long()
	case "unsigned long", "unsigned long long":
		return ctypes.Tlong{Sign: ctypes.Unsigned}
	case "float":
		return ctypes.Float()
	case "double":
		return ctypes.Double()
	// Standard integer typedefs from <stdint.h>
	case "int8_t":
		return ctypes.Char() // signed 8-bit
	case "uint8_t":
		return ctypes.UChar() // unsigned 8-bit
	case "int16_t":
		return ctypes.Short() // signed 16-bit
	case "uint16_t":
		return ctypes.Tint{Size: ctypes.I16, Sign: ctypes.Unsigned} // unsigned 16-bit
	case "int32_t":
		return ctypes.Int() // signed 32-bit
	case "uint32_t":
		return ctypes.UInt() // unsigned 32-bit
	case "int64_t":
		return ctypes.Long() // signed 64-bit
	case "uint64_t":
		return ctypes.Tlong{Sign: ctypes.Unsigned} // unsigned 64-bit
	case "size_t":
		return ctypes.Tlong{Sign: ctypes.Unsigned} // unsigned long on 64-bit
	case "ssize_t", "ptrdiff_t":
		return ctypes.Long() // signed long on 64-bit
	default:
		// Check for pointer types
		if len(typeName) > 2 && typeName[len(typeName)-1] == '*' {
			baseType := t.typeFromString(typeName[:len(typeName)-2])
			return ctypes.Pointer(baseType)
		}
		return ctypes.Int() // default fallback
	}
}

// processEscapeSequences converts escape sequences in a string literal to their actual characters.
// For example, `\n` becomes a newline character (byte 10).
func processEscapeSequences(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result = append(result, '\n')
				i++
			case 't':
				result = append(result, '\t')
				i++
			case 'r':
				result = append(result, '\r')
				i++
			case '0':
				result = append(result, 0)
				i++
			case '\\':
				result = append(result, '\\')
				i++
			case '"':
				result = append(result, '"')
				i++
			case '\'':
				result = append(result, '\'')
				i++
			default:
				// Unknown escape - keep backslash and character
				result = append(result, s[i])
			}
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// transformLogicalAnd implements short-circuit && evaluation.
// Transforms: a && b => if (a) { if (b) temp=1 else temp=0 } else { temp=0 }
func (t *Transformer) transformLogicalAnd(left, right cabs.Expr) TransformResult {
	leftResult := t.TransformExpr(left)
	rightResult := t.TransformExpr(right)

	// Result type is always int (0 or 1)
	resultType := ctypes.Int()
	tempID := t.newTemp(resultType)

	one := clight.Econst_int{Value: 1, Typ: resultType}
	zero := clight.Econst_int{Value: 0, Typ: resultType}

	// Build: if (left) { stmts(right); if (right) temp=1 else temp=0 } else { temp=0 }
	var stmts []clight.Stmt
	stmts = append(stmts, leftResult.Stmts...)

	// Inner if: if (right) temp=1 else temp=0
	innerIf := clight.Sifthenelse{
		Cond: rightResult.Expr,
		Then: clight.Sset{TempID: tempID, RHS: one},
		Else: clight.Sset{TempID: tempID, RHS: zero},
	}

	// Then branch: rightResult.Stmts + innerIf
	thenBranch := clight.Seq(append(rightResult.Stmts, innerIf)...)

	// Outer if
	outerIf := clight.Sifthenelse{
		Cond: leftResult.Expr,
		Then: thenBranch,
		Else: clight.Sset{TempID: tempID, RHS: zero},
	}
	stmts = append(stmts, outerIf)

	return TransformResult{
		Stmts: stmts,
		Expr:  clight.Etempvar{ID: tempID, Typ: resultType},
	}
}

// usualArithmeticConversion computes the result type of a binary arithmetic
// operation according to C's "usual arithmetic conversions" (C99 6.3.1.8).
// Key rules:
// - Types smaller than int (char, short, int8, int16, uint8, uint16) are
//   promoted to int before arithmetic
// - If both operands become int, the result is int
// - If one operand is unsigned int and the other is signed int (with same
//   rank), the result is unsigned int
// - For long types, similar rules apply with long/unsigned long
func usualArithmeticConversion(left, right ctypes.Type) ctypes.Type {
	// Helper to check if type needs integer promotion (smaller than int)
	needsPromotion := func(t ctypes.Type) bool {
		switch typ := t.(type) {
		case ctypes.Tint:
			// int8, int16, uint8, uint16 all promote to int
			return typ.Size == ctypes.I8 || typ.Size == ctypes.I16
		}
		return false
	}

	// Helper to check if type is unsigned int (32-bit)
	isUnsignedInt := func(t ctypes.Type) bool {
		if typ, ok := t.(ctypes.Tint); ok {
			return typ.Size == ctypes.I32 && typ.Sign == ctypes.Unsigned
		}
		return false
	}

	// Helper to check if type is unsigned long
	isUnsignedLong := func(t ctypes.Type) bool {
		if typ, ok := t.(ctypes.Tlong); ok {
			return typ.Sign == ctypes.Unsigned
		}
		return false
	}

	// Handle float types - use wider float type
	leftFloat, leftIsFloat := left.(ctypes.Tfloat)
	rightFloat, rightIsFloat := right.(ctypes.Tfloat)
	if leftIsFloat || rightIsFloat {
		if leftIsFloat && rightIsFloat {
			if leftFloat.Size == ctypes.F64 || rightFloat.Size == ctypes.F64 {
				return ctypes.Double()
			}
			return ctypes.Float()
		}
		if leftIsFloat {
			return left
		}
		return right
	}

	// Handle long types
	_, leftIsLong := left.(ctypes.Tlong)
	_, rightIsLong := right.(ctypes.Tlong)
	if leftIsLong || rightIsLong {
		if isUnsignedLong(left) || isUnsignedLong(right) {
			return ctypes.Tlong{Sign: ctypes.Unsigned}
		}
		return ctypes.Long()
	}

	// For pointer arithmetic, result is typically pointer or long
	if _, ok := left.(ctypes.Tpointer); ok {
		return left
	}
	if _, ok := right.(ctypes.Tpointer); ok {
		return right
	}

	// If either operand needs promotion (smaller than int), result is int
	if needsPromotion(left) || needsPromotion(right) {
		return ctypes.Int()
	}

	// If either operand is unsigned int, result is unsigned int
	if isUnsignedInt(left) || isUnsignedInt(right) {
		return ctypes.UInt()
	}

	// Default: use left operand type (typically int)
	return left
}

// transformLogicalOr implements short-circuit || evaluation.
// Transforms: a || b => if (a) { temp=1 } else { if (b) temp=1 else temp=0 }
func (t *Transformer) transformLogicalOr(left, right cabs.Expr) TransformResult {
	leftResult := t.TransformExpr(left)
	rightResult := t.TransformExpr(right)

	// Result type is always int (0 or 1)
	resultType := ctypes.Int()
	tempID := t.newTemp(resultType)

	one := clight.Econst_int{Value: 1, Typ: resultType}
	zero := clight.Econst_int{Value: 0, Typ: resultType}

	// Build: if (left) { temp=1 } else { stmts(right); if (right) temp=1 else temp=0 }
	var stmts []clight.Stmt
	stmts = append(stmts, leftResult.Stmts...)

	// Inner if: if (right) temp=1 else temp=0
	innerIf := clight.Sifthenelse{
		Cond: rightResult.Expr,
		Then: clight.Sset{TempID: tempID, RHS: one},
		Else: clight.Sset{TempID: tempID, RHS: zero},
	}

	// Else branch: rightResult.Stmts + innerIf
	elseBranch := clight.Seq(append(rightResult.Stmts, innerIf)...)

	// Outer if
	outerIf := clight.Sifthenelse{
		Cond: leftResult.Expr,
		Then: clight.Sset{TempID: tempID, RHS: one},
		Else: elseBranch,
	}
	stmts = append(stmts, outerIf)

	return TransformResult{
		Stmts: stmts,
		Expr:  clight.Etempvar{ID: tempID, Typ: resultType},
	}
}
