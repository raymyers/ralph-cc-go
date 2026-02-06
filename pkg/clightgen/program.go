// Package clightgen implements the transformation from Cabs (CompCert C) to Clight.
// This includes the SimplExpr and SimplLocals passes.
package clightgen

import (
	"github.com/raymyers/ralph-cc/pkg/cabs"
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
	"github.com/raymyers/ralph-cc/pkg/simplexpr"
	"github.com/raymyers/ralph-cc/pkg/simpllocals"
)

// TranslateProgram transforms a Cabs program to a Clight program.
func TranslateProgram(prog *cabs.Program) *clight.Program {
	result := &clight.Program{}

	// First pass: collect struct and union definitions
	structDefs := make(map[string]ctypes.Tstruct)
	for _, def := range prog.Definitions {
		switch d := def.(type) {
		case cabs.StructDef:
			s := ctypes.Tstruct{
				Name:   d.Name,
				Fields: make([]ctypes.Field, len(d.Fields)),
			}
			for i, f := range d.Fields {
				s.Fields[i] = ctypes.Field{
					Name: f.Name,
					Type: TypeFromString(f.TypeSpec),
				}
			}
			result.Structs = append(result.Structs, s)
			structDefs[s.Name] = s
		case cabs.UnionDef:
			u := ctypes.Tunion{
				Name:   d.Name,
				Fields: make([]ctypes.Field, len(d.Fields)),
			}
			for i, f := range d.Fields {
				u.Fields[i] = ctypes.Field{
					Name: f.Name,
					Type: TypeFromString(f.TypeSpec),
				}
			}
			result.Unions = append(result.Unions, u)
		}
	}

	// Second pass: collect global variable types first
	globalTypes := make(map[string]ctypes.Type)
	for _, def := range prog.Definitions {
		if d, ok := def.(cabs.VarDef); ok {
			// Skip extern declarations without initializer (they're just declarations)
			if d.StorageClass == "extern" && d.Initializer == nil {
				continue
			}
			typ := TypeFromString(d.TypeSpec)
			globalTypes[d.Name] = typ
			var init []byte
			if d.Initializer != nil {
				init = evaluateConstantInitializer(d.Initializer, typ)
			}
			result.Globals = append(result.Globals, clight.VarDecl{
				Name: d.Name,
				Type: typ,
				Init: init,
			})
		}
	}

	// Third pass: translate functions with global type information
	for _, def := range prog.Definitions {
		if d, ok := def.(cabs.FunDef); ok {
			// Skip function declarations (prototypes) - only process definitions with bodies
			if d.Body == nil {
				continue
			}
			fn := translateFunctionWithStructsAndGlobals(&d, structDefs, globalTypes)
			result.Functions = append(result.Functions, fn)
		}
	}

	return result
}

// translateFunction transforms a Cabs function to a Clight function.
// Deprecated: use translateFunctionWithStructsAndGlobals instead.
func translateFunction(fn *cabs.FunDef) clight.Function {
	return translateFunctionWithStructsAndGlobals(fn, nil, nil)
}

// translateFunctionWithStructs transforms a Cabs function to a Clight function,
// using the provided struct definitions for field resolution.
// Deprecated: use translateFunctionWithStructsAndGlobals instead.
func translateFunctionWithStructs(fn *cabs.FunDef, structDefs map[string]ctypes.Tstruct) clight.Function {
	return translateFunctionWithStructsAndGlobals(fn, structDefs, nil)
}

// translateFunctionWithStructsAndGlobals transforms a Cabs function to a Clight function,
// using the provided struct definitions for field resolution and global variable types.
func translateFunctionWithStructsAndGlobals(fn *cabs.FunDef, structDefs map[string]ctypes.Tstruct, globalTypes map[string]ctypes.Type) clight.Function {
	// Create transformers
	simplExpr := simplexpr.New()
	simplLoc := simpllocals.New()

	// Register struct definitions for field resolution
	for _, s := range structDefs {
		simplExpr.SetStructDef(s)
	}

	// Register global variable types
	for name, typ := range globalTypes {
		simplExpr.SetType(name, typ)
	}

	// Set up type environment for parameters
	for _, param := range fn.Params {
		typ := TypeFromString(param.TypeSpec)
		simplExpr.SetType(param.Name, typ)
	}

	// Analyze the function for address-taken variables
	if fn.Body != nil {
		simplLoc.AnalyzeFunction(fn)
	}

	// Collect local variables from the body
	var locals []clight.VarDecl
	if fn.Body != nil {
		collectLocals(fn.Body, &locals, simplExpr)
	}

	// Analyze which locals can be promoted to temps
	localInfos := simplLoc.AnalyzeLocals(locals)
	remainingLocals := simpllocals.FilterUnpromotedLocals(localInfos)

	// Continue temp IDs from simpllocals
	simplExpr.Reset()
	// Re-register struct definitions after reset
	for _, s := range structDefs {
		simplExpr.SetStructDef(s)
	}
	// Re-register global types after reset
	for name, typ := range globalTypes {
		simplExpr.SetType(name, typ)
	}
	for _, param := range fn.Params {
		simplExpr.SetType(param.Name, TypeFromString(param.TypeSpec))
	}

	// Set starting temp ID after simpllocals temps to avoid collision
	nextTemp := 1
	for _, info := range localInfos {
		if info.Promoted && info.TempID >= nextTemp {
			nextTemp = info.TempID + 1
		}
	}
	simplExpr.SetNextTempID(nextTemp)

	// Transform the body
	var body clight.Stmt = clight.Sskip{}
	if fn.Body != nil {
		body = transformBlock(fn.Body, simplExpr)
	}

	// Apply simpllocals transformation to the body
	body = simplLoc.TransformStmt(body)

	// Collect temp types
	var temps []ctypes.Type
	temps = append(temps, simplLoc.TempTypes()...)
	temps = append(temps, simplExpr.TempTypes()...)

	// Build params
	params := make([]clight.VarDecl, len(fn.Params))
	for i, p := range fn.Params {
		params[i] = clight.VarDecl{
			Name: p.Name,
			Type: TypeFromString(p.TypeSpec),
		}
	}

	return clight.Function{
		Name:   fn.Name,
		Return: TypeFromString(fn.ReturnType),
		Params: params,
		Locals: remainingLocals,
		Temps:  temps,
		Body:   body,
	}
}

