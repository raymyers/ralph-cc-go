// Package cshmgen implements the Cshmgen pass: Clight → Csharpminor
// This file handles expression translation.
package cshmgen

import (
	"fmt"

	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

// StringLiteral holds info about a string literal for later emission.
type StringLiteral struct {
	Label string
	Value string
}

// ExprTranslator translates Clight expressions to Csharpminor expressions.
type ExprTranslator struct {
	// globals tracks which variables are global (vs local temporaries)
	globals map[string]bool
	// stringCounter generates unique labels for string literals
	stringCounter int
	// strings collects all string literals for later emission
	strings []StringLiteral
	// paramTemps maps modified parameter names to their shadow temp IDs
	// This is set externally when parameters are modified
	paramTemps map[string]int
}

// NewExprTranslator creates a new expression translator.
func NewExprTranslator(globals map[string]bool) *ExprTranslator {
	if globals == nil {
		globals = make(map[string]bool)
	}
	return &ExprTranslator{globals: globals, stringCounter: 0, strings: nil, paramTemps: make(map[string]int)}
}

// SetParamTemps sets the parameter-to-temp mapping for reading modified parameters.
func (t *ExprTranslator) SetParamTemps(temps map[string]int) {
	t.paramTemps = temps
}

// GetStrings returns all collected string literals.
func (t *ExprTranslator) GetStrings() []StringLiteral {
	return t.strings
}

// TranslateExpr translates a Clight expression to a Csharpminor expression.
func (t *ExprTranslator) TranslateExpr(e clight.Expr) csharpminor.Expr {
	switch expr := e.(type) {
	case clight.Econst_int:
		return t.translateConstInt(expr)
	case clight.Econst_long:
		return t.translateConstLong(expr)
	case clight.Econst_float:
		return t.translateConstFloat(expr)
	case clight.Econst_single:
		return t.translateConstSingle(expr)
	case clight.Estring:
		return t.translateString(expr)
	case clight.Evar:
		return t.translateVar(expr)
	case clight.Etempvar:
		return t.translateTempvar(expr)
	case clight.Eunop:
		return t.translateUnop(expr)
	case clight.Ebinop:
		return t.translateBinop(expr)
	case clight.Ecast:
		return t.translateCast(expr)
	case clight.Ederef:
		return t.translateDeref(expr)
	case clight.Eaddrof:
		return t.translateAddrof(expr)
	case clight.Efield:
		return t.translateField(expr)
	case clight.Esizeof:
		return t.translateSizeof(expr)
	case clight.Ealignof:
		return t.translateAlignof(expr)
	}
	panic("unhandled expression type")
}

// translateConstInt translates an integer constant.
func (t *ExprTranslator) translateConstInt(e clight.Econst_int) csharpminor.Expr {
	return csharpminor.Econst{Const: csharpminor.Ointconst{Value: int32(e.Value)}}
}

// translateConstLong translates a long constant.
func (t *ExprTranslator) translateConstLong(e clight.Econst_long) csharpminor.Expr {
	return csharpminor.Econst{Const: csharpminor.Olongconst{Value: e.Value}}
}

// translateConstFloat translates a float64 constant.
func (t *ExprTranslator) translateConstFloat(e clight.Econst_float) csharpminor.Expr {
	return csharpminor.Econst{Const: csharpminor.Ofloatconst{Value: e.Value}}
}

// translateConstSingle translates a float32 constant.
func (t *ExprTranslator) translateConstSingle(e clight.Econst_single) csharpminor.Expr {
	return csharpminor.Econst{Const: csharpminor.Osingleconst{Value: e.Value}}
}

// translateString translates a string literal to a symbol address constant.
// Generates a unique label and stores the string for later emission.
func (t *ExprTranslator) translateString(e clight.Estring) csharpminor.Expr {
	// Generate unique label for this string
	label := fmt.Sprintf(".Lstr%d", t.stringCounter)
	t.stringCounter++
	// Store for later emission in rodata section
	t.strings = append(t.strings, StringLiteral{Label: label, Value: e.Value})
	return csharpminor.Econst{Const: csharpminor.Oaddrsymbol{Name: label, Offset: 0}}
}

// translateVar translates a variable reference.
// In Clight, Evar is always a memory location. We produce an Evar (global reference).
// For modified parameters, we read from the shadow temp instead.
func (t *ExprTranslator) translateVar(e clight.Evar) csharpminor.Expr {
	// Check if this is a modified parameter that should read from a temp
	if tempID, ok := t.paramTemps[e.Name]; ok {
		return csharpminor.Etempvar{ID: tempID}
	}
	return csharpminor.Evar{Name: e.Name}
}

// translateTempvar translates a temporary variable reference.
func (t *ExprTranslator) translateTempvar(e clight.Etempvar) csharpminor.Expr {
	return csharpminor.Etempvar{ID: e.ID}
}

// translateUnop translates a unary operation.
func (t *ExprTranslator) translateUnop(e clight.Eunop) csharpminor.Expr {
	arg := t.TranslateExpr(e.Arg)
	argType := e.Arg.ExprType()
	op := TranslateUnaryOp(e.Op, argType)
	return csharpminor.Eunop{Op: op, Arg: arg}
}

// translateBinop translates a binary operation.
func (t *ExprTranslator) translateBinop(e clight.Ebinop) csharpminor.Expr {
	left := t.TranslateExpr(e.Left)
	right := t.TranslateExpr(e.Right)
	leftType := e.Left.ExprType()
	rightType := e.Right.ExprType()

	op, cmp := TranslateBinaryOp(e.Op, leftType, rightType)

	// For comparison operators, use Ecmp
	if IsComparisonOp(e.Op) {
		return csharpminor.Ecmp{
			Op:    op,
			Cmp:   cmp,
			Left:  left,
			Right: right,
		}
	}

	return csharpminor.Ebinop{Op: op, Left: left, Right: right}
}

// translateCast translates a type cast.
func (t *ExprTranslator) translateCast(e clight.Ecast) csharpminor.Expr {
	arg := t.TranslateExpr(e.Arg)
	fromType := e.Arg.ExprType()
	toType := e.Typ

	op, needsCast := TranslateCast(fromType, toType)
	if !needsCast {
		return arg // no conversion needed
	}
	return csharpminor.Eunop{Op: op, Arg: arg}
}

// translateDeref translates a pointer dereference (*p).
// This becomes an explicit Eload with the appropriate memory chunk.
func (t *ExprTranslator) translateDeref(e clight.Ederef) csharpminor.Expr {
	addr := t.TranslateExpr(e.Ptr)
	chunk := csharpminor.ChunkForType(e.Typ)
	return csharpminor.Eload{Chunk: chunk, Addr: addr}
}

// translateAddrof translates address-of (&x).
func (t *ExprTranslator) translateAddrof(e clight.Eaddrof) csharpminor.Expr {
	// The inner expression should be an l-value.
	// For Evar, we produce Eaddrof with the variable name.
	switch inner := e.Arg.(type) {
	case clight.Evar:
		return csharpminor.Eaddrof{Name: inner.Name}
	case clight.Ederef:
		// &(*p) = p
		return t.TranslateExpr(inner.Ptr)
	case clight.Efield:
		// &(s.f) - address of struct field
		return t.TranslateFieldAddr(inner)
	}
	panic("cannot take address of expression")
}

// translateField translates struct field access (s.f).
// This becomes address computation + Eload.
func (t *ExprTranslator) translateField(e clight.Efield) csharpminor.Expr {
	addr := t.TranslateFieldAddr(e)
	chunk := csharpminor.ChunkForType(e.Typ)
	return csharpminor.Eload{Chunk: chunk, Addr: addr}
}

// TranslateFieldAddr computes the address of a struct field.
func (t *ExprTranslator) TranslateFieldAddr(e clight.Efield) csharpminor.Expr {
	// Get the address of the base struct
	baseAddr := t.translateLvalueAddr(e.Arg)

	// Get the struct type to find field offset
	baseType := e.Arg.ExprType()
	offset := fieldOffset(baseType, e.FieldName)

	if offset == 0 {
		return baseAddr
	}

	// Add offset: baseAddr + offset
	offsetExpr := csharpminor.Econst{Const: csharpminor.Olongconst{Value: offset}}
	return csharpminor.Ebinop{
		Op:    csharpminor.Oaddl,
		Left:  baseAddr,
		Right: offsetExpr,
	}
}

// translateLvalueAddr translates an l-value to its address.
func (t *ExprTranslator) translateLvalueAddr(e clight.Expr) csharpminor.Expr {
	switch expr := e.(type) {
	case clight.Evar:
		return csharpminor.Eaddrof{Name: expr.Name}
	case clight.Ederef:
		return t.TranslateExpr(expr.Ptr)
	case clight.Efield:
		return t.TranslateFieldAddr(expr)
	}
	panic("not an l-value")
}

// translateSizeof translates sizeof(type) to a constant.
func (t *ExprTranslator) translateSizeof(e clight.Esizeof) csharpminor.Expr {
	size := sizeofType(e.ArgType)
	return csharpminor.Econst{Const: csharpminor.Ointconst{Value: int32(size)}}
}

// translateAlignof translates alignof(type) to a constant.
func (t *ExprTranslator) translateAlignof(e clight.Ealignof) csharpminor.Expr {
	align := alignofType(e.ArgType)
	return csharpminor.Econst{Const: csharpminor.Ointconst{Value: int32(align)}}
}

// --- Helper functions for type layout ---

// sizeofType returns the size of a type in bytes.
func sizeofType(t ctypes.Type) int64 {
	switch typ := t.(type) {
	case ctypes.Tvoid:
		return 1 // void has size 1 in CompCert
	case ctypes.Tint:
		switch typ.Size {
		case ctypes.I8:
			return 1
		case ctypes.I16:
			return 2
		case ctypes.I32, ctypes.IBool:
			return 4
		}
	case ctypes.Tlong:
		return 8
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return 4
		}
		return 8
	case ctypes.Tpointer:
		return 8 // 64-bit pointers on aarch64
	case ctypes.Tarray:
		if typ.Size < 0 {
			return 0 // incomplete array
		}
		return typ.Size * sizeofType(typ.Elem)
	case ctypes.Tstruct:
		return sizeofStruct(typ)
	case ctypes.Tunion:
		return sizeofUnion(typ)
	}
	return 4 // default
}

