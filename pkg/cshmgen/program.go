// Package cshmgen implements the Cshmgen pass: Clight → Csharpminor
// This file handles program-level translation.
package cshmgen

import (
	"github.com/raymyers/ralph-cc/pkg/clight"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
	"github.com/raymyers/ralph-cc/pkg/ctypes"
)

// TranslateProgram translates a complete Clight program to Csharpminor.
func TranslateProgram(prog *clight.Program) *csharpminor.Program {
	result := &csharpminor.Program{}

	// Build struct definitions map for type resolution
	structDefs := make(map[string]ctypes.Tstruct)
	for _, s := range prog.Structs {
		structDefs[s.Name] = s
	}

	// Build global variable set
	globals := make(map[string]bool)
	for _, g := range prog.Globals {
		globals[g.Name] = true
	}

	// Translate global variables
	for _, g := range prog.Globals {
		typ := resolveStructType(g.Type, structDefs)
		size := sizeofType(typ)
		result.Globals = append(result.Globals, csharpminor.VarDecl{
			Name: g.Name,
			Size: size,
			Init: g.Init,
		})
	}

	// Create a shared expression translator to collect strings across all functions
	exprTr := NewExprTranslator(globals)

	// Translate functions
	for _, fn := range prog.Functions {
		csharpFn := translateFunctionWithStructs(&fn, exprTr, structDefs)
		result.Functions = append(result.Functions, csharpFn)
	}

	// Add collected string literals as read-only globals
	for _, str := range exprTr.GetStrings() {
		// String data with null terminator
		data := append([]byte(str.Value), 0)
		result.Globals = append(result.Globals, csharpminor.VarDecl{
			Name:     str.Label,
			Size:     int64(len(data)),
			Init:     data,
			ReadOnly: true,
		})
	}

	return result
}

// resolveStructType returns a struct type with its fields if available.
func resolveStructType(t ctypes.Type, defs map[string]ctypes.Tstruct) ctypes.Type {
	if s, ok := t.(ctypes.Tstruct); ok {
		if def, found := defs[s.Name]; found {
			return def
		}
	}
	return t
}

// translateFunction translates a single function from Clight to Csharpminor.
func translateFunction(fn *clight.Function, globals map[string]bool) csharpminor.Function {
	// Build expression and statement translators
	exprTr := NewExprTranslator(globals)
	return translateFunctionWithStructs(fn, exprTr, nil)
}

// translateFunctionWithTranslator translates a function using the provided translator.
// This allows string literals to be collected across all functions.
func translateFunctionWithTranslator(fn *clight.Function, exprTr *ExprTranslator) csharpminor.Function {
	return translateFunctionWithStructs(fn, exprTr, nil)
}

// translateFunctionWithStructs translates a function with struct definitions for type resolution.
func translateFunctionWithStructs(fn *clight.Function, exprTr *ExprTranslator, structDefs map[string]ctypes.Tstruct) csharpminor.Function {
	stmtTr := NewStmtTranslator(exprTr)

	// Build signature
	sig := csharpminor.Sig{
		Return: fn.Return,
	}
	for _, p := range fn.Params {
		sig.Args = append(sig.Args, p.Type)
	}

	// Translate locals, resolving struct types
	var locals []csharpminor.VarDecl
	for _, l := range fn.Locals {
		typ := resolveStructType(l.Type, structDefs)
		size := sizeofType(typ)
		locals = append(locals, csharpminor.VarDecl{
			Name: l.Name,
			Size: size,
		})
	}

	// Build parameter names
	var params []string
	for _, p := range fn.Params {
		params = append(params, p.Name)
	}

	// Translate body
	body := stmtTr.TranslateStmt(fn.Body)

	return csharpminor.Function{
		Name:   fn.Name,
		Sig:    sig,
		Params: params,
		Locals: locals,
		Temps:  fn.Temps,
		Body:   body,
	}
}
