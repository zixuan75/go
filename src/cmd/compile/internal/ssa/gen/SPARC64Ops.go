// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import "strings"

// Notes:
//  - Less-than-64-bit integer types live in the low portion of registers.
//    For now, the upper portion is junk; sign/zero-extension might be optimized in the future, but not yet.
//  - Boolean types are zero or 1; stored in a byte, but loaded with AMOVUB so the upper bytes of a register are zero.
//  - *const instructions may use a constant larger than the instuction can encode.
//    In this case the assembler expands to multiple instructions and uses TMP
//    register.

// Note: registers not used in regalloc are not included in this list,
// so that regmask stays within int64
// Be careful when hand coding regmasks.
var regNamesSPARC64 = []string{
	// "G0", zero register, not used in regalloc
	"RT1",
	"CTXT",
	"g",
	"RT2",
	// "TMP", not used in regalloc
	// "G6", reserved for the operating system
	// "TLS", reserved for the operating system
	"O0",
	"O1",
	"O2",
	"O3",
	"O4",
	"O5",
	"RSP",
	// "OLR", output link register, not used in regalloc
	// "TMP2", not used in regalloc
	"L1",
	"L2",
	"L3",
	"L4",
	"L5",
	"L6",
	"L7",
	"I0",
	"I1",
	"I2",
	"I3",
	"I4",
	"I5",
	"RFP",
	// "ILR", input link register, not used in regalloc

	"Y0",
	"Y1",
	"Y2",
	"Y3",
	"Y4",
	"Y5",
	"Y6",
	"Y7",
	"Y8",
	"Y9",
	"Y10",
	"Y11",
	"Y12",
	"Y13",
	// "YTWO", not used in regalloc
	// "YTMP", not used in regalloc

	// pseudo-registers
	"SB",
	"SP",
	"FP",
}

