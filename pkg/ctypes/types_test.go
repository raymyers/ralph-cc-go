package ctypes

import "testing"

func TestTypeConstructors(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		wantStr  string
	}{
		{"void", Void(), "void"},
		{"int", Int(), "int"},
		{"unsigned int", UInt(), "unsigned int"},
		{"char", Char(), "char"},
		{"unsigned char", UChar(), "unsigned char"},
		{"short", Short(), "short"},
		{"long", Long(), "long"},
		{"float", Float(), "float"},
		{"double", Double(), "double"},
		{"pointer to int", Pointer(Int()), "int *"},
		{"pointer to void", Pointer(Void()), "void *"},
		{"array of int", Array(Int(), 10), "int[...]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.wantStr {
				t.Errorf("String() = %q, want %q", got, tt.wantStr)
			}
		})
	}
}

func TestTypeEquality(t *testing.T) {
	tests := []struct {
		name  string
		a, b  Type
		equal bool
	}{
		{"int == int", Int(), Int(), true},
		{"int != unsigned int", Int(), UInt(), false},
		{"int != long", Int(), Long(), false},
		{"int != void", Int(), Void(), false},
		{"void == void", Void(), Void(), true},
		{"pointer to int == pointer to int", Pointer(Int()), Pointer(Int()), true},
		{"pointer to int != pointer to char", Pointer(Int()), Pointer(Char()), false},
		{"array[10] of int == array[10] of int", Array(Int(), 10), Array(Int(), 10), true},
		{"array[10] of int != array[20] of int", Array(Int(), 10), Array(Int(), 20), false},
		{"struct A == struct A", Tstruct{Name: "A"}, Tstruct{Name: "A"}, true},
		{"struct A != struct B", Tstruct{Name: "A"}, Tstruct{Name: "B"}, false},
		{"nil == nil", nil, nil, true},
		{"nil != int", nil, Int(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Equal(tt.a, tt.b); got != tt.equal {
				t.Errorf("Equal(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.equal)
			}
		})
	}
}

func TestFunctionTypeEquality(t *testing.T) {
	fn1 := Tfunction{Params: []Type{Int(), Int()}, Return: Int()}
	fn2 := Tfunction{Params: []Type{Int(), Int()}, Return: Int()}
	fn3 := Tfunction{Params: []Type{Int()}, Return: Int()}
	fn4 := Tfunction{Params: []Type{Int(), Int()}, Return: Void()}

	if !Equal(fn1, fn2) {
		t.Error("identical function types should be equal")
	}
	if Equal(fn1, fn3) {
		t.Error("functions with different param counts should not be equal")
	}
	if Equal(fn1, fn4) {
		t.Error("functions with different return types should not be equal")
	}
}

func TestSignednessString(t *testing.T) {
	if Signed.String() != "signed" {
		t.Errorf("Signed.String() = %q, want %q", Signed.String(), "signed")
	}
	if Unsigned.String() != "unsigned" {
		t.Errorf("Unsigned.String() = %q, want %q", Unsigned.String(), "unsigned")
	}
}

func TestIntSizeString(t *testing.T) {
	tests := []struct {
		size IntSize
		want string
	}{
		{I8, "i8"},
		{I16, "i16"},
		{I32, "i32"},
		{IBool, "ibool"},
	}
	for _, tt := range tests {
		if got := tt.size.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestFloatSizeString(t *testing.T) {
	if F32.String() != "f32" {
		t.Errorf("F32.String() = %q, want %q", F32.String(), "f32")
	}
	if F64.String() != "f64" {
		t.Errorf("F64.String() = %q, want %q", F64.String(), "f64")
	}
}
