// Package cshmgen implements the Cshmgen pass: Clight → Csharpminor
// This pass makes type-dependent operations explicit.
package cshmgen

import (
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

// TranslateUnaryOp maps a Clight unary operator to a Csharpminor typed unary operator
// based on the operand type.
func TranslateUnaryOp(op clight.UnaryOp, argType ctypes.Type) csharpminor.UnaryOp {
	switch op {
	case clight.Oneg:
		return translateNeg(argType)
	case clight.Onotint:
		return translateBitnot(argType)
	case clight.Onotbool:
		return csharpminor.Onotbool
	case clight.Oabsfloat:
		// No direct Csharpminor equivalent; would be lowered differently
		// For now, return a placeholder (handled at a higher level)
		return csharpminor.Onegf // placeholder
	}
	panic("unhandled unary operator")
}

// translateNeg maps negation to typed negation operator
func translateNeg(t ctypes.Type) csharpminor.UnaryOp {
	switch typ := t.(type) {
	case ctypes.Tint:
		return csharpminor.Onegint
	case ctypes.Tlong:
		return csharpminor.Onegl
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return csharpminor.Onegs
		}
		return csharpminor.Onegf
	}
	return csharpminor.Onegint // default
}

// translateBitnot maps bitwise not to typed operator
func translateBitnot(t ctypes.Type) csharpminor.UnaryOp {
	switch t.(type) {
	case ctypes.Tlong:
		return csharpminor.Onotl
	default:
		return csharpminor.Onotint
	}
}

// TranslateBinaryOp maps a Clight binary operator to a Csharpminor typed binary operator
// based on the operand types. For comparison operators, also returns the Comparison kind.
func TranslateBinaryOp(op clight.BinaryOp, leftType, rightType ctypes.Type) (csharpminor.BinaryOp, csharpminor.Comparison) {
	// For arithmetic/bitwise ops, use left type (operands should have same type after conversions)
	switch op {
	case clight.Oadd:
		return translateAdd(leftType), csharpminor.Ceq
	case clight.Osub:
		return translateSub(leftType), csharpminor.Ceq
	case clight.Omul:
		return translateMul(leftType), csharpminor.Ceq
	case clight.Odiv:
		return translateDiv(leftType), csharpminor.Ceq
	case clight.Omod:
		return translateMod(leftType), csharpminor.Ceq
	case clight.Oand:
		return translateAnd(leftType), csharpminor.Ceq
	case clight.Oor:
		return translateOr(leftType), csharpminor.Ceq
	case clight.Oxor:
		return translateXor(leftType), csharpminor.Ceq
	case clight.Oshl:
		return translateShl(leftType), csharpminor.Ceq
	case clight.Oshr:
		return translateShr(leftType), csharpminor.Ceq
	// Comparison operators - check both operand types for signedness
	// If either operand is unsigned, use unsigned comparison (C standard)
	case clight.Oeq:
		return translateCmpBoth(leftType, rightType), csharpminor.Ceq
	case clight.One:
		return translateCmpBoth(leftType, rightType), csharpminor.Cne
	case clight.Olt:
		return translateCmpBoth(leftType, rightType), csharpminor.Clt
	case clight.Ole:
		return translateCmpBoth(leftType, rightType), csharpminor.Cle
	case clight.Ogt:
		return translateCmpBoth(leftType, rightType), csharpminor.Cgt
	case clight.Oge:
		return translateCmpBoth(leftType, rightType), csharpminor.Cge
	}
	panic("unhandled binary operator")
}

// translateAdd maps addition to typed operator
func translateAdd(t ctypes.Type) csharpminor.BinaryOp {
	switch typ := t.(type) {
	case ctypes.Tint:
		return csharpminor.Oadd
	case ctypes.Tlong:
		return csharpminor.Oaddl
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return csharpminor.Oadds
		}
		return csharpminor.Oaddf
	case ctypes.Tpointer:
		return csharpminor.Oaddl // pointer arithmetic uses long
	}
	return csharpminor.Oadd // default
}

// translateSub maps subtraction to typed operator
func translateSub(t ctypes.Type) csharpminor.BinaryOp {
	switch typ := t.(type) {
	case ctypes.Tint:
		return csharpminor.Osub
	case ctypes.Tlong:
		return csharpminor.Osubl
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return csharpminor.Osubs
		}
		return csharpminor.Osubf
	case ctypes.Tpointer:
		return csharpminor.Osubl // pointer arithmetic uses long
	}
	return csharpminor.Osub // default
}

