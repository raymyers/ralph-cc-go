// Package asm defines the ARM64 assembly representation.
// This is the final output of the compiler - actual assembly code.
// This mirrors CompCert's aarch64/Asm.v
package asm

import "github.com/raymyers/ralph-cc/pkg/ltl"

// Re-export types
type (
	MReg = ltl.MReg
	Typ  = ltl.Typ
)

// Re-export register constants
const (
	X0  = ltl.X0
	X1  = ltl.X1
	X2  = ltl.X2
	X3  = ltl.X3
	X4  = ltl.X4
	X5  = ltl.X5
	X6  = ltl.X6
	X7  = ltl.X7
	X8  = ltl.X8
	X9  = ltl.X9
	X10 = ltl.X10
	X11 = ltl.X11
	X12 = ltl.X12
	X13 = ltl.X13
	X14 = ltl.X14
	X15 = ltl.X15
	X16 = ltl.X16
	X17 = ltl.X17
	X18 = ltl.X18
	X19 = ltl.X19
	X20 = ltl.X20
	X21 = ltl.X21
	X22 = ltl.X22
	X23 = ltl.X23
	X24 = ltl.X24
	X25 = ltl.X25
	X26 = ltl.X26
	X27 = ltl.X27
	X28 = ltl.X28
	X29 = ltl.X29 // FP
	X30 = ltl.X30 // LR
	D0  = ltl.D0
	D1  = ltl.D1
	D2  = ltl.D2
	D3  = ltl.D3
	D4  = ltl.D4
	D5  = ltl.D5
	D6  = ltl.D6
	D7  = ltl.D7
	D8  = ltl.D8
	D9  = ltl.D9
	D10 = ltl.D10
	D11 = ltl.D11
	D12 = ltl.D12
	D13 = ltl.D13
	D14 = ltl.D14
	D15 = ltl.D15
)

// Label represents a branch target label
type Label string

// --- Instruction Interface ---

// Instruction is the interface for ARM64 instructions
type Instruction interface {
	implInstruction()
}

// --- Data Processing Instructions ---

// ADD - Add
type ADD struct {
	Rd, Rn, Rm MReg
	Is64       bool // true for X registers, false for W
}

// ADDi - Add immediate
type ADDi struct {
	Rd, Rn MReg
	Imm    int64
	Is64   bool
}

// SUB - Subtract
type SUB struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// SUBi - Subtract immediate
type SUBi struct {
	Rd, Rn MReg
	Imm    int64
	Is64   bool
}

// MUL - Multiply
type MUL struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// MADD - Multiply-add (Rd = Ra + Rn * Rm)
type MADD struct {
	Rd, Rn, Rm, Ra MReg
	Is64           bool
}

// SMULL - Signed multiply long (64 = 32 * 32)
type SMULL struct {
	Rd, Rn, Rm MReg
}

// UMULL - Unsigned multiply long
type UMULL struct {
	Rd, Rn, Rm MReg
}

// SDIV - Signed divide
type SDIV struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// UDIV - Unsigned divide
type UDIV struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// AND - Bitwise AND
type AND struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// ANDi - Bitwise AND immediate
type ANDi struct {
	Rd, Rn MReg
	Imm    int64
	Is64   bool
}

// ORR - Bitwise OR
type ORR struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// ORRi - Bitwise OR immediate
type ORRi struct {
	Rd, Rn MReg
	Imm    int64
	Is64   bool
}

// EOR - Bitwise exclusive OR
type EOR struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// EORi - Bitwise exclusive OR immediate
type EORi struct {
	Rd, Rn MReg
	Imm    int64
	Is64   bool
}

// MVN - Bitwise NOT (Rd = ~Rm)
type MVN struct {
	Rd, Rm MReg
	Is64   bool
}

// NEG - Negate (Rd = -Rm)
type NEG struct {
	Rd, Rm MReg
	Is64   bool
}

// --- Shift Instructions ---

// LSL - Logical shift left
type LSL struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// LSLi - Logical shift left immediate
type LSLi struct {
	Rd, Rn MReg
	Shift  int
	Is64   bool
}

// LSR - Logical shift right
type LSR struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// LSRi - Logical shift right immediate
type LSRi struct {
	Rd, Rn MReg
	Shift  int
	Is64   bool
}

// ASR - Arithmetic shift right
type ASR struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// ASRi - Arithmetic shift right immediate
type ASRi struct {
	Rd, Rn MReg
	Shift  int
	Is64   bool
}

// ROR - Rotate right
type ROR struct {
	Rd, Rn, Rm MReg
	Is64       bool
}