func init() {
	// Make map from reg names to reg integers.
	if len(regNamesSPARC64) > 64 {
		panic("too many registers")
	}
	num := map[string]int{}
	for i, name := range regNamesSPARC64 {
		num[name] = i
	}
	buildReg := func(s string) regMask {
		m := regMask(0)
		for _, r := range strings.Split(s, " ") {
			if n, ok := num[r]; ok {
				m |= regMask(1) << uint(n)
				continue
			}
			panic("register " + r + " not found")
		}
		return m
	}

	// Common individual register masks
	var (
		gp = buildReg("O0 O1 O2 O3 O4 O5 L1 L2 L3 L4 L5 L6 L7 I0 I1 I2 I3 I4 I5")
		gprt = buildReg("RT1 RT2")
		fp = buildReg("Y0 Y1 Y2 Y3 Y4 Y5 Y6 Y7 Y8 Y9 Y10 Y11 Y12 Y13")
		sp = buildReg("SP")
		sb = buildReg("SB")
		g = buildReg("g")
		ctxt = buildReg("CTXT")
		gpin = gp | gprt | ctxt | g
		gpout = gp
		multmp = buildReg("O0 O1 O2")
		mulin = gpin&^multmp
		mulout = gpout&^multmp

		gp01        = regInfo{inputs: nil, outputs: []regMask{gpout}}
		gp11        = regInfo{inputs: []regMask{gpin}, outputs: []regMask{gpout}}
		gp21        = regInfo{inputs: []regMask{gpin, gpin}, outputs: []regMask{gpout}}
		gp1flags    = regInfo{inputs: []regMask{gpin}}
		gp2flags    = regInfo{inputs: []regMask{gpin, gpin}}
		gpload      = regInfo{inputs: []regMask{gpin | sp | sb}, outputs: []regMask{gpout}}
		gpstore     = regInfo{inputs: []regMask{gpin | sp | sb, gpin | sp | sb}}
		gpstore0    = regInfo{inputs: []regMask{gpin | sp | sb}}
		fp01        = regInfo{inputs: nil, outputs: []regMask{fp}}
		fp11        = regInfo{inputs: []regMask{fp}, outputs: []regMask{fp}}
		fpgp        = regInfo{inputs: []regMask{fp}, outputs: []regMask{gpout}}
		gpfp        = regInfo{inputs: []regMask{gpin}, outputs: []regMask{fp}}
		fp21        = regInfo{inputs: []regMask{fp, fp}, outputs: []regMask{fp}}
		fp2flags    = regInfo{inputs: []regMask{fp, fp}}
		fpload      = regInfo{inputs: []regMask{gpin | sp | sb}, outputs: []regMask{fp}}
		fpstore     = regInfo{inputs: []regMask{gpin | sp | sb, fp}}
		readflags   = regInfo{inputs: nil, outputs: []regMask{gpout}}
		callerSave  = gpin | fp // runtime.setg (and anything calling it) may clobber g
	)
	ops := []opData{
		{name: "ADD", argLength: 2, reg: gp21, asm: "ADD", commutative: true}, // arg0 + arg1
		{name: "SUB", argLength: 2, reg: gp21, asm: "SUB"}, // arg0 - arg1

		{name: "AND", argLength: 2, reg: gp21, asm: "AND", commutative: true}, // arg0 & arg1
		{name: "OR", argLength: 2, reg: gp21, asm: "OR", commutative: true},  // arg0 | arg1
		{name: "XOR", argLength: 2, reg: gp21, asm: "XOR", commutative: true}, // arg0 ^ arg1

		{name: "ADDconst", argLength: 1, reg: regInfo{inputs: []regMask{gpin | sp}, outputs: []regMask{gpout}}, asm: "ADD", aux: "Int64"}, // arg0 + auxInt
		{name: "SUBconst", argLength: 1, reg: gp11, asm: "SUB", aux: "Int64"}, // arg0 - auxInt
		{name: "ANDconst", argLength: 1, reg: gp11, asm: "AND", aux: "Int64"}, // arg0 & auxInt
		{name: "ORconst", argLength: 1, reg: gp11, asm: "OR", aux: "Int64"},  // arg0 | auxInt
		{name: "XORconst", argLength: 1, reg: gp11, asm: "XOR", aux: "Int64"}, // arg0 ^ auxInt

		{name: "MULD", argLength: 2, reg: gp21, typ: "Int64", asm: "MULD", commutative: true},     // arg0 * arg1
		{name: "SDIVD", argLength: 2, reg: gp21, typ: "Int64", asm: "SDIVD"},                       // arg0 / arg1, signed
		{name: "UDIVD", argLength: 2, reg: gp21, typ: "UInt64", asm: "UDIVD"},                       // arg0 / arg1, unsigned

		{
			name: "MULXHI",
			argLength: 2,
			reg: regInfo{inputs: []regMask{mulin, mulin}, outputs: []regMask{mulout}, clobbers: multmp},
			clobberFlags: true,
			commutative: true,
		}, // (arg0 * arg1) >> 64, signed
		{
			name: "UMULXHI",
			argLength: 2,
			reg: regInfo{inputs: []regMask{mulin, mulin}, outputs: []regMask{mulout}, clobbers: multmp},
			clobberFlags: true,
			commutative: true,
		}, // (arg0 * arg1) >> 64, unsigned

		{name: "FADDS", argLength: 2, reg: fp21, asm: "FADDS", commutative: true}, // arg0 + arg1
		{name: "FADDD", argLength: 2, reg: fp21, asm: "FADDD", commutative: true}, // arg0 + arg1
		{name: "FSUBS", argLength: 2, reg: fp21, asm: "FSUBS"},                    // arg0 - arg1
		{name: "FSUBD", argLength: 2, reg: fp21, asm: "FSUBD"},                    // arg0 - arg1
		{name: "FMULS", argLength: 2, reg: fp21, asm: "FMULS", commutative: true}, // arg0 * arg1
		{name: "FMULD", argLength: 2, reg: fp21, asm: "FMULD", commutative: true}, // arg0 * arg1
		{name: "FDIVS", argLength: 2, reg: fp21, asm: "FDIVS"},                    // arg0 / arg1
		{name: "FDIVD", argLength: 2, reg: fp21, asm: "FDIVD"},                    // arg0 / arg1

		// unary ops
		{name: "NEG", argLength: 1, reg: gp11, asm: "NEG"},       // -arg0
		{name: "FNEGS", argLength: 1, reg: fp11, asm: "FNEGS"},   // -arg0, float32
		{name: "FNEGD", argLength: 1, reg: fp11, asm: "FNEGD"},   // -arg0, float64
		{name: "FSQRTD", argLength: 1, reg: fp11, asm: "FSQRTD"}, // sqrt(arg0), float64
		// moves
		{name: "MOVDaddr", argLength: 1, reg: regInfo{inputs: []regMask{sp | sb}, outputs: []regMask{gp}}, aux: "SymOff", asm: "MOVD", rematerializeable: true}, // arg0 + auxInt + aux.(*gc.Sym), arg0=SP/SB

		{name: "MOVBload", argLength: 2, reg: gpload, aux: "SymOff", asm: "MOVB", typ: "Int8"},      // load from arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVUBload", argLength: 2, reg: gpload, aux: "SymOff", asm: "MOVUB", typ: "UInt8"},   // load from arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVHload", argLength: 2, reg: gpload, aux: "SymOff", asm: "MOVH", typ: "Int16"},     // load from arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVUHload", argLength: 2, reg: gpload, aux: "SymOff", asm: "MOVUH", typ: "UInt16"},  // load from arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVWload", argLength: 2, reg: gpload, aux: "SymOff", asm: "MOVW", typ: "Int32"},     // load from arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVUWload", argLength: 2, reg: gpload, aux: "SymOff", asm: "MOVUW", typ: "UInt32"},  // load from arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVDload", argLength: 2, reg: gpload, aux: "SymOff", asm: "MOVD", typ: "UInt64"},    // load from arg0 + auxInt + aux.  arg1=mem.
		{name: "FMOVSload", argLength: 2, reg: fpload, aux: "SymOff", asm: "FMOVS", typ: "Float32"},  // load from arg0 + auxInt + aux.  arg1=mem.
		{name: "FMOVDload", argLength: 2, reg: fpload, aux: "SymOff", asm: "FMOVD", typ: "Float64"},  // load from arg0 + auxInt + aux.  arg1=mem.

		{name: "MOVDstore", argLength: 3, reg: gpstore, asm: "MOVD", aux: "SymOff", typ: "Mem"},
		{name: "MOVWstore", argLength: 3, reg: gpstore, asm: "MOVW", aux: "SymOff", typ: "Mem"},
		{name: "MOVHstore", argLength: 3, reg: gpstore, asm: "MOVH", aux: "SymOff", typ: "Mem"},
		{name: "MOVBstore", argLength: 3, reg: gpstore, asm: "MOVB", aux: "SymOff", typ: "Mem"},
		{name: "FMOVSstore", argLength: 3, reg: fpstore, aux: "SymOff", asm: "FMOVS", typ: "Mem"}, // store 4 bytes of arg1 to arg0 + auxInt + aux.  arg2=mem.
		{name: "FMOVDstore", argLength: 3, reg: fpstore, aux: "SymOff", asm: "FMOVD", typ: "Mem"}, // store 8 bytes of arg1 to arg0 + auxInt + aux.  arg2=mem.

		{name: "MOVBstorezero", argLength: 2, reg: gpstore0, aux: "SymOff", asm: "MOVB", typ: "Mem"}, // store 1 byte of zero to arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVHstorezero", argLength: 2, reg: gpstore0, aux: "SymOff", asm: "MOVH", typ: "Mem"}, // store 2 bytes of zero to arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVWstorezero", argLength: 2, reg: gpstore0, aux: "SymOff", asm: "MOVW", typ: "Mem"}, // store 4 bytes of zero to arg0 + auxInt + aux.  arg1=mem.
		{name: "MOVDstorezero", argLength: 2, reg: gpstore0, aux: "SymOff", asm: "MOVD", typ: "Mem"}, // store 8 bytes of zero to arg0 + auxInt + aux.  ar12=mem.

		{name: "MOVDconst", argLength: 0, reg: gp01, aux: "Int64", asm: "MOVD", typ: "UInt64", rematerializeable: true},
		{name: "MOVWconst", argLength: 0, reg: gp01, aux: "Int32", asm: "MOVW", rematerializeable: true},     // 32 low bits of auxint
		{name: "FMOVSconst", argLength: 0, reg: fp01, aux: "Float64", asm: "FMOVS", typ: "Float32", rematerializeable: true}, // auxint as 64-bit float, convert to 32-bit float
		{name: "FMOVDconst", argLength: 0, reg: fp01, aux: "Float64", asm: "FMOVD", typ: "Float64", rematerializeable: true}, // auxint as 64-bit float

		// shifts
		{
			name: "SLLmax",
			argLength: 2,
			reg: gp21,
			asm: "SLLD",
			clobberFlags: true,
			aux: "Int64",
		},     // arg0 << arg1, shift amount is mod 64, aux is max shift until zero result
		{name: "SLLconst", argLength: 1, reg: gp11, asm: "SLLD", aux: "Int64"},   // arg0 << auxInt
		{
			name: "SRLmax",
			argLength: 2,
			reg: gp21,
			asm: "SRLD",
			clobberFlags: true,
			aux: "Int64",
		},     // arg0 >> arg1, shift amount is mod 64, aux is max shift until zero result
		{name: "SRLconst", argLength: 1, reg: gp11, asm: "SRLD", aux: "Int64"},   // arg0 >> auxInt, unsigned
		{
			name: "SRAmax",
			argLength: 2,
			reg: gp21,
			asm: "SRAD",
			clobberFlags: true,
			aux: "Int64",
		},     // arg0 >> arg1, signed, shift amount is mod 64, aux is max shift
		{name: "SRAconst", argLength: 1, reg: gp11, asm: "SRAD", aux: "Int64"},   // arg0 >> auxInt, signed

		// comparisons
		{name: "CMP", argLength: 2, reg: gp2flags, asm: "CMP", typ: "Flags"},                      // arg0 compare to arg1
		{name: "CMPconst", argLength: 1, reg: gp1flags, asm: "CMP", aux: "Int64", typ: "Flags"},   // arg0 compare to auxInt
		{name: "FCMPS", argLength: 2, reg: fp2flags, asm: "FCMPS", typ: "Flags"},                  // arg0 compare to arg1, float32
		{name: "FCMPD", argLength: 2, reg: fp2flags, asm: "FCMPD", typ: "Flags"},                  // arg0 compare to arg1, float64

		// conversions
		{name: "MOVBreg", argLength: 1, reg: gp11, asm: "MOVB"},   // move from arg0, sign-extended from byte
		{name: "MOVUBreg", argLength: 1, reg: gp11, asm: "MOVUB"}, // move from arg0, unsign-extended from byte
		{name: "MOVHreg", argLength: 1, reg: gp11, asm: "MOVH"},   // move from arg0, sign-extended from half
		{name: "MOVUHreg", argLength: 1, reg: gp11, asm: "MOVUH"}, // move from arg0, unsign-extended from half
		{name: "MOVWreg", argLength: 1, reg: gp11, asm: "MOVW"},   // move from arg0, sign-extended from word
		{name: "MOVUWreg", argLength: 1, reg: gp11, asm: "MOVUW"}, // move from arg0, unsign-extended from word
		{name: "MOVDreg", argLength: 1, reg: gp11, asm: "MOVD"},   // move from arg0

		{name: "FITOS", argLength: 1, reg: gpfp, asm: "FITOS"}, // int32/uint32 -> float32
		{name: "FITOD", argLength: 1, reg: gpfp, asm: "FITOD"}, // int32/uint32 -> float64
		{name: "FXTOS", argLength: 1, reg: gpfp, clobberFlags: true, asm: "FXTOS"}, // int64/uint64 -> float32
		{name: "FXTOD", argLength: 1, reg: gpfp, clobberFlags: true, asm: "FXTOD"}, // int64/uint64 -> float64
		{name: "FSTOI", argLength: 1, reg: fpgp, asm: "FSTOI"}, // float32 -> int32/uint32
		{name: "FDTOI", argLength: 1, reg: fpgp, asm: "FDTOI"}, // float64 -> int32/uint32
		{name: "FSTOX", argLength: 1, reg: fpgp, asm: "FSTOX"}, // float32 -> int64/uint64
		{name: "FDTOX", argLength: 1, reg: fpgp, clobberFlags: true, asm: "FDTOX"}, // float64 -> int64/uint64
		{name: "FSTOD", argLength: 1, reg: fp11, asm: "FSTOD"}, // float32 -> float64
		{name: "FDTOS", argLength: 1, reg: fp11, asm: "FDTOS"}, // float64 -> float32

		// function calls
		{name: "CALLstatic", argLength: 1, reg: regInfo{clobbers: callerSave}, aux: "SymOff", clobberFlags: true, call: true},                                              // call static function aux.(*gc.Sym).  arg0=mem, auxint=argsize, returns mem
		{name: "CALLclosure", argLength: 3, reg: regInfo{inputs: []regMask{gp | sp, ctxt, 0}, clobbers: callerSave}, aux: "Int64", clobberFlags: true, call: true}, // call function via closure.  arg0=codeptr, arg1=closure, arg2=mem, auxint=argsize, returns mem
		{name: "CALLdefer", argLength: 1, reg: regInfo{clobbers: callerSave}, aux: "Int64", clobberFlags: true, call: true},                                                // call deferproc.  arg0=mem, auxint=argsize, returns mem
		{name: "CALLgo", argLength: 1, reg: regInfo{clobbers: callerSave}, aux: "Int64", clobberFlags: true, call: true},                                                   // call newproc.  arg0=mem, auxint=argsize, returns mem
		{name: "CALLinter", argLength: 2, reg: regInfo{inputs: []regMask{gp}, clobbers: callerSave}, aux: "Int64", clobberFlags: true, call: true},                         // call fn by pointer.  arg0=codeptr, arg1=mem, auxint=argsize, returns mem

		// pseudo-ops
		{name: "LoweredNilCheck", argLength: 2, reg: regInfo{inputs: []regMask{gpin}}}, // panic if arg0 is nil.  arg1=mem.

		{name: "Equal32", argLength: 1, reg: readflags},         // bool, true flags encode x==y false otherwise.
		{name: "Equal64", argLength: 1, reg: readflags},         // bool, true flags encode x==y false otherwise.
		{name: "EqualF", argLength: 1, reg: readflags},          // bool, true flags encode x==y false otherwise.

		{name: "NotEqual32", argLength: 1, reg: readflags},      // bool, true flags encode x!=y false otherwise.
		{name: "NotEqual64", argLength: 1, reg: readflags},      // bool, true flags encode x!=y false otherwise.
		{name: "NotEqualF", argLength: 1, reg: readflags},       // bool, true flags encode x!=y false otherwise.

		{name: "LessThan32", argLength: 1, reg: readflags},      // bool, true flags encode signed x<y false otherwise.
		{name: "LessThan64", argLength: 1, reg: readflags},      // bool, true flags encode signed x<y false otherwise.
		{name: "LessThan32U", argLength: 1, reg: readflags},     // bool, true flags encode unsigned x<y false otherwise.
		{name: "LessThan64U", argLength: 1, reg: readflags},     // bool, true flags encode unsigned x<y false otherwise.
		{name: "LessThanF", argLength: 1, reg: readflags},       // bool, true flags encode x<y false otherwise.

		{name: "LessEqual32", argLength: 1, reg: readflags},     // bool, true flags encode signed x<=y false otherwise.
		{name: "LessEqual64", argLength: 1, reg: readflags},     // bool, true flags encode signed x<=y false otherwise.
		{name: "LessEqual32U", argLength: 1, reg: readflags},    // bool, true flags encode unsigned x<=y false otherwise.
		{name: "LessEqual64U", argLength: 1, reg: readflags},    // bool, true flags encode unsigned x<=y false otherwise.
		{name: "LessEqualF", argLength: 1, reg: readflags},      // bool, true flags encode x<=y false otherwise.

		{name: "GreaterThan32", argLength: 1, reg: readflags},   // bool, true flags encode signed x>y false otherwise.
		{name: "GreaterThan64", argLength: 1, reg: readflags},   // bool, true flags encode signed x>y false otherwise.
		{name: "GreaterThan32U", argLength: 1, reg: readflags},  // bool, true flags encode unsigned x>y false otherwise.
		{name: "GreaterThan64U", argLength: 1, reg: readflags},  // bool, true flags encode unsigned x>y false otherwise.
		{name: "GreaterThanF", argLength: 1, reg: readflags},    // bool, true flags encode x>y false otherwise.

		{name: "GreaterEqual32", argLength: 1, reg: readflags},  // bool, true flags encode signed x>=y false otherwise.
		{name: "GreaterEqual64", argLength: 1, reg: readflags},  // bool, true flags encode signed x>=y false otherwise.
		{name: "GreaterEqual32U", argLength: 1, reg: readflags}, // bool, true flags encode unsigned x>=y false otherwise.
		{name: "GreaterEqual64U", argLength: 1, reg: readflags}, // bool, true flags encode unsigned x>=y false otherwise.
		{name: "GreaterEqualF", argLength: 1, reg: readflags},   // bool, true flags encode x>=y false otherwise.

		// large zeroing
		// arg0 = address of memory to zero (in REG_RT1, changed as side effect)
		// arg1 = address of the last element to zero
		// arg2 = mem
		// returns mem
		// auxInt is alignment
		// Note: the-end-of-the-memory may be not a valid pointer. it's a problem if it is spilled.
		// the-end-of-the-memory - 8 is with the area to zero, ok to spill.
		{
			name:      "LoweredZero",
			aux:       "Int64",
			argLength: 3,
			reg: regInfo{
				inputs:   []regMask{buildReg("RT1"), gp},
				clobbers: buildReg("RT1"),
			},
			clobberFlags: true,
		},

		// large move
		// arg0 = address of dst memory (in REG_RT2, changed as side effect)
		// arg1 = address of src memory (in REG_RT1, changed as side effect)
		// arg2 = address of the last element of src
		// arg3 = mem
		// returns mem
		// auxInt is alignment
		// Note: the-end-of-src may be not a valid pointer. it's a problem if it is spilled.
		// the-end-of-src - 8 is within the area to copy, ok to spill.
		{
			name:      "LoweredMove",
			aux:       "Int64",
			argLength: 4,
			reg: regInfo{
				inputs:   []regMask{buildReg("RT2"), buildReg("RT1"), gp},
				clobbers: buildReg("RT1 RT2"),
			},
			clobberFlags: true,
		},

		// Scheduler ensures LoweredGetClosurePtr occurs only in entry block,
		// and sorts it to the very beginning of the block to prevent other
		// use of CTXT (sparc64.REG_CTXT, the closure pointer)
		{name: "LoweredGetClosurePtr", reg: regInfo{outputs: []regMask{buildReg("CTXT")}}},

		// MOVDconvert converts between pointers and integers.
		// We have a special op for this so as to not confuse GC
		// (particularly stack maps).  It takes a memory arg so it
		// gets correctly ordered with respect to GC safepoints.
		// arg0=ptr/int arg1=mem, output=int/ptr
		{name: "MOVDconvert", argLength: 2, reg: gp11, asm: "MOVD"},
	}

	blocks := []blockData{
		// int
		{name: "ND"},
		{name: "NED"},
		{name: "ED"},
		{name: "GD"},
		{name: "LED"},
		{name: "GED"},
		{name: "LD"},
		{name: "GUD"},
		{name: "LEUD"},
		{name: "CCD"},
		{name: "CSD"},
		{name: "POSD"},
		{name: "NEGD"},
		{name: "VCD"},
		{name: "VSD"},
		{name: "NW"},
		{name: "NEW"},
		{name: "EW"},
		{name: "GW"},
		{name: "LEW"},
		{name: "GEW"},
		{name: "LW"},
		{name: "GUW"},
		{name: "LEUW"},
		{name: "CCW"},
		{name: "CSW"},
		{name: "POSW"},
		{name: "NEGW"},
		{name: "VCW"},
		{name: "VSW"},
		// float
		{name: "NF"},
		{name: "EF"},
		{name: "NEF"},
		{name: "LF"},
		{name: "LEF"},
		{name: "GF"},
		{name: "GEF"},
	}

	archs = append(archs, arch{
		name:            "SPARC64",
		pkg:             "cmd/internal/obj/sparc64",
		genfile:         "../../sparc64/ssa.go",
		ops:             ops,
		blocks:          blocks,
		regnames:        regNamesSPARC64,
		gpregmask:       gp | gprt | ctxt, // must not include g
		fpregmask:       fp,
		framepointerreg: int8(num["RFP"]),
	})
}