// translateMul maps multiplication to typed operator
func translateMul(t ctypes.Type) csharpminor.BinaryOp {
	switch typ := t.(type) {
	case ctypes.Tint:
		return csharpminor.Omul
	case ctypes.Tlong:
		return csharpminor.Omull
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return csharpminor.Omuls
		}
		return csharpminor.Omulf
	}
	return csharpminor.Omul // default
}

// translateDiv maps division to typed operator
func translateDiv(t ctypes.Type) csharpminor.BinaryOp {
	switch typ := t.(type) {
	case ctypes.Tint:
		if typ.Sign == ctypes.Unsigned {
			return csharpminor.Odivu
		}
		return csharpminor.Odiv
	case ctypes.Tlong:
		if typ.Sign == ctypes.Unsigned {
			return csharpminor.Odivlu
		}
		return csharpminor.Odivl
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return csharpminor.Odivs
		}
		return csharpminor.Odivf
	}
	return csharpminor.Odiv // default
}

// translateMod maps modulo to typed operator
func translateMod(t ctypes.Type) csharpminor.BinaryOp {
	switch typ := t.(type) {
	case ctypes.Tint:
		if typ.Sign == ctypes.Unsigned {
			return csharpminor.Omodu
		}
		return csharpminor.Omod
	case ctypes.Tlong:
		if typ.Sign == ctypes.Unsigned {
			return csharpminor.Omodlu
		}
		return csharpminor.Omodl
	}
	return csharpminor.Omod // default
}

// translateAnd maps bitwise and to typed operator
func translateAnd(t ctypes.Type) csharpminor.BinaryOp {
	switch t.(type) {
	case ctypes.Tlong:
		return csharpminor.Oandl
	}
	return csharpminor.Oand
}

// translateOr maps bitwise or to typed operator
func translateOr(t ctypes.Type) csharpminor.BinaryOp {
	switch t.(type) {
	case ctypes.Tlong:
		return csharpminor.Oorl
	}
	return csharpminor.Oor
}

// translateXor maps bitwise xor to typed operator
func translateXor(t ctypes.Type) csharpminor.BinaryOp {
	switch t.(type) {
	case ctypes.Tlong:
		return csharpminor.Oxorl
	}
	return csharpminor.Oxor
}

// translateShl maps shift left to typed operator
func translateShl(t ctypes.Type) csharpminor.BinaryOp {
	switch t.(type) {
	case ctypes.Tlong:
		return csharpminor.Oshll
	}
	return csharpminor.Oshl
}

// translateShr maps shift right to typed operator (signed vs unsigned)
func translateShr(t ctypes.Type) csharpminor.BinaryOp {
	switch typ := t.(type) {
	case ctypes.Tint:
		if typ.Sign == ctypes.Unsigned {
			return csharpminor.Oshru
		}
		return csharpminor.Oshr
	case ctypes.Tlong:
		if typ.Sign == ctypes.Unsigned {
			return csharpminor.Oshrlu
		}
		return csharpminor.Oshrl
	}
	return csharpminor.Oshr // default signed
}

// isUnsigned returns true if the type is an unsigned integer type
func isUnsigned(t ctypes.Type) bool {
	switch typ := t.(type) {
	case ctypes.Tint:
		return typ.Sign == ctypes.Unsigned
	case ctypes.Tlong:
		return typ.Sign == ctypes.Unsigned
	}
	return false
}

// translateCmpBoth maps comparison to typed comparison operator,
// checking both operand types. If either is unsigned, use unsigned comparison.
func translateCmpBoth(left, right ctypes.Type) csharpminor.BinaryOp {
	// If either operand is unsigned, use unsigned comparison
	if isUnsigned(left) || isUnsigned(right) {
		switch left.(type) {
		case ctypes.Tlong:
			return csharpminor.Ocmplu
		default:
			return csharpminor.Ocmpu
		}
	}
	return translateCmp(left)
}

// translateCmp maps comparison to typed comparison operator
func translateCmp(t ctypes.Type) csharpminor.BinaryOp {
	switch typ := t.(type) {
	case ctypes.Tint:
		if typ.Sign == ctypes.Unsigned {
			return csharpminor.Ocmpu
		}
		return csharpminor.Ocmp
	case ctypes.Tlong:
		if typ.Sign == ctypes.Unsigned {
			return csharpminor.Ocmplu
		}
		return csharpminor.Ocmpl
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return csharpminor.Ocmps
		}
		return csharpminor.Ocmpf
	case ctypes.Tpointer:
		return csharpminor.Ocmplu // pointer comparison is unsigned long
	}
	return csharpminor.Ocmp // default
}

