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