// alignofType returns the alignment of a type in bytes.
func alignofType(t ctypes.Type) int64 {
	switch typ := t.(type) {
	case ctypes.Tvoid:
		return 1
	case ctypes.Tint:
		switch typ.Size {
		case ctypes.I8:
			return 1
		case ctypes.I16:
			return 2
		case ctypes.I32, ctypes.IBool:
			return 4
		}
	case ctypes.Tlong:
		return 8
	case ctypes.Tfloat:
		if typ.Size == ctypes.F32 {
			return 4
		}
		return 8
	case ctypes.Tpointer:
		return 8
	case ctypes.Tarray:
		return alignofType(typ.Elem)
	case ctypes.Tstruct:
		return alignofStruct(typ)
	case ctypes.Tunion:
		return alignofUnion(typ)
	}
	return 4
}

// sizeofStruct computes the size of a struct with padding.
func sizeofStruct(s ctypes.Tstruct) int64 {
	var size int64
	for _, f := range s.Fields {
		align := alignofType(f.Type)
		size = alignUp(size, align)
		size += sizeofType(f.Type)
	}
	// Final alignment
	structAlign := alignofStruct(s)
	return alignUp(size, structAlign)
}

// alignofStruct returns the alignment of a struct.
func alignofStruct(s ctypes.Tstruct) int64 {
	var maxAlign int64 = 1
	for _, f := range s.Fields {
		a := alignofType(f.Type)
		if a > maxAlign {
			maxAlign = a
		}
	}
	return maxAlign
}