// TranslateCast maps a Clight cast to a Csharpminor unary operator
// Returns the operator and whether a conversion is needed.
// If no conversion is needed (same type), returns ok=false.
func TranslateCast(fromType, toType ctypes.Type) (op csharpminor.UnaryOp, ok bool) {
	if ctypes.Equal(fromType, toType) {
		return 0, false
	}

	// Integer truncation/sign-extension
	if to, ok := toType.(ctypes.Tint); ok {
		switch to.Size {
		case ctypes.I8:
			if to.Sign == ctypes.Signed {
				return csharpminor.Ocast8signed, true
			}
			return csharpminor.Ocast8unsigned, true
		case ctypes.I16:
			if to.Sign == ctypes.Signed {
				return csharpminor.Ocast16signed, true
			}
			return csharpminor.Ocast16unsigned, true
		}
	}

	// Float conversions
	fromFloat, fromIsFloat := fromType.(ctypes.Tfloat)
	toFloat, toIsFloat := toType.(ctypes.Tfloat)

	if fromIsFloat && toIsFloat {
		if fromFloat.Size == ctypes.F64 && toFloat.Size == ctypes.F32 {
			return csharpminor.Osingleoffloat, true
		}
		if fromFloat.Size == ctypes.F32 && toFloat.Size == ctypes.F64 {
			return csharpminor.Ofloatofsingle, true
		}
		return 0, false // same float size
	}

	// Int/Long to Float
	if toIsFloat {
		if fromLong, ok := fromType.(ctypes.Tlong); ok {
			if toFloat.Size == ctypes.F32 {
				if fromLong.Sign == ctypes.Unsigned {
					return csharpminor.Osingleoflongu, true
				}
				return csharpminor.Osingleoflong, true
			}
			if fromLong.Sign == ctypes.Unsigned {
				return csharpminor.Ofloatoflongu, true
			}
			return csharpminor.Ofloatoflong, true
		}
		if fromInt, ok := fromType.(ctypes.Tint); ok {
			if toFloat.Size == ctypes.F32 {
				// int -> float32 is int -> float64 -> float32
				if fromInt.Sign == ctypes.Unsigned {
					// We'd chain: intu -> float64 -> float32
					return csharpminor.Ofloatofintu, true // caller must chain
				}
				return csharpminor.Ofloatofint, true // caller must chain
			}
			if fromInt.Sign == ctypes.Unsigned {
				return csharpminor.Ofloatofintu, true
			}
			return csharpminor.Ofloatofint, true
		}
	}

	// Float to Int/Long
	if fromIsFloat {
		if toLong, ok := toType.(ctypes.Tlong); ok {
			if fromFloat.Size == ctypes.F32 {
				if toLong.Sign == ctypes.Unsigned {
					return csharpminor.Olonguofsingle, true
				}
				return csharpminor.Olongofsingle, true
			}
			if toLong.Sign == ctypes.Unsigned {
				return csharpminor.Olonguoffloat, true
			}
			return csharpminor.Olongoffloat, true
		}
		if toInt, ok := toType.(ctypes.Tint); ok {
			if fromFloat.Size == ctypes.F32 {
				// float32 -> int is float32 -> float64 -> int
				// Return first step; caller chains
				return csharpminor.Ofloatofsingle, true
			}
			if toInt.Sign == ctypes.Unsigned {
				return csharpminor.Ointuoffloat, true
			}
			return csharpminor.Ointoffloat, true
		}
	}

	// Long/Int conversions
	if _, ok := toType.(ctypes.Tlong); ok {
		if fromInt, ok := fromType.(ctypes.Tint); ok {
			if fromInt.Sign == ctypes.Unsigned {
				return csharpminor.Olongofintu, true
			}
			return csharpminor.Olongofint, true
		}
	}
	if _, ok := toType.(ctypes.Tint); ok {
		if _, ok := fromType.(ctypes.Tlong); ok {
			return csharpminor.Ointoflong, true
		}
	}

	return 0, false
}

// IsComparisonOp returns true if the Clight binary operator is a comparison
func IsComparisonOp(op clight.BinaryOp) bool {
	switch op {
	case clight.Oeq, clight.One, clight.Olt, clight.Ogt, clight.Ole, clight.Oge:
		return true
	}
	return false
}
