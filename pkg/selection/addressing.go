// Package selection implements the instruction selection pass: Cminor → CminorSel
// This file handles addressing mode selection for memory operations.
// Addressing modes allow more efficient memory access patterns by
// recognizing complex address computations that match hardware capabilities.
package selection

import (
	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/cminorsel"
)

// AddressResult holds the result of addressing mode selection:
// the selected mode plus any argument expressions needed.
type AddressResult struct {
	Mode cminorsel.AddressingMode
	Args []cminorsel.Expr
}

// SelectAddressing analyzes an address expression and selects the best
// addressing mode for the target architecture (ARM64).
// The input is a Cminor expression representing the address.
// Returns the addressing mode and the expressions to evaluate for it.
func SelectAddressing(addr cminor.Expr, globals map[string]bool, stackVars map[string]int64) AddressResult {
	// Try patterns in order of specificity (most specific first)

	// Pattern: global symbol + offset
	if result, ok := tryAglobal(addr, globals); ok {
		return result
	}

	// Pattern: stack slot access (sp + offset)
	if result, ok := tryAinstack(addr, stackVars); ok {
		return result
	}

	// Pattern: base + (index << shift) - ARM64 scaled addressing
	if result, ok := tryAindexed2shift(addr); ok {
		return result
	}

	// Pattern: base + index
	if result, ok := tryAindexed2(addr); ok {
		return result
	}

	// Pattern: base + constant offset
	if result, ok := tryAindexed(addr); ok {
		return result
	}

	// Fallback: base with zero offset
	return AddressResult{
		Mode: cminorsel.Aindexed{Offset: 0},
		Args: []cminorsel.Expr{translateExpr(addr)},
	}
}

// tryAglobal tries to match: &global or &global + offset
func tryAglobal(addr cminor.Expr, globals map[string]bool) (AddressResult, bool) {
	// Direct global variable reference: Evar where name is global
	if v, ok := addr.(cminor.Evar); ok {
		if globals[v.Name] {
			return AddressResult{
				Mode: cminorsel.Aglobal{Symbol: v.Name, Offset: 0},
				Args: nil, // No runtime arguments needed
			}, true
		}
	}

	// Global + constant: Ebinop(Oadd/Oaddl, Evar(global), Econst)
	if binop, ok := addr.(cminor.Ebinop); ok {
		if binop.Op == cminor.Oadd || binop.Op == cminor.Oaddl {
			// Check for global + const
			if v, vok := binop.Left.(cminor.Evar); vok && globals[v.Name] {
				if off := extractConstantOffset(binop.Right); off != nil {
					return AddressResult{
						Mode: cminorsel.Aglobal{Symbol: v.Name, Offset: *off},
						Args: nil,
					}, true
				}
			}
			// Check for const + global (commutative)
			if v, vok := binop.Right.(cminor.Evar); vok && globals[v.Name] {
				if off := extractConstantOffset(binop.Left); off != nil {
					return AddressResult{
						Mode: cminorsel.Aglobal{Symbol: v.Name, Offset: *off},
						Args: nil,
					}, true
				}
			}
		}
	}

	return AddressResult{}, false
}

