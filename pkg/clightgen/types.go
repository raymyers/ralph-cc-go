package clightgen

import (
	"strings"

	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

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
	case "long":
		return ctypes.Long()
	case "unsigned long":
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
