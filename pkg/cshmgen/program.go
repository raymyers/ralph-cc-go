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

	// Build parameter names and set them on the translator
	var params []string
	for _, p := range fn.Params {
		params = append(params, p.Name)
	}
	stmtTr.SetParams(params)
	
	// Set starting temp ID after any existing temps
	stmtTr.SetNextTempID(len(fn.Temps))
	
	// First pass: find which parameters are modified
	// We scan the body to identify assignments to parameter names
	modifiedParams := findModifiedParams(fn.Body, params)
	
	// Allocate temp IDs for modified parameters and set up the mapping
	// for both writing (in stmtTr) and reading (in exprTr)
	nextTempID := len(fn.Temps)
	paramTemps := make(map[string]int)
	for _, name := range modifiedParams {
		paramTemps[name] = nextTempID
		nextTempID++
	}
	
	// Set the param temps on both translators
	for name, id := range paramTemps {
		stmtTr.paramTemps[name] = id
	}
	stmtTr.nextTempID = nextTempID
	exprTr.SetParamTemps(paramTemps)

	// Translate body
	body := stmtTr.TranslateStmt(fn.Body)
	
	// Generate initialization code: copy original param values to temps
	// This needs to happen at the beginning of the function
	if len(paramTemps) > 0 {
		var initStmts []csharpminor.Stmt
		for name, tempID := range paramTemps {
			// Generate: temp = param
			initStmts = append(initStmts, csharpminor.Sset{
				TempID: tempID,
				RHS:    csharpminor.Evar{Name: name},
			})
		}
		// Prepend initialization to body
		body = csharpminor.Seq(append(initStmts, body)...)
	}
	
	// Clear param temps from exprTr so it doesn't affect other functions
	exprTr.SetParamTemps(make(map[string]int))

	// Extend temps list to include param shadow temps
	temps := make([]ctypes.Type, nextTempID)
	copy(temps, fn.Temps)
	// Fill in types for param temps (look up from params)
	paramTypes := make(map[string]ctypes.Type)
	for _, p := range fn.Params {
		paramTypes[p.Name] = p.Type
	}
	for name, id := range paramTemps {
		if typ, ok := paramTypes[name]; ok {
			temps[id] = typ
		}
	}

	return csharpminor.Function{
		Name:   fn.Name,
		Sig:    sig,
		Params: params,
		Locals: locals,
		Temps:  temps,
		Body:   body,
	}
}

// findModifiedParams scans a Clight statement tree to find which parameters are assigned to.
func findModifiedParams(body clight.Stmt, params []string) []string {
	paramSet := make(map[string]bool)
	for _, p := range params {
		paramSet[p] = true
	}
	
	modified := make(map[string]bool)
	scanForModifiedParams(body, paramSet, modified)
	
	var result []string
	for name := range modified {
		result = append(result, name)
	}
	return result
}

// scanForModifiedParams recursively scans statements for parameter assignments.
func scanForModifiedParams(s clight.Stmt, params map[string]bool, modified map[string]bool) {
	switch stmt := s.(type) {
	case clight.Sassign:
		if evar, ok := stmt.LHS.(clight.Evar); ok {
			if params[evar.Name] {
				modified[evar.Name] = true
			}
		}
	case clight.Ssequence:
		scanForModifiedParams(stmt.First, params, modified)
		scanForModifiedParams(stmt.Second, params, modified)
	case clight.Sifthenelse:
		scanForModifiedParams(stmt.Then, params, modified)
		scanForModifiedParams(stmt.Else, params, modified)
	case clight.Sloop:
		scanForModifiedParams(stmt.Body, params, modified)
		scanForModifiedParams(stmt.Continue, params, modified)
	case clight.Sswitch:
		for _, c := range stmt.Cases {
			scanForModifiedParams(c.Body, params, modified)
		}
		scanForModifiedParams(stmt.Default, params, modified)
	case clight.Slabel:
		scanForModifiedParams(stmt.Stmt, params, modified)
	}
}