// tryAinstack tries to match: stack variable address (stackptr + offset)
func tryAinstack(addr cminor.Expr, stackVars map[string]int64) (AddressResult, bool) {
	// Direct Oaddrstack constant (new form from cminorgen)
	if c, ok := addr.(cminor.Econst); ok {
		if stk, ok := c.Const.(cminor.Oaddrstack); ok {
			return AddressResult{
				Mode: cminorsel.Ainstack{Offset: stk.Offset},
				Args: nil,
			}, true
		}
	}

	// Direct stack variable reference (legacy form)
	if v, ok := addr.(cminor.Evar); ok {
		if off, found := stackVars[v.Name]; found {
			return AddressResult{
				Mode: cminorsel.Ainstack{Offset: off},
				Args: nil,
			}, true
		}
	}

	// Oaddrstack + constant: Ebinop(Oaddl, Econst{Oaddrstack{off1}}, Econst{off2})
	if binop, ok := addr.(cminor.Ebinop); ok {
		if binop.Op == cminor.Oadd || binop.Op == cminor.Oaddl {
			// Check for Oaddrstack + const
			if c, cok := binop.Left.(cminor.Econst); cok {
				if stk, sok := c.Const.(cminor.Oaddrstack); sok {
					if addOff := extractConstantOffset(binop.Right); addOff != nil {
						return AddressResult{
							Mode: cminorsel.Ainstack{Offset: stk.Offset + *addOff},
							Args: nil,
						}, true
					}
				}
			}
			// Commutative: const + Oaddrstack
			if c, cok := binop.Right.(cminor.Econst); cok {
				if stk, sok := c.Const.(cminor.Oaddrstack); sok {
					if addOff := extractConstantOffset(binop.Left); addOff != nil {
						return AddressResult{
							Mode: cminorsel.Ainstack{Offset: stk.Offset + *addOff},
							Args: nil,
						}, true
					}
				}
			}

			// Stack var + constant (legacy form with Evar)
			if v, vok := binop.Left.(cminor.Evar); vok {
				if baseOff, found := stackVars[v.Name]; found {
					if addOff := extractConstantOffset(binop.Right); addOff != nil {
						return AddressResult{
							Mode: cminorsel.Ainstack{Offset: baseOff + *addOff},
							Args: nil,
						}, true
					}
				}
			}
			// Commutative: const + stackvar
			if v, vok := binop.Right.(cminor.Evar); vok {
				if baseOff, found := stackVars[v.Name]; found {
					if addOff := extractConstantOffset(binop.Left); addOff != nil {
						return AddressResult{
							Mode: cminorsel.Ainstack{Offset: baseOff + *addOff},
							Args: nil,
						}, true
					}
				}
			}
		}
	}

	return AddressResult{}, false
}

// tryAindexed2shift tries to match: base + (index << shift)
// This is ARM64's scaled addressing mode for array access.
// Valid shifts are 0-3 (scales 1,2,4,8).
func tryAindexed2shift(addr cminor.Expr) (AddressResult, bool) {
	binop, ok := addr.(cminor.Ebinop)
	if !ok || (binop.Op != cminor.Oadd && binop.Op != cminor.Oaddl) {
		return AddressResult{}, false
	}

	// Check for base + (index << shift)
	if shift, base, index, ok := extractShiftAdd(binop.Left, binop.Right); ok {
		if shift >= 0 && shift <= 3 {
			return AddressResult{
				Mode: cminorsel.Aindexed2shift{Shift: shift},
				Args: []cminorsel.Expr{translateExpr(base), translateExpr(index)},
			}, true
		}
	}

	// Check commutative: (index << shift) + base
	if shift, base, index, ok := extractShiftAdd(binop.Right, binop.Left); ok {
		if shift >= 0 && shift <= 3 {
			return AddressResult{
				Mode: cminorsel.Aindexed2shift{Shift: shift},
				Args: []cminorsel.Expr{translateExpr(base), translateExpr(index)},
			}, true
		}
	}

	return AddressResult{}, false
}

// extractShiftAdd checks if 'shifted' is (index << const) and returns (shift, base, index)
func extractShiftAdd(base, shifted cminor.Expr) (int, cminor.Expr, cminor.Expr, bool) {
	shiftOp, ok := shifted.(cminor.Ebinop)
	if !ok {
		return 0, nil, nil, false
	}

	// Check for shift left operations
	if shiftOp.Op != cminor.Oshl && shiftOp.Op != cminor.Oshll {
		return 0, nil, nil, false
	}

	// Get the shift amount (must be constant)
	shiftAmt := extractConstantInt(shiftOp.Right)
	if shiftAmt == nil {
		return 0, nil, nil, false
	}

	return int(*shiftAmt), base, shiftOp.Left, true
}

// tryAindexed2 tries to match: base + index
func tryAindexed2(addr cminor.Expr) (AddressResult, bool) {
	binop, ok := addr.(cminor.Ebinop)
	if !ok || (binop.Op != cminor.Oadd && binop.Op != cminor.Oaddl) {
		return AddressResult{}, false
	}

	// Don't use Aindexed2 if one operand is a constant
	// (Aindexed would be more efficient)
	if isConstant(binop.Left) || isConstant(binop.Right) {
		return AddressResult{}, false
	}

	return AddressResult{
		Mode: cminorsel.Aindexed2{},
		Args: []cminorsel.Expr{translateExpr(binop.Left), translateExpr(binop.Right)},
	}, true
}

