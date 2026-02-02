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
	nextTempID int                    // counter for generating unique temp IDs
	tempTypes  []ctypes.Type          // types of generated temporaries
	typeEnv    map[string]ctypes.Type // variable name -> type
}

// New creates a new SimplExpr transformer.
func New() *Transformer {
	return &Transformer{
		nextTempID: 1,
		tempTypes:  nil,
		typeEnv:    make(map[string]ctypes.Type),
	}
}

// Reset resets the transformer state for a new function.
func (t *Transformer) Reset() {
	t.nextTempID = 1
	t.tempTypes = nil
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
		return TransformResult{
			Expr: clight.Econst_int{Value: expr.Value, Typ: ctypes.Int()},
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

	default:
		// Pure binary operators
		left := t.TransformExpr(expr.Left)
		right := t.TransformExpr(expr.Right)

		var stmts []clight.Stmt
		stmts = append(stmts, left.Stmts...)
		stmts = append(stmts, right.Stmts...)

		clightOp := t.cabsToBinaryOp(expr.Op)
		typ := left.Expr.ExprType() // simplified type inference

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

	// Get element type
	elemTyp := ctypes.Int() // default
	switch at := array.Expr.ExprType().(type) {
	case ctypes.Tarray:
		elemTyp = at.Elem
	case ctypes.Tpointer:
		elemTyp = at.Elem
	}

	// Compute address: a + i
	ptrAdd := clight.Ebinop{
		Op:    clight.Oadd,
		Left:  array.Expr,
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
	if expr.IsArrow {
		// Dereference the pointer first
		ptrTyp := inner.Expr.ExprType()
		elemTyp := ctypes.Int()
		if ptr, ok := ptrTyp.(ctypes.Tpointer); ok {
			elemTyp = ptr.Elem
		}
		base = clight.Ederef{Ptr: inner.Expr, Typ: elemTyp}
	}

	// Look up field type (simplified - assume int)
	fieldTyp := ctypes.Int()
	if st, ok := base.ExprType().(ctypes.Tstruct); ok {
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
		// Logical && is not directly in Clight - handled via conditionals
		return clight.Oand // placeholder
	case cabs.OpOr:
		// Logical || is not directly in Clight - handled via conditionals
		return clight.Oor // placeholder
	}
	return clight.Oadd // fallback
}

func (t *Transformer) typeFromString(typeName string) ctypes.Type {
	switch typeName {
	case "void":
		return ctypes.Void()
	case "char":
		return ctypes.Char()
	case "unsigned char":
		return ctypes.UChar()
	case "short":
		return ctypes.Short()
	case "int":
		return ctypes.Int()
	case "unsigned int", "unsigned":
		return ctypes.UInt()
	case "long":
		return ctypes.Long()
	case "float":
		return ctypes.Float()
	case "double":
		return ctypes.Double()
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
