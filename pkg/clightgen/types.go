package clightgen

import (
	"strings"

	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

// SizeofType returns the size in bytes for a given type.
func SizeofType(t ctypes.Type) int64 {
	switch t := t.(type) {
	case ctypes.Tvoid:
		return 0
	case ctypes.Tint:
		switch t.Size {
		case ctypes.I8:
			return 1
		case ctypes.I16:
			return 2
		case ctypes.I32:
			return 4
		}
		return 4
	case ctypes.Tlong:
		return 8
	case ctypes.Tfloat:
		switch t.Size {
		case ctypes.F32:
			return 4
		case ctypes.F64:
			return 8
		}
		return 8
	case ctypes.Tpointer:
		return 8 // 64-bit pointers
	case ctypes.Tarray:
		return t.Size * SizeofType(t.Elem)
	case ctypes.Tstruct:
		var total int64
		for _, f := range t.Fields {
			total += SizeofType(f.Type)
		}
		return total
	case ctypes.Tunion:
		var maxSize int64
		for _, f := range t.Fields {
			if sz := SizeofType(f.Type); sz > maxSize {
				maxSize = sz
			}
		}
		return maxSize
	default:
		return 4 // default to int size
	}
}

// TypeFromString converts a C type string to a ctypes.Type.
func TypeFromString(typeName string) ctypes.Type {
	// Remove any leading/trailing whitespace
	typeName = strings.TrimSpace(typeName)

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
	case "long", "long long", "signed long long":
		return ctypes.Long()
	case "unsigned long", "unsigned long long":
		return ctypes.Tlong{Sign: ctypes.Unsigned}
	case "float":
		return ctypes.Float()
	case "double":
		return ctypes.Double()
	default:
		// Check for pointer types
		if strings.HasSuffix(typeName, "*") {
			baseType := TypeFromString(strings.TrimSpace(typeName[:len(typeName)-1]))
			return ctypes.Pointer(baseType)
		}
		// Check for struct types
		if strings.HasPrefix(typeName, "struct ") {
			structName := strings.TrimPrefix(typeName, "struct ")
			return ctypes.Tstruct{Name: strings.TrimSpace(structName)}
		}
		// Check for union types
		if strings.HasPrefix(typeName, "union ") {
			unionName := strings.TrimPrefix(typeName, "union ")
			return ctypes.Tunion{Name: strings.TrimSpace(unionName)}
		}
		return ctypes.Int() // default fallback
	}
}
