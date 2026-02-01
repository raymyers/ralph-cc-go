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

	for _, def := range prog.Definitions {
		switch d := def.(type) {
		case cabs.FunDef:
			fn := translateFunction(&d)
			result.Functions = append(result.Functions, fn)
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

	return result
}

// translateFunction transforms a Cabs function to a Clight function.
func translateFunction(fn *cabs.FunDef) clight.Function {
	// Create transformers
	simplExpr := simplexpr.New()
	simplLoc := simpllocals.New()

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
	for _, param := range fn.Params {
		simplExpr.SetType(param.Name, TypeFromString(param.TypeSpec))
	}

	// Set starting temp ID after simpllocals temps
	nextTemp := 1
	for _, info := range localInfos {
		if info.Promoted && info.TempID >= nextTemp {
			nextTemp = info.TempID + 1
		}
	}

	// Transform the body
	var body clight.Stmt = clight.Sskip{}
	if fn.Body != nil {
		body = transformBlock(fn.Body, simplExpr)
	}

	// Apply simpllocals transformation to the body
	body = simplLoc.TransformStmt(body)

	// Collect temp types
	var temps []ctypes.Type
	temps = append(temps, simplExpr.TempTypes()...)
	temps = append(temps, simplLoc.TempTypes()...)

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
		switch s := item.(type) {
		case cabs.DeclStmt:
			for _, decl := range s.Decls {
				typ := TypeFromString(decl.TypeSpec)
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
