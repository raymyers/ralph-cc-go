// Package ctypes defines the C type system, mirroring CompCert's Ctypes.v
package ctypes

// Type is the interface for all C types
type Type interface {
	implType()
	String() string
}

// Signedness represents signed/unsigned for integer types
type Signedness int

const (
	Signed Signedness = iota
	Unsigned
)

func (s Signedness) String() string {
	if s == Signed {
		return "signed"
	}
	return "unsigned"
}

// IntSize represents the size of integer types
type IntSize int

const (
	I8 IntSize = iota
	I16
	I32
	IBool
)

func (s IntSize) String() string {
	names := []string{"i8", "i16", "i32", "ibool"}
	if int(s) < len(names) {
		return names[s]
	}
	return "?"
}

// FloatSize represents the size of floating-point types
type FloatSize int

const (
	F32 FloatSize = iota
	F64
)

func (s FloatSize) String() string {
	if s == F32 {
		return "f32"
	}
	return "f64"
}

// Tvoid represents the void type
type Tvoid struct{}

// Tint represents integer types (char, short, int, _Bool)
type Tint struct {
	Size IntSize
	Sign Signedness
}

// Tlong represents the long type (64-bit)
type Tlong struct {
	Sign Signedness
}

// Tfloat represents floating-point types (float, double)
type Tfloat struct {
	Size FloatSize
}

// Tpointer represents pointer types
type Tpointer struct {
	Elem Type
}

// Tarray represents array types
type Tarray struct {
	Elem Type
	Size int64 // -1 for incomplete array
}

// Tfunction represents function types
type Tfunction struct {
	Params []Type
	Return Type
	VarArg bool
}

// Tstruct represents struct types
type Tstruct struct {
	Name   string
	Fields []Field
}

// Tunion represents union types
type Tunion struct {
	Name   string
	Fields []Field
}

// Field represents a struct or union field
type Field struct {
	Name string
	Type Type
}

// Marker methods for Type interface
func (Tvoid) implType()     {}
func (Tint) implType()      {}
func (Tlong) implType()     {}
func (Tfloat) implType()    {}
func (Tpointer) implType()  {}
func (Tarray) implType()    {}
func (Tfunction) implType() {}
func (Tstruct) implType()   {}
func (Tunion) implType()    {}

// String methods for types
func (Tvoid) String() string { return "void" }

func (t Tint) String() string {
	sign := ""
	if t.Sign == Unsigned {
		sign = "unsigned "
	}
	switch t.Size {
	case I8:
		return sign + "char"
	case I16:
		return sign + "short"
	case I32:
		return sign + "int"
	case IBool:
		return "_Bool"
	}
	return sign + "int"
}

func (t Tlong) String() string {
	if t.Sign == Unsigned {
		return "unsigned long"
	}
	return "long"
}

func (t Tfloat) String() string {
	if t.Size == F32 {
		return "float"
	}
	return "double"
}

func (t Tpointer) String() string {
	if t.Elem == nil {
		return "void *"
	}
	return t.Elem.String() + " *"
}

func (t Tarray) String() string {
	if t.Elem == nil {
		return "?[]"
	}
	if t.Size < 0 {
		return t.Elem.String() + "[]"
	}
	return t.Elem.String() + "[...]"
}

func (t Tfunction) String() string {
	return "function"
}

func (t Tstruct) String() string {
	if t.Name == "" {
		return "struct <anonymous>"
	}
	return "struct " + t.Name
}

func (t Tunion) String() string {
	if t.Name == "" {
		return "union <anonymous>"
	}
	return "union " + t.Name
}

// Common type constructors

// Int returns a signed 32-bit int type
func Int() Type {
	return Tint{Size: I32, Sign: Signed}
}

// UInt returns an unsigned 32-bit int type
func UInt() Type {
	return Tint{Size: I32, Sign: Unsigned}
}

// Char returns a signed char type
func Char() Type {
	return Tint{Size: I8, Sign: Signed}
}

// UChar returns an unsigned char type
func UChar() Type {
	return Tint{Size: I8, Sign: Unsigned}
}

// Short returns a signed short type
func Short() Type {
	return Tint{Size: I16, Sign: Signed}
}

// Long returns a signed long type
func Long() Type {
	return Tlong{Sign: Signed}
}

// Float returns a float (32-bit) type
func Float() Type {
	return Tfloat{Size: F32}
}

// Double returns a double (64-bit) type
func Double() Type {
	return Tfloat{Size: F64}
}

// Void returns the void type
func Void() Type {
	return Tvoid{}
}

// Pointer returns a pointer to the given type
func Pointer(elem Type) Type {
	return Tpointer{Elem: elem}
}

// Array returns an array type
func Array(elem Type, size int64) Type {
	return Tarray{Elem: elem, Size: size}
}

// Equal checks if two types are equal
func Equal(a, b Type) bool {
	if a == nil || b == nil {
		return a == b
	}
	switch ta := a.(type) {
	case Tvoid:
		_, ok := b.(Tvoid)
		return ok
	case Tint:
		tb, ok := b.(Tint)
		return ok && ta.Size == tb.Size && ta.Sign == tb.Sign
	case Tlong:
		tb, ok := b.(Tlong)
		return ok && ta.Sign == tb.Sign
	case Tfloat:
		tb, ok := b.(Tfloat)
		return ok && ta.Size == tb.Size
	case Tpointer:
		tb, ok := b.(Tpointer)
		return ok && Equal(ta.Elem, tb.Elem)
	case Tarray:
		tb, ok := b.(Tarray)
		return ok && ta.Size == tb.Size && Equal(ta.Elem, tb.Elem)
	case Tstruct:
		tb, ok := b.(Tstruct)
		return ok && ta.Name == tb.Name
	case Tunion:
		tb, ok := b.(Tunion)
		return ok && ta.Name == tb.Name
	case Tfunction:
		tb, ok := b.(Tfunction)
		if !ok || ta.VarArg != tb.VarArg || len(ta.Params) != len(tb.Params) {
			return false
		}
		if !Equal(ta.Return, tb.Return) {
			return false
		}
		for i, p := range ta.Params {
			if !Equal(p, tb.Params[i]) {
				return false
			}
		}
		return true
	}
	return false
}