// RORi - Rotate right immediate
type RORi struct {
	Rd, Rn MReg
	Shift  int
	Is64   bool
}

// --- Load/Store Instructions ---

// LDR - Load register
type LDR struct {
	Rt   MReg
	Rn   MReg  // base register
	Ofs  int64 // immediate offset
	Is64 bool
}

// LDRr - Load register (register offset)
type LDRr struct {
	Rt, Rn, Rm MReg
	Is64       bool
}

// LDRB - Load byte
type LDRB struct {
	Rt  MReg
	Rn  MReg
	Ofs int64
}

// LDRH - Load halfword
type LDRH struct {
	Rt  MReg
	Rn  MReg
	Ofs int64
}

// LDRSB - Load signed byte
type LDRSB struct {
	Rt   MReg
	Rn   MReg
	Ofs  int64
	Is64 bool // extend to 64-bit
}

// LDRSH - Load signed halfword
type LDRSH struct {
	Rt   MReg
	Rn   MReg
	Ofs  int64
	Is64 bool
}

// LDRSW - Load signed word (32->64)
type LDRSW struct {
	Rt  MReg
	Rn  MReg
	Ofs int64
}

// STR - Store register
type STR struct {
	Rt   MReg
	Rn   MReg
	Ofs  int64
	Is64 bool
}

// STRr - Store register (register offset)
type STRr struct {
	Rt, Rn, Rm MReg
	Is64       bool
}

// STRB - Store byte
type STRB struct {
	Rt  MReg
	Rn  MReg
	Ofs int64
}

// STRH - Store halfword
type STRH struct {
	Rt  MReg
	Rn  MReg
	Ofs int64
}

// LDP - Load pair
type LDP struct {
	Rt1, Rt2 MReg
	Rn       MReg
	Ofs      int64
	Is64     bool
}

// STP - Store pair
type STP struct {
	Rt1, Rt2 MReg
	Rn       MReg
	Ofs      int64
	Is64     bool
}

// --- Floating Point Load/Store ---

// FLDRs - Load single-precision float
type FLDRs struct {
	Ft  MReg
	Rn  MReg
	Ofs int64
}

// FLDRd - Load double-precision float
type FLDRd struct {
	Ft  MReg
	Rn  MReg
	Ofs int64
}

// FSTRs - Store single-precision float
type FSTRs struct {
	Ft  MReg
	Rn  MReg
	Ofs int64
}

// FSTRd - Store double-precision float
type FSTRd struct {
	Ft  MReg
	Rn  MReg
	Ofs int64
}

// --- Branch Instructions ---

// B - Unconditional branch
type B struct {
	Target Label
}

// BL - Branch with link (call)
type BL struct {
	Target Label
}

// BR - Branch to register
type BR struct {
	Rn MReg
}

// BLR - Branch with link to register
type BLR struct {
	Rn MReg
}

// RET - Return (branch to LR)
type RET struct{}

// --- Conditional Branch ---

// Bcond represents a conditional branch (B.cond)
type Bcond struct {
	Cond   CondCode
	Target Label
}

// CondCode represents ARM64 condition codes
type CondCode int

const (
	CondEQ CondCode = iota // Equal (Z=1)
	CondNE                 // Not equal (Z=0)
	CondCS                 // Carry set / unsigned higher or same
	CondCC                 // Carry clear / unsigned lower
	CondMI                 // Minus / negative
	CondPL                 // Plus / positive or zero
	CondVS                 // Overflow
	CondVC                 // No overflow
	CondHI                 // Unsigned higher
	CondLS                 // Unsigned lower or same
	CondGE                 // Signed greater or equal
	CondLT                 // Signed less than
	CondGT                 // Signed greater than
	CondLE                 // Signed less or equal
	CondAL                 // Always
)

// String returns the condition code as a string
func (c CondCode) String() string {
	names := []string{
		"eq", "ne", "cs", "cc", "mi", "pl", "vs", "vc",
		"hi", "ls", "ge", "lt", "gt", "le", "al",
	}
	if int(c) < len(names) {
		return names[c]
	}
	return "?"
}

// --- Compare Instructions ---

// CMP - Compare (Rn - Rm)
type CMP struct {
	Rn, Rm MReg
	Is64   bool
}

// CMPi - Compare immediate
type CMPi struct {
	Rn   MReg
	Imm  int64
	Is64 bool
}

// CMN - Compare negative (Rn + Rm)
type CMN struct {
	Rn, Rm MReg
	Is64   bool
}

// CMNi - Compare negative immediate
type CMNi struct {
	Rn   MReg
	Imm  int64
	Is64 bool
}

// TST - Test bits (Rn & Rm)
type TST struct {
	Rn, Rm MReg
	Is64   bool
}