// tryAindexed tries to match: base + constant
func tryAindexed(addr cminor.Expr) (AddressResult, bool) {
	binop, ok := addr.(cminor.Ebinop)
	if !ok {
		return AddressResult{}, false
	}

	switch binop.Op {
	case cminor.Oadd, cminor.Oaddl:
		// Check for base + const
		if off := extractConstantOffset(binop.Right); off != nil {
			return AddressResult{
				Mode: cminorsel.Aindexed{Offset: *off},
				Args: []cminorsel.Expr{translateExpr(binop.Left)},
			}, true
		}
		// Check for const + base (commutative)
		if off := extractConstantOffset(binop.Left); off != nil {
			return AddressResult{
				Mode: cminorsel.Aindexed{Offset: *off},
				Args: []cminorsel.Expr{translateExpr(binop.Right)},
			}, true
		}

	case cminor.Osub, cminor.Osubl:
		// base - const => base + (-const)
		if off := extractConstantOffset(binop.Right); off != nil {
			return AddressResult{
				Mode: cminorsel.Aindexed{Offset: -*off},
				Args: []cminorsel.Expr{translateExpr(binop.Left)},
			}, true
		}
	}

	return AddressResult{}, false
}

// extractConstantOffset extracts a constant offset from an expression.
// Returns nil if the expression is not a constant.
func extractConstantOffset(e cminor.Expr) *int64 {
	cnst, ok := e.(cminor.Econst)
	if !ok {
		return nil
	}

	switch c := cnst.Const.(type) {
	case cminor.Ointconst:
		v := int64(c.Value)
		return &v
	case cminor.Olongconst:
		return &c.Value
	}
	return nil
}

// extractConstantInt extracts an int32 constant from an expression.
func extractConstantInt(e cminor.Expr) *int32 {
	cnst, ok := e.(cminor.Econst)
	if !ok {
		return nil
	}

	if c, ok := cnst.Const.(cminor.Ointconst); ok {
		return &c.Value
	}
	return nil
}

// isConstant returns true if the expression is a constant.
func isConstant(e cminor.Expr) bool {
	_, ok := e.(cminor.Econst)
	return ok
}

// translateExpr is a simple placeholder that converts a Cminor expression
// to a CminorSel expression. Full expression translation is in expr.go.
func translateExpr(e cminor.Expr) cminorsel.Expr {
	switch expr := e.(type) {
	case cminor.Evar:
		return cminorsel.Evar{Name: expr.Name}
	case cminor.Econst:
		return translateConst(expr.Const)
	case cminor.Eunop:
		return cminorsel.Eunop{
			Op:  cminorsel.UnaryOp(expr.Op),
			Arg: translateExpr(expr.Arg),
		}
	case cminor.Ebinop:
		return cminorsel.Ebinop{
			Op:    cminorsel.BinaryOp(expr.Op),
			Left:  translateExpr(expr.Left),
			Right: translateExpr(expr.Right),
		}
	case cminor.Ecmp:
		return cminorsel.Ebinop{
			Op:    cminorsel.BinaryOp(expr.Op),
			Left:  translateExpr(expr.Left),
			Right: translateExpr(expr.Right),
		}
	case cminor.Eload:
		// For nested loads, use simple Aindexed{0} addressing
		return cminorsel.Eload{
			Chunk: cminorsel.Chunk(expr.Chunk),
			Mode:  cminorsel.Aindexed{Offset: 0},
			Args:  []cminorsel.Expr{translateExpr(expr.Addr)},
		}
	}
	// Unknown expression type - return as-is wrapped in a var for safety
	return cminorsel.Evar{Name: "?unknown?"}
}

// translateConst converts a Cminor constant to a CminorSel expression.
func translateConst(c cminor.Constant) cminorsel.Expr {
	switch cnst := c.(type) {
	case cminor.Ointconst:
		return cminorsel.Econst{Const: cminorsel.Ointconst{Value: cnst.Value}}
	case cminor.Ofloatconst:
		return cminorsel.Econst{Const: cminorsel.Ofloatconst{Value: cnst.Value}}
	case cminor.Olongconst:
		return cminorsel.Econst{Const: cminorsel.Olongconst{Value: cnst.Value}}
	case cminor.Osingleconst:
		return cminorsel.Econst{Const: cminorsel.Osingleconst{Value: cnst.Value}}
	case cminor.Oaddrsymbol:
		return cminorsel.Econst{Const: cminorsel.Oaddrsymbol{Symbol: cnst.Name, Offset: cnst.Offset}}
	case cminor.Oaddrstack:
		return cminorsel.Econst{Const: cminorsel.Oaddrstack{Offset: cnst.Offset}}
	}
	return cminorsel.Econst{Const: cminorsel.Ointconst{Value: 0}}
}