// sizeofUnion computes the size of a union.
func sizeofUnion(u ctypes.Tunion) int64 {
	var maxSize int64
	for _, f := range u.Fields {
		sz := sizeofType(f.Type)
		if sz > maxSize {
			maxSize = sz
		}
	}
	// Align to union alignment
	return alignUp(maxSize, alignofUnion(u))
}

// alignofUnion returns the alignment of a union.
func alignofUnion(u ctypes.Tunion) int64 {
	var maxAlign int64 = 1
	for _, f := range u.Fields {
		a := alignofType(f.Type)
		if a > maxAlign {
			maxAlign = a
		}
	}
	return maxAlign
}

// fieldOffset computes the offset of a field within a struct.
func fieldOffset(t ctypes.Type, fieldName string) int64 {
	s, ok := t.(ctypes.Tstruct)
	if !ok {
		return 0
	}

	var offset int64
	for _, f := range s.Fields {
		align := alignofType(f.Type)
		offset = alignUp(offset, align)
		if f.Name == fieldName {
			return offset
		}
		offset += sizeofType(f.Type)
	}
	return 0 // field not found
}

// alignUp rounds n up to the nearest multiple of align.
func alignUp(n, align int64) int64 {
	if align == 0 {
		return n
	}
	return (n + align - 1) / align * align
}