// collectLocals extracts local variable declarations from a block.
func collectLocals(block *cabs.Block, locals *[]clight.VarDecl, simplExpr *simplexpr.Transformer) {
	for _, item := range block.Items {
		collectLocalsFromStmt(item, locals, simplExpr)
	}
}

// collectLocalsFromStmt extracts local variable declarations from a statement.
func collectLocalsFromStmt(item cabs.Stmt, locals *[]clight.VarDecl, simplExpr *simplexpr.Transformer) {
	switch s := item.(type) {
	case cabs.DeclStmt:
		for _, decl := range s.Decls {
			typ := TypeFromString(decl.TypeSpec)
			// Resolve struct types to include field information
			if st, ok := typ.(ctypes.Tstruct); ok {
				typ = simplExpr.ResolveStruct(st)
			}
			// Handle array declarations
			if len(decl.ArrayDims) > 0 {
				// Build array type from innermost to outermost dimension
				for i := len(decl.ArrayDims) - 1; i >= 0; i-- {
					dim := decl.ArrayDims[i]
					size := int64(-1) // default: incomplete array
					if c, ok := dim.(cabs.Constant); ok {
						size = c.Value
					}
					typ = ctypes.Tarray{Elem: typ, Size: size}
				}
			}
			simplExpr.SetType(decl.Name, typ)
			*locals = append(*locals, clight.VarDecl{
				Name: decl.Name,
				Type: typ,
			})
		}
	case cabs.Block:
		collectLocals(&s, locals, simplExpr)
	case *cabs.Block:
		collectLocals(s, locals, simplExpr)
	case cabs.For:
		// C99 for-loop declarations
		for _, decl := range s.InitDecl {
			typ := TypeFromString(decl.TypeSpec)
			// Resolve struct types to include field information
			if st, ok := typ.(ctypes.Tstruct); ok {
				typ = simplExpr.ResolveStruct(st)
			}
			simplExpr.SetType(decl.Name, typ)
			*locals = append(*locals, clight.VarDecl{
				Name: decl.Name,
				Type: typ,
			})
		}
		// Recurse into body
		collectLocalsFromStmt(s.Body, locals, simplExpr)
	case cabs.While:
		collectLocalsFromStmt(s.Body, locals, simplExpr)
	case cabs.DoWhile:
		collectLocalsFromStmt(s.Body, locals, simplExpr)
	case cabs.If:
		collectLocalsFromStmt(s.Then, locals, simplExpr)
		if s.Else != nil {
			collectLocalsFromStmt(s.Else, locals, simplExpr)
		}
	case cabs.Switch:
		for _, c := range s.Cases {
			for _, stmt := range c.Stmts {
				collectLocalsFromStmt(stmt, locals, simplExpr)
			}
		}
	}
}

// transformBlock transforms a Cabs block to a Clight statement.
func transformBlock(block *cabs.Block, simplExpr *simplexpr.Transformer) clight.Stmt {
	var stmts []clight.Stmt
	for _, item := range block.Items {
		stmt := transformStmt(item, simplExpr)
		stmts = append(stmts, stmt)
	}
	return clight.Seq(stmts...)
}

// evaluateConstantInitializer evaluates a constant expression to bytes.
// For now, handles simple integer constants only.
func evaluateConstantInitializer(expr cabs.Expr, typ ctypes.Type) []byte {
	size := SizeofType(typ)
	switch e := expr.(type) {
	case cabs.Paren:
		// Unwrap parenthesized expressions: (-3) -> -3
		return evaluateConstantInitializer(e.Expr, typ)
	case cabs.Constant:
		val := e.Value
		result := make([]byte, size)
		// Write little-endian integer
		for i := int64(0); i < size; i++ {
			result[i] = byte(val & 0xff)
			val >>= 8
		}
		return result
	case cabs.Unary:
		// Handle negative constants: -42
		if e.Op == cabs.OpNeg {
			if c, ok := e.Expr.(cabs.Constant); ok {
				negVal := -c.Value
				result := make([]byte, size)
				for i := int64(0); i < size; i++ {
					result[i] = byte(negVal & 0xff)
					negVal >>= 8
				}
				return result
			}
		}
	}
	return nil
}
