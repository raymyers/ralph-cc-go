// Package cminorgen implements the Cminorgen pass: Csharpminor → Cminor
// This file handles variable classification and transformation.
package cminorgen

import (
	"github.com/raymyers/ralph-cc/pkg/cminor"
	"github.com/raymyers/ralph-cc/pkg/csharpminor"
)

// VarKind represents where a variable is stored
type VarKind int

const (
	VarRegister VarKind = iota // Not address-taken, can stay in register
	VarStack                   // Address-taken, must be on stack
)

// VarInfo holds classification and location info for a variable
type VarInfo struct {
	Name   string
	Kind   VarKind
	Size   int64
	Offset int64 // Stack offset (only valid if Kind == VarStack)
	Chunk  cminor.Chunk
}

// VarEnv holds the variable environment for a function transformation
type VarEnv struct {
	Vars       map[string]*VarInfo // All local variables
	StackSize  int64               // Total stack frame size
	TempPrefix string              // Prefix for generated temp names
}

// ClassifyVariables analyzes locals and the function body to classify variables.
// Address-taken variables go to stack, others can stay in registers.
func ClassifyVariables(locals []csharpminor.VarDecl, body csharpminor.Stmt) *VarEnv {
	env := &VarEnv{
		Vars:       make(map[string]*VarInfo),
		TempPrefix: "_t",
	}

	// Find all address-taken variables
	addrTaken := FindAddressTaken(body, locals)

	// Build initial var info map
	for _, local := range locals {
		kind := VarRegister
		if addrTaken[local.Name] {
			kind = VarStack
		}
		env.Vars[local.Name] = &VarInfo{
			Name:  local.Name,
			Kind:  kind,
			Size:  local.Size,
			Chunk: chunkForSize(local.Size),
		}
	}

	// Compute stack layout for address-taken variables
	var stackLocals []csharpminor.VarDecl
	for _, local := range locals {
		if env.Vars[local.Name].Kind == VarStack {
			stackLocals = append(stackLocals, local)
		}
	}

	layout := ComputeStackLayout(stackLocals)
	env.StackSize = layout.TotalSize

	// Update offsets for stack variables
	for i := range layout.Slots {
		slot := &layout.Slots[i]
		if info, ok := env.Vars[slot.Name]; ok {
			info.Offset = slot.Offset
		}
	}

	return env
}

// chunkForSize returns the appropriate memory chunk for a given byte size.
// Defaults to signed for backward compatibility.
func chunkForSize(size int64) cminor.Chunk {
	return chunkForSizeAndSign(size, true)
}

// chunkForSizeAndSign returns the appropriate memory chunk for a given byte size and signedness.
func chunkForSizeAndSign(size int64, signed bool) cminor.Chunk {
	switch size {
	case 1:
		if signed {
			return cminor.Mint8signed
		}
		return cminor.Mint8unsigned
	case 2:
		if signed {
			return cminor.Mint16signed
		}
		return cminor.Mint16unsigned
	case 4:
		return cminor.Mint32
	case 8:
		return cminor.Mint64
	default:
		return cminor.Many64 // for arrays/structs, handle element-wise
	}
}

// IsRegister returns true if the variable is a register candidate.
func (env *VarEnv) IsRegister(name string) bool {
	if info, ok := env.Vars[name]; ok {
		return info.Kind == VarRegister
	}
	return false
}

// IsStack returns true if the variable is stack-allocated.
func (env *VarEnv) IsStack(name string) bool {
	if info, ok := env.Vars[name]; ok {
		return info.Kind == VarStack
	}
	return false
}

// GetStackOffset returns the stack offset for a stack variable.
// Returns -1 if not found or not a stack variable.
func (env *VarEnv) GetStackOffset(name string) int64 {
	if info, ok := env.Vars[name]; ok && info.Kind == VarStack {
		return info.Offset
	}
	return -1
}

// GetChunk returns the memory chunk for a variable.
func (env *VarEnv) GetChunk(name string) cminor.Chunk {
	if info, ok := env.Vars[name]; ok {
		return info.Chunk
	}
	return cminor.Many32
}

// RegisterVars returns a list of variable names that are register candidates.
func (env *VarEnv) RegisterVars() []string {
	var result []string
	for name, info := range env.Vars {
		if info.Kind == VarRegister {
			result = append(result, name)
		}
	}
	return result
}

// StackVars returns a list of variable names that are stack-allocated.
func (env *VarEnv) StackVars() []string {
	var result []string
	for name, info := range env.Vars {
		if info.Kind == VarStack {
			result = append(result, name)
		}
	}
	return result
}

// TransformAddrOf transforms address-of on a local variable to a stack address.
// For stack variable "x" at offset 8, &x becomes Oaddrstack{Offset: 8}.
func (env *VarEnv) TransformAddrOf(name string) cminor.Expr {
	offset := env.GetStackOffset(name)
	if offset < 0 {
		panic("TransformAddrOf called on non-stack variable: " + name)
	}
	return cminor.Econst{Const: cminor.Oaddrstack{Offset: offset}}
}

// TransformVarRead transforms a read of a local variable.
// - Register vars: simple Evar reference
// - Stack vars: Eload from stack address
func (env *VarEnv) TransformVarRead(name string) cminor.Expr {
	info, ok := env.Vars[name]
	if !ok {
		// Not a local, must be global
		return cminor.Evar{Name: name}
	}

	if info.Kind == VarRegister {
		return cminor.Evar{Name: name}
	}

	// Stack variable: load from stack offset
	return cminor.Eload{
		Chunk: info.Chunk,
		Addr:  cminor.Econst{Const: cminor.Oaddrstack{Offset: info.Offset}},
	}
}

// TransformVarWrite transforms a write to a local variable.
// - Register vars: Sassign(name, value)
// - Stack vars: Sstore(chunk, stackaddr, value)
func (env *VarEnv) TransformVarWrite(name string, value cminor.Expr) cminor.Stmt {
	info, ok := env.Vars[name]
	if !ok {
		// Not a local, treat as register-like (could be param)
		return cminor.Sassign{Name: name, RHS: value}
	}

	if info.Kind == VarRegister {
		return cminor.Sassign{Name: name, RHS: value}
	}

	// Stack variable: store to stack offset
	return cminor.Sstore{
		Chunk: info.Chunk,
		Addr:  cminor.Econst{Const: cminor.Oaddrstack{Offset: info.Offset}},
		Value: value,
	}
}