// TSTi - Test bits immediate
type TSTi struct {
	Rn   MReg
	Imm  int64
	Is64 bool
}

// --- Conditional Select ---

// CSEL - Conditional select
type CSEL struct {
	Rd, Rn, Rm MReg
	Cond       CondCode
	Is64       bool
}

// CSET - Conditional set (Rd = cond ? 1 : 0)
type CSET struct {
	Rd   MReg
	Cond CondCode
	Is64 bool
}

// CSINC - Conditional select increment
type CSINC struct {
	Rd, Rn, Rm MReg
	Cond       CondCode
	Is64       bool
}

// --- Move Instructions ---

// MOV - Move (alias for ORR with XZR)
type MOV struct {
	Rd, Rm MReg
	Is64   bool
}

// MOVi - Move immediate (small values)
type MOVi struct {
	Rd   MReg
	Imm  int64
	Is64 bool
}

// MOVZ - Move wide with zero
type MOVZ struct {
	Rd    MReg
	Imm   uint16
	Shift int // 0, 16, 32, or 48
	Is64  bool
}

// MOVK - Move wide with keep
type MOVK struct {
	Rd    MReg
	Imm   uint16
	Shift int
	Is64  bool
}

// MOVN - Move wide with NOT
type MOVN struct {
	Rd    MReg
	Imm   uint16
	Shift int
	Is64  bool
}

// --- Address Computation ---

// ADR - Compute PC-relative address
type ADR struct {
	Rd     MReg
	Target Label
}

// ADRP - Compute PC-relative page address
type ADRP struct {
	Rd     MReg
	Target Label
}

// --- Floating Point Operations ---

// FADD - Floating-point add
type FADD struct {
	Fd, Fn, Fm MReg
	IsDouble   bool
}

// FSUB - Floating-point subtract
type FSUB struct {
	Fd, Fn, Fm MReg
	IsDouble   bool
}

// FMUL - Floating-point multiply
type FMUL struct {
	Fd, Fn, Fm MReg
	IsDouble   bool
}

// FDIV - Floating-point divide
type FDIV struct {
	Fd, Fn, Fm MReg
	IsDouble   bool
}

// FNEG - Floating-point negate
type FNEG struct {
	Fd, Fn   MReg
	IsDouble bool
}

// FABS - Floating-point absolute value
type FABS struct {
	Fd, Fn   MReg
	IsDouble bool
}

// FSQRT - Floating-point square root
type FSQRT struct {
	Fd, Fn   MReg
	IsDouble bool
}

// FMOV - Floating-point move
type FMOV struct {
	Fd, Fn   MReg
	IsDouble bool
}

// FMOVi - Floating-point move immediate
type FMOVi struct {
	Fd       MReg
	Imm      float64
	IsDouble bool
}

// --- Floating Point Conversions ---

// SCVTF - Signed integer to float
type SCVTF struct {
	Fd       MReg
	Rn       MReg
	IsDouble bool // result is double
	Is64Src  bool // source is 64-bit int
}

// UCVTF - Unsigned integer to float
type UCVTF struct {
	Fd       MReg
	Rn       MReg
	IsDouble bool
	Is64Src  bool
}

// FCVTZS - Float to signed integer (round toward zero)
type FCVTZS struct {
	Rd       MReg
	Fn       MReg
	IsDouble bool // source is double
	Is64Dst  bool // dest is 64-bit int
}

// FCVTZU - Float to unsigned integer
type FCVTZU struct {
	Rd       MReg
	Fn       MReg
	IsDouble bool
	Is64Dst  bool
}

// FCVT - Float conversion (single <-> double)
type FCVT struct {
	Fd        MReg
	Fn        MReg
	DstDouble bool // true: single->double, false: double->single
}

// --- Floating Point Compare ---

// FCMP - Floating-point compare
type FCMP struct {
	Fn, Fm   MReg
	IsDouble bool
}

// FCMPz - Floating-point compare with zero
type FCMPz struct {
	Fn       MReg
	IsDouble bool
}

// --- Sign/Zero Extension ---

// SXTB - Sign extend byte to 32/64 bit
type SXTB struct {
	Rd, Rn MReg
	Is64   bool
}

// SXTH - Sign extend halfword to 32/64 bit
type SXTH struct {
	Rd, Rn MReg
	Is64   bool
}

// SXTW - Sign extend word to 64 bit
type SXTW struct {
	Rd, Rn MReg
}

// UXTB - Zero extend byte to 32 bit
type UXTB struct {
	Rd, Rn MReg
}

// UXTH - Zero extend halfword to 32 bit
type UXTH struct {
	Rd, Rn MReg
}

// --- Labels and Directives ---

// LabelDef defines a label
type LabelDef struct {
	Name Label
}

// --- Marker methods for Instruction interface ---

func (ADD) implInstruction()      {}
func (ADDi) implInstruction()     {}
func (SUB) implInstruction()      {}
func (SUBi) implInstruction()     {}
func (MUL) implInstruction()      {}
func (MADD) implInstruction()     {}
func (SMULL) implInstruction()    {}
func (UMULL) implInstruction()    {}
func (SDIV) implInstruction()     {}
func (UDIV) implInstruction()     {}
func (AND) implInstruction()      {}
func (ANDi) implInstruction()     {}
func (ORR) implInstruction()      {}
func (ORRi) implInstruction()     {}
func (EOR) implInstruction()      {}
func (EORi) implInstruction()     {}
func (MVN) implInstruction()      {}
func (NEG) implInstruction()      {}
func (LSL) implInstruction()      {}
func (LSLi) implInstruction()     {}
func (LSR) implInstruction()      {}
func (LSRi) implInstruction()     {}
func (ASR) implInstruction()      {}
func (ASRi) implInstruction()     {}
func (ROR) implInstruction()      {}
func (RORi) implInstruction()     {}
func (LDR) implInstruction()      {}
func (LDRr) implInstruction()     {}
func (LDRB) implInstruction()     {}
func (LDRH) implInstruction()     {}
func (LDRSB) implInstruction()    {}
func (LDRSH) implInstruction()    {}
func (LDRSW) implInstruction()    {}
func (STR) implInstruction()      {}
func (STRr) implInstruction()     {}
func (STRB) implInstruction()     {}
func (STRH) implInstruction()     {}
func (LDP) implInstruction()      {}
func (STP) implInstruction()      {}
func (FLDRs) implInstruction()    {}
func (FLDRd) implInstruction()    {}
func (FSTRs) implInstruction()    {}
func (FSTRd) implInstruction()    {}
func (B) implInstruction()        {}
func (BL) implInstruction()       {}
func (BR) implInstruction()       {}
func (BLR) implInstruction()      {}
func (RET) implInstruction()      {}
func (Bcond) implInstruction()    {}
func (CMP) implInstruction()      {}
func (CMPi) implInstruction()     {}
func (CMN) implInstruction()      {}
func (CMNi) implInstruction()     {}
func (TST) implInstruction()      {}
func (TSTi) implInstruction()     {}
func (CSEL) implInstruction()     {}
func (CSET) implInstruction()     {}
func (CSINC) implInstruction()    {}
func (MOV) implInstruction()      {}
func (MOVi) implInstruction()     {}
func (MOVZ) implInstruction()     {}
func (MOVK) implInstruction()     {}
func (MOVN) implInstruction()     {}
func (ADR) implInstruction()      {}
func (ADRP) implInstruction()     {}
func (FADD) implInstruction()     {}
func (FSUB) implInstruction()     {}
func (FMUL) implInstruction()     {}
func (FDIV) implInstruction()     {}
func (FNEG) implInstruction()     {}
func (FABS) implInstruction()     {}
func (FSQRT) implInstruction()    {}
func (FMOV) implInstruction()     {}
func (FMOVi) implInstruction()    {}
func (SCVTF) implInstruction()    {}
func (UCVTF) implInstruction()    {}
func (FCVTZS) implInstruction()   {}
func (FCVTZU) implInstruction()   {}
func (FCVT) implInstruction()     {}
func (FCMP) implInstruction()     {}
func (FCMPz) implInstruction()    {}
func (SXTB) implInstruction()     {}
func (SXTH) implInstruction()     {}
func (SXTW) implInstruction()     {}
func (UXTB) implInstruction()     {}
func (UXTH) implInstruction()     {}
func (LabelDef) implInstruction() {}

// --- Function and Program ---

// Function represents an assembly function
type Function struct {
	Name string
	Code []Instruction
	Size int64 // function size in bytes (computed after assembly)
}

// GlobVar represents a global variable
type GlobVar struct {
	Name  string
	Size  int64
	Init  []byte
	Align int
}

// Program represents a complete assembly program
type Program struct {
	Globals   []GlobVar
	Functions []Function
}

// NewFunction creates a new assembly function
func NewFunction(name string) *Function {
	return &Function{
		Name: name,
		Code: make([]Instruction, 0),
	}
}

// Append adds an instruction to the function
func (f *Function) Append(inst Instruction) {
	f.Code = append(f.Code, inst)
}

// AppendLabel adds a label definition
func (f *Function) AppendLabel(name Label) {
	f.Code = append(f.Code, LabelDef{Name: name})
}
