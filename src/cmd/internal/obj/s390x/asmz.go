// Based on cmd/internal/obj/ppc64/asm9.go.
//
//    Copyright © 1994-1999 Lucent Technologies Inc.  All rights reserved.
//    Portions Copyright © 1995-1997 C H Forsyth (forsyth@terzarima.net)
//    Portions Copyright © 1997-1999 Vita Nuova Limited
//    Portions Copyright © 2000-2008 Vita Nuova Holdings Limited (www.vitanuova.com)
//    Portions Copyright © 2004,2006 Bruce Ellis
//    Portions Copyright © 2005-2007 C H Forsyth (forsyth@terzarima.net)
//    Revisions Copyright © 2000-2008 Lucent Technologies Inc. and others
//    Portions Copyright © 2009 The Go Authors.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package s390x

import (
	"cmd/internal/obj"
	"log"
	"math"
	"sort"
)

// instruction layout.
const (
	FuncAlign = 16
)

type Optab struct {
	as    int16 // opcode
	a1    uint8 // From
	a2    uint8 // Reg
	a3    uint8 // From3
	a4    uint8 // To
	type_ int8
	param int16 // REGSP for auto variables
}

var optab = []Optab{
	// instruction,  From,   Reg,    From3,  To, type, param
	Optab{obj.ATEXT, C_ADDR, C_NONE, C_NONE, C_TEXTSIZE, 0, 0},
	Optab{obj.ATEXT, C_ADDR, C_NONE, C_LCON, C_TEXTSIZE, 0, 0},

	// move register
	Optab{AMOVD, C_REG, C_NONE, C_NONE, C_REG, 1, 0},
	Optab{AMOVB, C_REG, C_NONE, C_NONE, C_REG, 1, 0},
	Optab{AMOVBZ, C_REG, C_NONE, C_NONE, C_REG, 1, 0},
	Optab{AMOVW, C_REG, C_NONE, C_NONE, C_REG, 1, 0},
	Optab{AMOVWZ, C_REG, C_NONE, C_NONE, C_REG, 1, 0},
	Optab{AFMOVD, C_FREG, C_NONE, C_NONE, C_FREG, 1, 0},
	Optab{AMOVDBR, C_REG, C_NONE, C_NONE, C_REG, 1, 0},

	// load constant
	Optab{AMOVD, C_LACON, C_NONE, C_NONE, C_REG, 26, REGSP},
	Optab{AMOVW, C_LACON, C_NONE, C_NONE, C_REG, 26, REGSP},
	Optab{AMOVWZ, C_LACON, C_NONE, C_NONE, C_REG, 26, REGSP},
	Optab{AMOVD, C_DCON, C_NONE, C_NONE, C_REG, 3, 0},
	Optab{AMOVW, C_DCON, C_NONE, C_NONE, C_REG, 3, 0},
	Optab{AMOVWZ, C_DCON, C_NONE, C_NONE, C_REG, 3, 0},
	Optab{AMOVB, C_DCON, C_NONE, C_NONE, C_REG, 3, 0},
	Optab{AMOVBZ, C_DCON, C_NONE, C_NONE, C_REG, 3, 0},

	// store constant
	Optab{AMOVD, C_SYMADDR, C_NONE, C_NONE, C_ADDR, 73, 0},
	Optab{AMOVD, C_LCON, C_NONE, C_NONE, C_ADDR, 73, 0},
	Optab{AMOVW, C_LCON, C_NONE, C_NONE, C_ADDR, 73, 0},
	Optab{AMOVWZ, C_LCON, C_NONE, C_NONE, C_ADDR, 73, 0},
	Optab{AMOVBZ, C_LCON, C_NONE, C_NONE, C_ADDR, 73, 0},
	Optab{AMOVB, C_LCON, C_NONE, C_NONE, C_ADDR, 73, 0},
	Optab{AMOVD, C_SYMADDR, C_NONE, C_NONE, C_LAUTO, 72, REGSP},
	Optab{AMOVD, C_LCON, C_NONE, C_NONE, C_LAUTO, 72, REGSP},
	Optab{AMOVW, C_LCON, C_NONE, C_NONE, C_LAUTO, 72, REGSP},
	Optab{AMOVWZ, C_LCON, C_NONE, C_NONE, C_LAUTO, 72, REGSP},
	Optab{AMOVB, C_LCON, C_NONE, C_NONE, C_LAUTO, 72, REGSP},
	Optab{AMOVBZ, C_LCON, C_NONE, C_NONE, C_LAUTO, 72, REGSP},
	Optab{AMOVD, C_SYMADDR, C_NONE, C_NONE, C_LOREG, 72, 0},
	Optab{AMOVD, C_LCON, C_NONE, C_NONE, C_LOREG, 72, 0},
	Optab{AMOVW, C_LCON, C_NONE, C_NONE, C_LOREG, 72, 0},
	Optab{AMOVWZ, C_LCON, C_NONE, C_NONE, C_LOREG, 72, 0},
	Optab{AMOVB, C_LCON, C_NONE, C_NONE, C_LOREG, 72, 0},
	Optab{AMOVBZ, C_LCON, C_NONE, C_NONE, C_LOREG, 72, 0},

	// store
	Optab{AMOVD, C_REG, C_NONE, C_NONE, C_LAUTO, 35, REGSP},
	Optab{AMOVW, C_REG, C_NONE, C_NONE, C_LAUTO, 35, REGSP},
	Optab{AMOVWZ, C_REG, C_NONE, C_NONE, C_LAUTO, 35, REGSP},
	Optab{AMOVBZ, C_REG, C_NONE, C_NONE, C_LAUTO, 35, REGSP},
	Optab{AMOVB, C_REG, C_NONE, C_NONE, C_LAUTO, 35, REGSP},
	Optab{AMOVDBR, C_REG, C_NONE, C_NONE, C_LAUTO, 35, REGSP},
	Optab{AMOVHBR, C_REG, C_NONE, C_NONE, C_LAUTO, 35, REGSP},
	Optab{AMOVD, C_REG, C_NONE, C_NONE, C_LOREG, 35, 0},
	Optab{AMOVW, C_REG, C_NONE, C_NONE, C_LOREG, 35, 0},
	Optab{AMOVWZ, C_REG, C_NONE, C_NONE, C_LOREG, 35, 0},
	Optab{AMOVBZ, C_REG, C_NONE, C_NONE, C_LOREG, 35, 0},
	Optab{AMOVB, C_REG, C_NONE, C_NONE, C_LOREG, 35, 0},
	Optab{AMOVDBR, C_REG, C_NONE, C_NONE, C_LOREG, 35, 0},
	Optab{AMOVHBR, C_REG, C_NONE, C_NONE, C_LOREG, 35, 0},
	Optab{AMOVD, C_REG, C_NONE, C_NONE, C_ADDR, 74, 0},
	Optab{AMOVW, C_REG, C_NONE, C_NONE, C_ADDR, 74, 0},
	Optab{AMOVWZ, C_REG, C_NONE, C_NONE, C_ADDR, 74, 0},
	Optab{AMOVBZ, C_REG, C_NONE, C_NONE, C_ADDR, 74, 0},
	Optab{AMOVB, C_REG, C_NONE, C_NONE, C_ADDR, 74, 0},

	// load
	Optab{AMOVD, C_LAUTO, C_NONE, C_NONE, C_REG, 36, REGSP},
	Optab{AMOVW, C_LAUTO, C_NONE, C_NONE, C_REG, 36, REGSP},
	Optab{AMOVWZ, C_LAUTO, C_NONE, C_NONE, C_REG, 36, REGSP},
	Optab{AMOVBZ, C_LAUTO, C_NONE, C_NONE, C_REG, 36, REGSP},
	Optab{AMOVB, C_LAUTO, C_NONE, C_NONE, C_REG, 36, REGSP},
	Optab{AMOVDBR, C_LAUTO, C_NONE, C_NONE, C_REG, 36, REGSP},
	Optab{AMOVHBR, C_LAUTO, C_NONE, C_NONE, C_REG, 36, REGSP},
	Optab{AMOVD, C_LOREG, C_NONE, C_NONE, C_REG, 36, 0},
	Optab{AMOVW, C_LOREG, C_NONE, C_NONE, C_REG, 36, 0},
	Optab{AMOVWZ, C_LOREG, C_NONE, C_NONE, C_REG, 36, 0},
	Optab{AMOVBZ, C_LOREG, C_NONE, C_NONE, C_REG, 36, 0},
	Optab{AMOVB, C_LOREG, C_NONE, C_NONE, C_REG, 36, 0},
	Optab{AMOVDBR, C_LOREG, C_NONE, C_NONE, C_REG, 36, 0},
	Optab{AMOVHBR, C_LOREG, C_NONE, C_NONE, C_REG, 36, 0},
	Optab{AMOVD, C_ADDR, C_NONE, C_NONE, C_REG, 75, 0},
	Optab{AMOVW, C_ADDR, C_NONE, C_NONE, C_REG, 75, 0},
	Optab{AMOVWZ, C_ADDR, C_NONE, C_NONE, C_REG, 75, 0},
	Optab{AMOVBZ, C_ADDR, C_NONE, C_NONE, C_REG, 75, 0},
	Optab{AMOVB, C_ADDR, C_NONE, C_NONE, C_REG, 75, 0},

	// integer arithmetic
	Optab{AADD, C_REG, C_REG, C_NONE, C_REG, 2, 0},
	Optab{AADD, C_REG, C_NONE, C_NONE, C_REG, 2, 0},
	Optab{AADD, C_LCON, C_REG, C_NONE, C_REG, 22, 0},
	Optab{AADD, C_LCON, C_NONE, C_NONE, C_REG, 22, 0},
	Optab{AADDC, C_REG, C_REG, C_NONE, C_REG, 2, 0},
	Optab{AADDC, C_REG, C_NONE, C_NONE, C_REG, 2, 0},
	Optab{AADDC, C_LCON, C_REG, C_NONE, C_REG, 22, 0},
	Optab{AADDC, C_LCON, C_NONE, C_NONE, C_REG, 22, 0},
	Optab{AMULLW, C_REG, C_REG, C_NONE, C_REG, 2, 0},
	Optab{AMULLW, C_REG, C_NONE, C_NONE, C_REG, 2, 0},
	Optab{AMULLW, C_LCON, C_REG, C_NONE, C_REG, 22, 0},
	Optab{AMULLW, C_LCON, C_NONE, C_NONE, C_REG, 22, 0},
	Optab{ASUBC, C_REG, C_REG, C_NONE, C_REG, 10, 0},
	Optab{ASUBC, C_REG, C_NONE, C_NONE, C_REG, 10, 0},
	Optab{ADIVW, C_REG, C_REG, C_NONE, C_REG, 2, 0},
	Optab{ADIVW, C_REG, C_NONE, C_NONE, C_REG, 2, 0},
	Optab{ASUB, C_REG, C_REG, C_NONE, C_REG, 10, 0},
	Optab{ASUB, C_REG, C_NONE, C_NONE, C_REG, 10, 0},
	Optab{AADDME, C_REG, C_NONE, C_NONE, C_REG, 47, 0},
	Optab{ANEG, C_REG, C_NONE, C_NONE, C_REG, 47, 0},
	Optab{ANEG, C_NONE, C_NONE, C_NONE, C_REG, 47, 0},

	// integer logical
	Optab{AAND, C_REG, C_REG, C_NONE, C_REG, 6, 0},
	Optab{AAND, C_REG, C_NONE, C_NONE, C_REG, 6, 0},
	Optab{AAND, C_LCON, C_NONE, C_NONE, C_REG, 23, 0},
	Optab{AAND, C_LCON, C_REG, C_NONE, C_REG, 23, 0},
	Optab{AOR, C_REG, C_REG, C_NONE, C_REG, 6, 0},
	Optab{AOR, C_REG, C_NONE, C_NONE, C_REG, 6, 0},
	Optab{AOR, C_LCON, C_NONE, C_NONE, C_REG, 23, 0},
	Optab{AOR, C_LCON, C_REG, C_NONE, C_REG, 23, 0},
	Optab{ASLD, C_REG, C_NONE, C_NONE, C_REG, 7, 0},
	Optab{ASLD, C_REG, C_REG, C_NONE, C_REG, 7, 0},
	Optab{ASLD, C_SCON, C_REG, C_NONE, C_REG, 7, 0},
	Optab{ASLD, C_SCON, C_NONE, C_NONE, C_REG, 7, 0},

	// compare and swap
	Optab{ACSG, C_REG, C_REG, C_NONE, C_SOREG, 79, 0},

	// floating point
	Optab{AFADD, C_FREG, C_NONE, C_NONE, C_FREG, 2, 0},
	Optab{AFADD, C_FREG, C_FREG, C_NONE, C_FREG, 2, 0},
	Optab{AFABS, C_FREG, C_NONE, C_NONE, C_FREG, 33, 0},
	Optab{AFABS, C_NONE, C_NONE, C_NONE, C_FREG, 33, 0},
	Optab{AFMADD, C_FREG, C_FREG, C_FREG, C_FREG, 34, 0},
	Optab{AFMUL, C_FREG, C_NONE, C_NONE, C_FREG, 32, 0},
	Optab{AFMUL, C_FREG, C_FREG, C_NONE, C_FREG, 32, 0},
	Optab{AFMOVD, C_LAUTO, C_NONE, C_NONE, C_FREG, 36, REGSP},
	Optab{AFMOVD, C_LOREG, C_NONE, C_NONE, C_FREG, 36, 0},
	Optab{AFMOVD, C_ADDR, C_NONE, C_NONE, C_FREG, 75, 0},
	Optab{AFMOVD, C_FREG, C_NONE, C_NONE, C_LAUTO, 35, REGSP},
	Optab{AFMOVD, C_FREG, C_NONE, C_NONE, C_LOREG, 35, 0},
	Optab{AFMOVD, C_FREG, C_NONE, C_NONE, C_ADDR, 74, 0},
	Optab{AFMOVD, C_ZCON, C_NONE, C_NONE, C_FREG, 67, 0},
	Optab{ACEFBRA, C_REG, C_NONE, C_NONE, C_FREG, 82, 0},
	Optab{ACFEBRA, C_FREG, C_NONE, C_NONE, C_REG, 83, 0},

	// load symbol address (plus offset)
	Optab{AMOVD, C_SYMADDR, C_NONE, C_NONE, C_REG, 19, 0},
	Optab{AMOVD, C_GOTADDR, C_NONE, C_NONE, C_REG, 93, 0},
	Optab{AMOVD, C_TLS_LE, C_NONE, C_NONE, C_REG, 94, 0},
	Optab{AMOVD, C_TLS_IE, C_NONE, C_NONE, C_REG, 95, 0},

	// system call
	Optab{ASYSCALL, C_NONE, C_NONE, C_NONE, C_NONE, 5, 0},
	Optab{ASYSCALL, C_SCON, C_NONE, C_NONE, C_NONE, 77, 0},

	// branch
	Optab{ABEQ, C_NONE, C_NONE, C_NONE, C_SBRA, 16, 0},
	Optab{ABR, C_NONE, C_NONE, C_NONE, C_LBRA, 11, 0},
	Optab{ABC, C_SCON, C_REG, C_NONE, C_SBRA, 16, 0},
	Optab{ABC, C_SCON, C_REG, C_NONE, C_LBRA, 17, 0},
	Optab{ABR, C_NONE, C_NONE, C_NONE, C_REG, 18, 0},
	Optab{ABR, C_REG, C_NONE, C_NONE, C_REG, 18, 0},
	Optab{ABR, C_NONE, C_NONE, C_NONE, C_ZOREG, 15, 0},
	Optab{ABC, C_NONE, C_NONE, C_NONE, C_ZOREG, 15, 0},
	Optab{ACMPBEQ, C_REG, C_REG, C_NONE, C_SBRA, 89, 0},
	Optab{ACMPBEQ, C_REG, C_NONE, C_ADDCON, C_SBRA, 90, 0},
	Optab{ACMPBEQ, C_REG, C_NONE, C_SCON, C_SBRA, 90, 0},
	Optab{ACMPUBEQ, C_REG, C_REG, C_NONE, C_SBRA, 89, 0},
	Optab{ACMPUBEQ, C_REG, C_NONE, C_ANDCON, C_SBRA, 90, 0},

	// compare
	Optab{ACMP, C_REG, C_NONE, C_NONE, C_REG, 70, 0},
	Optab{ACMP, C_REG, C_NONE, C_NONE, C_LCON, 71, 0},
	Optab{ACMPU, C_REG, C_NONE, C_NONE, C_REG, 70, 0},
	Optab{ACMPU, C_REG, C_NONE, C_NONE, C_LCON, 71, 0},
	Optab{AFCMPO, C_FREG, C_NONE, C_NONE, C_FREG, 70, 0},
	Optab{AFCMPO, C_FREG, C_REG, C_NONE, C_FREG, 70, 0},

	// 32-bit access registers
	Optab{AMOVW, C_AREG, C_NONE, C_NONE, C_REG, 68, 0},
	Optab{AMOVWZ, C_AREG, C_NONE, C_NONE, C_REG, 68, 0},
	Optab{AMOVW, C_REG, C_NONE, C_NONE, C_AREG, 69, 0},
	Optab{AMOVWZ, C_REG, C_NONE, C_NONE, C_AREG, 69, 0},

	// macros
	Optab{ACLEAR, C_LCON, C_NONE, C_NONE, C_LOREG, 96, 0},
	Optab{ACLEAR, C_LCON, C_NONE, C_NONE, C_LAUTO, 96, REGSP},

	// load/store multiple
	Optab{ASTMG, C_REG, C_REG, C_NONE, C_LOREG, 97, 0},
	Optab{ASTMG, C_REG, C_REG, C_NONE, C_LAUTO, 97, REGSP},
	Optab{ALMG, C_LOREG, C_REG, C_NONE, C_REG, 98, 0},
	Optab{ALMG, C_LAUTO, C_REG, C_NONE, C_REG, 98, REGSP},

	// bytes
	Optab{ABYTE, C_SCON, C_NONE, C_NONE, C_NONE, 40, 0},
	Optab{AWORD, C_LCON, C_NONE, C_NONE, C_NONE, 40, 0},
	Optab{ADWORD, C_LCON, C_NONE, C_NONE, C_NONE, 31, 0},
	Optab{ADWORD, C_DCON, C_NONE, C_NONE, C_NONE, 31, 0},

	// fast synchronization
	Optab{ASYNC, C_NONE, C_NONE, C_NONE, C_NONE, 81, 0},

	// store clock
	Optab{ASTCK, C_NONE, C_NONE, C_NONE, C_SAUTO, 88, REGSP},
	Optab{ASTCK, C_NONE, C_NONE, C_NONE, C_SOREG, 88, 0},

	// storage and storage
	Optab{AMVC, C_LOREG, C_NONE, C_SCON, C_LOREG, 84, 0},
	Optab{AMVC, C_LOREG, C_NONE, C_SCON, C_LAUTO, 84, REGSP},
	Optab{AMVC, C_LAUTO, C_NONE, C_SCON, C_LAUTO, 84, REGSP},

	// address
	Optab{ALARL, C_LCON, C_NONE, C_NONE, C_REG, 85, 0},
	Optab{ALARL, C_SYMADDR, C_NONE, C_NONE, C_REG, 85, 0},
	Optab{ALA, C_SOREG, C_NONE, C_NONE, C_REG, 86, 0},
	Optab{ALA, C_SAUTO, C_NONE, C_NONE, C_REG, 86, REGSP},
	Optab{AEXRL, C_SYMADDR, C_NONE, C_NONE, C_REG, 87, 0},

	// misc
	Optab{obj.AUNDEF, C_NONE, C_NONE, C_NONE, C_NONE, 78, 0},
	Optab{obj.APCDATA, C_LCON, C_NONE, C_NONE, C_LCON, 0, 0},
	Optab{obj.AFUNCDATA, C_SCON, C_NONE, C_NONE, C_ADDR, 0, 0},
	Optab{obj.ANOP, C_NONE, C_NONE, C_NONE, C_NONE, 0, 0},
	Optab{obj.ANOP, C_SAUTO, C_NONE, C_NONE, C_NONE, 0, 0},

	// vector instructions

	// VRX store
	Optab{AVST, C_VREG, C_NONE, C_NONE, C_SOREG, 100, 0},
	Optab{AVST, C_VREG, C_NONE, C_NONE, C_SAUTO, 100, REGSP},
	Optab{AVSTEG, C_VREG, C_NONE, C_SCON, C_SOREG, 100, 0},
	Optab{AVSTEG, C_VREG, C_NONE, C_SCON, C_SAUTO, 100, REGSP},

	// VRX load
	Optab{AVL, C_SOREG, C_NONE, C_NONE, C_VREG, 101, 0},
	Optab{AVL, C_SAUTO, C_NONE, C_NONE, C_VREG, 101, REGSP},
	Optab{AVLEG, C_SOREG, C_NONE, C_SCON, C_VREG, 101, 0},
	Optab{AVLEG, C_SAUTO, C_NONE, C_SCON, C_VREG, 101, REGSP},

	// VRV scatter
	Optab{AVSCEG, C_VREG, C_NONE, C_SCON, C_SOREG, 102, 0},
	Optab{AVSCEG, C_VREG, C_NONE, C_SCON, C_SAUTO, 102, REGSP},

	// VRV gather
	Optab{AVGEG, C_SOREG, C_NONE, C_SCON, C_VREG, 103, 0},
	Optab{AVGEG, C_SAUTO, C_NONE, C_SCON, C_VREG, 103, REGSP},

	// VRS element shift/rotate and load gr to/from vr element
	Optab{AVESLG, C_SCON, C_VREG, C_NONE, C_VREG, 104, 0},
	Optab{AVESLG, C_REG, C_VREG, C_NONE, C_VREG, 104, 0},
	Optab{AVESLG, C_SCON, C_NONE, C_NONE, C_VREG, 104, 0},
	Optab{AVESLG, C_REG, C_NONE, C_NONE, C_VREG, 104, 0},
	Optab{AVLGVG, C_SCON, C_VREG, C_NONE, C_REG, 104, 0},
	Optab{AVLGVG, C_REG, C_VREG, C_NONE, C_REG, 104, 0},
	Optab{AVLVGG, C_SCON, C_REG, C_NONE, C_VREG, 104, 0},
	Optab{AVLVGG, C_REG, C_REG, C_NONE, C_VREG, 104, 0},

	// VRS store multiple
	Optab{AVSTM, C_VREG, C_VREG, C_NONE, C_SOREG, 105, 0},
	Optab{AVSTM, C_VREG, C_VREG, C_NONE, C_SAUTO, 105, REGSP},

	// VRS load multiple
	Optab{AVLM, C_SOREG, C_VREG, C_NONE, C_VREG, 106, 0},
	Optab{AVLM, C_SAUTO, C_VREG, C_NONE, C_VREG, 106, REGSP},

	// VRS store with length
	Optab{AVSTL, C_VREG, C_NONE, C_REG, C_SOREG, 107, 0},
	Optab{AVSTL, C_VREG, C_NONE, C_REG, C_SAUTO, 107, REGSP},

	// VRS load with length
	Optab{AVLL, C_SOREG, C_NONE, C_REG, C_VREG, 108, 0},
	Optab{AVLL, C_SAUTO, C_NONE, C_REG, C_VREG, 108, REGSP},

	// VRI-a
	Optab{AVGBM, C_ANDCON, C_NONE, C_NONE, C_VREG, 109, 0},
	Optab{AVZERO, C_NONE, C_NONE, C_NONE, C_VREG, 109, 0},
	Optab{AVREPIG, C_ADDCON, C_NONE, C_NONE, C_VREG, 109, 0},
	Optab{AVREPIG, C_SCON, C_NONE, C_NONE, C_VREG, 109, 0},
	Optab{AVLEIG, C_ADDCON, C_NONE, C_SCON, C_VREG, 109, 0},
	Optab{AVLEIG, C_SCON, C_NONE, C_SCON, C_VREG, 109, 0},

	// VRI-b generate mask
	Optab{AVGMG, C_SCON, C_NONE, C_SCON, C_VREG, 110, 0},

	// VRI-c replicate
	Optab{AVREPG, C_UCON, C_VREG, C_NONE, C_VREG, 111, 0},

	// VRI-d element rotate and insert under mask and
	// shift left double by byte
	Optab{AVERIMG, C_VREG, C_VREG, C_SCON, C_VREG, 112, 0},
	Optab{AVSLDB, C_VREG, C_VREG, C_SCON, C_VREG, 112, 0},

	// VRI-d fp test data class immediate
	Optab{AVFTCIDB, C_SCON, C_VREG, C_NONE, C_VREG, 113, 0},

	// VRR-a load reg
	Optab{AVLR, C_VREG, C_NONE, C_NONE, C_VREG, 114, 0},

	// VRR-a compare
	Optab{AVECG, C_VREG, C_NONE, C_NONE, C_VREG, 115, 0},

	// VRR-b
	Optab{AVCEQG, C_VREG, C_VREG, C_NONE, C_VREG, 117, 0},
	Optab{AVFAEF, C_VREG, C_VREG, C_NONE, C_VREG, 117, 0},
	Optab{AVPKSG, C_VREG, C_VREG, C_NONE, C_VREG, 117, 0},

	// VRR-c
	Optab{AVAQ, C_VREG, C_VREG, C_NONE, C_VREG, 118, 0},
	Optab{AVAQ, C_VREG, C_NONE, C_NONE, C_VREG, 118, 0},
	Optab{AVNOT, C_VREG, C_NONE, C_NONE, C_VREG, 118, 0},
	Optab{AVPDI, C_VREG, C_VREG, C_SCON, C_VREG, 123, 0},

	// VRR-c shifts
	Optab{AVERLLVG, C_VREG, C_VREG, C_NONE, C_VREG, 119, 0},
	Optab{AVERLLVG, C_VREG, C_NONE, C_NONE, C_VREG, 119, 0},

	// VRR-d
	//             2       3       1       4
	Optab{AVACQ, C_VREG, C_VREG, C_VREG, C_VREG, 120, 0},

	// VRR-e
	Optab{AVSEL, C_VREG, C_VREG, C_VREG, C_VREG, 121, 0},

	// VRR-f
	Optab{AVLVGP, C_REG, C_REG, C_NONE, C_VREG, 122, 0},

	Optab{obj.AXXX, C_NONE, C_NONE, C_NONE, C_NONE, 0, 0},
}

type Oprang struct {
	start []Optab
	stop  []Optab
}

var oprange [ALAST & obj.AMask]Oprang

var xcmp [C_NCLASS][C_NCLASS]uint8

func spanz(ctxt *obj.Link, cursym *obj.LSym) {
	p := cursym.Text
	if p == nil || p.Link == nil { // handle external functions and ELF section symbols
		return
	}
	ctxt.Cursym = cursym
	ctxt.Autosize = int32(p.To.Offset)

	if oprange[AANDN&obj.AMask].start == nil {
		buildop(ctxt)
	}

	buffer := make([]byte, 0)
	changed := true
	loop := 0
	for changed {
		if loop > 10 {
			ctxt.Diag("stuck in spanz loop")
			break
		}
		changed = false
		buffer = buffer[:0]
		ctxt.Cursym.R = make([]obj.Reloc, 0)
		for p := cursym.Text; p != nil; p = p.Link {
			pc := int64(len(buffer))
			if pc != p.Pc {
				changed = true
			}
			p.Pc = pc
			ctxt.Pc = p.Pc
			ctxt.Curp = p
			asmout(ctxt, &buffer)
			if pc == int64(len(buffer)) {
				switch p.As {
				case obj.ANOP, obj.AFUNCDATA, obj.APCDATA, obj.ATEXT:
					// ok
				default:
					ctxt.Diag("zero-width instruction\n%v", p)
				}
			}
		}
		loop++
	}

	cursym.Size = int64(len(buffer))
	if cursym.Size%FuncAlign != 0 {
		cursym.Size += FuncAlign - (cursym.Size % FuncAlign)
	}
	obj.Symgrow(ctxt, cursym, cursym.Size)
	copy(cursym.P, buffer)
}

func isint32(v int64) bool {
	return int64(int32(v)) == v
}

func isuint32(v uint64) bool {
	return uint64(uint32(v)) == v
}

func aclass(ctxt *obj.Link, a *obj.Addr) int {
	switch a.Type {
	case obj.TYPE_NONE:
		return C_NONE

	case obj.TYPE_REG:
		if REG_R0 <= a.Reg && a.Reg <= REG_R15 {
			return C_REG
		}
		if REG_F0 <= a.Reg && a.Reg <= REG_F15 {
			return C_FREG
		}
		if REG_AR0 <= a.Reg && a.Reg <= REG_AR15 {
			return C_AREG
		}
		if REG_V0 <= a.Reg && a.Reg <= REG_V31 {
			return C_VREG
		}
		return C_GOK

	case obj.TYPE_MEM:
		switch a.Name {
		case obj.NAME_EXTERN,
			obj.NAME_STATIC:
			if a.Sym == nil {
				// must have a symbol
				break
			}
			ctxt.Instoffset = a.Offset
			if a.Sym.Type == obj.STLSBSS {
				if ctxt.Flag_shared != 0 {
					return C_TLS_IE // initial exec model
				}
				return C_TLS_LE // local exec model
			}
			return C_ADDR

		case obj.NAME_GOTREF:
			return C_GOTADDR

		case obj.NAME_AUTO:
			ctxt.Instoffset = int64(ctxt.Autosize) + a.Offset
			if ctxt.Instoffset >= -BIG && ctxt.Instoffset < BIG {
				return C_SAUTO
			}
			return C_LAUTO

		case obj.NAME_PARAM:
			ctxt.Instoffset = int64(ctxt.Autosize) + a.Offset + ctxt.FixedFrameSize()
			if ctxt.Instoffset >= -BIG && ctxt.Instoffset < BIG {
				return C_SAUTO
			}
			return C_LAUTO

		case obj.NAME_NONE:
			ctxt.Instoffset = a.Offset
			if ctxt.Instoffset == 0 {
				return C_ZOREG
			}
			if ctxt.Instoffset >= -BIG && ctxt.Instoffset < BIG {
				return C_SOREG
			}
			return C_LOREG
		}

		return C_GOK

	case obj.TYPE_TEXTSIZE:
		return C_TEXTSIZE

	case obj.TYPE_FCONST:
		if f64, ok := a.Val.(float64); ok && math.Float64bits(f64) == 0 {
			return C_ZCON
		}
		ctxt.Diag("cannot handle the floating point constant %v", a.Val)

	case obj.TYPE_CONST,
		obj.TYPE_ADDR:
		switch a.Name {
		case obj.TYPE_NONE:
			ctxt.Instoffset = a.Offset
			if a.Reg != 0 {
				if -BIG <= ctxt.Instoffset && ctxt.Instoffset <= BIG {
					return C_SACON
				}
				if isint32(ctxt.Instoffset) {
					return C_LACON
				}
				return C_DACON
			}
			goto consize

		case obj.NAME_EXTERN,
			obj.NAME_STATIC:
			s := a.Sym
			if s == nil {
				break
			}
			ctxt.Instoffset = s.Value + a.Offset
			if s.Type == obj.SCONST {
				goto consize
			}

			return C_SYMADDR

		case obj.NAME_AUTO:
			ctxt.Instoffset = int64(ctxt.Autosize) + a.Offset
			if ctxt.Instoffset >= -BIG && ctxt.Instoffset < BIG {
				return C_SACON
			}
			return C_LACON

		case obj.NAME_PARAM:
			ctxt.Instoffset = int64(ctxt.Autosize) + a.Offset + ctxt.FixedFrameSize()
			if ctxt.Instoffset >= -BIG && ctxt.Instoffset < BIG {
				return C_SACON
			}
			return C_LACON
		}

		return C_GOK

	consize:
		if ctxt.Instoffset == 0 {
			return C_ZCON
		}
		if ctxt.Instoffset >= 0 {
			if ctxt.Instoffset <= 0x7fff {
				return C_SCON
			}
			if ctxt.Instoffset <= 0xffff {
				return C_ANDCON
			}
			if ctxt.Instoffset&0xffff == 0 && isuint32(uint64(ctxt.Instoffset)) { /* && (instoffset & (1<<31)) == 0) */
				return C_UCON
			}
			if isint32(ctxt.Instoffset) || isuint32(uint64(ctxt.Instoffset)) {
				return C_LCON
			}
			return C_DCON
		}

		if ctxt.Instoffset >= -0x8000 {
			return C_ADDCON
		}
		if ctxt.Instoffset&0xffff == 0 && isint32(ctxt.Instoffset) {
			return C_UCON
		}
		if isint32(ctxt.Instoffset) {
			return C_LCON
		}
		return C_DCON

	case obj.TYPE_BRANCH:
		return C_SBRA
	}

	return C_GOK
}

func oplook(ctxt *obj.Link, p *obj.Prog) *Optab {
	a1 := int(p.Optab)
	if a1 != 0 {
		return &optab[a1-1:][0]
	}
	a1 = int(p.From.Class)
	if a1 == 0 {
		a1 = aclass(ctxt, &p.From) + 1
		p.From.Class = int8(a1)
	}

	a1--
	a3 := C_NONE + 1
	if p.From3 != nil {
		a3 = int(p.From3.Class)
		if a3 == 0 {
			a3 = aclass(ctxt, p.From3) + 1
			p.From3.Class = int8(a3)
		}
	}

	a3--
	a4 := int(p.To.Class)
	if a4 == 0 {
		a4 = aclass(ctxt, &p.To) + 1
		p.To.Class = int8(a4)
	}

	a4--
	a2 := C_NONE
	if p.Reg != 0 {
		if REG_R0 <= p.Reg && p.Reg <= REG_R15 {
			a2 = C_REG
		} else if REG_V0 <= p.Reg && p.Reg <= REG_V31 {
			a2 = C_VREG
		} else if REG_F0 <= p.Reg && p.Reg <= REG_F15 {
			a2 = C_FREG
		} else if REG_AR0 <= p.Reg && p.Reg <= REG_AR15 {
			a2 = C_AREG
		}
	}

	r0 := p.As & obj.AMask

	o := oprange[r0].start
	if o == nil {
		o = oprange[r0].stop /* just generate an error */
	}

	e := oprange[r0].stop
	c1 := xcmp[a1][:]
	c3 := xcmp[a3][:]
	c4 := xcmp[a4][:]
	for ; -cap(o) < -cap(e); o = o[1:] {
		if int(o[0].a2) == a2 {
			if c1[o[0].a1] != 0 {
				if c3[o[0].a3] != 0 {
					if c4[o[0].a4] != 0 {
						p.Optab = uint16((-cap(o) + cap(optab)) + 1)
						return &o[0]
					}
				}
			}
		}
	}

	// cannot find a case; abort
	ctxt.Diag("illegal combination %v %v %v %v %v\n", obj.Aconv(int(p.As)), DRconv(a1), DRconv(a2), DRconv(a3), DRconv(a4))
	ctxt.Diag("prog: %v\n", p)
	return nil
}

func cmp(a int, b int) bool {
	if a == b {
		return true
	}
	switch a {
	case C_DCON:
		if b == C_LCON {
			return true
		}
		fallthrough
	case C_LCON:
		if b == C_ZCON || b == C_SCON || b == C_UCON || b == C_ADDCON || b == C_ANDCON {
			return true
		}

	case C_ADDCON:
		if b == C_ZCON || b == C_SCON {
			return true
		}

	case C_ANDCON:
		if b == C_ZCON || b == C_SCON {
			return true
		}

	case C_UCON:
		if b == C_ZCON || b == C_SCON {
			return true
		}

	case C_SCON:
		if b == C_ZCON {
			return true
		}

	case C_LACON:
		if b == C_SACON {
			return true
		}

	case C_LBRA:
		if b == C_SBRA {
			return true
		}

	case C_LAUTO:
		if b == C_SAUTO {
			return true
		}

	case C_LOREG:
		if b == C_ZOREG || b == C_SOREG {
			return true
		}

	case C_SOREG:
		if b == C_ZOREG {
			return true
		}

	case C_ANY:
		return true
	}

	return false
}

type ocmp []Optab

func (x ocmp) Len() int {
	return len(x)
}

func (x ocmp) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x ocmp) Less(i, j int) bool {
	p1 := &x[i]
	p2 := &x[j]
	n := int(p1.as) - int(p2.as)
	if n != 0 {
		return n < 0
	}
	n = int(p1.a1) - int(p2.a1)
	if n != 0 {
		return n < 0
	}
	n = int(p1.a2) - int(p2.a2)
	if n != 0 {
		return n < 0
	}
	n = int(p1.a3) - int(p2.a3)
	if n != 0 {
		return n < 0
	}
	n = int(p1.a4) - int(p2.a4)
	if n != 0 {
		return n < 0
	}
	return false
}
func opset(a, b0 int16) {
	oprange[a&obj.AMask] = oprange[b0]
}

func buildop(ctxt *obj.Link) {
	var n int

	for i := 0; i < C_NCLASS; i++ {
		for n = 0; n < C_NCLASS; n++ {
			if cmp(n, i) {
				xcmp[i][n] = 1
			}
		}
	}
	for n = 0; optab[n].as != obj.AXXX; n++ {
	}
	sort.Sort(ocmp(optab[:n]))
	for i := 0; i < n; i++ {
		r := optab[i].as
		r0 := r & obj.AMask
		oprange[r0].start = optab[i:]
		for optab[i].as == r {
			i++
		}
		oprange[r0].stop = optab[i:]
		i--

		// opset() aliases optab ranges for similar instructions, to reduce the number of optabs in the array.
		// oprange[] is used by oplook() to find the Optab entry that applies to a given Prog.
		switch r {
		default:
			ctxt.Diag("unknown op in build: %v", obj.Aconv(int(r)))
			log.Fatalf("bad code")

		case ADIVW: /* op Rb[,Ra],Rd */
			opset(AADDE, r0)
			opset(AMULLD, r0)
			opset(AMULHDU, r0)
			opset(ADIVD, r0)
			opset(ADIVDU, r0)
			opset(ADIVWU, r0)

		case AMOVBZ: /* lbz, stz, rlwm(r/r), lhz, lha, stz, and x variants */
			opset(AMOVH, r0)
			opset(AMOVHZ, r0)

		case ALA:
			opset(ALAY, r0)

		case ALARL:

		case AMVC:
			opset(ACLC, r0)
			opset(AXC, r0)
			opset(AOC, r0)
			opset(ANC, r0)

		case AEXRL:

		case ASTCK:
			opset(ASTCKC, r0)
			opset(ASTCKE, r0)
			opset(ASTCKF, r0)

		case ACLEAR:

		case ASTMG:
			opset(ASTMY, r0)

		case ALMG:
			opset(ALMY, r0)

		case AAND: /* logical op Rb,Rs,Ra; no literal */
			opset(AANDN, r0)
			opset(ANAND, r0)
			opset(ANOR, r0)
			opset(AORN, r0)

		case AADDME: /* op Ra, Rd */
			opset(AADDZE, r0)
			opset(ASUBME, r0)
			opset(ASUBZE, r0)

		case AADDC:

		case ABEQ:
			opset(ABGE, r0)
			opset(ABGT, r0)
			opset(ABLE, r0)
			opset(ABLT, r0)
			opset(ABNE, r0)
			opset(ABVC, r0)
			opset(ABVS, r0)

		case ABR:
			opset(ABL, r0)

		case ABC:
			opset(ABCL, r0)

		case AFABS: /* fop [s,]d */
			opset(AFNABS, r0)
			opset(AFNEG, r0)
			opset(ALEDBR, r0)
			opset(ALDEBR, r0)
			opset(AFSQRT, r0)
			opset(AFSQRTS, r0)

		case AFADD:
			opset(AFADDS, r0)
			opset(AFDIV, r0)
			opset(AFDIVS, r0)
			opset(AFSUB, r0)
			opset(AFSUBS, r0)

		case AFMADD:
			opset(AFMADDS, r0)
			opset(AFMSUB, r0)
			opset(AFMSUBS, r0)
			opset(AFNMADD, r0)
			opset(AFNMADDS, r0)
			opset(AFNMSUB, r0)
			opset(AFNMSUBS, r0)

		case AFMUL:
			opset(AFMULS, r0)

		case AFCMPO:
			opset(AFCMPU, r0)
			opset(ACEBR, r0)

		case ANEG: /* op [Ra,] Rd */

		case AOR: /* or/xor Rb,Rs,Ra; ori/xori $uimm,Rs,Ra; oris/xoris $uimm,Rs,Ra */
			opset(AXOR, r0)

		case ASLD:
			opset(ASRD, r0)
			opset(ASLW, r0)
			opset(ASRW, r0)
			opset(ASRAD, r0)
			opset(ASRAW, r0)
			opset(ARLL, r0)
			opset(ARLLG, r0)

		case ACSG:
			opset(ACS, r0)

		case ASUB: /* SUB Ra,Rb,Rd => subf Rd,ra,rb */
			opset(ASUBC, r0)
			opset(ASUBE, r0)

		case ASYNC:

		case AFMOVD:
			opset(AFMOVS, r0)

		case ASYSCALL: /* just the op; flow of control */

		case AMOVDBR:
			opset(AMOVWBR, r0)

		case AMOVHBR: // no reg-reg moves

		case ACMP:
			opset(ACMPW, r0)

		case ACMPU:
			opset(ACMPWU, r0)

		case ACEFBRA:
			opset(ACDFBRA, r0)
			opset(ACEGBRA, r0)
			opset(ACDGBRA, r0)
			opset(ACELFBR, r0)
			opset(ACDLFBR, r0)
			opset(ACELGBR, r0)
			opset(ACDLGBR, r0)

		case ACFEBRA:
			opset(ACFDBRA, r0)
			opset(ACGEBRA, r0)
			opset(ACGDBRA, r0)
			opset(ACLFEBR, r0)
			opset(ACLFDBR, r0)
			opset(ACLGEBR, r0)
			opset(ACLGDBR, r0)

		case ACMPBEQ:
			opset(ACMPBGE, r0)
			opset(ACMPBGT, r0)
			opset(ACMPBLE, r0)
			opset(ACMPBLT, r0)
			opset(ACMPBNE, r0)

		case ACMPUBEQ:
			opset(ACMPUBGE, r0)
			opset(ACMPUBGT, r0)
			opset(ACMPUBLE, r0)
			opset(ACMPUBLT, r0)
			opset(ACMPUBNE, r0)

		case AVL:
			opset(AVLLEZB, r0)
			opset(AVLLEZH, r0)
			opset(AVLLEZF, r0)
			opset(AVLLEZG, r0)
			opset(AVLREPB, r0)
			opset(AVLREPH, r0)
			opset(AVLREPF, r0)
			opset(AVLREPG, r0)

		case AVST:

		case AVLEG:
			opset(AVLBB, r0)
			opset(AVLEB, r0)
			opset(AVLEH, r0)
			opset(AVLEF, r0)
			opset(AVLEG, r0)
			opset(AVLREP, r0)

		case AVSTEG:
			opset(AVSTEB, r0)
			opset(AVSTEH, r0)
			opset(AVSTEF, r0)

		case AVSCEG:
			opset(AVSCEF, r0)

		case AVGEG:
			opset(AVGEF, r0)

		case AVESLG:
			opset(AVESLB, r0)
			opset(AVESLH, r0)
			opset(AVESLF, r0)
			opset(AVERLLB, r0)
			opset(AVERLLH, r0)
			opset(AVERLLF, r0)
			opset(AVERLLG, r0)
			opset(AVESRAB, r0)
			opset(AVESRAH, r0)
			opset(AVESRAF, r0)
			opset(AVESRAG, r0)
			opset(AVESRLB, r0)
			opset(AVESRLH, r0)
			opset(AVESRLF, r0)
			opset(AVESRLG, r0)

		case AVLGVG:
			opset(AVLGVB, r0)
			opset(AVLGVH, r0)
			opset(AVLGVF, r0)

		case AVLVGG:
			opset(AVLVGB, r0)
			opset(AVLVGH, r0)
			opset(AVLVGF, r0)

		case AVLL:

		case AVSTL:

		case AVLM:

		case AVSTM:

		case AVGBM:

		case AVZERO:
			opset(AVONE, r0)

		case AVREPIG:
			opset(AVREPIB, r0)
			opset(AVREPIH, r0)
			opset(AVREPIF, r0)

		case AVLEIG:
			opset(AVLEIB, r0)
			opset(AVLEIH, r0)
			opset(AVLEIF, r0)

		case AVGMG:
			opset(AVGMB, r0)
			opset(AVGMH, r0)
			opset(AVGMF, r0)

		case AVREPG:
			opset(AVREPB, r0)
			opset(AVREPH, r0)
			opset(AVREPF, r0)

		case AVERIMG:
			opset(AVERIMB, r0)
			opset(AVERIMH, r0)
			opset(AVERIMF, r0)

		case AVSLDB:

		case AVFTCIDB:
			opset(AWFTCIDB, r0)

		case AVLR:
			opset(AVUPHB, r0)
			opset(AVUPHH, r0)
			opset(AVUPHF, r0)
			opset(AVUPLHB, r0)
			opset(AVUPLHH, r0)
			opset(AVUPLHF, r0)
			opset(AVUPLB, r0)
			opset(AVUPLHW, r0)
			opset(AVUPLF, r0)
			opset(AVUPLLB, r0)
			opset(AVUPLLH, r0)
			opset(AVUPLLF, r0)
			opset(AVCLZB, r0)
			opset(AVCLZH, r0)
			opset(AVCLZF, r0)
			opset(AVCLZG, r0)
			opset(AVCTZB, r0)
			opset(AVCTZH, r0)
			opset(AVCTZF, r0)
			opset(AVCTZG, r0)
			opset(AVLDEB, r0)
			opset(AWLDEB, r0)
			opset(AVFLCDB, r0)
			opset(AWFLCDB, r0)
			opset(AVFLNDB, r0)
			opset(AWFLNDB, r0)
			opset(AVFLPDB, r0)
			opset(AWFLPDB, r0)
			opset(AVFSQDB, r0)
			opset(AWFSQDB, r0)
			opset(AVISTRB, r0)
			opset(AVISTRH, r0)
			opset(AVISTRF, r0)
			opset(AVISTRBS, r0)
			opset(AVISTRHS, r0)
			opset(AVISTRFS, r0)
			opset(AVLCB, r0)
			opset(AVLCH, r0)
			opset(AVLCF, r0)
			opset(AVLCG, r0)
			opset(AVLPB, r0)
			opset(AVLPH, r0)
			opset(AVLPF, r0)
			opset(AVLPG, r0)
			opset(AVPOPCT, r0)
			opset(AVSEGB, r0)
			opset(AVSEGH, r0)
			opset(AVSEGF, r0)

		case AVECG:
			opset(AVECB, r0)
			opset(AVECH, r0)
			opset(AVECF, r0)
			opset(AVECLB, r0)
			opset(AVECLH, r0)
			opset(AVECLF, r0)
			opset(AVECLG, r0)
			opset(AWFCDB, r0)
			opset(AWFKDB, r0)

		case AVCEQG:
			opset(AVCEQB, r0)
			opset(AVCEQH, r0)
			opset(AVCEQF, r0)
			opset(AVCEQBS, r0)
			opset(AVCEQHS, r0)
			opset(AVCEQFS, r0)
			opset(AVCEQGS, r0)
			opset(AVCHB, r0)
			opset(AVCHH, r0)
			opset(AVCHF, r0)
			opset(AVCHG, r0)
			opset(AVCHBS, r0)
			opset(AVCHHS, r0)
			opset(AVCHFS, r0)
			opset(AVCHGS, r0)
			opset(AVCHLB, r0)
			opset(AVCHLH, r0)
			opset(AVCHLF, r0)
			opset(AVCHLG, r0)
			opset(AVCHLBS, r0)
			opset(AVCHLHS, r0)
			opset(AVCHLFS, r0)
			opset(AVCHLGS, r0)

		case AVFAEF:
			opset(AVFAEB, r0)
			opset(AVFAEH, r0)
			opset(AVFAEBS, r0)
			opset(AVFAEHS, r0)
			opset(AVFAEFS, r0)
			opset(AVFAEZB, r0)
			opset(AVFAEZH, r0)
			opset(AVFAEZF, r0)
			opset(AVFAEZBS, r0)
			opset(AVFAEZHS, r0)
			opset(AVFAEZFS, r0)
			opset(AVFEEB, r0)
			opset(AVFEEH, r0)
			opset(AVFEEF, r0)
			opset(AVFEEBS, r0)
			opset(AVFEEHS, r0)
			opset(AVFEEFS, r0)
			opset(AVFEEZB, r0)
			opset(AVFEEZH, r0)
			opset(AVFEEZF, r0)
			opset(AVFEEZBS, r0)
			opset(AVFEEZHS, r0)
			opset(AVFEEZFS, r0)
			opset(AVFENEB, r0)
			opset(AVFENEH, r0)
			opset(AVFENEF, r0)
			opset(AVFENEBS, r0)
			opset(AVFENEHS, r0)
			opset(AVFENEFS, r0)
			opset(AVFENEZB, r0)
			opset(AVFENEZH, r0)
			opset(AVFENEZF, r0)
			opset(AVFENEZBS, r0)
			opset(AVFENEZHS, r0)
			opset(AVFENEZFS, r0)

		case AVPKSG:
			opset(AVPKSH, r0)
			opset(AVPKSF, r0)
			opset(AVPKSHS, r0)
			opset(AVPKSFS, r0)
			opset(AVPKSGS, r0)
			opset(AVPKLSH, r0)
			opset(AVPKLSF, r0)
			opset(AVPKLSG, r0)
			opset(AVPKLSHS, r0)
			opset(AVPKLSFS, r0)
			opset(AVPKLSGS, r0)

		case AVAQ:
			opset(AVAB, r0)
			opset(AVAH, r0)
			opset(AVAF, r0)
			opset(AVAG, r0)
			opset(AVACCB, r0)
			opset(AVACCH, r0)
			opset(AVACCF, r0)
			opset(AVACCG, r0)
			opset(AVACCQ, r0)
			opset(AVN, r0)
			opset(AVNC, r0)
			opset(AVAVGB, r0)
			opset(AVAVGH, r0)
			opset(AVAVGF, r0)
			opset(AVAVGG, r0)
			opset(AVAVGLB, r0)
			opset(AVAVGLH, r0)
			opset(AVAVGLF, r0)
			opset(AVAVGLG, r0)
			opset(AVCKSM, r0)
			opset(AVX, r0)
			opset(AVFADB, r0)
			opset(AWFADB, r0)
			opset(AVFCEDB, r0)
			opset(AVFCEDBS, r0)
			opset(AWFCEDB, r0)
			opset(AWFCEDBS, r0)
			opset(AVFCHDB, r0)
			opset(AVFCHDBS, r0)
			opset(AWFCHDB, r0)
			opset(AWFCHDBS, r0)
			opset(AVFCHEDB, r0)
			opset(AVFCHEDBS, r0)
			opset(AWFCHEDB, r0)
			opset(AWFCHEDBS, r0)
			opset(AVFMDB, r0)
			opset(AWFMDB, r0)
			opset(AVGFMB, r0)
			opset(AVGFMH, r0)
			opset(AVGFMF, r0)
			opset(AVGFMG, r0)
			opset(AVMXB, r0)
			opset(AVMXH, r0)
			opset(AVMXF, r0)
			opset(AVMXG, r0)
			opset(AVMXLB, r0)
			opset(AVMXLH, r0)
			opset(AVMXLF, r0)
			opset(AVMXLG, r0)
			opset(AVMNB, r0)
			opset(AVMNH, r0)
			opset(AVMNF, r0)
			opset(AVMNG, r0)
			opset(AVMNLB, r0)
			opset(AVMNLH, r0)
			opset(AVMNLF, r0)
			opset(AVMNLG, r0)
			opset(AVMRHB, r0)
			opset(AVMRHH, r0)
			opset(AVMRHF, r0)
			opset(AVMRHG, r0)
			opset(AVMRLB, r0)
			opset(AVMRLH, r0)
			opset(AVMRLF, r0)
			opset(AVMRLG, r0)
			opset(AVMEB, r0)
			opset(AVMEH, r0)
			opset(AVMEF, r0)
			opset(AVMLEB, r0)
			opset(AVMLEH, r0)
			opset(AVMLEF, r0)
			opset(AVMOB, r0)
			opset(AVMOH, r0)
			opset(AVMOF, r0)
			opset(AVMLOB, r0)
			opset(AVMLOH, r0)
			opset(AVMLOF, r0)
			opset(AVMHB, r0)
			opset(AVMHH, r0)
			opset(AVMHF, r0)
			opset(AVMLHB, r0)
			opset(AVMLHH, r0)
			opset(AVMLHF, r0)
			opset(AVMLH, r0)
			opset(AVMLHW, r0)
			opset(AVMLF, r0)
			opset(AVNO, r0)
			opset(AVO, r0)
			opset(AVPKH, r0)
			opset(AVPKF, r0)
			opset(AVPKG, r0)
			opset(AVSUMGH, r0)
			opset(AVSUMGF, r0)
			opset(AVSUMQF, r0)
			opset(AVSUMQG, r0)
			opset(AVSUMB, r0)
			opset(AVSUMH, r0)

		case AVNOT:

		case AVERLLVG:
			opset(AVERLLVB, r0)
			opset(AVERLLVH, r0)
			opset(AVERLLVF, r0)
			opset(AVESLVB, r0)
			opset(AVESLVH, r0)
			opset(AVESLVF, r0)
			opset(AVESLVG, r0)
			opset(AVESRAVB, r0)
			opset(AVESRAVH, r0)
			opset(AVESRAVF, r0)
			opset(AVESRAVG, r0)
			opset(AVESRLVB, r0)
			opset(AVESRLVH, r0)
			opset(AVESRLVF, r0)
			opset(AVESRLVG, r0)
			opset(AVFDDB, r0)
			opset(AWFDDB, r0)
			opset(AVFSDB, r0)
			opset(AWFSDB, r0)
			opset(AVSL, r0)
			opset(AVSLB, r0)
			opset(AVSRA, r0)
			opset(AVSRAB, r0)
			opset(AVSRL, r0)
			opset(AVSRLB, r0)
			opset(AVSF, r0)
			opset(AVSG, r0)
			opset(AVSQ, r0)
			opset(AVSCBIB, r0)
			opset(AVSCBIH, r0)
			opset(AVSCBIF, r0)
			opset(AVSCBIG, r0)
			opset(AVSCBIQ, r0)

		case AVACQ:
			opset(AVACCCQ, r0)
			opset(AVGFMAB, r0)
			opset(AVGFMAH, r0)
			opset(AVGFMAF, r0)
			opset(AVGFMAG, r0)
			opset(AVMALB, r0)
			opset(AVMALHW, r0)
			opset(AVMALF, r0)
			opset(AVMAHB, r0)
			opset(AVMAHH, r0)
			opset(AVMAHF, r0)
			opset(AVMALHB, r0)
			opset(AVMALHH, r0)
			opset(AVMALHF, r0)
			opset(AVMAEB, r0)
			opset(AVMAEH, r0)
			opset(AVMAEF, r0)
			opset(AVMALEB, r0)
			opset(AVMALEH, r0)
			opset(AVMALEF, r0)
			opset(AVMAOB, r0)
			opset(AVMAOH, r0)
			opset(AVMAOF, r0)
			opset(AVMALOB, r0)
			opset(AVMALOH, r0)
			opset(AVMALOF, r0)
			opset(AVSTRCB, r0)
			opset(AVSTRCH, r0)
			opset(AVSTRCF, r0)
			opset(AVSTRCBS, r0)
			opset(AVSTRCHS, r0)
			opset(AVSTRCFS, r0)
			opset(AVSTRCZB, r0)
			opset(AVSTRCZH, r0)
			opset(AVSTRCZF, r0)
			opset(AVSTRCZBS, r0)
			opset(AVSTRCZHS, r0)
			opset(AVSTRCZFS, r0)
			opset(AVSBCBIQ, r0)
			opset(AVSBIQ, r0)

		case AVSEL:
			opset(AVFMADB, r0)
			opset(AWFMADB, r0)
			opset(AVFMSDB, r0)
			opset(AWFMSDB, r0)
			opset(AVPERM, r0)

		case AVLVGP:

		case AVPDI:

		case AADD,
			AMOVW,
			/* load/store/move word with sign extension; special 32-bit move; move 32-bit literals */
			AMOVWZ, /* load/store/move word with zero extension; move 32-bit literals  */
			AMOVD,  /* load/store/move 64-bit values, including 32-bit literals with/without sign-extension */
			AMOVB,  /* macro: move byte with sign extension */
			AMULLW,
			/* op $s[,r2],r3; op r1[,r2],r3; no cc/v */
			ASUBC, /* op r1,$s,r3; op r1[,r2],r3 */
			ABYTE,
			AWORD,
			ADWORD,
			obj.ANOP,
			obj.ATEXT,
			obj.AUNDEF,
			obj.AFUNCDATA,
			obj.APCDATA:
			break
		}
	}
}

const (
	op_A       uint32 = 0x5A00 // FORMAT_RX1        ADD (32)
	op_AD      uint32 = 0x6A00 // FORMAT_RX1        ADD NORMALIZED (long HFP)
	op_ADB     uint32 = 0xED1A // FORMAT_RXE        ADD (long BFP)
	op_ADBR    uint32 = 0xB31A // FORMAT_RRE        ADD (long BFP)
	op_ADR     uint32 = 0x2A00 // FORMAT_RR         ADD NORMALIZED (long HFP)
	op_ADTR    uint32 = 0xB3D2 // FORMAT_RRF1       ADD (long DFP)
	op_ADTRA   uint32 = 0xB3D2 // FORMAT_RRF1       ADD (long DFP)
	op_AE      uint32 = 0x7A00 // FORMAT_RX1        ADD NORMALIZED (short HFP)
	op_AEB     uint32 = 0xED0A // FORMAT_RXE        ADD (short BFP)
	op_AEBR    uint32 = 0xB30A // FORMAT_RRE        ADD (short BFP)
	op_AER     uint32 = 0x3A00 // FORMAT_RR         ADD NORMALIZED (short HFP)
	op_AFI     uint32 = 0xC209 // FORMAT_RIL1       ADD IMMEDIATE (32)
	op_AG      uint32 = 0xE308 // FORMAT_RXY1       ADD (64)
	op_AGF     uint32 = 0xE318 // FORMAT_RXY1       ADD (64<-32)
	op_AGFI    uint32 = 0xC208 // FORMAT_RIL1       ADD IMMEDIATE (64<-32)
	op_AGFR    uint32 = 0xB918 // FORMAT_RRE        ADD (64<-32)
	op_AGHI    uint32 = 0xA70B // FORMAT_RI1        ADD HALFWORD IMMEDIATE (64)
	op_AGHIK   uint32 = 0xECD9 // FORMAT_RIE4       ADD IMMEDIATE (64<-16)
	op_AGR     uint32 = 0xB908 // FORMAT_RRE        ADD (64)
	op_AGRK    uint32 = 0xB9E8 // FORMAT_RRF1       ADD (64)
	op_AGSI    uint32 = 0xEB7A // FORMAT_SIY        ADD IMMEDIATE (64<-8)
	op_AH      uint32 = 0x4A00 // FORMAT_RX1        ADD HALFWORD
	op_AHHHR   uint32 = 0xB9C8 // FORMAT_RRF1       ADD HIGH (32)
	op_AHHLR   uint32 = 0xB9D8 // FORMAT_RRF1       ADD HIGH (32)
	op_AHI     uint32 = 0xA70A // FORMAT_RI1        ADD HALFWORD IMMEDIATE (32)
	op_AHIK    uint32 = 0xECD8 // FORMAT_RIE4       ADD IMMEDIATE (32<-16)
	op_AHY     uint32 = 0xE37A // FORMAT_RXY1       ADD HALFWORD
	op_AIH     uint32 = 0xCC08 // FORMAT_RIL1       ADD IMMEDIATE HIGH (32)
	op_AL      uint32 = 0x5E00 // FORMAT_RX1        ADD LOGICAL (32)
	op_ALC     uint32 = 0xE398 // FORMAT_RXY1       ADD LOGICAL WITH CARRY (32)
	op_ALCG    uint32 = 0xE388 // FORMAT_RXY1       ADD LOGICAL WITH CARRY (64)
	op_ALCGR   uint32 = 0xB988 // FORMAT_RRE        ADD LOGICAL WITH CARRY (64)
	op_ALCR    uint32 = 0xB998 // FORMAT_RRE        ADD LOGICAL WITH CARRY (32)
	op_ALFI    uint32 = 0xC20B // FORMAT_RIL1       ADD LOGICAL IMMEDIATE (32)
	op_ALG     uint32 = 0xE30A // FORMAT_RXY1       ADD LOGICAL (64)
	op_ALGF    uint32 = 0xE31A // FORMAT_RXY1       ADD LOGICAL (64<-32)
	op_ALGFI   uint32 = 0xC20A // FORMAT_RIL1       ADD LOGICAL IMMEDIATE (64<-32)
	op_ALGFR   uint32 = 0xB91A // FORMAT_RRE        ADD LOGICAL (64<-32)
	op_ALGHSIK uint32 = 0xECDB // FORMAT_RIE4       ADD LOGICAL WITH SIGNED IMMEDIATE (64<-16)
	op_ALGR    uint32 = 0xB90A // FORMAT_RRE        ADD LOGICAL (64)
	op_ALGRK   uint32 = 0xB9EA // FORMAT_RRF1       ADD LOGICAL (64)
	op_ALGSI   uint32 = 0xEB7E // FORMAT_SIY        ADD LOGICAL WITH SIGNED IMMEDIATE (64<-8)
	op_ALHHHR  uint32 = 0xB9CA // FORMAT_RRF1       ADD LOGICAL HIGH (32)
	op_ALHHLR  uint32 = 0xB9DA // FORMAT_RRF1       ADD LOGICAL HIGH (32)
	op_ALHSIK  uint32 = 0xECDA // FORMAT_RIE4       ADD LOGICAL WITH SIGNED IMMEDIATE (32<-16)
	op_ALR     uint32 = 0x1E00 // FORMAT_RR         ADD LOGICAL (32)
	op_ALRK    uint32 = 0xB9FA // FORMAT_RRF1       ADD LOGICAL (32)
	op_ALSI    uint32 = 0xEB6E // FORMAT_SIY        ADD LOGICAL WITH SIGNED IMMEDIATE (32<-8)
	op_ALSIH   uint32 = 0xCC0A // FORMAT_RIL1       ADD LOGICAL WITH SIGNED IMMEDIATE HIGH (32)
	op_ALSIHN  uint32 = 0xCC0B // FORMAT_RIL1       ADD LOGICAL WITH SIGNED IMMEDIATE HIGH (32)
	op_ALY     uint32 = 0xE35E // FORMAT_RXY1       ADD LOGICAL (32)
	op_AP      uint32 = 0xFA00 // FORMAT_SS2        ADD DECIMAL
	op_AR      uint32 = 0x1A00 // FORMAT_RR         ADD (32)
	op_ARK     uint32 = 0xB9F8 // FORMAT_RRF1       ADD (32)
	op_ASI     uint32 = 0xEB6A // FORMAT_SIY        ADD IMMEDIATE (32<-8)
	op_AU      uint32 = 0x7E00 // FORMAT_RX1        ADD UNNORMALIZED (short HFP)
	op_AUR     uint32 = 0x3E00 // FORMAT_RR         ADD UNNORMALIZED (short HFP)
	op_AW      uint32 = 0x6E00 // FORMAT_RX1        ADD UNNORMALIZED (long HFP)
	op_AWR     uint32 = 0x2E00 // FORMAT_RR         ADD UNNORMALIZED (long HFP)
	op_AXBR    uint32 = 0xB34A // FORMAT_RRE        ADD (extended BFP)
	op_AXR     uint32 = 0x3600 // FORMAT_RR         ADD NORMALIZED (extended HFP)
	op_AXTR    uint32 = 0xB3DA // FORMAT_RRF1       ADD (extended DFP)
	op_AXTRA   uint32 = 0xB3DA // FORMAT_RRF1       ADD (extended DFP)
	op_AY      uint32 = 0xE35A // FORMAT_RXY1       ADD (32)
	op_BAKR    uint32 = 0xB240 // FORMAT_RRE        BRANCH AND STACK
	op_BAL     uint32 = 0x4500 // FORMAT_RX1        BRANCH AND LINK
	op_BALR    uint32 = 0x0500 // FORMAT_RR         BRANCH AND LINK
	op_BAS     uint32 = 0x4D00 // FORMAT_RX1        BRANCH AND SAVE
	op_BASR    uint32 = 0x0D00 // FORMAT_RR         BRANCH AND SAVE
	op_BASSM   uint32 = 0x0C00 // FORMAT_RR         BRANCH AND SAVE AND SET MODE
	op_BC      uint32 = 0x4700 // FORMAT_RX2        BRANCH ON CONDITION
	op_BCR     uint32 = 0x0700 // FORMAT_RR         BRANCH ON CONDITION
	op_BCT     uint32 = 0x4600 // FORMAT_RX1        BRANCH ON COUNT (32)
	op_BCTG    uint32 = 0xE346 // FORMAT_RXY1       BRANCH ON COUNT (64)
	op_BCTGR   uint32 = 0xB946 // FORMAT_RRE        BRANCH ON COUNT (64)
	op_BCTR    uint32 = 0x0600 // FORMAT_RR         BRANCH ON COUNT (32)
	op_BPP     uint32 = 0xC700 // FORMAT_SMI        BRANCH PREDICTION PRELOAD
	op_BPRP    uint32 = 0xC500 // FORMAT_MII        BRANCH PREDICTION RELATIVE PRELOAD
	op_BRAS    uint32 = 0xA705 // FORMAT_RI2        BRANCH RELATIVE AND SAVE
	op_BRASL   uint32 = 0xC005 // FORMAT_RIL2       BRANCH RELATIVE AND SAVE LONG
	op_BRC     uint32 = 0xA704 // FORMAT_RI3        BRANCH RELATIVE ON CONDITION
	op_BRCL    uint32 = 0xC004 // FORMAT_RIL3       BRANCH RELATIVE ON CONDITION LONG
	op_BRCT    uint32 = 0xA706 // FORMAT_RI2        BRANCH RELATIVE ON COUNT (32)
	op_BRCTG   uint32 = 0xA707 // FORMAT_RI2        BRANCH RELATIVE ON COUNT (64)
	op_BRCTH   uint32 = 0xCC06 // FORMAT_RIL2       BRANCH RELATIVE ON COUNT HIGH (32)
	op_BRXH    uint32 = 0x8400 // FORMAT_RSI        BRANCH RELATIVE ON INDEX HIGH (32)
	op_BRXHG   uint32 = 0xEC44 // FORMAT_RIE5       BRANCH RELATIVE ON INDEX HIGH (64)
	op_BRXLE   uint32 = 0x8500 // FORMAT_RSI        BRANCH RELATIVE ON INDEX LOW OR EQ. (32)
	op_BRXLG   uint32 = 0xEC45 // FORMAT_RIE5       BRANCH RELATIVE ON INDEX LOW OR EQ. (64)
	op_BSA     uint32 = 0xB25A // FORMAT_RRE        BRANCH AND SET AUTHORITY
	op_BSG     uint32 = 0xB258 // FORMAT_RRE        BRANCH IN SUBSPACE GROUP
	op_BSM     uint32 = 0x0B00 // FORMAT_RR         BRANCH AND SET MODE
	op_BXH     uint32 = 0x8600 // FORMAT_RS1        BRANCH ON INDEX HIGH (32)
	op_BXHG    uint32 = 0xEB44 // FORMAT_RSY1       BRANCH ON INDEX HIGH (64)
	op_BXLE    uint32 = 0x8700 // FORMAT_RS1        BRANCH ON INDEX LOW OR EQUAL (32)
	op_BXLEG   uint32 = 0xEB45 // FORMAT_RSY1       BRANCH ON INDEX LOW OR EQUAL (64)
	op_C       uint32 = 0x5900 // FORMAT_RX1        COMPARE (32)
	op_CD      uint32 = 0x6900 // FORMAT_RX1        COMPARE (long HFP)
	op_CDB     uint32 = 0xED19 // FORMAT_RXE        COMPARE (long BFP)
	op_CDBR    uint32 = 0xB319 // FORMAT_RRE        COMPARE (long BFP)
	op_CDFBR   uint32 = 0xB395 // FORMAT_RRE        CONVERT FROM FIXED (32 to long BFP)
	op_CDFBRA  uint32 = 0xB395 // FORMAT_RRF5       CONVERT FROM FIXED (32 to long BFP)
	op_CDFR    uint32 = 0xB3B5 // FORMAT_RRE        CONVERT FROM FIXED (32 to long HFP)
	op_CDFTR   uint32 = 0xB951 // FORMAT_RRE        CONVERT FROM FIXED (32 to long DFP)
	op_CDGBR   uint32 = 0xB3A5 // FORMAT_RRE        CONVERT FROM FIXED (64 to long BFP)
	op_CDGBRA  uint32 = 0xB3A5 // FORMAT_RRF5       CONVERT FROM FIXED (64 to long BFP)
	op_CDGR    uint32 = 0xB3C5 // FORMAT_RRE        CONVERT FROM FIXED (64 to long HFP)
	op_CDGTR   uint32 = 0xB3F1 // FORMAT_RRE        CONVERT FROM FIXED (64 to long DFP)
	op_CDGTRA  uint32 = 0xB3F1 // FORMAT_RRF5       CONVERT FROM FIXED (64 to long DFP)
	op_CDLFBR  uint32 = 0xB391 // FORMAT_RRF5       CONVERT FROM LOGICAL (32 to long BFP)
	op_CDLFTR  uint32 = 0xB953 // FORMAT_RRF5       CONVERT FROM LOGICAL (32 to long DFP)
	op_CDLGBR  uint32 = 0xB3A1 // FORMAT_RRF5       CONVERT FROM LOGICAL (64 to long BFP)
	op_CDLGTR  uint32 = 0xB952 // FORMAT_RRF5       CONVERT FROM LOGICAL (64 to long DFP)
	op_CDR     uint32 = 0x2900 // FORMAT_RR         COMPARE (long HFP)
	op_CDS     uint32 = 0xBB00 // FORMAT_RS1        COMPARE DOUBLE AND SWAP (32)
	op_CDSG    uint32 = 0xEB3E // FORMAT_RSY1       COMPARE DOUBLE AND SWAP (64)
	op_CDSTR   uint32 = 0xB3F3 // FORMAT_RRE        CONVERT FROM SIGNED PACKED (64 to long DFP)
	op_CDSY    uint32 = 0xEB31 // FORMAT_RSY1       COMPARE DOUBLE AND SWAP (32)
	op_CDTR    uint32 = 0xB3E4 // FORMAT_RRE        COMPARE (long DFP)
	op_CDUTR   uint32 = 0xB3F2 // FORMAT_RRE        CONVERT FROM UNSIGNED PACKED (64 to long DFP)
	op_CDZT    uint32 = 0xEDAA // FORMAT_RSL        CONVERT FROM ZONED (to long DFP)
	op_CE      uint32 = 0x7900 // FORMAT_RX1        COMPARE (short HFP)
	op_CEB     uint32 = 0xED09 // FORMAT_RXE        COMPARE (short BFP)
	op_CEBR    uint32 = 0xB309 // FORMAT_RRE        COMPARE (short BFP)
	op_CEDTR   uint32 = 0xB3F4 // FORMAT_RRE        COMPARE BIASED EXPONENT (long DFP)
	op_CEFBR   uint32 = 0xB394 // FORMAT_RRE        CONVERT FROM FIXED (32 to short BFP)
	op_CEFBRA  uint32 = 0xB394 // FORMAT_RRF5       CONVERT FROM FIXED (32 to short BFP)
	op_CEFR    uint32 = 0xB3B4 // FORMAT_RRE        CONVERT FROM FIXED (32 to short HFP)
	op_CEGBR   uint32 = 0xB3A4 // FORMAT_RRE        CONVERT FROM FIXED (64 to short BFP)
	op_CEGBRA  uint32 = 0xB3A4 // FORMAT_RRF5       CONVERT FROM FIXED (64 to short BFP)
	op_CEGR    uint32 = 0xB3C4 // FORMAT_RRE        CONVERT FROM FIXED (64 to short HFP)
	op_CELFBR  uint32 = 0xB390 // FORMAT_RRF5       CONVERT FROM LOGICAL (32 to short BFP)
	op_CELGBR  uint32 = 0xB3A0 // FORMAT_RRF5       CONVERT FROM LOGICAL (64 to short BFP)
	op_CER     uint32 = 0x3900 // FORMAT_RR         COMPARE (short HFP)
	op_CEXTR   uint32 = 0xB3FC // FORMAT_RRE        COMPARE BIASED EXPONENT (extended DFP)
	op_CFC     uint32 = 0xB21A // FORMAT_S          COMPARE AND FORM CODEWORD
	op_CFDBR   uint32 = 0xB399 // FORMAT_RRF5       CONVERT TO FIXED (long BFP to 32)
	op_CFDBRA  uint32 = 0xB399 // FORMAT_RRF5       CONVERT TO FIXED (long BFP to 32)
	op_CFDR    uint32 = 0xB3B9 // FORMAT_RRF5       CONVERT TO FIXED (long HFP to 32)
	op_CFDTR   uint32 = 0xB941 // FORMAT_RRF5       CONVERT TO FIXED (long DFP to 32)
	op_CFEBR   uint32 = 0xB398 // FORMAT_RRF5       CONVERT TO FIXED (short BFP to 32)
	op_CFEBRA  uint32 = 0xB398 // FORMAT_RRF5       CONVERT TO FIXED (short BFP to 32)
	op_CFER    uint32 = 0xB3B8 // FORMAT_RRF5       CONVERT TO FIXED (short HFP to 32)
	op_CFI     uint32 = 0xC20D // FORMAT_RIL1       COMPARE IMMEDIATE (32)
	op_CFXBR   uint32 = 0xB39A // FORMAT_RRF5       CONVERT TO FIXED (extended BFP to 32)
	op_CFXBRA  uint32 = 0xB39A // FORMAT_RRF5       CONVERT TO FIXED (extended BFP to 32)
	op_CFXR    uint32 = 0xB3BA // FORMAT_RRF5       CONVERT TO FIXED (extended HFP to 32)
	op_CFXTR   uint32 = 0xB949 // FORMAT_RRF5       CONVERT TO FIXED (extended DFP to 32)
	op_CG      uint32 = 0xE320 // FORMAT_RXY1       COMPARE (64)
	op_CGDBR   uint32 = 0xB3A9 // FORMAT_RRF5       CONVERT TO FIXED (long BFP to 64)
	op_CGDBRA  uint32 = 0xB3A9 // FORMAT_RRF5       CONVERT TO FIXED (long BFP to 64)
	op_CGDR    uint32 = 0xB3C9 // FORMAT_RRF5       CONVERT TO FIXED (long HFP to 64)
	op_CGDTR   uint32 = 0xB3E1 // FORMAT_RRF5       CONVERT TO FIXED (long DFP to 64)
	op_CGDTRA  uint32 = 0xB3E1 // FORMAT_RRF5       CONVERT TO FIXED (long DFP to 64)
	op_CGEBR   uint32 = 0xB3A8 // FORMAT_RRF5       CONVERT TO FIXED (short BFP to 64)
	op_CGEBRA  uint32 = 0xB3A8 // FORMAT_RRF5       CONVERT TO FIXED (short BFP to 64)
	op_CGER    uint32 = 0xB3C8 // FORMAT_RRF5       CONVERT TO FIXED (short HFP to 64)
	op_CGF     uint32 = 0xE330 // FORMAT_RXY1       COMPARE (64<-32)
	op_CGFI    uint32 = 0xC20C // FORMAT_RIL1       COMPARE IMMEDIATE (64<-32)
	op_CGFR    uint32 = 0xB930 // FORMAT_RRE        COMPARE (64<-32)
	op_CGFRL   uint32 = 0xC60C // FORMAT_RIL2       COMPARE RELATIVE LONG (64<-32)
	op_CGH     uint32 = 0xE334 // FORMAT_RXY1       COMPARE HALFWORD (64<-16)
	op_CGHI    uint32 = 0xA70F // FORMAT_RI1        COMPARE HALFWORD IMMEDIATE (64<-16)
	op_CGHRL   uint32 = 0xC604 // FORMAT_RIL2       COMPARE HALFWORD RELATIVE LONG (64<-16)
	op_CGHSI   uint32 = 0xE558 // FORMAT_SIL        COMPARE HALFWORD IMMEDIATE (64<-16)
	op_CGIB    uint32 = 0xECFC // FORMAT_RIS        COMPARE IMMEDIATE AND BRANCH (64<-8)
	op_CGIJ    uint32 = 0xEC7C // FORMAT_RIE3       COMPARE IMMEDIATE AND BRANCH RELATIVE (64<-8)
	op_CGIT    uint32 = 0xEC70 // FORMAT_RIE1       COMPARE IMMEDIATE AND TRAP (64<-16)
	op_CGR     uint32 = 0xB920 // FORMAT_RRE        COMPARE (64)
	op_CGRB    uint32 = 0xECE4 // FORMAT_RRS        COMPARE AND BRANCH (64)
	op_CGRJ    uint32 = 0xEC64 // FORMAT_RIE2       COMPARE AND BRANCH RELATIVE (64)
	op_CGRL    uint32 = 0xC608 // FORMAT_RIL2       COMPARE RELATIVE LONG (64)
	op_CGRT    uint32 = 0xB960 // FORMAT_RRF3       COMPARE AND TRAP (64)
	op_CGXBR   uint32 = 0xB3AA // FORMAT_RRF5       CONVERT TO FIXED (extended BFP to 64)
	op_CGXBRA  uint32 = 0xB3AA // FORMAT_RRF5       CONVERT TO FIXED (extended BFP to 64)
	op_CGXR    uint32 = 0xB3CA // FORMAT_RRF5       CONVERT TO FIXED (extended HFP to 64)
	op_CGXTR   uint32 = 0xB3E9 // FORMAT_RRF5       CONVERT TO FIXED (extended DFP to 64)
	op_CGXTRA  uint32 = 0xB3E9 // FORMAT_RRF5       CONVERT TO FIXED (extended DFP to 64)
	op_CH      uint32 = 0x4900 // FORMAT_RX1        COMPARE HALFWORD (32<-16)
	op_CHF     uint32 = 0xE3CD // FORMAT_RXY1       COMPARE HIGH (32)
	op_CHHR    uint32 = 0xB9CD // FORMAT_RRE        COMPARE HIGH (32)
	op_CHHSI   uint32 = 0xE554 // FORMAT_SIL        COMPARE HALFWORD IMMEDIATE (16)
	op_CHI     uint32 = 0xA70E // FORMAT_RI1        COMPARE HALFWORD IMMEDIATE (32<-16)
	op_CHLR    uint32 = 0xB9DD // FORMAT_RRE        COMPARE HIGH (32)
	op_CHRL    uint32 = 0xC605 // FORMAT_RIL2       COMPARE HALFWORD RELATIVE LONG (32<-16)
	op_CHSI    uint32 = 0xE55C // FORMAT_SIL        COMPARE HALFWORD IMMEDIATE (32<-16)
	op_CHY     uint32 = 0xE379 // FORMAT_RXY1       COMPARE HALFWORD (32<-16)
	op_CIB     uint32 = 0xECFE // FORMAT_RIS        COMPARE IMMEDIATE AND BRANCH (32<-8)
	op_CIH     uint32 = 0xCC0D // FORMAT_RIL1       COMPARE IMMEDIATE HIGH (32)
	op_CIJ     uint32 = 0xEC7E // FORMAT_RIE3       COMPARE IMMEDIATE AND BRANCH RELATIVE (32<-8)
	op_CIT     uint32 = 0xEC72 // FORMAT_RIE1       COMPARE IMMEDIATE AND TRAP (32<-16)
	op_CKSM    uint32 = 0xB241 // FORMAT_RRE        CHECKSUM
	op_CL      uint32 = 0x5500 // FORMAT_RX1        COMPARE LOGICAL (32)
	op_CLC     uint32 = 0xD500 // FORMAT_SS1        COMPARE LOGICAL (character)
	op_CLCL    uint32 = 0x0F00 // FORMAT_RR         COMPARE LOGICAL LONG
	op_CLCLE   uint32 = 0xA900 // FORMAT_RS1        COMPARE LOGICAL LONG EXTENDED
	op_CLCLU   uint32 = 0xEB8F // FORMAT_RSY1       COMPARE LOGICAL LONG UNICODE
	op_CLFDBR  uint32 = 0xB39D // FORMAT_RRF5       CONVERT TO LOGICAL (long BFP to 32)
	op_CLFDTR  uint32 = 0xB943 // FORMAT_RRF5       CONVERT TO LOGICAL (long DFP to 32)
	op_CLFEBR  uint32 = 0xB39C // FORMAT_RRF5       CONVERT TO LOGICAL (short BFP to 32)
	op_CLFHSI  uint32 = 0xE55D // FORMAT_SIL        COMPARE LOGICAL IMMEDIATE (32<-16)
	op_CLFI    uint32 = 0xC20F // FORMAT_RIL1       COMPARE LOGICAL IMMEDIATE (32)
	op_CLFIT   uint32 = 0xEC73 // FORMAT_RIE1       COMPARE LOGICAL IMMEDIATE AND TRAP (32<-16)
	op_CLFXBR  uint32 = 0xB39E // FORMAT_RRF5       CONVERT TO LOGICAL (extended BFP to 32)
	op_CLFXTR  uint32 = 0xB94B // FORMAT_RRF5       CONVERT TO LOGICAL (extended DFP to 32)
	op_CLG     uint32 = 0xE321 // FORMAT_RXY1       COMPARE LOGICAL (64)
	op_CLGDBR  uint32 = 0xB3AD // FORMAT_RRF5       CONVERT TO LOGICAL (long BFP to 64)
	op_CLGDTR  uint32 = 0xB942 // FORMAT_RRF5       CONVERT TO LOGICAL (long DFP to 64)
	op_CLGEBR  uint32 = 0xB3AC // FORMAT_RRF5       CONVERT TO LOGICAL (short BFP to 64)
	op_CLGF    uint32 = 0xE331 // FORMAT_RXY1       COMPARE LOGICAL (64<-32)
	op_CLGFI   uint32 = 0xC20E // FORMAT_RIL1       COMPARE LOGICAL IMMEDIATE (64<-32)
	op_CLGFR   uint32 = 0xB931 // FORMAT_RRE        COMPARE LOGICAL (64<-32)
	op_CLGFRL  uint32 = 0xC60E // FORMAT_RIL2       COMPARE LOGICAL RELATIVE LONG (64<-32)
	op_CLGHRL  uint32 = 0xC606 // FORMAT_RIL2       COMPARE LOGICAL RELATIVE LONG (64<-16)
	op_CLGHSI  uint32 = 0xE559 // FORMAT_SIL        COMPARE LOGICAL IMMEDIATE (64<-16)
	op_CLGIB   uint32 = 0xECFD // FORMAT_RIS        COMPARE LOGICAL IMMEDIATE AND BRANCH (64<-8)
	op_CLGIJ   uint32 = 0xEC7D // FORMAT_RIE3       COMPARE LOGICAL IMMEDIATE AND BRANCH RELATIVE (64<-8)
	op_CLGIT   uint32 = 0xEC71 // FORMAT_RIE1       COMPARE LOGICAL IMMEDIATE AND TRAP (64<-16)
	op_CLGR    uint32 = 0xB921 // FORMAT_RRE        COMPARE LOGICAL (64)
	op_CLGRB   uint32 = 0xECE5 // FORMAT_RRS        COMPARE LOGICAL AND BRANCH (64)
	op_CLGRJ   uint32 = 0xEC65 // FORMAT_RIE2       COMPARE LOGICAL AND BRANCH RELATIVE (64)
	op_CLGRL   uint32 = 0xC60A // FORMAT_RIL2       COMPARE LOGICAL RELATIVE LONG (64)
	op_CLGRT   uint32 = 0xB961 // FORMAT_RRF3       COMPARE LOGICAL AND TRAP (64)
	op_CLGT    uint32 = 0xEB2B // FORMAT_RSY2       COMPARE LOGICAL AND TRAP (64)
	op_CLGXBR  uint32 = 0xB3AE // FORMAT_RRF5       CONVERT TO LOGICAL (extended BFP to 64)
	op_CLGXTR  uint32 = 0xB94A // FORMAT_RRF5       CONVERT TO LOGICAL (extended DFP to 64)
	op_CLHF    uint32 = 0xE3CF // FORMAT_RXY1       COMPARE LOGICAL HIGH (32)
	op_CLHHR   uint32 = 0xB9CF // FORMAT_RRE        COMPARE LOGICAL HIGH (32)
	op_CLHHSI  uint32 = 0xE555 // FORMAT_SIL        COMPARE LOGICAL IMMEDIATE (16)
	op_CLHLR   uint32 = 0xB9DF // FORMAT_RRE        COMPARE LOGICAL HIGH (32)
	op_CLHRL   uint32 = 0xC607 // FORMAT_RIL2       COMPARE LOGICAL RELATIVE LONG (32<-16)
	op_CLI     uint32 = 0x9500 // FORMAT_SI         COMPARE LOGICAL (immediate)
	op_CLIB    uint32 = 0xECFF // FORMAT_RIS        COMPARE LOGICAL IMMEDIATE AND BRANCH (32<-8)
	op_CLIH    uint32 = 0xCC0F // FORMAT_RIL1       COMPARE LOGICAL IMMEDIATE HIGH (32)
	op_CLIJ    uint32 = 0xEC7F // FORMAT_RIE3       COMPARE LOGICAL IMMEDIATE AND BRANCH RELATIVE (32<-8)
	op_CLIY    uint32 = 0xEB55 // FORMAT_SIY        COMPARE LOGICAL (immediate)
	op_CLM     uint32 = 0xBD00 // FORMAT_RS2        COMPARE LOGICAL CHAR. UNDER MASK (low)
	op_CLMH    uint32 = 0xEB20 // FORMAT_RSY2       COMPARE LOGICAL CHAR. UNDER MASK (high)
	op_CLMY    uint32 = 0xEB21 // FORMAT_RSY2       COMPARE LOGICAL CHAR. UNDER MASK (low)
	op_CLR     uint32 = 0x1500 // FORMAT_RR         COMPARE LOGICAL (32)
	op_CLRB    uint32 = 0xECF7 // FORMAT_RRS        COMPARE LOGICAL AND BRANCH (32)
	op_CLRJ    uint32 = 0xEC77 // FORMAT_RIE2       COMPARE LOGICAL AND BRANCH RELATIVE (32)
	op_CLRL    uint32 = 0xC60F // FORMAT_RIL2       COMPARE LOGICAL RELATIVE LONG (32)
	op_CLRT    uint32 = 0xB973 // FORMAT_RRF3       COMPARE LOGICAL AND TRAP (32)
	op_CLST    uint32 = 0xB25D // FORMAT_RRE        COMPARE LOGICAL STRING
	op_CLT     uint32 = 0xEB23 // FORMAT_RSY2       COMPARE LOGICAL AND TRAP (32)
	op_CLY     uint32 = 0xE355 // FORMAT_RXY1       COMPARE LOGICAL (32)
	op_CMPSC   uint32 = 0xB263 // FORMAT_RRE        COMPRESSION CALL
	op_CP      uint32 = 0xF900 // FORMAT_SS2        COMPARE DECIMAL
	op_CPSDR   uint32 = 0xB372 // FORMAT_RRF2       COPY SIGN (long)
	op_CPYA    uint32 = 0xB24D // FORMAT_RRE        COPY ACCESS
	op_CR      uint32 = 0x1900 // FORMAT_RR         COMPARE (32)
	op_CRB     uint32 = 0xECF6 // FORMAT_RRS        COMPARE AND BRANCH (32)
	op_CRDTE   uint32 = 0xB98F // FORMAT_RRF2       COMPARE AND REPLACE DAT TABLE ENTRY
	op_CRJ     uint32 = 0xEC76 // FORMAT_RIE2       COMPARE AND BRANCH RELATIVE (32)
	op_CRL     uint32 = 0xC60D // FORMAT_RIL2       COMPARE RELATIVE LONG (32)
	op_CRT     uint32 = 0xB972 // FORMAT_RRF3       COMPARE AND TRAP (32)
	op_CS      uint32 = 0xBA00 // FORMAT_RS1        COMPARE AND SWAP (32)
	op_CSCH    uint32 = 0xB230 // FORMAT_S          CLEAR SUBCHANNEL
	op_CSDTR   uint32 = 0xB3E3 // FORMAT_RRF4       CONVERT TO SIGNED PACKED (long DFP to 64)
	op_CSG     uint32 = 0xEB30 // FORMAT_RSY1       COMPARE AND SWAP (64)
	op_CSP     uint32 = 0xB250 // FORMAT_RRE        COMPARE AND SWAP AND PURGE
	op_CSPG    uint32 = 0xB98A // FORMAT_RRE        COMPARE AND SWAP AND PURGE
	op_CSST    uint32 = 0xC802 // FORMAT_SSF        COMPARE AND SWAP AND STORE
	op_CSXTR   uint32 = 0xB3EB // FORMAT_RRF4       CONVERT TO SIGNED PACKED (extended DFP to 128)
	op_CSY     uint32 = 0xEB14 // FORMAT_RSY1       COMPARE AND SWAP (32)
	op_CU12    uint32 = 0xB2A7 // FORMAT_RRF3       CONVERT UTF-8 TO UTF-16
	op_CU14    uint32 = 0xB9B0 // FORMAT_RRF3       CONVERT UTF-8 TO UTF-32
	op_CU21    uint32 = 0xB2A6 // FORMAT_RRF3       CONVERT UTF-16 TO UTF-8
	op_CU24    uint32 = 0xB9B1 // FORMAT_RRF3       CONVERT UTF-16 TO UTF-32
	op_CU41    uint32 = 0xB9B2 // FORMAT_RRE        CONVERT UTF-32 TO UTF-8
	op_CU42    uint32 = 0xB9B3 // FORMAT_RRE        CONVERT UTF-32 TO UTF-16
	op_CUDTR   uint32 = 0xB3E2 // FORMAT_RRE        CONVERT TO UNSIGNED PACKED (long DFP to 64)
	op_CUSE    uint32 = 0xB257 // FORMAT_RRE        COMPARE UNTIL SUBSTRING EQUAL
	op_CUTFU   uint32 = 0xB2A7 // FORMAT_RRF3       CONVERT UTF-8 TO UNICODE
	op_CUUTF   uint32 = 0xB2A6 // FORMAT_RRF3       CONVERT UNICODE TO UTF-8
	op_CUXTR   uint32 = 0xB3EA // FORMAT_RRE        CONVERT TO UNSIGNED PACKED (extended DFP to 128)
	op_CVB     uint32 = 0x4F00 // FORMAT_RX1        CONVERT TO BINARY (32)
	op_CVBG    uint32 = 0xE30E // FORMAT_RXY1       CONVERT TO BINARY (64)
	op_CVBY    uint32 = 0xE306 // FORMAT_RXY1       CONVERT TO BINARY (32)
	op_CVD     uint32 = 0x4E00 // FORMAT_RX1        CONVERT TO DECIMAL (32)
	op_CVDG    uint32 = 0xE32E // FORMAT_RXY1       CONVERT TO DECIMAL (64)
	op_CVDY    uint32 = 0xE326 // FORMAT_RXY1       CONVERT TO DECIMAL (32)
	op_CXBR    uint32 = 0xB349 // FORMAT_RRE        COMPARE (extended BFP)
	op_CXFBR   uint32 = 0xB396 // FORMAT_RRE        CONVERT FROM FIXED (32 to extended BFP)
	op_CXFBRA  uint32 = 0xB396 // FORMAT_RRF5       CONVERT FROM FIXED (32 to extended BFP)
	op_CXFR    uint32 = 0xB3B6 // FORMAT_RRE        CONVERT FROM FIXED (32 to extended HFP)
	op_CXFTR   uint32 = 0xB959 // FORMAT_RRE        CONVERT FROM FIXED (32 to extended DFP)
	op_CXGBR   uint32 = 0xB3A6 // FORMAT_RRE        CONVERT FROM FIXED (64 to extended BFP)
	op_CXGBRA  uint32 = 0xB3A6 // FORMAT_RRF5       CONVERT FROM FIXED (64 to extended BFP)
	op_CXGR    uint32 = 0xB3C6 // FORMAT_RRE        CONVERT FROM FIXED (64 to extended HFP)
	op_CXGTR   uint32 = 0xB3F9 // FORMAT_RRE        CONVERT FROM FIXED (64 to extended DFP)
	op_CXGTRA  uint32 = 0xB3F9 // FORMAT_RRF5       CONVERT FROM FIXED (64 to extended DFP)
	op_CXLFBR  uint32 = 0xB392 // FORMAT_RRF5       CONVERT FROM LOGICAL (32 to extended BFP)
	op_CXLFTR  uint32 = 0xB95B // FORMAT_RRF5       CONVERT FROM LOGICAL (32 to extended DFP)
	op_CXLGBR  uint32 = 0xB3A2 // FORMAT_RRF5       CONVERT FROM LOGICAL (64 to extended BFP)
	op_CXLGTR  uint32 = 0xB95A // FORMAT_RRF5       CONVERT FROM LOGICAL (64 to extended DFP)
	op_CXR     uint32 = 0xB369 // FORMAT_RRE        COMPARE (extended HFP)
	op_CXSTR   uint32 = 0xB3FB // FORMAT_RRE        CONVERT FROM SIGNED PACKED (128 to extended DFP)
	op_CXTR    uint32 = 0xB3EC // FORMAT_RRE        COMPARE (extended DFP)
	op_CXUTR   uint32 = 0xB3FA // FORMAT_RRE        CONVERT FROM UNSIGNED PACKED (128 to ext. DFP)
	op_CXZT    uint32 = 0xEDAB // FORMAT_RSL        CONVERT FROM ZONED (to extended DFP)
	op_CY      uint32 = 0xE359 // FORMAT_RXY1       COMPARE (32)
	op_CZDT    uint32 = 0xEDA8 // FORMAT_RSL        CONVERT TO ZONED (from long DFP)
	op_CZXT    uint32 = 0xEDA9 // FORMAT_RSL        CONVERT TO ZONED (from extended DFP)
	op_D       uint32 = 0x5D00 // FORMAT_RX1        DIVIDE (32<-64)
	op_DD      uint32 = 0x6D00 // FORMAT_RX1        DIVIDE (long HFP)
	op_DDB     uint32 = 0xED1D // FORMAT_RXE        DIVIDE (long BFP)
	op_DDBR    uint32 = 0xB31D // FORMAT_RRE        DIVIDE (long BFP)
	op_DDR     uint32 = 0x2D00 // FORMAT_RR         DIVIDE (long HFP)
	op_DDTR    uint32 = 0xB3D1 // FORMAT_RRF1       DIVIDE (long DFP)
	op_DDTRA   uint32 = 0xB3D1 // FORMAT_RRF1       DIVIDE (long DFP)
	op_DE      uint32 = 0x7D00 // FORMAT_RX1        DIVIDE (short HFP)
	op_DEB     uint32 = 0xED0D // FORMAT_RXE        DIVIDE (short BFP)
	op_DEBR    uint32 = 0xB30D // FORMAT_RRE        DIVIDE (short BFP)
	op_DER     uint32 = 0x3D00 // FORMAT_RR         DIVIDE (short HFP)
	op_DIDBR   uint32 = 0xB35B // FORMAT_RRF2       DIVIDE TO INTEGER (long BFP)
	op_DIEBR   uint32 = 0xB353 // FORMAT_RRF2       DIVIDE TO INTEGER (short BFP)
	op_DL      uint32 = 0xE397 // FORMAT_RXY1       DIVIDE LOGICAL (32<-64)
	op_DLG     uint32 = 0xE387 // FORMAT_RXY1       DIVIDE LOGICAL (64<-128)
	op_DLGR    uint32 = 0xB987 // FORMAT_RRE        DIVIDE LOGICAL (64<-128)
	op_DLR     uint32 = 0xB997 // FORMAT_RRE        DIVIDE LOGICAL (32<-64)
	op_DP      uint32 = 0xFD00 // FORMAT_SS2        DIVIDE DECIMAL
	op_DR      uint32 = 0x1D00 // FORMAT_RR         DIVIDE (32<-64)
	op_DSG     uint32 = 0xE30D // FORMAT_RXY1       DIVIDE SINGLE (64)
	op_DSGF    uint32 = 0xE31D // FORMAT_RXY1       DIVIDE SINGLE (64<-32)
	op_DSGFR   uint32 = 0xB91D // FORMAT_RRE        DIVIDE SINGLE (64<-32)
	op_DSGR    uint32 = 0xB90D // FORMAT_RRE        DIVIDE SINGLE (64)
	op_DXBR    uint32 = 0xB34D // FORMAT_RRE        DIVIDE (extended BFP)
	op_DXR     uint32 = 0xB22D // FORMAT_RRE        DIVIDE (extended HFP)
	op_DXTR    uint32 = 0xB3D9 // FORMAT_RRF1       DIVIDE (extended DFP)
	op_DXTRA   uint32 = 0xB3D9 // FORMAT_RRF1       DIVIDE (extended DFP)
	op_EAR     uint32 = 0xB24F // FORMAT_RRE        EXTRACT ACCESS
	op_ECAG    uint32 = 0xEB4C // FORMAT_RSY1       EXTRACT CACHE ATTRIBUTE
	op_ECTG    uint32 = 0xC801 // FORMAT_SSF        EXTRACT CPU TIME
	op_ED      uint32 = 0xDE00 // FORMAT_SS1        EDIT
	op_EDMK    uint32 = 0xDF00 // FORMAT_SS1        EDIT AND MARK
	op_EEDTR   uint32 = 0xB3E5 // FORMAT_RRE        EXTRACT BIASED EXPONENT (long DFP to 64)
	op_EEXTR   uint32 = 0xB3ED // FORMAT_RRE        EXTRACT BIASED EXPONENT (extended DFP to 64)
	op_EFPC    uint32 = 0xB38C // FORMAT_RRE        EXTRACT FPC
	op_EPAIR   uint32 = 0xB99A // FORMAT_RRE        EXTRACT PRIMARY ASN AND INSTANCE
	op_EPAR    uint32 = 0xB226 // FORMAT_RRE        EXTRACT PRIMARY ASN
	op_EPSW    uint32 = 0xB98D // FORMAT_RRE        EXTRACT PSW
	op_EREG    uint32 = 0xB249 // FORMAT_RRE        EXTRACT STACKED REGISTERS (32)
	op_EREGG   uint32 = 0xB90E // FORMAT_RRE        EXTRACT STACKED REGISTERS (64)
	op_ESAIR   uint32 = 0xB99B // FORMAT_RRE        EXTRACT SECONDARY ASN AND INSTANCE
	op_ESAR    uint32 = 0xB227 // FORMAT_RRE        EXTRACT SECONDARY ASN
	op_ESDTR   uint32 = 0xB3E7 // FORMAT_RRE        EXTRACT SIGNIFICANCE (long DFP)
	op_ESEA    uint32 = 0xB99D // FORMAT_RRE        EXTRACT AND SET EXTENDED AUTHORITY
	op_ESTA    uint32 = 0xB24A // FORMAT_RRE        EXTRACT STACKED STATE
	op_ESXTR   uint32 = 0xB3EF // FORMAT_RRE        EXTRACT SIGNIFICANCE (extended DFP)
	op_ETND    uint32 = 0xB2EC // FORMAT_RRE        EXTRACT TRANSACTION NESTING DEPTH
	op_EX      uint32 = 0x4400 // FORMAT_RX1        EXECUTE
	op_EXRL    uint32 = 0xC600 // FORMAT_RIL2       EXECUTE RELATIVE LONG
	op_FIDBR   uint32 = 0xB35F // FORMAT_RRF5       LOAD FP INTEGER (long BFP)
	op_FIDBRA  uint32 = 0xB35F // FORMAT_RRF5       LOAD FP INTEGER (long BFP)
	op_FIDR    uint32 = 0xB37F // FORMAT_RRE        LOAD FP INTEGER (long HFP)
	op_FIDTR   uint32 = 0xB3D7 // FORMAT_RRF5       LOAD FP INTEGER (long DFP)
	op_FIEBR   uint32 = 0xB357 // FORMAT_RRF5       LOAD FP INTEGER (short BFP)
	op_FIEBRA  uint32 = 0xB357 // FORMAT_RRF5       LOAD FP INTEGER (short BFP)
	op_FIER    uint32 = 0xB377 // FORMAT_RRE        LOAD FP INTEGER (short HFP)
	op_FIXBR   uint32 = 0xB347 // FORMAT_RRF5       LOAD FP INTEGER (extended BFP)
	op_FIXBRA  uint32 = 0xB347 // FORMAT_RRF5       LOAD FP INTEGER (extended BFP)
	op_FIXR    uint32 = 0xB367 // FORMAT_RRE        LOAD FP INTEGER (extended HFP)
	op_FIXTR   uint32 = 0xB3DF // FORMAT_RRF5       LOAD FP INTEGER (extended DFP)
	op_FLOGR   uint32 = 0xB983 // FORMAT_RRE        FIND LEFTMOST ONE
	op_HDR     uint32 = 0x2400 // FORMAT_RR         HALVE (long HFP)
	op_HER     uint32 = 0x3400 // FORMAT_RR         HALVE (short HFP)
	op_HSCH    uint32 = 0xB231 // FORMAT_S          HALT SUBCHANNEL
	op_IAC     uint32 = 0xB224 // FORMAT_RRE        INSERT ADDRESS SPACE CONTROL
	op_IC      uint32 = 0x4300 // FORMAT_RX1        INSERT CHARACTER
	op_ICM     uint32 = 0xBF00 // FORMAT_RS2        INSERT CHARACTERS UNDER MASK (low)
	op_ICMH    uint32 = 0xEB80 // FORMAT_RSY2       INSERT CHARACTERS UNDER MASK (high)
	op_ICMY    uint32 = 0xEB81 // FORMAT_RSY2       INSERT CHARACTERS UNDER MASK (low)
	op_ICY     uint32 = 0xE373 // FORMAT_RXY1       INSERT CHARACTER
	op_IDTE    uint32 = 0xB98E // FORMAT_RRF2       INVALIDATE DAT TABLE ENTRY
	op_IEDTR   uint32 = 0xB3F6 // FORMAT_RRF2       INSERT BIASED EXPONENT (64 to long DFP)
	op_IEXTR   uint32 = 0xB3FE // FORMAT_RRF2       INSERT BIASED EXPONENT (64 to extended DFP)
	op_IIHF    uint32 = 0xC008 // FORMAT_RIL1       INSERT IMMEDIATE (high)
	op_IIHH    uint32 = 0xA500 // FORMAT_RI1        INSERT IMMEDIATE (high high)
	op_IIHL    uint32 = 0xA501 // FORMAT_RI1        INSERT IMMEDIATE (high low)
	op_IILF    uint32 = 0xC009 // FORMAT_RIL1       INSERT IMMEDIATE (low)
	op_IILH    uint32 = 0xA502 // FORMAT_RI1        INSERT IMMEDIATE (low high)
	op_IILL    uint32 = 0xA503 // FORMAT_RI1        INSERT IMMEDIATE (low low)
	op_IPK     uint32 = 0xB20B // FORMAT_S          INSERT PSW KEY
	op_IPM     uint32 = 0xB222 // FORMAT_RRE        INSERT PROGRAM MASK
	op_IPTE    uint32 = 0xB221 // FORMAT_RRF1       INVALIDATE PAGE TABLE ENTRY
	op_ISKE    uint32 = 0xB229 // FORMAT_RRE        INSERT STORAGE KEY EXTENDED
	op_IVSK    uint32 = 0xB223 // FORMAT_RRE        INSERT VIRTUAL STORAGE KEY
	op_KDB     uint32 = 0xED18 // FORMAT_RXE        COMPARE AND SIGNAL (long BFP)
	op_KDBR    uint32 = 0xB318 // FORMAT_RRE        COMPARE AND SIGNAL (long BFP)
	op_KDTR    uint32 = 0xB3E0 // FORMAT_RRE        COMPARE AND SIGNAL (long DFP)
	op_KEB     uint32 = 0xED08 // FORMAT_RXE        COMPARE AND SIGNAL (short BFP)
	op_KEBR    uint32 = 0xB308 // FORMAT_RRE        COMPARE AND SIGNAL (short BFP)
	op_KIMD    uint32 = 0xB93E // FORMAT_RRE        COMPUTE INTERMEDIATE MESSAGE DIGEST
	op_KLMD    uint32 = 0xB93F // FORMAT_RRE        COMPUTE LAST MESSAGE DIGEST
	op_KM      uint32 = 0xB92E // FORMAT_RRE        CIPHER MESSAGE
	op_KMAC    uint32 = 0xB91E // FORMAT_RRE        COMPUTE MESSAGE AUTHENTICATION CODE
	op_KMC     uint32 = 0xB92F // FORMAT_RRE        CIPHER MESSAGE WITH CHAINING
	op_KMCTR   uint32 = 0xB92D // FORMAT_RRF2       CIPHER MESSAGE WITH COUNTER
	op_KMF     uint32 = 0xB92A // FORMAT_RRE        CIPHER MESSAGE WITH CFB
	op_KMO     uint32 = 0xB92B // FORMAT_RRE        CIPHER MESSAGE WITH OFB
	op_KXBR    uint32 = 0xB348 // FORMAT_RRE        COMPARE AND SIGNAL (extended BFP)
	op_KXTR    uint32 = 0xB3E8 // FORMAT_RRE        COMPARE AND SIGNAL (extended DFP)
	op_L       uint32 = 0x5800 // FORMAT_RX1        LOAD (32)
	op_LA      uint32 = 0x4100 // FORMAT_RX1        LOAD ADDRESS
	op_LAA     uint32 = 0xEBF8 // FORMAT_RSY1       LOAD AND ADD (32)
	op_LAAG    uint32 = 0xEBE8 // FORMAT_RSY1       LOAD AND ADD (64)
	op_LAAL    uint32 = 0xEBFA // FORMAT_RSY1       LOAD AND ADD LOGICAL (32)
	op_LAALG   uint32 = 0xEBEA // FORMAT_RSY1       LOAD AND ADD LOGICAL (64)
	op_LAE     uint32 = 0x5100 // FORMAT_RX1        LOAD ADDRESS EXTENDED
	op_LAEY    uint32 = 0xE375 // FORMAT_RXY1       LOAD ADDRESS EXTENDED
	op_LAM     uint32 = 0x9A00 // FORMAT_RS1        LOAD ACCESS MULTIPLE
	op_LAMY    uint32 = 0xEB9A // FORMAT_RSY1       LOAD ACCESS MULTIPLE
	op_LAN     uint32 = 0xEBF4 // FORMAT_RSY1       LOAD AND AND (32)
	op_LANG    uint32 = 0xEBE4 // FORMAT_RSY1       LOAD AND AND (64)
	op_LAO     uint32 = 0xEBF6 // FORMAT_RSY1       LOAD AND OR (32)
	op_LAOG    uint32 = 0xEBE6 // FORMAT_RSY1       LOAD AND OR (64)
	op_LARL    uint32 = 0xC000 // FORMAT_RIL2       LOAD ADDRESS RELATIVE LONG
	op_LASP    uint32 = 0xE500 // FORMAT_SSE        LOAD ADDRESS SPACE PARAMETERS
	op_LAT     uint32 = 0xE39F // FORMAT_RXY1       LOAD AND TRAP (32L<-32)
	op_LAX     uint32 = 0xEBF7 // FORMAT_RSY1       LOAD AND EXCLUSIVE OR (32)
	op_LAXG    uint32 = 0xEBE7 // FORMAT_RSY1       LOAD AND EXCLUSIVE OR (64)
	op_LAY     uint32 = 0xE371 // FORMAT_RXY1       LOAD ADDRESS
	op_LB      uint32 = 0xE376 // FORMAT_RXY1       LOAD BYTE (32)
	op_LBH     uint32 = 0xE3C0 // FORMAT_RXY1       LOAD BYTE HIGH (32<-8)
	op_LBR     uint32 = 0xB926 // FORMAT_RRE        LOAD BYTE (32)
	op_LCDBR   uint32 = 0xB313 // FORMAT_RRE        LOAD COMPLEMENT (long BFP)
	op_LCDFR   uint32 = 0xB373 // FORMAT_RRE        LOAD COMPLEMENT (long)
	op_LCDR    uint32 = 0x2300 // FORMAT_RR         LOAD COMPLEMENT (long HFP)
	op_LCEBR   uint32 = 0xB303 // FORMAT_RRE        LOAD COMPLEMENT (short BFP)
	op_LCER    uint32 = 0x3300 // FORMAT_RR         LOAD COMPLEMENT (short HFP)
	op_LCGFR   uint32 = 0xB913 // FORMAT_RRE        LOAD COMPLEMENT (64<-32)
	op_LCGR    uint32 = 0xB903 // FORMAT_RRE        LOAD COMPLEMENT (64)
	op_LCR     uint32 = 0x1300 // FORMAT_RR         LOAD COMPLEMENT (32)
	op_LCTL    uint32 = 0xB700 // FORMAT_RS1        LOAD CONTROL (32)
	op_LCTLG   uint32 = 0xEB2F // FORMAT_RSY1       LOAD CONTROL (64)
	op_LCXBR   uint32 = 0xB343 // FORMAT_RRE        LOAD COMPLEMENT (extended BFP)
	op_LCXR    uint32 = 0xB363 // FORMAT_RRE        LOAD COMPLEMENT (extended HFP)
	op_LD      uint32 = 0x6800 // FORMAT_RX1        LOAD (long)
	op_LDE     uint32 = 0xED24 // FORMAT_RXE        LOAD LENGTHENED (short to long HFP)
	op_LDEB    uint32 = 0xED04 // FORMAT_RXE        LOAD LENGTHENED (short to long BFP)
	op_LDEBR   uint32 = 0xB304 // FORMAT_RRE        LOAD LENGTHENED (short to long BFP)
	op_LDER    uint32 = 0xB324 // FORMAT_RRE        LOAD LENGTHENED (short to long HFP)
	op_LDETR   uint32 = 0xB3D4 // FORMAT_RRF4       LOAD LENGTHENED (short to long DFP)
	op_LDGR    uint32 = 0xB3C1 // FORMAT_RRE        LOAD FPR FROM GR (64 to long)
	op_LDR     uint32 = 0x2800 // FORMAT_RR         LOAD (long)
	op_LDXBR   uint32 = 0xB345 // FORMAT_RRE        LOAD ROUNDED (extended to long BFP)
	op_LDXBRA  uint32 = 0xB345 // FORMAT_RRF5       LOAD ROUNDED (extended to long BFP)
	op_LDXR    uint32 = 0x2500 // FORMAT_RR         LOAD ROUNDED (extended to long HFP)
	op_LDXTR   uint32 = 0xB3DD // FORMAT_RRF5       LOAD ROUNDED (extended to long DFP)
	op_LDY     uint32 = 0xED65 // FORMAT_RXY1       LOAD (long)
	op_LE      uint32 = 0x7800 // FORMAT_RX1        LOAD (short)
	op_LEDBR   uint32 = 0xB344 // FORMAT_RRE        LOAD ROUNDED (long to short BFP)
	op_LEDBRA  uint32 = 0xB344 // FORMAT_RRF5       LOAD ROUNDED (long to short BFP)
	op_LEDR    uint32 = 0x3500 // FORMAT_RR         LOAD ROUNDED (long to short HFP)
	op_LEDTR   uint32 = 0xB3D5 // FORMAT_RRF5       LOAD ROUNDED (long to short DFP)
	op_LER     uint32 = 0x3800 // FORMAT_RR         LOAD (short)
	op_LEXBR   uint32 = 0xB346 // FORMAT_RRE        LOAD ROUNDED (extended to short BFP)
	op_LEXBRA  uint32 = 0xB346 // FORMAT_RRF5       LOAD ROUNDED (extended to short BFP)
	op_LEXR    uint32 = 0xB366 // FORMAT_RRE        LOAD ROUNDED (extended to short HFP)
	op_LEY     uint32 = 0xED64 // FORMAT_RXY1       LOAD (short)
	op_LFAS    uint32 = 0xB2BD // FORMAT_S          LOAD FPC AND SIGNAL
	op_LFH     uint32 = 0xE3CA // FORMAT_RXY1       LOAD HIGH (32)
	op_LFHAT   uint32 = 0xE3C8 // FORMAT_RXY1       LOAD HIGH AND TRAP (32H<-32)
	op_LFPC    uint32 = 0xB29D // FORMAT_S          LOAD FPC
	op_LG      uint32 = 0xE304 // FORMAT_RXY1       LOAD (64)
	op_LGAT    uint32 = 0xE385 // FORMAT_RXY1       LOAD AND TRAP (64)
	op_LGB     uint32 = 0xE377 // FORMAT_RXY1       LOAD BYTE (64)
	op_LGBR    uint32 = 0xB906 // FORMAT_RRE        LOAD BYTE (64)
	op_LGDR    uint32 = 0xB3CD // FORMAT_RRE        LOAD GR FROM FPR (long to 64)
	op_LGF     uint32 = 0xE314 // FORMAT_RXY1       LOAD (64<-32)
	op_LGFI    uint32 = 0xC001 // FORMAT_RIL1       LOAD IMMEDIATE (64<-32)
	op_LGFR    uint32 = 0xB914 // FORMAT_RRE        LOAD (64<-32)
	op_LGFRL   uint32 = 0xC40C // FORMAT_RIL2       LOAD RELATIVE LONG (64<-32)
	op_LGH     uint32 = 0xE315 // FORMAT_RXY1       LOAD HALFWORD (64)
	op_LGHI    uint32 = 0xA709 // FORMAT_RI1        LOAD HALFWORD IMMEDIATE (64)
	op_LGHR    uint32 = 0xB907 // FORMAT_RRE        LOAD HALFWORD (64)
	op_LGHRL   uint32 = 0xC404 // FORMAT_RIL2       LOAD HALFWORD RELATIVE LONG (64<-16)
	op_LGR     uint32 = 0xB904 // FORMAT_RRE        LOAD (64)
	op_LGRL    uint32 = 0xC408 // FORMAT_RIL2       LOAD RELATIVE LONG (64)
	op_LH      uint32 = 0x4800 // FORMAT_RX1        LOAD HALFWORD (32)
	op_LHH     uint32 = 0xE3C4 // FORMAT_RXY1       LOAD HALFWORD HIGH (32<-16)
	op_LHI     uint32 = 0xA708 // FORMAT_RI1        LOAD HALFWORD IMMEDIATE (32)
	op_LHR     uint32 = 0xB927 // FORMAT_RRE        LOAD HALFWORD (32)
	op_LHRL    uint32 = 0xC405 // FORMAT_RIL2       LOAD HALFWORD RELATIVE LONG (32<-16)
	op_LHY     uint32 = 0xE378 // FORMAT_RXY1       LOAD HALFWORD (32)
	op_LLC     uint32 = 0xE394 // FORMAT_RXY1       LOAD LOGICAL CHARACTER (32)
	op_LLCH    uint32 = 0xE3C2 // FORMAT_RXY1       LOAD LOGICAL CHARACTER HIGH (32<-8)
	op_LLCR    uint32 = 0xB994 // FORMAT_RRE        LOAD LOGICAL CHARACTER (32)
	op_LLGC    uint32 = 0xE390 // FORMAT_RXY1       LOAD LOGICAL CHARACTER (64)
	op_LLGCR   uint32 = 0xB984 // FORMAT_RRE        LOAD LOGICAL CHARACTER (64)
	op_LLGF    uint32 = 0xE316 // FORMAT_RXY1       LOAD LOGICAL (64<-32)
	op_LLGFAT  uint32 = 0xE39D // FORMAT_RXY1       LOAD LOGICAL AND TRAP (64<-32)
	op_LLGFR   uint32 = 0xB916 // FORMAT_RRE        LOAD LOGICAL (64<-32)
	op_LLGFRL  uint32 = 0xC40E // FORMAT_RIL2       LOAD LOGICAL RELATIVE LONG (64<-32)
	op_LLGH    uint32 = 0xE391 // FORMAT_RXY1       LOAD LOGICAL HALFWORD (64)
	op_LLGHR   uint32 = 0xB985 // FORMAT_RRE        LOAD LOGICAL HALFWORD (64)
	op_LLGHRL  uint32 = 0xC406 // FORMAT_RIL2       LOAD LOGICAL HALFWORD RELATIVE LONG (64<-16)
	op_LLGT    uint32 = 0xE317 // FORMAT_RXY1       LOAD LOGICAL THIRTY ONE BITS
	op_LLGTAT  uint32 = 0xE39C // FORMAT_RXY1       LOAD LOGICAL THIRTY ONE BITS AND TRAP (64<-31)
	op_LLGTR   uint32 = 0xB917 // FORMAT_RRE        LOAD LOGICAL THIRTY ONE BITS
	op_LLH     uint32 = 0xE395 // FORMAT_RXY1       LOAD LOGICAL HALFWORD (32)
	op_LLHH    uint32 = 0xE3C6 // FORMAT_RXY1       LOAD LOGICAL HALFWORD HIGH (32<-16)
	op_LLHR    uint32 = 0xB995 // FORMAT_RRE        LOAD LOGICAL HALFWORD (32)
	op_LLHRL   uint32 = 0xC402 // FORMAT_RIL2       LOAD LOGICAL HALFWORD RELATIVE LONG (32<-16)
	op_LLIHF   uint32 = 0xC00E // FORMAT_RIL1       LOAD LOGICAL IMMEDIATE (high)
	op_LLIHH   uint32 = 0xA50C // FORMAT_RI1        LOAD LOGICAL IMMEDIATE (high high)
	op_LLIHL   uint32 = 0xA50D // FORMAT_RI1        LOAD LOGICAL IMMEDIATE (high low)
	op_LLILF   uint32 = 0xC00F // FORMAT_RIL1       LOAD LOGICAL IMMEDIATE (low)
	op_LLILH   uint32 = 0xA50E // FORMAT_RI1        LOAD LOGICAL IMMEDIATE (low high)
	op_LLILL   uint32 = 0xA50F // FORMAT_RI1        LOAD LOGICAL IMMEDIATE (low low)
	op_LM      uint32 = 0x9800 // FORMAT_RS1        LOAD MULTIPLE (32)
	op_LMD     uint32 = 0xEF00 // FORMAT_SS5        LOAD MULTIPLE DISJOINT
	op_LMG     uint32 = 0xEB04 // FORMAT_RSY1       LOAD MULTIPLE (64)
	op_LMH     uint32 = 0xEB96 // FORMAT_RSY1       LOAD MULTIPLE HIGH
	op_LMY     uint32 = 0xEB98 // FORMAT_RSY1       LOAD MULTIPLE (32)
	op_LNDBR   uint32 = 0xB311 // FORMAT_RRE        LOAD NEGATIVE (long BFP)
	op_LNDFR   uint32 = 0xB371 // FORMAT_RRE        LOAD NEGATIVE (long)
	op_LNDR    uint32 = 0x2100 // FORMAT_RR         LOAD NEGATIVE (long HFP)
	op_LNEBR   uint32 = 0xB301 // FORMAT_RRE        LOAD NEGATIVE (short BFP)
	op_LNER    uint32 = 0x3100 // FORMAT_RR         LOAD NEGATIVE (short HFP)
	op_LNGFR   uint32 = 0xB911 // FORMAT_RRE        LOAD NEGATIVE (64<-32)
	op_LNGR    uint32 = 0xB901 // FORMAT_RRE        LOAD NEGATIVE (64)
	op_LNR     uint32 = 0x1100 // FORMAT_RR         LOAD NEGATIVE (32)
	op_LNXBR   uint32 = 0xB341 // FORMAT_RRE        LOAD NEGATIVE (extended BFP)
	op_LNXR    uint32 = 0xB361 // FORMAT_RRE        LOAD NEGATIVE (extended HFP)
	op_LOC     uint32 = 0xEBF2 // FORMAT_RSY2       LOAD ON CONDITION (32)
	op_LOCG    uint32 = 0xEBE2 // FORMAT_RSY2       LOAD ON CONDITION (64)
	op_LOCGR   uint32 = 0xB9E2 // FORMAT_RRF3       LOAD ON CONDITION (64)
	op_LOCR    uint32 = 0xB9F2 // FORMAT_RRF3       LOAD ON CONDITION (32)
	op_LPD     uint32 = 0xC804 // FORMAT_SSF        LOAD PAIR DISJOINT (32)
	op_LPDBR   uint32 = 0xB310 // FORMAT_RRE        LOAD POSITIVE (long BFP)
	op_LPDFR   uint32 = 0xB370 // FORMAT_RRE        LOAD POSITIVE (long)
	op_LPDG    uint32 = 0xC805 // FORMAT_SSF        LOAD PAIR DISJOINT (64)
	op_LPDR    uint32 = 0x2000 // FORMAT_RR         LOAD POSITIVE (long HFP)
	op_LPEBR   uint32 = 0xB300 // FORMAT_RRE        LOAD POSITIVE (short BFP)
	op_LPER    uint32 = 0x3000 // FORMAT_RR         LOAD POSITIVE (short HFP)
	op_LPGFR   uint32 = 0xB910 // FORMAT_RRE        LOAD POSITIVE (64<-32)
	op_LPGR    uint32 = 0xB900 // FORMAT_RRE        LOAD POSITIVE (64)
	op_LPQ     uint32 = 0xE38F // FORMAT_RXY1       LOAD PAIR FROM QUADWORD
	op_LPR     uint32 = 0x1000 // FORMAT_RR         LOAD POSITIVE (32)
	op_LPSW    uint32 = 0x8200 // FORMAT_S          LOAD PSW
	op_LPSWE   uint32 = 0xB2B2 // FORMAT_S          LOAD PSW EXTENDED
	op_LPTEA   uint32 = 0xB9AA // FORMAT_RRF2       LOAD PAGE TABLE ENTRY ADDRESS
	op_LPXBR   uint32 = 0xB340 // FORMAT_RRE        LOAD POSITIVE (extended BFP)
	op_LPXR    uint32 = 0xB360 // FORMAT_RRE        LOAD POSITIVE (extended HFP)
	op_LR      uint32 = 0x1800 // FORMAT_RR         LOAD (32)
	op_LRA     uint32 = 0xB100 // FORMAT_RX1        LOAD REAL ADDRESS (32)
	op_LRAG    uint32 = 0xE303 // FORMAT_RXY1       LOAD REAL ADDRESS (64)
	op_LRAY    uint32 = 0xE313 // FORMAT_RXY1       LOAD REAL ADDRESS (32)
	op_LRDR    uint32 = 0x2500 // FORMAT_RR         LOAD ROUNDED (extended to long HFP)
	op_LRER    uint32 = 0x3500 // FORMAT_RR         LOAD ROUNDED (long to short HFP)
	op_LRL     uint32 = 0xC40D // FORMAT_RIL2       LOAD RELATIVE LONG (32)
	op_LRV     uint32 = 0xE31E // FORMAT_RXY1       LOAD REVERSED (32)
	op_LRVG    uint32 = 0xE30F // FORMAT_RXY1       LOAD REVERSED (64)
	op_LRVGR   uint32 = 0xB90F // FORMAT_RRE        LOAD REVERSED (64)
	op_LRVH    uint32 = 0xE31F // FORMAT_RXY1       LOAD REVERSED (16)
	op_LRVR    uint32 = 0xB91F // FORMAT_RRE        LOAD REVERSED (32)
	op_LT      uint32 = 0xE312 // FORMAT_RXY1       LOAD AND TEST (32)
	op_LTDBR   uint32 = 0xB312 // FORMAT_RRE        LOAD AND TEST (long BFP)
	op_LTDR    uint32 = 0x2200 // FORMAT_RR         LOAD AND TEST (long HFP)
	op_LTDTR   uint32 = 0xB3D6 // FORMAT_RRE        LOAD AND TEST (long DFP)
	op_LTEBR   uint32 = 0xB302 // FORMAT_RRE        LOAD AND TEST (short BFP)
	op_LTER    uint32 = 0x3200 // FORMAT_RR         LOAD AND TEST (short HFP)
	op_LTG     uint32 = 0xE302 // FORMAT_RXY1       LOAD AND TEST (64)
	op_LTGF    uint32 = 0xE332 // FORMAT_RXY1       LOAD AND TEST (64<-32)
	op_LTGFR   uint32 = 0xB912 // FORMAT_RRE        LOAD AND TEST (64<-32)
	op_LTGR    uint32 = 0xB902 // FORMAT_RRE        LOAD AND TEST (64)
	op_LTR     uint32 = 0x1200 // FORMAT_RR         LOAD AND TEST (32)
	op_LTXBR   uint32 = 0xB342 // FORMAT_RRE        LOAD AND TEST (extended BFP)
	op_LTXR    uint32 = 0xB362 // FORMAT_RRE        LOAD AND TEST (extended HFP)
	op_LTXTR   uint32 = 0xB3DE // FORMAT_RRE        LOAD AND TEST (extended DFP)
	op_LURA    uint32 = 0xB24B // FORMAT_RRE        LOAD USING REAL ADDRESS (32)
	op_LURAG   uint32 = 0xB905 // FORMAT_RRE        LOAD USING REAL ADDRESS (64)
	op_LXD     uint32 = 0xED25 // FORMAT_RXE        LOAD LENGTHENED (long to extended HFP)
	op_LXDB    uint32 = 0xED05 // FORMAT_RXE        LOAD LENGTHENED (long to extended BFP)
	op_LXDBR   uint32 = 0xB305 // FORMAT_RRE        LOAD LENGTHENED (long to extended BFP)
	op_LXDR    uint32 = 0xB325 // FORMAT_RRE        LOAD LENGTHENED (long to extended HFP)
	op_LXDTR   uint32 = 0xB3DC // FORMAT_RRF4       LOAD LENGTHENED (long to extended DFP)
	op_LXE     uint32 = 0xED26 // FORMAT_RXE        LOAD LENGTHENED (short to extended HFP)
	op_LXEB    uint32 = 0xED06 // FORMAT_RXE        LOAD LENGTHENED (short to extended BFP)
	op_LXEBR   uint32 = 0xB306 // FORMAT_RRE        LOAD LENGTHENED (short to extended BFP)
	op_LXER    uint32 = 0xB326 // FORMAT_RRE        LOAD LENGTHENED (short to extended HFP)
	op_LXR     uint32 = 0xB365 // FORMAT_RRE        LOAD (extended)
	op_LY      uint32 = 0xE358 // FORMAT_RXY1       LOAD (32)
	op_LZDR    uint32 = 0xB375 // FORMAT_RRE        LOAD ZERO (long)
	op_LZER    uint32 = 0xB374 // FORMAT_RRE        LOAD ZERO (short)
	op_LZXR    uint32 = 0xB376 // FORMAT_RRE        LOAD ZERO (extended)
	op_M       uint32 = 0x5C00 // FORMAT_RX1        MULTIPLY (64<-32)
	op_MAD     uint32 = 0xED3E // FORMAT_RXF        MULTIPLY AND ADD (long HFP)
	op_MADB    uint32 = 0xED1E // FORMAT_RXF        MULTIPLY AND ADD (long BFP)
	op_MADBR   uint32 = 0xB31E // FORMAT_RRD        MULTIPLY AND ADD (long BFP)
	op_MADR    uint32 = 0xB33E // FORMAT_RRD        MULTIPLY AND ADD (long HFP)
	op_MAE     uint32 = 0xED2E // FORMAT_RXF        MULTIPLY AND ADD (short HFP)
	op_MAEB    uint32 = 0xED0E // FORMAT_RXF        MULTIPLY AND ADD (short BFP)
	op_MAEBR   uint32 = 0xB30E // FORMAT_RRD        MULTIPLY AND ADD (short BFP)
	op_MAER    uint32 = 0xB32E // FORMAT_RRD        MULTIPLY AND ADD (short HFP)
	op_MAY     uint32 = 0xED3A // FORMAT_RXF        MULTIPLY & ADD UNNORMALIZED (long to ext. HFP)
	op_MAYH    uint32 = 0xED3C // FORMAT_RXF        MULTIPLY AND ADD UNNRM. (long to ext. high HFP)
	op_MAYHR   uint32 = 0xB33C // FORMAT_RRD        MULTIPLY AND ADD UNNRM. (long to ext. high HFP)
	op_MAYL    uint32 = 0xED38 // FORMAT_RXF        MULTIPLY AND ADD UNNRM. (long to ext. low HFP)
	op_MAYLR   uint32 = 0xB338 // FORMAT_RRD        MULTIPLY AND ADD UNNRM. (long to ext. low HFP)
	op_MAYR    uint32 = 0xB33A // FORMAT_RRD        MULTIPLY & ADD UNNORMALIZED (long to ext. HFP)
	op_MC      uint32 = 0xAF00 // FORMAT_SI         MONITOR CALL
	op_MD      uint32 = 0x6C00 // FORMAT_RX1        MULTIPLY (long HFP)
	op_MDB     uint32 = 0xED1C // FORMAT_RXE        MULTIPLY (long BFP)
	op_MDBR    uint32 = 0xB31C // FORMAT_RRE        MULTIPLY (long BFP)
	op_MDE     uint32 = 0x7C00 // FORMAT_RX1        MULTIPLY (short to long HFP)
	op_MDEB    uint32 = 0xED0C // FORMAT_RXE        MULTIPLY (short to long BFP)
	op_MDEBR   uint32 = 0xB30C // FORMAT_RRE        MULTIPLY (short to long BFP)
	op_MDER    uint32 = 0x3C00 // FORMAT_RR         MULTIPLY (short to long HFP)
	op_MDR     uint32 = 0x2C00 // FORMAT_RR         MULTIPLY (long HFP)
	op_MDTR    uint32 = 0xB3D0 // FORMAT_RRF1       MULTIPLY (long DFP)
	op_MDTRA   uint32 = 0xB3D0 // FORMAT_RRF1       MULTIPLY (long DFP)
	op_ME      uint32 = 0x7C00 // FORMAT_RX1        MULTIPLY (short to long HFP)
	op_MEE     uint32 = 0xED37 // FORMAT_RXE        MULTIPLY (short HFP)
	op_MEEB    uint32 = 0xED17 // FORMAT_RXE        MULTIPLY (short BFP)
	op_MEEBR   uint32 = 0xB317 // FORMAT_RRE        MULTIPLY (short BFP)
	op_MEER    uint32 = 0xB337 // FORMAT_RRE        MULTIPLY (short HFP)
	op_MER     uint32 = 0x3C00 // FORMAT_RR         MULTIPLY (short to long HFP)
	op_MFY     uint32 = 0xE35C // FORMAT_RXY1       MULTIPLY (64<-32)
	op_MGHI    uint32 = 0xA70D // FORMAT_RI1        MULTIPLY HALFWORD IMMEDIATE (64)
	op_MH      uint32 = 0x4C00 // FORMAT_RX1        MULTIPLY HALFWORD (32)
	op_MHI     uint32 = 0xA70C // FORMAT_RI1        MULTIPLY HALFWORD IMMEDIATE (32)
	op_MHY     uint32 = 0xE37C // FORMAT_RXY1       MULTIPLY HALFWORD (32)
	op_ML      uint32 = 0xE396 // FORMAT_RXY1       MULTIPLY LOGICAL (64<-32)
	op_MLG     uint32 = 0xE386 // FORMAT_RXY1       MULTIPLY LOGICAL (128<-64)
	op_MLGR    uint32 = 0xB986 // FORMAT_RRE        MULTIPLY LOGICAL (128<-64)
	op_MLR     uint32 = 0xB996 // FORMAT_RRE        MULTIPLY LOGICAL (64<-32)
	op_MP      uint32 = 0xFC00 // FORMAT_SS2        MULTIPLY DECIMAL
	op_MR      uint32 = 0x1C00 // FORMAT_RR         MULTIPLY (64<-32)
	op_MS      uint32 = 0x7100 // FORMAT_RX1        MULTIPLY SINGLE (32)
	op_MSCH    uint32 = 0xB232 // FORMAT_S          MODIFY SUBCHANNEL
	op_MSD     uint32 = 0xED3F // FORMAT_RXF        MULTIPLY AND SUBTRACT (long HFP)
	op_MSDB    uint32 = 0xED1F // FORMAT_RXF        MULTIPLY AND SUBTRACT (long BFP)
	op_MSDBR   uint32 = 0xB31F // FORMAT_RRD        MULTIPLY AND SUBTRACT (long BFP)
	op_MSDR    uint32 = 0xB33F // FORMAT_RRD        MULTIPLY AND SUBTRACT (long HFP)
	op_MSE     uint32 = 0xED2F // FORMAT_RXF        MULTIPLY AND SUBTRACT (short HFP)
	op_MSEB    uint32 = 0xED0F // FORMAT_RXF        MULTIPLY AND SUBTRACT (short BFP)
	op_MSEBR   uint32 = 0xB30F // FORMAT_RRD        MULTIPLY AND SUBTRACT (short BFP)
	op_MSER    uint32 = 0xB32F // FORMAT_RRD        MULTIPLY AND SUBTRACT (short HFP)
	op_MSFI    uint32 = 0xC201 // FORMAT_RIL1       MULTIPLY SINGLE IMMEDIATE (32)
	op_MSG     uint32 = 0xE30C // FORMAT_RXY1       MULTIPLY SINGLE (64)
	op_MSGF    uint32 = 0xE31C // FORMAT_RXY1       MULTIPLY SINGLE (64<-32)
	op_MSGFI   uint32 = 0xC200 // FORMAT_RIL1       MULTIPLY SINGLE IMMEDIATE (64<-32)
	op_MSGFR   uint32 = 0xB91C // FORMAT_RRE        MULTIPLY SINGLE (64<-32)
	op_MSGR    uint32 = 0xB90C // FORMAT_RRE        MULTIPLY SINGLE (64)
	op_MSR     uint32 = 0xB252 // FORMAT_RRE        MULTIPLY SINGLE (32)
	op_MSTA    uint32 = 0xB247 // FORMAT_RRE        MODIFY STACKED STATE
	op_MSY     uint32 = 0xE351 // FORMAT_RXY1       MULTIPLY SINGLE (32)
	op_MVC     uint32 = 0xD200 // FORMAT_SS1        MOVE (character)
	op_MVCDK   uint32 = 0xE50F // FORMAT_SSE        MOVE WITH DESTINATION KEY
	op_MVCIN   uint32 = 0xE800 // FORMAT_SS1        MOVE INVERSE
	op_MVCK    uint32 = 0xD900 // FORMAT_SS4        MOVE WITH KEY
	op_MVCL    uint32 = 0x0E00 // FORMAT_RR         MOVE LONG
	op_MVCLE   uint32 = 0xA800 // FORMAT_RS1        MOVE LONG EXTENDED
	op_MVCLU   uint32 = 0xEB8E // FORMAT_RSY1       MOVE LONG UNICODE
	op_MVCOS   uint32 = 0xC800 // FORMAT_SSF        MOVE WITH OPTIONAL SPECIFICATIONS
	op_MVCP    uint32 = 0xDA00 // FORMAT_SS4        MOVE TO PRIMARY
	op_MVCS    uint32 = 0xDB00 // FORMAT_SS4        MOVE TO SECONDARY
	op_MVCSK   uint32 = 0xE50E // FORMAT_SSE        MOVE WITH SOURCE KEY
	op_MVGHI   uint32 = 0xE548 // FORMAT_SIL        MOVE (64<-16)
	op_MVHHI   uint32 = 0xE544 // FORMAT_SIL        MOVE (16<-16)
	op_MVHI    uint32 = 0xE54C // FORMAT_SIL        MOVE (32<-16)
	op_MVI     uint32 = 0x9200 // FORMAT_SI         MOVE (immediate)
	op_MVIY    uint32 = 0xEB52 // FORMAT_SIY        MOVE (immediate)
	op_MVN     uint32 = 0xD100 // FORMAT_SS1        MOVE NUMERICS
	op_MVO     uint32 = 0xF100 // FORMAT_SS2        MOVE WITH OFFSET
	op_MVPG    uint32 = 0xB254 // FORMAT_RRE        MOVE PAGE
	op_MVST    uint32 = 0xB255 // FORMAT_RRE        MOVE STRING
	op_MVZ     uint32 = 0xD300 // FORMAT_SS1        MOVE ZONES
	op_MXBR    uint32 = 0xB34C // FORMAT_RRE        MULTIPLY (extended BFP)
	op_MXD     uint32 = 0x6700 // FORMAT_RX1        MULTIPLY (long to extended HFP)
	op_MXDB    uint32 = 0xED07 // FORMAT_RXE        MULTIPLY (long to extended BFP)
	op_MXDBR   uint32 = 0xB307 // FORMAT_RRE        MULTIPLY (long to extended BFP)
	op_MXDR    uint32 = 0x2700 // FORMAT_RR         MULTIPLY (long to extended HFP)
	op_MXR     uint32 = 0x2600 // FORMAT_RR         MULTIPLY (extended HFP)
	op_MXTR    uint32 = 0xB3D8 // FORMAT_RRF1       MULTIPLY (extended DFP)
	op_MXTRA   uint32 = 0xB3D8 // FORMAT_RRF1       MULTIPLY (extended DFP)
	op_MY      uint32 = 0xED3B // FORMAT_RXF        MULTIPLY UNNORMALIZED (long to ext. HFP)
	op_MYH     uint32 = 0xED3D // FORMAT_RXF        MULTIPLY UNNORM. (long to ext. high HFP)
	op_MYHR    uint32 = 0xB33D // FORMAT_RRD        MULTIPLY UNNORM. (long to ext. high HFP)
	op_MYL     uint32 = 0xED39 // FORMAT_RXF        MULTIPLY UNNORM. (long to ext. low HFP)
	op_MYLR    uint32 = 0xB339 // FORMAT_RRD        MULTIPLY UNNORM. (long to ext. low HFP)
	op_MYR     uint32 = 0xB33B // FORMAT_RRD        MULTIPLY UNNORMALIZED (long to ext. HFP)
	op_N       uint32 = 0x5400 // FORMAT_RX1        AND (32)
	op_NC      uint32 = 0xD400 // FORMAT_SS1        AND (character)
	op_NG      uint32 = 0xE380 // FORMAT_RXY1       AND (64)
	op_NGR     uint32 = 0xB980 // FORMAT_RRE        AND (64)
	op_NGRK    uint32 = 0xB9E4 // FORMAT_RRF1       AND (64)
	op_NI      uint32 = 0x9400 // FORMAT_SI         AND (immediate)
	op_NIAI    uint32 = 0xB2FA // FORMAT_IE         NEXT INSTRUCTION ACCESS INTENT
	op_NIHF    uint32 = 0xC00A // FORMAT_RIL1       AND IMMEDIATE (high)
	op_NIHH    uint32 = 0xA504 // FORMAT_RI1        AND IMMEDIATE (high high)
	op_NIHL    uint32 = 0xA505 // FORMAT_RI1        AND IMMEDIATE (high low)
	op_NILF    uint32 = 0xC00B // FORMAT_RIL1       AND IMMEDIATE (low)
	op_NILH    uint32 = 0xA506 // FORMAT_RI1        AND IMMEDIATE (low high)
	op_NILL    uint32 = 0xA507 // FORMAT_RI1        AND IMMEDIATE (low low)
	op_NIY     uint32 = 0xEB54 // FORMAT_SIY        AND (immediate)
	op_NR      uint32 = 0x1400 // FORMAT_RR         AND (32)
	op_NRK     uint32 = 0xB9F4 // FORMAT_RRF1       AND (32)
	op_NTSTG   uint32 = 0xE325 // FORMAT_RXY1       NONTRANSACTIONAL STORE
	op_NY      uint32 = 0xE354 // FORMAT_RXY1       AND (32)
	op_O       uint32 = 0x5600 // FORMAT_RX1        OR (32)
	op_OC      uint32 = 0xD600 // FORMAT_SS1        OR (character)
	op_OG      uint32 = 0xE381 // FORMAT_RXY1       OR (64)
	op_OGR     uint32 = 0xB981 // FORMAT_RRE        OR (64)
	op_OGRK    uint32 = 0xB9E6 // FORMAT_RRF1       OR (64)
	op_OI      uint32 = 0x9600 // FORMAT_SI         OR (immediate)
	op_OIHF    uint32 = 0xC00C // FORMAT_RIL1       OR IMMEDIATE (high)
	op_OIHH    uint32 = 0xA508 // FORMAT_RI1        OR IMMEDIATE (high high)
	op_OIHL    uint32 = 0xA509 // FORMAT_RI1        OR IMMEDIATE (high low)
	op_OILF    uint32 = 0xC00D // FORMAT_RIL1       OR IMMEDIATE (low)
	op_OILH    uint32 = 0xA50A // FORMAT_RI1        OR IMMEDIATE (low high)
	op_OILL    uint32 = 0xA50B // FORMAT_RI1        OR IMMEDIATE (low low)
	op_OIY     uint32 = 0xEB56 // FORMAT_SIY        OR (immediate)
	op_OR      uint32 = 0x1600 // FORMAT_RR         OR (32)
	op_ORK     uint32 = 0xB9F6 // FORMAT_RRF1       OR (32)
	op_OY      uint32 = 0xE356 // FORMAT_RXY1       OR (32)
	op_PACK    uint32 = 0xF200 // FORMAT_SS2        PACK
	op_PALB    uint32 = 0xB248 // FORMAT_RRE        PURGE ALB
	op_PC      uint32 = 0xB218 // FORMAT_S          PROGRAM CALL
	op_PCC     uint32 = 0xB92C // FORMAT_RRE        PERFORM CRYPTOGRAPHIC COMPUTATION
	op_PCKMO   uint32 = 0xB928 // FORMAT_RRE        PERFORM CRYPTOGRAPHIC KEY MGMT. OPERATIONS
	op_PFD     uint32 = 0xE336 // FORMAT_RXY2       PREFETCH DATA
	op_PFDRL   uint32 = 0xC602 // FORMAT_RIL3       PREFETCH DATA RELATIVE LONG
	op_PFMF    uint32 = 0xB9AF // FORMAT_RRE        PERFORM FRAME MANAGEMENT FUNCTION
	op_PFPO    uint32 = 0x010A // FORMAT_E          PERFORM FLOATING-POINT OPERATION
	op_PGIN    uint32 = 0xB22E // FORMAT_RRE        PAGE IN
	op_PGOUT   uint32 = 0xB22F // FORMAT_RRE        PAGE OUT
	op_PKA     uint32 = 0xE900 // FORMAT_SS6        PACK ASCII
	op_PKU     uint32 = 0xE100 // FORMAT_SS6        PACK UNICODE
	op_PLO     uint32 = 0xEE00 // FORMAT_SS5        PERFORM LOCKED OPERATION
	op_POPCNT  uint32 = 0xB9E1 // FORMAT_RRE        POPULATION COUNT
	op_PPA     uint32 = 0xB2E8 // FORMAT_RRF3       PERFORM PROCESSOR ASSIST
	op_PR      uint32 = 0x0101 // FORMAT_E          PROGRAM RETURN
	op_PT      uint32 = 0xB228 // FORMAT_RRE        PROGRAM TRANSFER
	op_PTF     uint32 = 0xB9A2 // FORMAT_RRE        PERFORM TOPOLOGY FUNCTION
	op_PTFF    uint32 = 0x0104 // FORMAT_E          PERFORM TIMING FACILITY FUNCTION
	op_PTI     uint32 = 0xB99E // FORMAT_RRE        PROGRAM TRANSFER WITH INSTANCE
	op_PTLB    uint32 = 0xB20D // FORMAT_S          PURGE TLB
	op_QADTR   uint32 = 0xB3F5 // FORMAT_RRF2       QUANTIZE (long DFP)
	op_QAXTR   uint32 = 0xB3FD // FORMAT_RRF2       QUANTIZE (extended DFP)
	op_RCHP    uint32 = 0xB23B // FORMAT_S          RESET CHANNEL PATH
	op_RISBG   uint32 = 0xEC55 // FORMAT_RIE6       ROTATE THEN INSERT SELECTED BITS
	op_RISBGN  uint32 = 0xEC59 // FORMAT_RIE6       ROTATE THEN INSERT SELECTED BITS
	op_RISBHG  uint32 = 0xEC5D // FORMAT_RIE6       ROTATE THEN INSERT SELECTED BITS HIGH
	op_RISBLG  uint32 = 0xEC51 // FORMAT_RIE6       ROTATE THEN INSERT SELECTED BITS LOW
	op_RLL     uint32 = 0xEB1D // FORMAT_RSY1       ROTATE LEFT SINGLE LOGICAL (32)
	op_RLLG    uint32 = 0xEB1C // FORMAT_RSY1       ROTATE LEFT SINGLE LOGICAL (64)
	op_RNSBG   uint32 = 0xEC54 // FORMAT_RIE6       ROTATE THEN AND SELECTED BITS
	op_ROSBG   uint32 = 0xEC56 // FORMAT_RIE6       ROTATE THEN OR SELECTED BITS
	op_RP      uint32 = 0xB277 // FORMAT_S          RESUME PROGRAM
	op_RRBE    uint32 = 0xB22A // FORMAT_RRE        RESET REFERENCE BIT EXTENDED
	op_RRBM    uint32 = 0xB9AE // FORMAT_RRE        RESET REFERENCE BITS MULTIPLE
	op_RRDTR   uint32 = 0xB3F7 // FORMAT_RRF2       REROUND (long DFP)
	op_RRXTR   uint32 = 0xB3FF // FORMAT_RRF2       REROUND (extended DFP)
	op_RSCH    uint32 = 0xB238 // FORMAT_S          RESUME SUBCHANNEL
	op_RXSBG   uint32 = 0xEC57 // FORMAT_RIE6       ROTATE THEN EXCLUSIVE OR SELECTED BITS
	op_S       uint32 = 0x5B00 // FORMAT_RX1        SUBTRACT (32)
	op_SAC     uint32 = 0xB219 // FORMAT_S          SET ADDRESS SPACE CONTROL
	op_SACF    uint32 = 0xB279 // FORMAT_S          SET ADDRESS SPACE CONTROL FAST
	op_SAL     uint32 = 0xB237 // FORMAT_S          SET ADDRESS LIMIT
	op_SAM24   uint32 = 0x010C // FORMAT_E          SET ADDRESSING MODE (24)
	op_SAM31   uint32 = 0x010D // FORMAT_E          SET ADDRESSING MODE (31)
	op_SAM64   uint32 = 0x010E // FORMAT_E          SET ADDRESSING MODE (64)
	op_SAR     uint32 = 0xB24E // FORMAT_RRE        SET ACCESS
	op_SCHM    uint32 = 0xB23C // FORMAT_S          SET CHANNEL MONITOR
	op_SCK     uint32 = 0xB204 // FORMAT_S          SET CLOCK
	op_SCKC    uint32 = 0xB206 // FORMAT_S          SET CLOCK COMPARATOR
	op_SCKPF   uint32 = 0x0107 // FORMAT_E          SET CLOCK PROGRAMMABLE FIELD
	op_SD      uint32 = 0x6B00 // FORMAT_RX1        SUBTRACT NORMALIZED (long HFP)
	op_SDB     uint32 = 0xED1B // FORMAT_RXE        SUBTRACT (long BFP)
	op_SDBR    uint32 = 0xB31B // FORMAT_RRE        SUBTRACT (long BFP)
	op_SDR     uint32 = 0x2B00 // FORMAT_RR         SUBTRACT NORMALIZED (long HFP)
	op_SDTR    uint32 = 0xB3D3 // FORMAT_RRF1       SUBTRACT (long DFP)
	op_SDTRA   uint32 = 0xB3D3 // FORMAT_RRF1       SUBTRACT (long DFP)
	op_SE      uint32 = 0x7B00 // FORMAT_RX1        SUBTRACT NORMALIZED (short HFP)
	op_SEB     uint32 = 0xED0B // FORMAT_RXE        SUBTRACT (short BFP)
	op_SEBR    uint32 = 0xB30B // FORMAT_RRE        SUBTRACT (short BFP)
	op_SER     uint32 = 0x3B00 // FORMAT_RR         SUBTRACT NORMALIZED (short HFP)
	op_SFASR   uint32 = 0xB385 // FORMAT_RRE        SET FPC AND SIGNAL
	op_SFPC    uint32 = 0xB384 // FORMAT_RRE        SET FPC
	op_SG      uint32 = 0xE309 // FORMAT_RXY1       SUBTRACT (64)
	op_SGF     uint32 = 0xE319 // FORMAT_RXY1       SUBTRACT (64<-32)
	op_SGFR    uint32 = 0xB919 // FORMAT_RRE        SUBTRACT (64<-32)
	op_SGR     uint32 = 0xB909 // FORMAT_RRE        SUBTRACT (64)
	op_SGRK    uint32 = 0xB9E9 // FORMAT_RRF1       SUBTRACT (64)
	op_SH      uint32 = 0x4B00 // FORMAT_RX1        SUBTRACT HALFWORD
	op_SHHHR   uint32 = 0xB9C9 // FORMAT_RRF1       SUBTRACT HIGH (32)
	op_SHHLR   uint32 = 0xB9D9 // FORMAT_RRF1       SUBTRACT HIGH (32)
	op_SHY     uint32 = 0xE37B // FORMAT_RXY1       SUBTRACT HALFWORD
	op_SIGP    uint32 = 0xAE00 // FORMAT_RS1        SIGNAL PROCESSOR
	op_SL      uint32 = 0x5F00 // FORMAT_RX1        SUBTRACT LOGICAL (32)
	op_SLA     uint32 = 0x8B00 // FORMAT_RS1        SHIFT LEFT SINGLE (32)
	op_SLAG    uint32 = 0xEB0B // FORMAT_RSY1       SHIFT LEFT SINGLE (64)
	op_SLAK    uint32 = 0xEBDD // FORMAT_RSY1       SHIFT LEFT SINGLE (32)
	op_SLB     uint32 = 0xE399 // FORMAT_RXY1       SUBTRACT LOGICAL WITH BORROW (32)
	op_SLBG    uint32 = 0xE389 // FORMAT_RXY1       SUBTRACT LOGICAL WITH BORROW (64)
	op_SLBGR   uint32 = 0xB989 // FORMAT_RRE        SUBTRACT LOGICAL WITH BORROW (64)
	op_SLBR    uint32 = 0xB999 // FORMAT_RRE        SUBTRACT LOGICAL WITH BORROW (32)
	op_SLDA    uint32 = 0x8F00 // FORMAT_RS1        SHIFT LEFT DOUBLE
	op_SLDL    uint32 = 0x8D00 // FORMAT_RS1        SHIFT LEFT DOUBLE LOGICAL
	op_SLDT    uint32 = 0xED40 // FORMAT_RXF        SHIFT SIGNIFICAND LEFT (long DFP)
	op_SLFI    uint32 = 0xC205 // FORMAT_RIL1       SUBTRACT LOGICAL IMMEDIATE (32)
	op_SLG     uint32 = 0xE30B // FORMAT_RXY1       SUBTRACT LOGICAL (64)
	op_SLGF    uint32 = 0xE31B // FORMAT_RXY1       SUBTRACT LOGICAL (64<-32)
	op_SLGFI   uint32 = 0xC204 // FORMAT_RIL1       SUBTRACT LOGICAL IMMEDIATE (64<-32)
	op_SLGFR   uint32 = 0xB91B // FORMAT_RRE        SUBTRACT LOGICAL (64<-32)
	op_SLGR    uint32 = 0xB90B // FORMAT_RRE        SUBTRACT LOGICAL (64)
	op_SLGRK   uint32 = 0xB9EB // FORMAT_RRF1       SUBTRACT LOGICAL (64)
	op_SLHHHR  uint32 = 0xB9CB // FORMAT_RRF1       SUBTRACT LOGICAL HIGH (32)
	op_SLHHLR  uint32 = 0xB9DB // FORMAT_RRF1       SUBTRACT LOGICAL HIGH (32)
	op_SLL     uint32 = 0x8900 // FORMAT_RS1        SHIFT LEFT SINGLE LOGICAL (32)
	op_SLLG    uint32 = 0xEB0D // FORMAT_RSY1       SHIFT LEFT SINGLE LOGICAL (64)
	op_SLLK    uint32 = 0xEBDF // FORMAT_RSY1       SHIFT LEFT SINGLE LOGICAL (32)
	op_SLR     uint32 = 0x1F00 // FORMAT_RR         SUBTRACT LOGICAL (32)
	op_SLRK    uint32 = 0xB9FB // FORMAT_RRF1       SUBTRACT LOGICAL (32)
	op_SLXT    uint32 = 0xED48 // FORMAT_RXF        SHIFT SIGNIFICAND LEFT (extended DFP)
	op_SLY     uint32 = 0xE35F // FORMAT_RXY1       SUBTRACT LOGICAL (32)
	op_SP      uint32 = 0xFB00 // FORMAT_SS2        SUBTRACT DECIMAL
	op_SPKA    uint32 = 0xB20A // FORMAT_S          SET PSW KEY FROM ADDRESS
	op_SPM     uint32 = 0x0400 // FORMAT_RR         SET PROGRAM MASK
	op_SPT     uint32 = 0xB208 // FORMAT_S          SET CPU TIMER
	op_SPX     uint32 = 0xB210 // FORMAT_S          SET PREFIX
	op_SQD     uint32 = 0xED35 // FORMAT_RXE        SQUARE ROOT (long HFP)
	op_SQDB    uint32 = 0xED15 // FORMAT_RXE        SQUARE ROOT (long BFP)
	op_SQDBR   uint32 = 0xB315 // FORMAT_RRE        SQUARE ROOT (long BFP)
	op_SQDR    uint32 = 0xB244 // FORMAT_RRE        SQUARE ROOT (long HFP)
	op_SQE     uint32 = 0xED34 // FORMAT_RXE        SQUARE ROOT (short HFP)
	op_SQEB    uint32 = 0xED14 // FORMAT_RXE        SQUARE ROOT (short BFP)
	op_SQEBR   uint32 = 0xB314 // FORMAT_RRE        SQUARE ROOT (short BFP)
	op_SQER    uint32 = 0xB245 // FORMAT_RRE        SQUARE ROOT (short HFP)
	op_SQXBR   uint32 = 0xB316 // FORMAT_RRE        SQUARE ROOT (extended BFP)
	op_SQXR    uint32 = 0xB336 // FORMAT_RRE        SQUARE ROOT (extended HFP)
	op_SR      uint32 = 0x1B00 // FORMAT_RR         SUBTRACT (32)
	op_SRA     uint32 = 0x8A00 // FORMAT_RS1        SHIFT RIGHT SINGLE (32)
	op_SRAG    uint32 = 0xEB0A // FORMAT_RSY1       SHIFT RIGHT SINGLE (64)
	op_SRAK    uint32 = 0xEBDC // FORMAT_RSY1       SHIFT RIGHT SINGLE (32)
	op_SRDA    uint32 = 0x8E00 // FORMAT_RS1        SHIFT RIGHT DOUBLE
	op_SRDL    uint32 = 0x8C00 // FORMAT_RS1        SHIFT RIGHT DOUBLE LOGICAL
	op_SRDT    uint32 = 0xED41 // FORMAT_RXF        SHIFT SIGNIFICAND RIGHT (long DFP)
	op_SRK     uint32 = 0xB9F9 // FORMAT_RRF1       SUBTRACT (32)
	op_SRL     uint32 = 0x8800 // FORMAT_RS1        SHIFT RIGHT SINGLE LOGICAL (32)
	op_SRLG    uint32 = 0xEB0C // FORMAT_RSY1       SHIFT RIGHT SINGLE LOGICAL (64)
	op_SRLK    uint32 = 0xEBDE // FORMAT_RSY1       SHIFT RIGHT SINGLE LOGICAL (32)
	op_SRNM    uint32 = 0xB299 // FORMAT_S          SET BFP ROUNDING MODE (2 bit)
	op_SRNMB   uint32 = 0xB2B8 // FORMAT_S          SET BFP ROUNDING MODE (3 bit)
	op_SRNMT   uint32 = 0xB2B9 // FORMAT_S          SET DFP ROUNDING MODE
	op_SRP     uint32 = 0xF000 // FORMAT_SS3        SHIFT AND ROUND DECIMAL
	op_SRST    uint32 = 0xB25E // FORMAT_RRE        SEARCH STRING
	op_SRSTU   uint32 = 0xB9BE // FORMAT_RRE        SEARCH STRING UNICODE
	op_SRXT    uint32 = 0xED49 // FORMAT_RXF        SHIFT SIGNIFICAND RIGHT (extended DFP)
	op_SSAIR   uint32 = 0xB99F // FORMAT_RRE        SET SECONDARY ASN WITH INSTANCE
	op_SSAR    uint32 = 0xB225 // FORMAT_RRE        SET SECONDARY ASN
	op_SSCH    uint32 = 0xB233 // FORMAT_S          START SUBCHANNEL
	op_SSKE    uint32 = 0xB22B // FORMAT_RRF3       SET STORAGE KEY EXTENDED
	op_SSM     uint32 = 0x8000 // FORMAT_S          SET SYSTEM MASK
	op_ST      uint32 = 0x5000 // FORMAT_RX1        STORE (32)
	op_STAM    uint32 = 0x9B00 // FORMAT_RS1        STORE ACCESS MULTIPLE
	op_STAMY   uint32 = 0xEB9B // FORMAT_RSY1       STORE ACCESS MULTIPLE
	op_STAP    uint32 = 0xB212 // FORMAT_S          STORE CPU ADDRESS
	op_STC     uint32 = 0x4200 // FORMAT_RX1        STORE CHARACTER
	op_STCH    uint32 = 0xE3C3 // FORMAT_RXY1       STORE CHARACTER HIGH (8)
	op_STCK    uint32 = 0xB205 // FORMAT_S          STORE CLOCK
	op_STCKC   uint32 = 0xB207 // FORMAT_S          STORE CLOCK COMPARATOR
	op_STCKE   uint32 = 0xB278 // FORMAT_S          STORE CLOCK EXTENDED
	op_STCKF   uint32 = 0xB27C // FORMAT_S          STORE CLOCK FAST
	op_STCM    uint32 = 0xBE00 // FORMAT_RS2        STORE CHARACTERS UNDER MASK (low)
	op_STCMH   uint32 = 0xEB2C // FORMAT_RSY2       STORE CHARACTERS UNDER MASK (high)
	op_STCMY   uint32 = 0xEB2D // FORMAT_RSY2       STORE CHARACTERS UNDER MASK (low)
	op_STCPS   uint32 = 0xB23A // FORMAT_S          STORE CHANNEL PATH STATUS
	op_STCRW   uint32 = 0xB239 // FORMAT_S          STORE CHANNEL REPORT WORD
	op_STCTG   uint32 = 0xEB25 // FORMAT_RSY1       STORE CONTROL (64)
	op_STCTL   uint32 = 0xB600 // FORMAT_RS1        STORE CONTROL (32)
	op_STCY    uint32 = 0xE372 // FORMAT_RXY1       STORE CHARACTER
	op_STD     uint32 = 0x6000 // FORMAT_RX1        STORE (long)
	op_STDY    uint32 = 0xED67 // FORMAT_RXY1       STORE (long)
	op_STE     uint32 = 0x7000 // FORMAT_RX1        STORE (short)
	op_STEY    uint32 = 0xED66 // FORMAT_RXY1       STORE (short)
	op_STFH    uint32 = 0xE3CB // FORMAT_RXY1       STORE HIGH (32)
	op_STFL    uint32 = 0xB2B1 // FORMAT_S          STORE FACILITY LIST
	op_STFLE   uint32 = 0xB2B0 // FORMAT_S          STORE FACILITY LIST EXTENDED
	op_STFPC   uint32 = 0xB29C // FORMAT_S          STORE FPC
	op_STG     uint32 = 0xE324 // FORMAT_RXY1       STORE (64)
	op_STGRL   uint32 = 0xC40B // FORMAT_RIL2       STORE RELATIVE LONG (64)
	op_STH     uint32 = 0x4000 // FORMAT_RX1        STORE HALFWORD
	op_STHH    uint32 = 0xE3C7 // FORMAT_RXY1       STORE HALFWORD HIGH (16)
	op_STHRL   uint32 = 0xC407 // FORMAT_RIL2       STORE HALFWORD RELATIVE LONG
	op_STHY    uint32 = 0xE370 // FORMAT_RXY1       STORE HALFWORD
	op_STIDP   uint32 = 0xB202 // FORMAT_S          STORE CPU ID
	op_STM     uint32 = 0x9000 // FORMAT_RS1        STORE MULTIPLE (32)
	op_STMG    uint32 = 0xEB24 // FORMAT_RSY1       STORE MULTIPLE (64)
	op_STMH    uint32 = 0xEB26 // FORMAT_RSY1       STORE MULTIPLE HIGH
	op_STMY    uint32 = 0xEB90 // FORMAT_RSY1       STORE MULTIPLE (32)
	op_STNSM   uint32 = 0xAC00 // FORMAT_SI         STORE THEN AND SYSTEM MASK
	op_STOC    uint32 = 0xEBF3 // FORMAT_RSY2       STORE ON CONDITION (32)
	op_STOCG   uint32 = 0xEBE3 // FORMAT_RSY2       STORE ON CONDITION (64)
	op_STOSM   uint32 = 0xAD00 // FORMAT_SI         STORE THEN OR SYSTEM MASK
	op_STPQ    uint32 = 0xE38E // FORMAT_RXY1       STORE PAIR TO QUADWORD
	op_STPT    uint32 = 0xB209 // FORMAT_S          STORE CPU TIMER
	op_STPX    uint32 = 0xB211 // FORMAT_S          STORE PREFIX
	op_STRAG   uint32 = 0xE502 // FORMAT_SSE        STORE REAL ADDRESS
	op_STRL    uint32 = 0xC40F // FORMAT_RIL2       STORE RELATIVE LONG (32)
	op_STRV    uint32 = 0xE33E // FORMAT_RXY1       STORE REVERSED (32)
	op_STRVG   uint32 = 0xE32F // FORMAT_RXY1       STORE REVERSED (64)
	op_STRVH   uint32 = 0xE33F // FORMAT_RXY1       STORE REVERSED (16)
	op_STSCH   uint32 = 0xB234 // FORMAT_S          STORE SUBCHANNEL
	op_STSI    uint32 = 0xB27D // FORMAT_S          STORE SYSTEM INFORMATION
	op_STURA   uint32 = 0xB246 // FORMAT_RRE        STORE USING REAL ADDRESS (32)
	op_STURG   uint32 = 0xB925 // FORMAT_RRE        STORE USING REAL ADDRESS (64)
	op_STY     uint32 = 0xE350 // FORMAT_RXY1       STORE (32)
	op_SU      uint32 = 0x7F00 // FORMAT_RX1        SUBTRACT UNNORMALIZED (short HFP)
	op_SUR     uint32 = 0x3F00 // FORMAT_RR         SUBTRACT UNNORMALIZED (short HFP)
	op_SVC     uint32 = 0x0A00 // FORMAT_I          SUPERVISOR CALL
	op_SW      uint32 = 0x6F00 // FORMAT_RX1        SUBTRACT UNNORMALIZED (long HFP)
	op_SWR     uint32 = 0x2F00 // FORMAT_RR         SUBTRACT UNNORMALIZED (long HFP)
	op_SXBR    uint32 = 0xB34B // FORMAT_RRE        SUBTRACT (extended BFP)
	op_SXR     uint32 = 0x3700 // FORMAT_RR         SUBTRACT NORMALIZED (extended HFP)
	op_SXTR    uint32 = 0xB3DB // FORMAT_RRF1       SUBTRACT (extended DFP)
	op_SXTRA   uint32 = 0xB3DB // FORMAT_RRF1       SUBTRACT (extended DFP)
	op_SY      uint32 = 0xE35B // FORMAT_RXY1       SUBTRACT (32)
	op_TABORT  uint32 = 0xB2FC // FORMAT_S          TRANSACTION ABORT
	op_TAM     uint32 = 0x010B // FORMAT_E          TEST ADDRESSING MODE
	op_TAR     uint32 = 0xB24C // FORMAT_RRE        TEST ACCESS
	op_TB      uint32 = 0xB22C // FORMAT_RRE        TEST BLOCK
	op_TBDR    uint32 = 0xB351 // FORMAT_RRF5       CONVERT HFP TO BFP (long)
	op_TBEDR   uint32 = 0xB350 // FORMAT_RRF5       CONVERT HFP TO BFP (long to short)
	op_TBEGIN  uint32 = 0xE560 // FORMAT_SIL        TRANSACTION BEGIN
	op_TBEGINC uint32 = 0xE561 // FORMAT_SIL        TRANSACTION BEGIN
	op_TCDB    uint32 = 0xED11 // FORMAT_RXE        TEST DATA CLASS (long BFP)
	op_TCEB    uint32 = 0xED10 // FORMAT_RXE        TEST DATA CLASS (short BFP)
	op_TCXB    uint32 = 0xED12 // FORMAT_RXE        TEST DATA CLASS (extended BFP)
	op_TDCDT   uint32 = 0xED54 // FORMAT_RXE        TEST DATA CLASS (long DFP)
	op_TDCET   uint32 = 0xED50 // FORMAT_RXE        TEST DATA CLASS (short DFP)
	op_TDCXT   uint32 = 0xED58 // FORMAT_RXE        TEST DATA CLASS (extended DFP)
	op_TDGDT   uint32 = 0xED55 // FORMAT_RXE        TEST DATA GROUP (long DFP)
	op_TDGET   uint32 = 0xED51 // FORMAT_RXE        TEST DATA GROUP (short DFP)
	op_TDGXT   uint32 = 0xED59 // FORMAT_RXE        TEST DATA GROUP (extended DFP)
	op_TEND    uint32 = 0xB2F8 // FORMAT_S          TRANSACTION END
	op_THDER   uint32 = 0xB358 // FORMAT_RRE        CONVERT BFP TO HFP (short to long)
	op_THDR    uint32 = 0xB359 // FORMAT_RRE        CONVERT BFP TO HFP (long)
	op_TM      uint32 = 0x9100 // FORMAT_SI         TEST UNDER MASK
	op_TMH     uint32 = 0xA700 // FORMAT_RI1        TEST UNDER MASK HIGH
	op_TMHH    uint32 = 0xA702 // FORMAT_RI1        TEST UNDER MASK (high high)
	op_TMHL    uint32 = 0xA703 // FORMAT_RI1        TEST UNDER MASK (high low)
	op_TML     uint32 = 0xA701 // FORMAT_RI1        TEST UNDER MASK LOW
	op_TMLH    uint32 = 0xA700 // FORMAT_RI1        TEST UNDER MASK (low high)
	op_TMLL    uint32 = 0xA701 // FORMAT_RI1        TEST UNDER MASK (low low)
	op_TMY     uint32 = 0xEB51 // FORMAT_SIY        TEST UNDER MASK
	op_TP      uint32 = 0xEBC0 // FORMAT_RSL        TEST DECIMAL
	op_TPI     uint32 = 0xB236 // FORMAT_S          TEST PENDING INTERRUPTION
	op_TPROT   uint32 = 0xE501 // FORMAT_SSE        TEST PROTECTION
	op_TR      uint32 = 0xDC00 // FORMAT_SS1        TRANSLATE
	op_TRACE   uint32 = 0x9900 // FORMAT_RS1        TRACE (32)
	op_TRACG   uint32 = 0xEB0F // FORMAT_RSY1       TRACE (64)
	op_TRAP2   uint32 = 0x01FF // FORMAT_E          TRAP
	op_TRAP4   uint32 = 0xB2FF // FORMAT_S          TRAP
	op_TRE     uint32 = 0xB2A5 // FORMAT_RRE        TRANSLATE EXTENDED
	op_TROO    uint32 = 0xB993 // FORMAT_RRF3       TRANSLATE ONE TO ONE
	op_TROT    uint32 = 0xB992 // FORMAT_RRF3       TRANSLATE ONE TO TWO
	op_TRT     uint32 = 0xDD00 // FORMAT_SS1        TRANSLATE AND TEST
	op_TRTE    uint32 = 0xB9BF // FORMAT_RRF3       TRANSLATE AND TEST EXTENDED
	op_TRTO    uint32 = 0xB991 // FORMAT_RRF3       TRANSLATE TWO TO ONE
	op_TRTR    uint32 = 0xD000 // FORMAT_SS1        TRANSLATE AND TEST REVERSE
	op_TRTRE   uint32 = 0xB9BD // FORMAT_RRF3       TRANSLATE AND TEST REVERSE EXTENDED
	op_TRTT    uint32 = 0xB990 // FORMAT_RRF3       TRANSLATE TWO TO TWO
	op_TS      uint32 = 0x9300 // FORMAT_S          TEST AND SET
	op_TSCH    uint32 = 0xB235 // FORMAT_S          TEST SUBCHANNEL
	op_UNPK    uint32 = 0xF300 // FORMAT_SS2        UNPACK
	op_UNPKA   uint32 = 0xEA00 // FORMAT_SS1        UNPACK ASCII
	op_UNPKU   uint32 = 0xE200 // FORMAT_SS1        UNPACK UNICODE
	op_UPT     uint32 = 0x0102 // FORMAT_E          UPDATE TREE
	op_X       uint32 = 0x5700 // FORMAT_RX1        EXCLUSIVE OR (32)
	op_XC      uint32 = 0xD700 // FORMAT_SS1        EXCLUSIVE OR (character)
	op_XG      uint32 = 0xE382 // FORMAT_RXY1       EXCLUSIVE OR (64)
	op_XGR     uint32 = 0xB982 // FORMAT_RRE        EXCLUSIVE OR (64)
	op_XGRK    uint32 = 0xB9E7 // FORMAT_RRF1       EXCLUSIVE OR (64)
	op_XI      uint32 = 0x9700 // FORMAT_SI         EXCLUSIVE OR (immediate)
	op_XIHF    uint32 = 0xC006 // FORMAT_RIL1       EXCLUSIVE OR IMMEDIATE (high)
	op_XILF    uint32 = 0xC007 // FORMAT_RIL1       EXCLUSIVE OR IMMEDIATE (low)
	op_XIY     uint32 = 0xEB57 // FORMAT_SIY        EXCLUSIVE OR (immediate)
	op_XR      uint32 = 0x1700 // FORMAT_RR         EXCLUSIVE OR (32)
	op_XRK     uint32 = 0xB9F7 // FORMAT_RRF1       EXCLUSIVE OR (32)
	op_XSCH    uint32 = 0xB276 // FORMAT_S          CANCEL SUBCHANNEL
	op_XY      uint32 = 0xE357 // FORMAT_RXY1       EXCLUSIVE OR (32)
	op_ZAP     uint32 = 0xF800 // FORMAT_SS2        ZERO AND ADD

	// added in z13
	op_CXPT   uint32 = 0xEDAF // 	RSL-b	CONVERT FROM PACKED (to extended DFP)
	op_CDPT   uint32 = 0xEDAE // 	RSL-b	CONVERT FROM PACKED (to long DFP)
	op_CPXT   uint32 = 0xEDAD // 	RSL-b	CONVERT TO PACKED (from extended DFP)
	op_CPDT   uint32 = 0xEDAC // 	RSL-b	CONVERT TO PACKED (from long DFP)
	op_LZRF   uint32 = 0xE33B // 	RXY-a	LOAD AND ZERO RIGHTMOST BYTE (32)
	op_LZRG   uint32 = 0xE32A // 	RXY-a	LOAD AND ZERO RIGHTMOST BYTE (64)
	op_LCCB   uint32 = 0xE727 // 	RXE	LOAD COUNT TO BLOCK BOUNDARY
	op_LOCHHI uint32 = 0xEC4E // 	RIE-g	LOAD HALFWORD HIGH IMMEDIATE ON CONDITION (32←16)
	op_LOCHI  uint32 = 0xEC42 // 	RIE-g	LOAD HALFWORD IMMEDIATE ON CONDITION (32←16)
	op_LOCGHI uint32 = 0xEC46 // 	RIE-g	LOAD HALFWORD IMMEDIATE ON CONDITION (64←16)
	op_LOCFH  uint32 = 0xEBE0 // 	RSY-b	LOAD HIGH ON CONDITION (32)
	op_LOCFHR uint32 = 0xB9E0 // 	RRF-c	LOAD HIGH ON CONDITION (32)
	op_LLZRGF uint32 = 0xE33A // 	RXY-a	LOAD LOGICAL AND ZERO RIGHTMOST BYTE (64←32)
	op_STOCFH uint32 = 0xEBE1 // 	RSY-b	STORE HIGH ON CONDITION
	op_VA     uint32 = 0xE7F3 // 	VRR-c	VECTOR ADD
	op_VACC   uint32 = 0xE7F1 // 	VRR-c	VECTOR ADD COMPUTE CARRY
	op_VAC    uint32 = 0xE7BB // 	VRR-d	VECTOR ADD WITH CARRY
	op_VACCC  uint32 = 0xE7B9 // 	VRR-d	VECTOR ADD WITH CARRY COMPUTE CARRY
	op_VN     uint32 = 0xE768 // 	VRR-c	VECTOR AND
	op_VNC    uint32 = 0xE769 // 	VRR-c	VECTOR AND WITH COMPLEMENT
	op_VAVG   uint32 = 0xE7F2 // 	VRR-c	VECTOR AVERAGE
	op_VAVGL  uint32 = 0xE7F0 // 	VRR-c	VECTOR AVERAGE LOGICAL
	op_VCKSM  uint32 = 0xE766 // 	VRR-c	VECTOR CHECKSUM
	op_VCEQ   uint32 = 0xE7F8 // 	VRR-b	VECTOR COMPARE EQUAL
	op_VCH    uint32 = 0xE7FB // 	VRR-b	VECTOR COMPARE HIGH
	op_VCHL   uint32 = 0xE7F9 // 	VRR-b	VECTOR COMPARE HIGH LOGICAL
	op_VCLZ   uint32 = 0xE753 // 	VRR-a	VECTOR COUNT LEADING ZEROS
	op_VCTZ   uint32 = 0xE752 // 	VRR-a	VECTOR COUNT TRAILING ZEROS
	op_VEC    uint32 = 0xE7DB // 	VRR-a	VECTOR ELEMENT COMPARE
	op_VECL   uint32 = 0xE7D9 // 	VRR-a	VECTOR ELEMENT COMPARE LOGICAL
	op_VERIM  uint32 = 0xE772 // 	VRI-d	VECTOR ELEMENT ROTATE AND INSERT UNDER MASK
	op_VERLL  uint32 = 0xE733 // 	VRS-a	VECTOR ELEMENT ROTATE LEFT LOGICAL
	op_VERLLV uint32 = 0xE773 // 	VRR-c	VECTOR ELEMENT ROTATE LEFT LOGICAL
	op_VESLV  uint32 = 0xE770 // 	VRR-c	VECTOR ELEMENT SHIFT LEFT
	op_VESL   uint32 = 0xE730 // 	VRS-a	VECTOR ELEMENT SHIFT LEFT
	op_VESRA  uint32 = 0xE73A // 	VRS-a	VECTOR ELEMENT SHIFT RIGHT ARITHMETIC
	op_VESRAV uint32 = 0xE77A // 	VRR-c	VECTOR ELEMENT SHIFT RIGHT ARITHMETIC
	op_VESRL  uint32 = 0xE738 // 	VRS-a	VECTOR ELEMENT SHIFT RIGHT LOGICAL
	op_VESRLV uint32 = 0xE778 // 	VRR-c	VECTOR ELEMENT SHIFT RIGHT LOGICAL
	op_VX     uint32 = 0xE76D // 	VRR-c	VECTOR EXCLUSIVE OR
	op_VFAE   uint32 = 0xE782 // 	VRR-b	VECTOR FIND ANY ELEMENT EQUAL
	op_VFEE   uint32 = 0xE780 // 	VRR-b	VECTOR FIND ELEMENT EQUAL
	op_VFENE  uint32 = 0xE781 // 	VRR-b	VECTOR FIND ELEMENT NOT EQUAL
	op_VFA    uint32 = 0xE7E3 // 	VRR-c	VECTOR FP ADD
	op_WFK    uint32 = 0xE7CA // 	VRR-a	VECTOR FP COMPARE AND SIGNAL SCALAR
	op_VFCE   uint32 = 0xE7E8 // 	VRR-c	VECTOR FP COMPARE EQUAL
	op_VFCH   uint32 = 0xE7EB // 	VRR-c	VECTOR FP COMPARE HIGH
	op_VFCHE  uint32 = 0xE7EA // 	VRR-c	VECTOR FP COMPARE HIGH OR EQUAL
	op_WFC    uint32 = 0xE7CB // 	VRR-a	VECTOR FP COMPARE SCALAR
	op_VCDG   uint32 = 0xE7C3 // 	VRR-a	VECTOR FP CONVERT FROM FIXED 64-BIT
	op_VCDLG  uint32 = 0xE7C1 // 	VRR-a	VECTOR FP CONVERT FROM LOGICAL 64-BIT
	op_VCGD   uint32 = 0xE7C2 // 	VRR-a	VECTOR FP CONVERT TO FIXED 64-BIT
	op_VCLGD  uint32 = 0xE7C0 // 	VRR-a	VECTOR FP CONVERT TO LOGICAL 64-BIT
	op_VFD    uint32 = 0xE7E5 // 	VRR-c	VECTOR FP DIVIDE
	op_VLDE   uint32 = 0xE7C4 // 	VRR-a	VECTOR FP LOAD LENGTHENED
	op_VLED   uint32 = 0xE7C5 // 	VRR-a	VECTOR FP LOAD ROUNDED
	op_VFM    uint32 = 0xE7E7 // 	VRR-c	VECTOR FP MULTIPLY
	op_VFMA   uint32 = 0xE78F // 	VRR-e	VECTOR FP MULTIPLY AND ADD
	op_VFMS   uint32 = 0xE78E // 	VRR-e	VECTOR FP MULTIPLY AND SUBTRACT
	op_VFPSO  uint32 = 0xE7CC // 	VRR-a	VECTOR FP PERFORM SIGN OPERATION
	op_VFSQ   uint32 = 0xE7CE // 	VRR-a	VECTOR FP SQUARE ROOT
	op_VFS    uint32 = 0xE7E2 // 	VRR-c	VECTOR FP SUBTRACT
	op_VFTCI  uint32 = 0xE74A // 	VRI-e	VECTOR FP TEST DATA CLASS IMMEDIATE
	op_VGFM   uint32 = 0xE7B4 // 	VRR-c	VECTOR GALOIS FIELD MULTIPLY SUM
	op_VGFMA  uint32 = 0xE7BC // 	VRR-d	VECTOR GALOIS FIELD MULTIPLY SUM AND ACCUMULATE
	op_VGEF   uint32 = 0xE713 // 	VRV	VECTOR GATHER ELEMENT (32)
	op_VGEG   uint32 = 0xE712 // 	VRV	VECTOR GATHER ELEMENT (64)
	op_VGBM   uint32 = 0xE744 // 	VRI-a	VECTOR GENERATE BYTE MASK
	op_VGM    uint32 = 0xE746 // 	VRI-b	VECTOR GENERATE MASK
	op_VISTR  uint32 = 0xE75C // 	VRR-a	VECTOR ISOLATE STRING
	op_VL     uint32 = 0xE706 // 	VRX	VECTOR LOAD
	op_VLR    uint32 = 0xE756 // 	VRR-a	VECTOR LOAD
	op_VLREP  uint32 = 0xE705 // 	VRX	VECTOR LOAD AND REPLICATE
	op_VLC    uint32 = 0xE7DE // 	VRR-a	VECTOR LOAD COMPLEMENT
	op_VLEH   uint32 = 0xE701 // 	VRX	VECTOR LOAD ELEMENT (16)
	op_VLEF   uint32 = 0xE703 // 	VRX	VECTOR LOAD ELEMENT (32)
	op_VLEG   uint32 = 0xE702 // 	VRX	VECTOR LOAD ELEMENT (64)
	op_VLEB   uint32 = 0xE700 // 	VRX	VECTOR LOAD ELEMENT (8)
	op_VLEIH  uint32 = 0xE741 // 	VRI-a	VECTOR LOAD ELEMENT IMMEDIATE (16)
	op_VLEIF  uint32 = 0xE743 // 	VRI-a	VECTOR LOAD ELEMENT IMMEDIATE (32)
	op_VLEIG  uint32 = 0xE742 // 	VRI-a	VECTOR LOAD ELEMENT IMMEDIATE (64)
	op_VLEIB  uint32 = 0xE740 // 	VRI-a	VECTOR LOAD ELEMENT IMMEDIATE (8)
	op_VFI    uint32 = 0xE7C7 // 	VRR-a	VECTOR LOAD FP INTEGER
	op_VLGV   uint32 = 0xE721 // 	VRS-c	VECTOR LOAD GR FROM VR ELEMENT
	op_VLLEZ  uint32 = 0xE704 // 	VRX	VECTOR LOAD LOGICAL ELEMENT AND ZERO
	op_VLM    uint32 = 0xE736 // 	VRS-a	VECTOR LOAD MULTIPLE
	op_VLP    uint32 = 0xE7DF // 	VRR-a	VECTOR LOAD POSITIVE
	op_VLBB   uint32 = 0xE707 // 	VRX	VECTOR LOAD TO BLOCK BOUNDARY
	op_VLVG   uint32 = 0xE722 // 	VRS-b	VECTOR LOAD VR ELEMENT FROM GR
	op_VLVGP  uint32 = 0xE762 // 	VRR-f	VECTOR LOAD VR FROM GRS DISJOINT
	op_VLL    uint32 = 0xE737 // 	VRS-b	VECTOR LOAD WITH LENGTH
	op_VMX    uint32 = 0xE7FF // 	VRR-c	VECTOR MAXIMUM
	op_VMXL   uint32 = 0xE7FD // 	VRR-c	VECTOR MAXIMUM LOGICAL
	op_VMRH   uint32 = 0xE761 // 	VRR-c	VECTOR MERGE HIGH
	op_VMRL   uint32 = 0xE760 // 	VRR-c	VECTOR MERGE LOW
	op_VMN    uint32 = 0xE7FE // 	VRR-c	VECTOR MINIMUM
	op_VMNL   uint32 = 0xE7FC // 	VRR-c	VECTOR MINIMUM LOGICAL
	op_VMAE   uint32 = 0xE7AE // 	VRR-d	VECTOR MULTIPLY AND ADD EVEN
	op_VMAH   uint32 = 0xE7AB // 	VRR-d	VECTOR MULTIPLY AND ADD HIGH
	op_VMALE  uint32 = 0xE7AC // 	VRR-d	VECTOR MULTIPLY AND ADD LOGICAL EVEN
	op_VMALH  uint32 = 0xE7A9 // 	VRR-d	VECTOR MULTIPLY AND ADD LOGICAL HIGH
	op_VMALO  uint32 = 0xE7AD // 	VRR-d	VECTOR MULTIPLY AND ADD LOGICAL ODD
	op_VMAL   uint32 = 0xE7AA // 	VRR-d	VECTOR MULTIPLY AND ADD LOW
	op_VMAO   uint32 = 0xE7AF // 	VRR-d	VECTOR MULTIPLY AND ADD ODD
	op_VME    uint32 = 0xE7A6 // 	VRR-c	VECTOR MULTIPLY EVEN
	op_VMH    uint32 = 0xE7A3 // 	VRR-c	VECTOR MULTIPLY HIGH
	op_VMLE   uint32 = 0xE7A4 // 	VRR-c	VECTOR MULTIPLY EVEN LOGICAL
	op_VMLH   uint32 = 0xE7A1 // 	VRR-c	VECTOR MULTIPLY HIGH LOGICAL
	op_VMLO   uint32 = 0xE7A5 // 	VRR-c	VECTOR MULTIPLY ODD LOGICAL
	op_VML    uint32 = 0xE7A2 // 	VRR-c	VECTOR MULTIPLY LOW
	op_VMO    uint32 = 0xE7A7 // 	VRR-c	VECTOR MULTIPLY ODD
	op_VNO    uint32 = 0xE76B // 	VRR-c	VECTOR NOR
	op_VO     uint32 = 0xE76A // 	VRR-c	VECTOR OR
	op_VPK    uint32 = 0xE794 // 	VRR-c	VECTOR PACK
	op_VPKLS  uint32 = 0xE795 // 	VRR-b	VECTOR PACK LOGICAL SATURATE
	op_VPKS   uint32 = 0xE797 // 	VRR-b	VECTOR PACK SATURATE
	op_VPERM  uint32 = 0xE78C // 	VRR-e	VECTOR PERMUTE
	op_VPDI   uint32 = 0xE784 // 	VRR-c	VECTOR PERMUTE DOUBLEWORD IMMEDIATE
	op_VPOPCT uint32 = 0xE750 // 	VRR-a	VECTOR POPULATION COUNT
	op_VREP   uint32 = 0xE74D // 	VRI-c	VECTOR REPLICATE
	op_VREPI  uint32 = 0xE745 // 	VRI-a	VECTOR REPLICATE IMMEDIATE
	op_VSCEF  uint32 = 0xE71B // 	VRV	VECTOR SCATTER ELEMENT (32)
	op_VSCEG  uint32 = 0xE71A // 	VRV	VECTOR SCATTER ELEMENT (64)
	op_VSEL   uint32 = 0xE78D // 	VRR-e	VECTOR SELECT
	op_VSL    uint32 = 0xE774 // 	VRR-c	VECTOR SHIFT LEFT
	op_VSLB   uint32 = 0xE775 // 	VRR-c	VECTOR SHIFT LEFT BY BYTE
	op_VSLDB  uint32 = 0xE777 // 	VRI-d	VECTOR SHIFT LEFT DOUBLE BY BYTE
	op_VSRA   uint32 = 0xE77E // 	VRR-c	VECTOR SHIFT RIGHT ARITHMETIC
	op_VSRAB  uint32 = 0xE77F // 	VRR-c	VECTOR SHIFT RIGHT ARITHMETIC BY BYTE
	op_VSRL   uint32 = 0xE77C // 	VRR-c	VECTOR SHIFT RIGHT LOGICAL
	op_VSRLB  uint32 = 0xE77D // 	VRR-c	VECTOR SHIFT RIGHT LOGICAL BY BYTE
	op_VSEG   uint32 = 0xE75F // 	VRR-a	VECTOR SIGN EXTEND TO DOUBLEWORD
	op_VST    uint32 = 0xE70E // 	VRX	VECTOR STORE
	op_VSTEH  uint32 = 0xE709 // 	VRX	VECTOR STORE ELEMENT (16)
	op_VSTEF  uint32 = 0xE70B // 	VRX	VECTOR STORE ELEMENT (32)
	op_VSTEG  uint32 = 0xE70A // 	VRX	VECTOR STORE ELEMENT (64)
	op_VSTEB  uint32 = 0xE708 // 	VRX	VECTOR STORE ELEMENT (8)
	op_VSTM   uint32 = 0xE73E // 	VRS-a	VECTOR STORE MULTIPLE
	op_VSTL   uint32 = 0xE73F // 	VRS-b	VECTOR STORE WITH LENGTH
	op_VSTRC  uint32 = 0xE78A // 	VRR-d	VECTOR STRING RANGE COMPARE
	op_VS     uint32 = 0xE7F7 // 	VRR-c	VECTOR SUBTRACT
	op_VSCBI  uint32 = 0xE7F5 // 	VRR-c	VECTOR SUBTRACT COMPUTE BORROW INDICATION
	op_VSBCBI uint32 = 0xE7BD // 	VRR-d	VECTOR SUBTRACT WITH BORROW COMPUTE BORROW INDICATION
	op_VSBI   uint32 = 0xE7BF // 	VRR-d	VECTOR SUBTRACT WITH BORROW INDICATION
	op_VSUMG  uint32 = 0xE765 // 	VRR-c	VECTOR SUM ACROSS DOUBLEWORD
	op_VSUMQ  uint32 = 0xE767 // 	VRR-c	VECTOR SUM ACROSS QUADWORD
	op_VSUM   uint32 = 0xE764 // 	VRR-c	VECTOR SUM ACROSS WORD
	op_VTM    uint32 = 0xE7D8 // 	VRR-a	VECTOR TEST UNDER MASK
	op_VUPH   uint32 = 0xE7D7 // 	VRR-a	VECTOR UNPACK HIGH
	op_VUPLH  uint32 = 0xE7D5 // 	VRR-a	VECTOR UNPACK LOGICAL HIGH
	op_VUPLL  uint32 = 0xE7D4 // 	VRR-a	VECTOR UNPACK LOGICAL LOW
	op_VUPL   uint32 = 0xE7D6 // 	VRR-a	VECTOR UNPACK LOW
)

func oclass(a *obj.Addr) int {
	return int(a.Class) - 1
}

// Add a relocation for the immediate in a RIL style instruction.
// The addend will be adjusted as required.
func addrilreloc(ctxt *obj.Link, sym *obj.LSym, add int64) *obj.Reloc {
	if sym == nil {
		ctxt.Diag("require symbol to apply relocation")
	}
	offset := int64(2) // relocation offset from start of instruction
	rel := obj.Addrel(ctxt.Cursym)
	rel.Off = int32(ctxt.Pc + offset)
	rel.Siz = 4
	rel.Sym = sym
	rel.Add = add + offset + int64(rel.Siz)
	rel.Type = obj.R_PCRELDBL
	return rel
}

func addrilrelocoffset(ctxt *obj.Link, sym *obj.LSym, add, offset int64) *obj.Reloc {
	if sym == nil {
		ctxt.Diag("require symbol to apply relocation")
	}
	offset += int64(2) // relocation offset from start of instruction
	rel := obj.Addrel(ctxt.Cursym)
	rel.Off = int32(ctxt.Pc + offset)
	rel.Siz = 4
	rel.Sym = sym
	rel.Add = add + offset + int64(rel.Siz)
	rel.Type = obj.R_PCRELDBL
	return rel
}

// Add a CALL relocation for the immediate in a RIL style instruction.
// The addend will be adjusted as required.
func addcallreloc(ctxt *obj.Link, sym *obj.LSym, add int64) *obj.Reloc {
	if sym == nil {
		ctxt.Diag("require symbol to apply relocation")
	}
	offset := int64(2) // relocation offset from start of instruction
	rel := obj.Addrel(ctxt.Cursym)
	rel.Off = int32(ctxt.Pc + offset)
	rel.Siz = 4
	rel.Sym = sym
	rel.Add = add + offset + int64(rel.Siz)
	rel.Type = obj.R_CALL
	return rel
}

func branchMask(ctxt *obj.Link, p *obj.Prog) uint32 {
	switch p.As {
	case ABEQ, ACMPBEQ, ACMPUBEQ:
		return 0x8
	case ABGE, ACMPBGE, ACMPUBGE:
		return 0xA
	case ABGT, ACMPBGT, ACMPUBGT:
		return 0x2
	case ABLE, ACMPBLE, ACMPUBLE:
		return 0xC
	case ABLT, ACMPBLT, ACMPUBLT:
		return 0x4
	case ABNE, ACMPBNE, ACMPUBNE:
		return 0x7
	case ABVC:
		return 0x0 //needs extra instruction
	case ABVS:
		return 0x1
	}
	ctxt.Diag("unknown conditional branch %v", p.As)
	return 0xF
}

func asmout(ctxt *obj.Link, asm *[]byte) {
	p := ctxt.Curp
	o := oplook(ctxt, p)
	ctxt.Printp = p

	switch o.type_ {
	default:
		ctxt.Diag("unknown type %d", o.type_)

	case 0: // PSEUDO OPS
		break

	case 1: // MOV REG TO REG
		switch p.As {
		default:
			ctxt.Diag("unhandled operation: %v", p.As)
		case AMOVD:
			zRRE(op_LGR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		// sign extend
		case AMOVW:
			zRRE(op_LGFR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		case AMOVH:
			zRRE(op_LGHR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		case AMOVB:
			zRRE(op_LGBR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		// zero extend
		case AMOVWZ:
			zRRE(op_LLGFR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		case AMOVHZ:
			zRRE(op_LLGHR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		case AMOVBZ:
			zRRE(op_LLGCR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		// reverse bytes
		case AMOVDBR:
			zRRE(op_LRVGR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		case AMOVWBR:
			zRRE(op_LRVR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		// floating point
		case AFMOVD, AFMOVS:
			zRR(op_LDR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		}

	case 2: /* int/cr/fp op Rb,[Ra],Rd */
		r := int(p.Reg)
		if r == 0 {
			r = int(p.To.Reg)
		}

		var opcode uint32

		switch p.As {
		default:
			ctxt.Diag("invalid opcode")
		case AADD:
			opcode = op_AGRK
		case AADDC:
			opcode = op_ALGRK
		case AADDE:
			opcode = op_ALCGR
		case AMULLW:
			opcode = op_MSGFR
		case AMULLD:
			opcode = op_MSGR
		case AMULHDU:
			opcode = op_MLGR
		case ADIVW:
			opcode = op_DSGFR
		case ADIVWU:
			opcode = op_DLR
		case ADIVD:
			opcode = op_DSGR
		case ADIVDU:
			opcode = op_DLGR
		case AFADD:
			opcode = op_ADBR
		case AFADDS:
			opcode = op_AEBR
		case AFSUB:
			opcode = op_SDBR
		case AFSUBS:
			opcode = op_SEBR
		case AFDIV:
			opcode = op_DDBR
		case AFDIVS:
			opcode = op_DEBR
		}

		switch p.As {
		default:

		case AADD, AADDC:
			zRRF(opcode, uint32(p.From.Reg), 0, uint32(p.To.Reg), uint32(r), asm)

		case AADDE, AMULLW, AMULLD:
			if r == int(p.To.Reg) {
				zRRE(opcode, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else if p.From.Reg == p.To.Reg {
				zRRE(opcode, uint32(p.To.Reg), uint32(r), asm)
			} else {
				zRRE(op_LGR, uint32(p.To.Reg), uint32(r), asm)
				zRRE(opcode, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			}

		case ADIVW, ADIVWU, ADIVD, ADIVDU:
			if p.As == ADIVWU || p.As == ADIVDU {
				zRRE(op_LGR, REGTMP, REGZERO, asm)
			}
			zRRE(op_LGR, REGTMP2, uint32(r), asm)
			zRRE(opcode, REGTMP, uint32(p.From.Reg), asm)
			zRRE(op_LGR, uint32(p.To.Reg), REGTMP2, asm)

		case AMULHDU:
			zRRE(op_LGR, REGTMP2, uint32(r), asm)
			zRRE(opcode, REGTMP, uint32(p.From.Reg), asm)
			zRRE(op_LGR, uint32(p.To.Reg), REGTMP, asm)

		case AFADD, AFADDS:
			if r == int(p.To.Reg) {
				zRRE(opcode, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else if p.From.Reg == p.To.Reg {
				zRRE(opcode, uint32(p.To.Reg), uint32(r), asm)
			} else {
				zRR(op_LDR, uint32(p.To.Reg), uint32(r), asm)
				zRRE(opcode, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			}

		case AFSUB, AFSUBS, AFDIV, AFDIVS:
			if r == int(p.To.Reg) {
				zRRE(opcode, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else if p.From.Reg == p.To.Reg {
				zRRE(op_LGDR, REGTMP, uint32(r), asm)
				zRRE(opcode, uint32(r), uint32(p.From.Reg), asm)
				zRR(op_LDR, uint32(p.To.Reg), uint32(r), asm)
				zRRE(op_LDGR, uint32(r), REGTMP, asm)
			} else {
				zRR(op_LDR, uint32(p.To.Reg), uint32(r), asm)
				zRRE(opcode, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			}

		}

	case 3: // MOV CONSTANT TO REG
		v := vregoff(ctxt, &p.From)
		switch p.As {
		case AMOVBZ:
			v = int64(uint8(v))
		case AMOVHZ:
			v = int64(uint16(v))
		case AMOVWZ:
			v = int64(uint32(v))
		case AMOVB:
			v = int64(int8(v))
		case AMOVH:
			v = int64(int16(v))
		case AMOVW:
			v = int64(int32(v))
		}
		if v&0xffff == v {
			zRI(op_LLILL, uint32(p.To.Reg), uint32(v), asm)
		} else if v&0xffff0000 == v {
			zRI(op_LLILH, uint32(p.To.Reg), uint32(v>>16), asm)
		} else if v&0xffff00000000 == v {
			zRI(op_LLIHL, uint32(p.To.Reg), uint32(v>>32), asm)
		} else if uint64(v)&0xffff000000000000 == uint64(v) {
			zRI(op_LLIHH, uint32(p.To.Reg), uint32(v>>48), asm)
		} else if int64(int16(v)) == v {
			zRI(op_LGHI, uint32(p.To.Reg), uint32(v), asm)
		} else if int64(int32(v)) == v {
			zRIL(a, op_LGFI, uint32(p.To.Reg), uint32(v), asm)
		} else if int64(uint32(v)) == v {
			zRIL(a, op_LLILF, uint32(p.To.Reg), uint32(v), asm)
		} else if uint64(v)&0xffffffff00000000 == uint64(v) {
			zRIL(a, op_LLIHF, uint32(p.To.Reg), uint32(v>>32), asm)
		} else {
			zRIL(a, op_LLILF, uint32(p.To.Reg), uint32(v), asm)
			zRIL(a, op_IIHF, uint32(p.To.Reg), uint32(v>>32), asm)
		}

	case 5: /* syscall */ // This might be right, assuming SVC is the same as Power's SC
		zI(op_SVC, 0, asm)

	case 6: /* logical op Rb,[Rs,]Ra; no literal */
		if p.To.Reg == 0 {
			ctxt.Diag("literal operation on R0\n%v", p)
		}

		switch p.As {
		case AAND, AOR, AXOR:
			var opcode1, opcode2 uint32
			switch p.As {
			default:
			case AAND:
				opcode1 = op_NGR
				opcode2 = op_NGRK
			case AOR:
				opcode1 = op_OGR
				opcode2 = op_OGRK
			case AXOR:
				opcode1 = op_XGR
				opcode2 = op_XGRK
			}

			r := int(p.Reg)
			if r == 0 {
				zRRE(opcode1, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else {
				zRRF(opcode2, uint32(r), 0, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			}

		case AANDN, AORN:
			var opcode1, opcode2 uint32
			switch p.As {
			default:
			case AANDN:
				opcode1 = op_NGR
				opcode2 = op_NGRK
			case AORN:
				opcode1 = op_OGR
				opcode2 = op_OGRK
			}

			r := int(p.Reg)
			if r == 0 {
				zRRE(op_LCGR, uint32(p.To.Reg), uint32(p.To.Reg), asm)
				zRRE(opcode1, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else {
				zRRE(op_LCGR, REGTMP, uint32(r), asm)
				zRRF(opcode2, REGTMP, 0, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			}

		case ANAND, ANOR:
			var opcode1, opcode2 uint32
			switch p.As {
			default:
			case ANAND:
				opcode1 = op_NGR
				opcode2 = op_NGRK
			case ANOR:
				opcode1 = op_OGR
				opcode2 = op_OGRK
			}

			r := int(p.Reg)
			if r == 0 {
				zRRE(opcode1, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else {
				zRRF(opcode2, uint32(r), 0, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			}

			zRRE(op_LCGR, uint32(p.To.Reg), uint32(p.To.Reg), asm)
		}

	case 7: // shift left/right and rotate left
		d2 := vregoff(ctxt, &p.From)
		b2 := p.From.Reg
		r3 := p.Reg
		if r3 == 0 {
			r3 = p.To.Reg
		}
		r1 := p.To.Reg
		var opcode uint32
		switch p.As {
		default:
		case ASLD:
			opcode = op_SLLG
		case ASRD:
			opcode = op_SRLG
		case ASLW:
			opcode = op_SLLK
		case ASRW:
			opcode = op_SRLK
		case ARLL:
			opcode = op_RLL
		case ARLLG:
			opcode = op_RLLG
		case ASRAW:
			opcode = op_SRAK
		case ASRAD:
			opcode = op_SRAG
		}
		zRSY(opcode, uint32(r1), uint32(r3), uint32(b2), uint32(d2), asm)

	case 10: /* sub Ra,[Rb],Rd => subf Rd,Ra,Rb */
		r := int(p.Reg)

		switch p.As {
		default:
		case ASUB:
			if r == 0 {
				zRRE(op_SGR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else {
				zRRF(op_SGRK, uint32(p.From.Reg), 0, uint32(p.To.Reg), uint32(r), asm)
			}
		case ASUBC:
			if r == 0 {
				zRRE(op_SLGR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else {
				zRRF(op_SLGRK, uint32(p.From.Reg), 0, uint32(p.To.Reg), uint32(r), asm)
			}

		case ASUBE:
			if r == 0 {
				r = int(p.To.Reg)
			}
			if r == int(p.To.Reg) {
				zRRE(op_SLBGR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			} else if p.From.Reg == p.To.Reg {
				zRRE(op_LGR, REGTMP, uint32(p.From.Reg), asm)
				zRRE(op_LGR, uint32(p.To.Reg), uint32(r), asm)
				zRRE(op_SLBGR, uint32(p.To.Reg), REGTMP, asm)
			} else {
				zRRE(op_LGR, uint32(p.To.Reg), uint32(r), asm)
				zRRE(op_SLBGR, uint32(p.To.Reg), uint32(p.From.Reg), asm)
			}
		}

	case 11: /* br/bl lbra */
		v := int32(0)

		if p.Pcond != nil {
			v = int32((p.Pcond.Pc - p.Pc) >> 1)
		}

		if p.As == ABR && p.To.Sym == nil && int32(int16(v)) == v {
			zRI(op_BRC, 0xF, uint32(v), asm)
		} else {
			if p.As == ABL {
				zRIL(b, op_BRASL, uint32(REG_LR), uint32(v), asm)
			} else {
				zRIL(c, op_BRCL, 0xF, uint32(v), asm)
			}
			if p.To.Sym != nil {
				addcallreloc(ctxt, p.To.Sym, p.To.Offset)
			}
		}

	case 15: /* br/bl (r) */
		r := p.To.Reg
		if p.As == ABCL || p.As == ABL {
			zRR(op_BASR, uint32(REG_LR), uint32(r), asm)
		} else {
			zRR(op_BCR, 0xF, uint32(r), asm)
		}

	case 17, /* bc bo,bi,lbra (same for now) */
		16: /* bc bo,bi,sbra */
		v := int32(0)
		if p.Pcond != nil {
			v = int32((p.Pcond.Pc - p.Pc) >> 1)
		}
		mask := branchMask(ctxt, p)
		if p.To.Sym == nil && int32(int16(v)) == v {
			zRI(op_BRC, mask, uint32(v), asm)
		} else {
			zRIL(c, op_BRCL, mask, uint32(v), asm)
		}
		if p.To.Sym != nil {
			addrilreloc(ctxt, p.To.Sym, p.To.Offset)
		}

	case 18: // br/bl r
		switch oclass(&p.To) {
		case C_REG:
			if p.As == ABL {
				zRR(op_BASR, uint32(REG_LR), uint32(p.To.Reg), asm)
			} else {
				zRR(op_BCR, 0xF, uint32(p.To.Reg), asm)
			}
		default:
			ctxt.Diag("bad optab entry (18): %d\n%v", p.To.Class, p)
		}

	case 19: // MOV $sym+n(SB) TO REG
		d := vregoff(ctxt, &p.From)
		zRIL(b, op_LARL, uint32(p.To.Reg), 0, asm)
		if d&1 != 0 {
			zRX(op_LA, uint32(p.To.Reg), uint32(p.To.Reg), 0, 1, asm)
			d -= 1
		}
		addrilreloc(ctxt, p.From.Sym, d)

	case 22: /* add $lcon,r1,r2 ==> cau+or+add */ /* could do add/sub more efficiently */

		if p.From.Sym != nil {
			ctxt.Diag("%v is not supported", p)
		}

		v := vregoff(ctxt, &p.From)
		r := p.Reg
		if r == 0 {
			r = p.To.Reg
		}
		switch p.As {
		default:
		case AADD:
			if r == p.To.Reg {
				zRIL(a, op_AGFI, uint32(p.To.Reg), uint32(v), asm)
			} else if int64(int16(v)) == v {
				zRIE(d, op_AGHIK, uint32(p.To.Reg), uint32(r), uint32(v), 0, 0, 0, 0, asm)
			} else {
				zRRE(op_LGR, uint32(p.To.Reg), uint32(r), asm)
				zRIL(a, op_AGFI, uint32(p.To.Reg), uint32(v), asm)
			}
		case AADDC:
			if r != p.To.Reg {
				zRRE(op_LGR, uint32(p.To.Reg), uint32(r), asm)
			}
			zRIL(a, op_ALGFI, uint32(p.To.Reg), uint32(v), asm)
		case AMULLW:
			if r != p.To.Reg {
				zRRE(op_LGR, uint32(p.To.Reg), uint32(r), asm)
			}
			zRIL(a, op_MSGFI, uint32(p.To.Reg), uint32(v), asm)
		}

	case 23: /* and $lcon,r1,r2 ==> cau+or+and */ /* masks could be done using rlnm etc. */

		v := vregoff(ctxt, &p.From)
		var opcode uint32
		r := p.Reg
		if r == 0 {
			r = p.To.Reg
		}
		if r == p.To.Reg {
			switch p.As {
			default:
				ctxt.Diag("%v is not supported", p)
			case AAND:
				if v >= 0 { // needs zero extend
					zRIL(a, op_LGFI, REGTMP, uint32(v), asm)
					zRRE(op_NGR, uint32(p.To.Reg), REGTMP, asm)
				} else if int64(int16(v)) == v {
					zRI(op_NILL, uint32(p.To.Reg), uint32(v), asm)
				} else { //  r.To.Reg & 0xffffffff00000000 & uint32(v)
					zRIL(a, op_NILF, uint32(p.To.Reg), uint32(v), asm)
				}
			case AOR:
				if int64(uint32(v)) != v { // needs sign extend
					zRIL(a, op_LGFI, REGTMP, uint32(v), asm)
					zRRE(op_OGR, uint32(p.To.Reg), REGTMP, asm)
				} else if int64(uint16(v)) == v {
					zRI(op_OILL, uint32(p.To.Reg), uint32(v), asm)
				} else {
					zRIL(a, op_OILF, uint32(p.To.Reg), uint32(v), asm)
				}
			case AXOR:
				if int64(uint32(v)) != v { // needs sign extend
					zRIL(a, op_LGFI, REGTMP, uint32(v), asm)
					zRRE(op_XGR, uint32(p.To.Reg), REGTMP, asm)
				} else {
					zRIL(a, op_XILF, uint32(p.To.Reg), uint32(v), asm)
				}
			}
		} else {
			switch p.As {
			default:
				ctxt.Diag("%v is not supported", p)
			case AAND:
				opcode = op_NGRK
			case AOR:
				opcode = op_OGRK
			case AXOR:
				opcode = op_XGRK
			}
			zRIL(a, op_LGFI, REGTMP, uint32(v), asm)
			zRRF(opcode, uint32(r), 0, uint32(p.To.Reg), REGTMP, asm)
		}

	case 26: // MOV LACON
		v := regoff(ctxt, &p.From)
		r := p.From.Reg
		if r == 0 {
			r = o.param
		}
		if v >= 0 && v < DISP12 {
			zRX(op_LA, uint32(p.To.Reg), uint32(r), 0, uint32(v), asm)
		} else if v >= -DISP20/2 && v < DISP20/2 {
			zRXY(a, op_LAY, uint32(p.To.Reg), uint32(r), 0, uint32(v), asm)
		} else {
			zRIL(a, op_LGFI, REGTMP, uint32(v), asm)
			zRX(op_LA, uint32(p.To.Reg), uint32(r), REGTMP, 0, asm)
		}

	case 31: /* dword */
		wd := uint64(vregoff(ctxt, &p.From))
		*asm = append(*asm,
			uint8(wd>>56),
			uint8(wd>>48),
			uint8(wd>>40),
			uint8(wd>>32),
			uint8(wd>>24),
			uint8(wd>>16),
			uint8(wd>>8),
			uint8(wd))

	case 32: /* fmul frc,fra,frd */
		r := int(p.Reg)
		if r == 0 {
			r = int(p.To.Reg)
		}

		var opcode uint32

		switch p.As {
		default:
			ctxt.Diag("invalid opcode")
		case AFMUL:
			opcode = op_MDBR
		case AFMULS:
			opcode = op_MEEBR
		}

		if r == int(p.To.Reg) {
			zRRE(opcode, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		} else if p.From.Reg == p.To.Reg {
			zRRE(opcode, uint32(p.To.Reg), uint32(r), asm)
		} else {
			zRR(op_LDR, uint32(p.To.Reg), uint32(r), asm)
			zRRE(opcode, uint32(p.To.Reg), uint32(p.From.Reg), asm)
		}

	case 33: /* fabs [frb,]frd; fmr. frb,frd */
		r := p.From.Reg
		if oclass(&p.From) == C_NONE {
			r = p.To.Reg
		}
		var opcode uint32
		switch p.As {
		default:
		case AFABS:
			opcode = op_LPDBR
		case AFNABS:
			opcode = op_LNDBR
		case AFNEG:
			opcode = op_LCDFR
		case ALEDBR:
			opcode = op_LEDBR
		case ALDEBR:
			opcode = op_LDEBR
		case AFSQRT:
			opcode = op_SQDBR
		case AFSQRTS:
			opcode = op_SQEBR
		}
		zRRE(opcode, uint32(p.To.Reg), uint32(r), asm)

	case 34: /* FMADDx fra,frb,frc,frd (d=a*b+c); FSELx a<0? (d=b): (d=c) */

		var opcode uint32

		switch p.As {
		default:
			ctxt.Diag("invalid opcode")
		case AFMADD:
			opcode = op_MADBR
		case AFMADDS:
			opcode = op_MAEBR
		case AFMSUB:
			opcode = op_MSDBR
		case AFMSUBS:
			opcode = op_MSEBR
		case AFNMADD:
			opcode = op_MADBR
		case AFNMADDS:
			opcode = op_MAEBR
		case AFNMSUB:
			opcode = op_MSDBR
		case AFNMSUBS:
			opcode = op_MSEBR
		}

		zRR(op_LDR, uint32(p.To.Reg), uint32(p.Reg), asm)
		zRRD(opcode, uint32(p.To.Reg), uint32(p.From.Reg), uint32(p.From3.Reg), asm)

		if p.As == AFNMADD || p.As == AFNMADDS || p.As == AFNMSUB || p.As == AFNMSUBS {
			zRRE(op_LCDFR, uint32(p.To.Reg), uint32(p.To.Reg), asm)
		}

	case 35: // MOVE REG TO LAUTO/LOREG
		d2 := regoff(ctxt, &p.To)
		b2 := p.To.Reg
		if b2 == 0 {
			b2 = o.param
		}
		x2 := p.To.Index
		if d2 < -DISP20/2 || d2 >= DISP20/2 {
			zRIL(a, op_LGFI, REGTMP, uint32(d2), asm)
			if x2 != 0 {
				zRX(op_LA, REGTMP, REGTMP, uint32(x2), 0, asm)
			}
			x2 = REGTMP
			d2 = 0
		}
		zRXY(0, zopstore(ctxt, p.As), uint32(p.From.Reg), uint32(x2), uint32(b2), uint32(d2), asm)

	case 36: // MOV LAUTO/LOREG TO REG
		d2 := regoff(ctxt, &p.From)
		b2 := p.From.Reg
		if b2 == 0 {
			b2 = o.param
		}
		x2 := p.From.Index
		if d2 < -DISP20/2 || d2 >= DISP20/2 {
			zRIL(a, op_LGFI, REGTMP, uint32(d2), asm)
			if x2 != 0 {
				zRX(op_LA, REGTMP, REGTMP, uint32(x2), 0, asm)
			}
			x2 = REGTMP
			d2 = 0
		}
		zRXY(0, zopload(ctxt, p.As), uint32(p.To.Reg), uint32(x2), uint32(b2), uint32(d2), asm)

	case 40: /* word and byte*/
		wd := uint32(regoff(ctxt, &p.From))
		if p.As == AWORD { //WORD
			*asm = append(*asm, uint8(wd>>24), uint8(wd>>16), uint8(wd>>8), uint8(wd))
		} else { //BYTE
			*asm = append(*asm, uint8(wd))
		}

	case 47: /* op Ra, Rd; also op [Ra,] Rd */
		switch p.As {
		default:

		case AADDME:
			r := int(p.From.Reg)
			if p.To.Reg == p.From.Reg {
				zRRE(op_LGR, REGTMP, uint32(p.From.Reg), asm)
				r = REGTMP
			}
			zRIL(a, op_LGFI, uint32(p.To.Reg), 0xffffffff, asm) // p.To.Reg <- -1
			zRRE(op_ALCGR, uint32(p.To.Reg), uint32(r), asm)

		case AADDZE:
			r := int(p.From.Reg)
			if p.To.Reg == p.From.Reg {
				zRRE(op_LGR, REGTMP, uint32(p.From.Reg), asm)
				r = REGTMP
			}
			zRRE(op_LGR, uint32(p.To.Reg), REGZERO, asm) // p.To.Reg <- 0
			zRRE(op_ALCGR, uint32(p.To.Reg), uint32(r), asm)

		case ASUBME:
			r := int(p.From.Reg)
			if p.To.Reg == p.From.Reg {
				zRRE(op_LGR, REGTMP, uint32(p.From.Reg), asm)
				r = REGTMP
			}
			zRIL(a, op_LGFI, uint32(p.To.Reg), 0xffffffff, asm) // p.To.Reg <- -1
			zRRE(op_SLBGR, uint32(p.To.Reg), uint32(r), asm)

		case ASUBZE:
			r := int(p.From.Reg)
			if p.To.Reg == p.From.Reg {
				zRRE(op_LGR, REGTMP, uint32(p.From.Reg), asm)
				r = REGTMP
			}
			zRRE(op_LGR, uint32(p.To.Reg), REGZERO, asm) // p.To.Reg <- 0
			zRRE(op_SLBGR, uint32(p.To.Reg), uint32(r), asm)

		case ANEG:
			r := int(p.From.Reg)
			if r == 0 {
				r = int(p.To.Reg)
			}
			zRRE(op_LCGR, uint32(p.To.Reg), uint32(r), asm)
		}

	case 67: // AFMOVx $0, Fy -- move +0 into reg
		var opcode uint32
		switch p.As {
		case AFMOVS:
			opcode = op_LZER
		case AFMOVD:
			opcode = op_LZDR
		}
		zRRE(opcode, uint32(p.To.Reg), 0, asm)

	case 68: /* ear arS,rD */
		zRRE(op_EAR, uint32(p.To.Reg), uint32(p.From.Reg-REG_AR0), asm)

	case 69: /* sar rS,arD */
		zRRE(op_SAR, uint32(p.To.Reg-REG_AR0), uint32(p.From.Reg), asm)

	case 70: /* [f]cmp r,r,cr*/
		if p.Reg != 0 {
			ctxt.Diag("unsupported nozero CC in Z")
		}
		if p.As == ACMPW || p.As == ACMPWU {
			zRR(zoprr(ctxt, p.As), uint32(p.From.Reg), uint32(p.To.Reg), asm)
		} else {
			zRRE(zoprre(ctxt, p.As), uint32(p.From.Reg), uint32(p.To.Reg), asm)
		}

	case 71: // cmp reg $constant
		v := vregoff(ctxt, &p.To)
		switch p.As {
		case ACMP, ACMPW:
			if int64(int32(v)) != v {
				ctxt.Diag("%v overflows an int32", v)
			}
		case ACMPU, ACMPWU:
			if int64(uint32(v)) != v {
				ctxt.Diag("%v overflows a uint32", v)
			}
		}
		zRIL(0, zopril(ctxt, p.As), uint32(p.From.Reg), uint32(regoff(ctxt, &p.To)), asm)

	case 72: // MOV int32 -> s+o(r)(i*1)
		v := regoff(ctxt, &p.From)
		d := regoff(ctxt, &p.To)
		r := p.To.Reg
		x := p.To.Index
		if r == 0 {
			r = o.param
		}
		if p.From.Sym != nil {
			zRIL(b, op_LARL, REGTMP, 0, asm)
			if v&0x1 != 0 {
				v -= 1
				zRX(op_LA, REGTMP, REGTMP, 0, 1, asm)
			}
			addrilreloc(ctxt, p.From.Sym, int64(v))
			if d < -DISP20/2 || d >= DISP20/2 {
				zRIL(a, op_LGFI, REGTMP2, uint32(d), asm)
				if x != 0 {
					zRRE(op_AGR, REGTMP2, uint32(x), asm)
				}
				d = 0
				x = REGTMP2
			}
			zRXY(0, zopstore(ctxt, p.As), REGTMP, uint32(x), uint32(r), uint32(d), asm)
		} else if int32(int16(v)) == v && x == 0 {
			if d < 0 || d >= DISP12 {
				if r == REGTMP || r == REGTMP2 {
					zRIL(a, op_AGFI, uint32(r), uint32(d), asm)
				} else {
					zRIL(a, op_LGFI, REGTMP, uint32(d), asm)
					zRRE(op_AGR, REGTMP, uint32(r), asm)
					r = REGTMP
				}
				d = 0
			}
			var opcode uint32
			switch p.As {
			case AMOVD:
				opcode = op_MVGHI
			case AMOVW, AMOVWZ:
				opcode = op_MVHI
			case AMOVH, AMOVHZ:
				opcode = op_MVHHI
			case AMOVB, AMOVBZ:
				opcode = op_MVI
			}
			if opcode == op_MVI {
				zSI(opcode, uint32(v), uint32(r), uint32(d), asm)
			} else {
				zSIL(opcode, uint32(r), uint32(d), uint32(v), asm)
			}
		} else {
			zRIL(a, op_LGFI, REGTMP2, uint32(v), asm)
			if d < -DISP20/2 || d >= DISP20/2 {
				if r == REGTMP {
					zRIL(a, op_AGFI, REGTMP, uint32(d), asm)
				} else {
					zRIL(a, op_LGFI, REGTMP, uint32(d), asm)
					if x != 0 {
						zRRE(op_AGR, REGTMP, uint32(x), asm)
					}
					x = REGTMP
				}
				d = 0
			}
			zRXY(0, zopstore(ctxt, p.As), REGTMP2, uint32(x), uint32(r), uint32(d), asm)
		}

	case 73: // MOV int32 -> addr
		v := regoff(ctxt, &p.From)
		d := regoff(ctxt, &p.To)
		a := uint32(0)
		if d&1 != 0 {
			d -= 1
			a = 1
		}
		zRIL(b, op_LARL, REGTMP, uint32(d), asm)
		addrilreloc(ctxt, p.To.Sym, int64(d))
		if p.From.Sym != nil {
			zRIL(b, op_LARL, REGTMP2, 0, asm)
			a := uint32(0)
			if v&0x1 != 0 {
				v -= 1
				zRX(op_LA, REGTMP2, REGTMP2, 0, 1, asm)
			}
			addrilrelocoffset(ctxt, p.From.Sym, int64(v), sizeRIL)
			zRXY(0, zopstore(ctxt, p.As), REGTMP2, 0, REGTMP, a, asm)
		} else if int32(int16(v)) == v {
			var opcode uint32
			switch p.As {
			case AMOVD:
				opcode = op_MVGHI
			case AMOVW, AMOVWZ:
				opcode = op_MVHI
			case AMOVH, AMOVHZ:
				opcode = op_MVHHI
			case AMOVB, AMOVBZ:
				opcode = op_MVI
			}
			if opcode == op_MVI {
				zSI(opcode, uint32(v), REGTMP, a, asm)
			} else {
				zSIL(opcode, REGTMP, a, uint32(v), asm)
			}
		} else {
			zRIL(a, op_LGFI, REGTMP2, uint32(v), asm)
			zRXY(0, zopstore(ctxt, p.As), REGTMP2, 0, REGTMP, a, asm)
		}

	case 74: // MOV sym+n(SB) TO REG (requires relocation)
		i2 := regoff(ctxt, &p.To)
		switch p.As {
		case AMOVD:
			zRIL(b, op_STGRL, uint32(p.From.Reg), 0, asm)
		case AMOVW, AMOVWZ: // The zero extension doesn't affect store instructions
			zRIL(b, op_STRL, uint32(p.From.Reg), 0, asm)
		case AMOVH, AMOVHZ: // The zero extension doesn't affect store instructions
			zRIL(b, op_STHRL, uint32(p.From.Reg), 0, asm)
		case AMOVB, AMOVBZ: // The zero extension doesn't affect store instructions
			zRIL(b, op_LARL, REGTMP, 0, asm)
			adj := uint32(0) // adjustment needed for odd addresses
			if i2&1 != 0 {
				i2 -= 1
				adj = 1
			}
			zRX(op_STC, uint32(p.From.Reg), 0, REGTMP, adj, asm)
		case AFMOVD:
			zRIL(b, op_LARL, REGTMP, 0, asm)
			zRX(op_STD, uint32(p.From.Reg), 0, REGTMP, 0, asm)
		case AFMOVS:
			zRIL(b, op_LARL, REGTMP, 0, asm)
			zRX(op_STE, uint32(p.From.Reg), 0, REGTMP, 0, asm)
		}
		addrilreloc(ctxt, p.To.Sym, int64(i2))

	case 75: // MOV REG TO sym+n(SB) (requires relocation)
		i2 := regoff(ctxt, &p.From)
		switch p.As {
		case AMOVD:
			if i2&1 != 0 {
				zRIL(b, op_LARL, REGTMP, 0, asm)
				zRXY(0, op_LG, uint32(p.To.Reg), REGTMP, 0, 1, asm)
				i2 -= 1
			} else {
				zRIL(b, op_LGRL, uint32(p.To.Reg), uint32(d), asm)
			}
		case AMOVW:
			zRIL(b, op_LGFRL, uint32(p.To.Reg), 0, asm)
		case AMOVWZ:
			zRIL(b, op_LLGFRL, uint32(p.To.Reg), 0, asm)
		case AMOVH:
			zRIL(b, op_LGHRL, uint32(p.To.Reg), 0, asm)
		case AMOVHZ:
			zRIL(b, op_LLGHRL, uint32(p.To.Reg), 0, asm)
		case AMOVB, AMOVBZ:
			zRIL(b, op_LARL, REGTMP, 0, asm)
			adj := uint32(0) // adjustment needed for odd addresses
			if i2&1 != 0 {
				i2 -= 1
				adj = 1
			}
			switch p.As {
			case AMOVB:
				zRXY(0, op_LGB, uint32(p.To.Reg), 0, REGTMP, adj, asm)
			case AMOVBZ:
				zRXY(0, op_LLGC, uint32(p.To.Reg), 0, REGTMP, adj, asm)
			}
		case AFMOVD:
			zRIL(a, op_LARL, REGTMP, 0, asm)
			zRX(op_LD, uint32(p.To.Reg), 0, REGTMP, 0, asm)
		case AFMOVS:
			zRIL(a, op_LARL, REGTMP, 0, asm)
			zRX(op_LE, uint32(p.To.Reg), 0, REGTMP, 0, asm)
		}
		addrilreloc(ctxt, p.From.Sym, int64(i2))

	case 77: /* syscall $scon */
		if p.From.Offset > 255 || p.From.Offset < 1 {
			ctxt.Diag("illegal system call; system call number out of range: %v", p)
			zE(op_TRAP2, asm) // trap always
		} else {
			zI(op_SVC, uint32(p.From.Offset), asm)
		}

	case 78: /* undef */
		/* "An instruction consisting entirely of binary 0s is guaranteed
		   always to be an illegal instruction."  */
		*asm = append(*asm, 0, 0, 0, 0)

	case 79: /* cs,csg  r1,r3,off(r2) -> compare & swap; if (r1 ==off(r2)) then off(r2)= r3 */
		v := regoff(ctxt, &p.To)
		if v < 0 {
			v = 0
		}
		if p.As == ACS {
			zRS(op_CS, uint32(p.From.Reg), uint32(p.Reg), uint32(p.To.Reg), uint32(v), asm)
		} else if p.As == ACSG {
			zRSY(op_CSG, uint32(p.From.Reg), uint32(p.Reg), uint32(p.To.Reg), uint32(v), asm)
		}

	case 81: /* SYNC-> BCR 14,0 */
		zRR(op_BCR, 0xE, 0, asm)

	case 82: /* conversion from GPR to FPR */
		var opcode uint32
		switch p.As {
		default:
			log.Fatalf("unexpected opcode %v", p.As)
		case ACEFBRA:
			opcode = op_CEFBRA
		case ACDFBRA:
			opcode = op_CDFBRA
		case ACEGBRA:
			opcode = op_CEGBRA
		case ACDGBRA:
			opcode = op_CDGBRA
		case ACELFBR:
			opcode = op_CELFBR
		case ACDLFBR:
			opcode = op_CDLFBR
		case ACELGBR:
			opcode = op_CELGBR
		case ACDLGBR:
			opcode = op_CDLGBR
		}
		/* set immediate operand M3 to 0 to use the default BFP rounding mode
		   (usually round to nearest, ties to even); M4 is reserved and must be 0 */
		zRRF(opcode, 0, 0, uint32(p.To.Reg), uint32(p.From.Reg), asm)

	case 83: /* conversion from FPR to GPR */
		var opcode uint32
		switch p.As {
		default:
			log.Fatalf("unexpected opcode %v", p.As)
		case ACFEBRA:
			opcode = op_CFEBRA
		case ACFDBRA:
			opcode = op_CFDBRA
		case ACGEBRA:
			opcode = op_CGEBRA
		case ACGDBRA:
			opcode = op_CGDBRA
		case ACLFEBR:
			opcode = op_CLFEBR
		case ACLFDBR:
			opcode = op_CLFDBR
		case ACLGEBR:
			opcode = op_CLGEBR
		case ACLGDBR:
			opcode = op_CLGDBR
		}
		/* set immediate operand M3 to 5 for rounding toward zero (required by Go spec); M4 is reserved and must be 0 */
		zRRF(opcode, 5, 0, uint32(p.To.Reg), uint32(p.From.Reg), asm)

	case 84: /* storage-and-storage operations (mvc, clc, xc, oc, nc) */
		l := regoff(ctxt, p.From3)
		if l < 1 || l > 256 {
			ctxt.Diag("number of bytes (%v) not in range [1,256]", l)
		}
		if p.From.Index != 0 || p.To.Index != 0 {
			ctxt.Diag("cannot use index reg")
		}
		b1 := p.To.Reg
		b2 := p.From.Reg
		if b1 == 0 {
			b1 = o.param
		}
		if b2 == 0 {
			b2 = o.param
		}
		d1 := regoff(ctxt, &p.To)
		d2 := regoff(ctxt, &p.From)
		if d1 < 0 || d1 >= DISP12 {
			if b2 == REGTMP {
				ctxt.Diag("REGTMP conflict")
			}
			if b1 != REGTMP {
				zRRE(op_LGR, REGTMP, uint32(b1), asm)
			}
			zRIL(a, op_AGFI, REGTMP, uint32(d1), asm)
			if d1 == d2 && b1 == b2 {
				d2 = 0
				b2 = REGTMP
			}
			d1 = 0
			b1 = REGTMP
		}
		if d2 < 0 || d2 >= DISP12 {
			if b1 == REGTMP2 {
				ctxt.Diag("REGTMP2 conflict")
			}
			if b2 != REGTMP2 {
				zRRE(op_LGR, REGTMP2, uint32(b2), asm)
			}
			zRIL(a, op_AGFI, REGTMP2, uint32(d2), asm)
			d2 = 0
			b2 = REGTMP2
		}
		var opcode uint32
		switch p.As {
		default:
			ctxt.Diag("unexpected opcode %v", p.As)
		case AMVC:
			opcode = op_MVC
		case ACLC:
			opcode = op_CLC
			// swap operand order for CLC so that it matches CMP
			b1, b2 = b2, b1
			d1, d2 = d2, d1
		case AXC:
			opcode = op_XC
		case AOC:
			opcode = op_OC
		case ANC:
			opcode = op_NC
		}
		zSS(a, opcode, uint32(l-1), 0, uint32(b1), uint32(d1), uint32(b2), uint32(d2), asm)

	case 85: /* larl: load address relative long */
		// When using larl directly, don't add a nop
		v := regoff(ctxt, &p.From)
		if p.From.Sym == nil {
			if (v & 1) != 0 {
				ctxt.Diag("cannot use LARL with odd offset: %v", v)
			}
		} else {
			addrilreloc(ctxt, p.From.Sym, int64(v))
			v = 0
		}
		zRIL(b, op_LARL, uint32(p.To.Reg), uint32(v>>1), asm)

	case 86: /* lay?: load address */
		d := vregoff(ctxt, &p.From)
		x := p.From.Index
		b := p.From.Reg
		if b == 0 {
			b = o.param
		}
		switch p.As {
		case ALA:
			zRX(op_LA, uint32(p.To.Reg), uint32(x), uint32(b), uint32(d), asm)
		case ALAY:
			zRXY(0, op_LAY, uint32(p.To.Reg), uint32(x), uint32(b), uint32(d), asm)
		}

	case 87: /* exrl: execute relative long */
		v := vregoff(ctxt, &p.From)
		if p.From.Sym == nil {
			if v&1 != 0 {
				ctxt.Diag("cannot use EXRL with odd offset: %v", v)
			}
		} else {
			addrilreloc(ctxt, p.From.Sym, v)
			v = 0
		}
		zRIL(b, op_EXRL, uint32(p.To.Reg), uint32(v>>1), asm)

	case 88: /* stck[cef]?: store clock (comparator/extended/fast) */
		var opcode uint32
		switch p.As {
		case ASTCK:
			opcode = op_STCK
		case ASTCKC:
			opcode = op_STCKC
		case ASTCKE:
			opcode = op_STCKE
		case ASTCKF:
			opcode = op_STCKF
		}
		v := vregoff(ctxt, &p.To)
		r := int(p.To.Reg)
		if r == 0 {
			r = int(o.param)
		}
		zS(opcode, uint32(r), uint32(v), asm)

	case 89:
		var v int32
		if p.Pcond != nil {
			v = int32((p.Pcond.Pc - p.Pc) >> 1)
		}
		var opcode, opcode2 uint32
		switch p.As {
		case ACMPBEQ, ACMPBGE, ACMPBGT, ACMPBLE, ACMPBLT, ACMPBNE:
			opcode = op_CGRJ
			opcode2 = op_CGR
		case ACMPUBEQ, ACMPUBGE, ACMPUBGT, ACMPUBLE, ACMPUBLT, ACMPUBNE:
			opcode = op_CLGRJ
			opcode2 = op_CLGR
		}
		mask := branchMask(ctxt, p)
		if int32(int16(v)) != v {
			zRRE(opcode2, uint32(p.From.Reg), uint32(p.Reg), asm)
			zRIL(c, op_BRCL, mask, uint32(v-sizeRRE/2), asm)
		} else {
			zRIE(b, opcode, uint32(p.From.Reg), uint32(p.Reg), uint32(v), 0, 0, mask, 0, asm)
		}

	case 90:
		var v int32
		if p.Pcond != nil {
			v = int32((p.Pcond.Pc - p.Pc) >> 1)
		}
		var opcode, opcode2 uint32
		switch p.As {
		case ACMPBEQ, ACMPBGE, ACMPBGT, ACMPBLE, ACMPBLT, ACMPBNE:
			opcode = op_CGIJ
			opcode2 = op_CGFI
		case ACMPUBEQ, ACMPUBGE, ACMPUBGT, ACMPUBLE, ACMPUBLT, ACMPUBNE:
			opcode = op_CLGIJ
			opcode2 = op_CLGFI
		}
		mask := branchMask(ctxt, p)
		if int32(int16(v)) != v {
			zRIL(0, opcode2, uint32(p.From.Reg), uint32(regoff(ctxt, p.From3)), asm)
			zRIL(c, op_BRCL, mask, uint32(v-sizeRIL/2), asm)
		} else {
			zRIE(c, opcode, uint32(p.From.Reg), mask, uint32(v), 0, 0, 0, uint32(regoff(ctxt, p.From3)), asm)
		}

	case 93: // GOT lookup
		v := vregoff(ctxt, &p.To)
		if v != 0 {
			ctxt.Diag("invalid offset against GOT slot %v", p)
		}
		zRIL(b, op_LGRL, uint32(p.To.Reg), 0, asm)
		rel := obj.Addrel(ctxt.Cursym)
		rel.Off = int32(ctxt.Pc + 2)
		rel.Siz = 4
		rel.Sym = p.From.Sym
		rel.Type = obj.R_GOTPCREL
		rel.Add = 2 + int64(rel.Siz)

	case 94: // TLS local exec model
		zRIL(b, op_LARL, REGTMP, (sizeRIL+sizeRXY+sizeRI)>>1, asm)
		zRXY(0, op_LG, uint32(p.To.Reg), REGTMP, 0, 0, asm)
		zRI(op_BRC, 0xF, (sizeRI+8)>>1, asm)
		*asm = append(*asm, 0, 0, 0, 0, 0, 0, 0, 0)
		rel := obj.Addrel(ctxt.Cursym)
		rel.Off = int32(ctxt.Pc + sizeRIL + sizeRXY + sizeRI)
		rel.Siz = 8
		rel.Sym = p.From.Sym
		rel.Type = obj.R_TLS_LE
		rel.Add = 0

	case 95: // TLS initial exec model
		// Assembly                   | Relocation symbol    | Done Here?
		// --------------------------------------------------------------
		// ear  %r11, %a0             |                      |
		// sllg %r11, %r11, 32        |                      |
		// ear  %r11, %a1             |                      |
		// larl %r10, <var>@indntpoff | R_390_TLS_IEENT      | Y
		// lg   %r10, 0(%r10)         | R_390_TLS_LOAD (tag) | Y
		// la   %r10, 0(%r10, %r11)   |                      |
		// --------------------------------------------------------------

		// R_390_TLS_IEENT
		zRIL(b, op_LARL, REGTMP, 0, asm)
		ieent := obj.Addrel(ctxt.Cursym)
		ieent.Off = int32(ctxt.Pc + 2)
		ieent.Siz = 4
		ieent.Sym = p.From.Sym
		ieent.Type = obj.R_TLS_IE
		ieent.Add = 2 + int64(ieent.Siz)

		// R_390_TLS_LOAD
		zRXY(0, op_LGF, uint32(p.To.Reg), REGTMP, 0, 0, asm)
		// TODO(mundaym): add R_390_TLS_LOAD relocation here
		// not strictly required but might allow the linker to optimize

	case 96: // CLEAR macro
		length := vregoff(ctxt, &p.From)
		offset := vregoff(ctxt, &p.To)
		reg := p.To.Reg
		if reg == 0 {
			reg = o.param
		}
		if length <= 0 {
			ctxt.Diag("cannot CLEAR %d bytes, must be greater than 0", length)
		}
		for length > 0 {
			if offset < 0 || offset >= DISP12 {
				if offset >= -DISP20/2 && offset < DISP20/2 {
					zRXY(0, op_LAY, REGTMP, uint32(reg), 0, uint32(offset), asm)
				} else {
					if reg != REGTMP {
						zRRE(op_LGR, REGTMP, uint32(reg), asm)
					}
					zRIL(a, op_AGFI, REGTMP, uint32(offset), asm)
				}
				reg = REGTMP
				offset = 0
			}
			size := length
			if size > 256 {
				size = 256
			}

			switch size {
			case 1:
				zSI(op_MVI, 0, uint32(reg), uint32(offset), asm)
			case 2:
				zSIL(op_MVHHI, uint32(reg), uint32(offset), 0, asm)
			case 4:
				zSIL(op_MVHI, uint32(reg), uint32(offset), 0, asm)
			case 8:
				zSIL(op_MVGHI, uint32(reg), uint32(offset), 0, asm)
			default:
				zSS(a, op_XC, uint32(size-1), 0, uint32(reg), uint32(offset), uint32(reg), uint32(offset), asm)
			}

			length -= size
			offset += size
		}

	case 97: // STORE MULTIPLE (STMG/STMY)
		rstart := p.From.Reg
		rend := p.Reg
		offset := regoff(ctxt, &p.To)
		reg := p.To.Reg
		if reg == 0 {
			reg = o.param
		}
		if offset < -DISP20/2 || offset >= DISP20/2 {
			if reg != REGTMP {
				zRRE(op_LGR, REGTMP, uint32(reg), asm)
			}
			zRIL(a, op_AGFI, REGTMP, uint32(offset), asm)
			reg = REGTMP
			offset = 0
		}
		switch p.As {
		case ASTMY:
			if offset >= 0 && offset < DISP12 {
				zRS(op_STM, uint32(rstart), uint32(rend), uint32(reg), uint32(offset), asm)
			} else {
				zRSY(op_STMY, uint32(rstart), uint32(rend), uint32(reg), uint32(offset), asm)
			}
		case ASTMG:
			zRSY(op_STMG, uint32(rstart), uint32(rend), uint32(reg), uint32(offset), asm)
		}

	case 98: // LOAD MULTIPLE (LMG/LMY)
		rstart := p.Reg
		rend := p.To.Reg
		offset := regoff(ctxt, &p.From)
		reg := p.From.Reg
		if reg == 0 {
			reg = o.param
		}
		if offset < -DISP20/2 || offset >= DISP20/2 {
			if reg != REGTMP {
				zRRE(op_LGR, REGTMP, uint32(reg), asm)
			}
			zRIL(a, op_AGFI, REGTMP, uint32(offset), asm)
			reg = REGTMP
			offset = 0
		}
		switch p.As {
		case ALMY:
			if offset >= 0 && offset < DISP12 {
				zRS(op_LM, uint32(rstart), uint32(rend), uint32(reg), uint32(offset), asm)
			} else {
				zRSY(op_LMY, uint32(rstart), uint32(rend), uint32(reg), uint32(offset), asm)
			}
		case ALMG:
			zRSY(op_LMG, uint32(rstart), uint32(rend), uint32(reg), uint32(offset), asm)
		}

	case 100: // VRX STORE
		op, m3, _ := vop(p.As)
		if p.From3 != nil {
			m3 = uint32(vregoff(ctxt, p.From3))
		}
		b2 := p.To.Reg
		if b2 == 0 {
			b2 = o.param
		}
		d2 := uint32(vregoff(ctxt, &p.To))
		zVRX(op, uint32(p.From.Reg), uint32(p.To.Index), uint32(b2), d2, m3, asm)

	case 101: // VRX LOAD
		op, m3, _ := vop(p.As)
		if p.From3 != nil {
			m3 = uint32(vregoff(ctxt, p.From3))
		}
		b2 := p.From.Reg
		if b2 == 0 {
			b2 = o.param
		}
		d2 := uint32(vregoff(ctxt, &p.From))
		zVRX(op, uint32(p.To.Reg), uint32(p.From.Index), uint32(b2), d2, m3, asm)

	case 102: // VRV SCATTER
		op, m3, _ := vop(p.As)
		if p.From3 != nil {
			m3 = uint32(vregoff(ctxt, p.From3))
		}
		b2 := p.To.Reg
		if b2 == 0 {
			b2 = o.param
		}
		d2 := uint32(vregoff(ctxt, &p.To))
		zVRV(op, uint32(p.From.Reg), uint32(p.To.Index), uint32(b2), d2, m3, asm)

	case 103: // VRV GATHER
		op, m3, _ := vop(p.As)
		if p.From3 != nil {
			m3 = uint32(vregoff(ctxt, p.From3))
		}
		b2 := p.From.Reg
		if b2 == 0 {
			b2 = o.param
		}
		d2 := uint32(vregoff(ctxt, &p.From))
		zVRV(op, uint32(p.To.Reg), uint32(p.From.Index), uint32(b2), d2, m3, asm)

	case 104: // VRS SHIFT/ROTATE and LOAD GR FROM VR ELEMENT
		op, m4, _ := vop(p.As)
		fr := p.Reg
		if fr == 0 {
			fr = p.To.Reg
		}
		bits := uint32(vregoff(ctxt, &p.From))
		zVRS(op, uint32(p.To.Reg), uint32(fr), uint32(p.From.Reg), bits, m4, asm)

	case 105: // VRS STORE MULTIPLE
		op, _, _ := vop(p.As)
		offset := uint32(vregoff(ctxt, &p.To))
		reg := p.To.Reg
		if reg == 0 {
			reg = o.param
		}
		zVRS(op, uint32(p.From.Reg), uint32(p.Reg), uint32(reg), offset, 0, asm)

	case 106: // VRS LOAD MULTIPLE
		op, _, _ := vop(p.As)
		offset := uint32(vregoff(ctxt, &p.From))
		reg := p.From.Reg
		if reg == 0 {
			reg = o.param
		}
		zVRS(op, uint32(p.Reg), uint32(p.To.Reg), uint32(reg), offset, 0, asm)

	case 107: // VRS STORE WITH LENGTH
		op, _, _ := vop(p.As)
		offset := uint32(vregoff(ctxt, &p.To))
		reg := p.To.Reg
		if reg == 0 {
			reg = o.param
		}
		zVRS(op, uint32(p.From.Reg), uint32(p.From3.Reg), uint32(reg), offset, 0, asm)

	case 108: // VRS LOAD WITH LENGTH
		op, _, _ := vop(p.As)
		offset := uint32(vregoff(ctxt, &p.From))
		reg := p.From.Reg
		if reg == 0 {
			reg = o.param
		}
		zVRS(op, uint32(p.To.Reg), uint32(p.From3.Reg), uint32(reg), offset, 0, asm)

	case 109: // VRI-a instructions
		op, _, _ := vop(p.As)
		i2 := uint32(vregoff(ctxt, &p.From))
		switch p.As {
		case AVZERO:
			i2 = 0
		case AVONE:
			i2 = 0xffff
		}
		m3 := uint32(0)
		if p.From3 != nil {
			m3 = uint32(vregoff(ctxt, p.From3))
		}
		zVRIa(op, uint32(p.To.Reg), i2, m3, asm)

	case 110:
		op, m4, _ := vop(p.As)
		i2 := uint32(vregoff(ctxt, p.From3))
		i3 := uint32(vregoff(ctxt, &p.From))
		zVRIb(op, uint32(p.To.Reg), i2, i3, m4, asm)

	case 111:
		op, m4, _ := vop(p.As)
		i2 := uint32(vregoff(ctxt, &p.From))
		zVRIc(op, uint32(p.To.Reg), uint32(p.Reg), i2, m4, asm)

	case 112:
		op, m5, _ := vop(p.As)
		i4 := uint32(vregoff(ctxt, p.From3))
		zVRId(op, uint32(p.To.Reg), uint32(p.From.Reg), uint32(p.Reg), i4, m5, asm)

	case 113:
		op, m4, _ := vop(p.As)
		m5 := singleElementMask(p.As)
		i3 := uint32(vregoff(ctxt, &p.From))
		zVRIe(op, uint32(p.To.Reg), uint32(p.Reg), i3, m5, m4, asm)

	case 114: // VRR-a
		op, m3, m5 := vop(p.As)
		m4 := singleElementMask(p.As)
		zVRRa(op, uint32(p.To.Reg), uint32(p.From.Reg), m5, m4, m3, asm)

	case 115: // VRR-a COMPARE
		op, m3, m5 := vop(p.As)
		m4 := singleElementMask(p.As)
		zVRRa(op, uint32(p.From.Reg), uint32(p.To.Reg), m5, m4, m3, asm)

	case 116: // VRR-a

	case 117: // VRR-b
		op, m4, m5 := vop(p.As)
		zVRRb(op, uint32(p.To.Reg), uint32(p.From.Reg), uint32(p.Reg), m5, m4, asm)

	case 118: // VRR-c
		op, m4, m6 := vop(p.As)
		m5 := singleElementMask(p.As)
		v3 := p.Reg
		if v3 == 0 {
			v3 = p.To.Reg
		}
		zVRRc(op, uint32(p.To.Reg), uint32(p.From.Reg), uint32(v3), m6, m5, m4, asm)

	case 119: // VRR-c SHIFT/ROTATE/DIVIDE/SUB (rhs value on the left, like SLD, DIV etc.)
		op, m4, m6 := vop(p.As)
		m5 := singleElementMask(p.As)
		v2 := p.Reg
		if v2 == 0 {
			v2 = p.To.Reg
		}
		zVRRc(op, uint32(p.To.Reg), uint32(v2), uint32(p.From.Reg), m6, m5, m4, asm)

	case 120: // VRR-d
		op, m6, _ := vop(p.As)
		m5 := singleElementMask(p.As)
		v1 := uint32(p.To.Reg)
		v2 := uint32(p.From3.Reg)
		v3 := uint32(p.From.Reg)
		v4 := uint32(p.Reg)
		zVRRd(op, v1, v2, v3, m6, m5, v4, asm)

	case 121: // VRR-e
		op, m6, _ := vop(p.As)
		m5 := singleElementMask(p.As)
		v1 := uint32(p.To.Reg)
		v2 := uint32(p.From3.Reg)
		v3 := uint32(p.From.Reg)
		v4 := uint32(p.Reg)
		zVRRe(op, v1, v2, v3, m5, m6, v4, asm)

	case 122: // VRR-f LOAD VRS FROM GRS DISJOINT
		op, _, _ := vop(p.As)
		zVRRf(op, uint32(p.To.Reg), uint32(p.From.Reg), uint32(p.Reg), asm)

	case 123: // VPDI $m4, V2, V3, V1
		op, _, _ := vop(p.As)
		m4 := regoff(ctxt, p.From3)
		zVRRc(op, uint32(p.To.Reg), uint32(p.From.Reg), uint32(p.Reg), 0, 0, uint32(m4), asm)
	}
}

func vregoff(ctxt *obj.Link, a *obj.Addr) int64 {
	ctxt.Instoffset = 0
	if a != nil {
		aclass(ctxt, a)
	}
	return ctxt.Instoffset
}

func regoff(ctxt *obj.Link, a *obj.Addr) int32 {
	return int32(vregoff(ctxt, a))
}

/*
 * load o(a), d
 */
func zopload(ctxt *obj.Link, a int16) uint32 {
	switch a {
	/* fixed point load */
	case AMOVD:
		return op_LG
	case AMOVW:
		return op_LGF
	case AMOVWZ:
		return op_LLGF
	case AMOVH:
		return op_LGH
	case AMOVHZ:
		return op_LLGH
	case AMOVB:
		return op_LGB
	case AMOVBZ:
		return op_LLGC

	/* floating point load */
	case AFMOVD:
		return op_LDY
	case AFMOVS:
		return op_LEY

	/* byte reversed load*/
	case AMOVDBR:
		return op_LRVG
	case AMOVWBR:
		return op_LRV
	case AMOVHBR:
		return op_LRVH
	}

	ctxt.Diag("unknown store opcode %v", obj.Aconv(int(a)))
	return 0
}

/*
 * store s,o(d)
 */
func zopstore(ctxt *obj.Link, a int16) uint32 {
	switch a {
	/* fixed point store */
	case AMOVD:
		return op_STG
	case AMOVW, AMOVWZ:
		return op_STY
	case AMOVH, AMOVHZ:
		return op_STHY
	case AMOVB, AMOVBZ:
		return op_STCY

	/* floating point store */
	case AFMOVD:
		return op_STDY
	case AFMOVS:
		return op_STEY

	/* byte reversed store */
	case AMOVDBR:
		return op_STRVG
	case AMOVWBR:
		return op_STRV
	case AMOVHBR:
		return op_STRVH
	}

	ctxt.Diag("unknown store opcode %v", obj.Aconv(int(a)))
	return 0
}

func zoprre(ctxt *obj.Link, a int16) uint32 {
	switch a {
	case ACMP:
		return op_CGR
	case ACMPU:
		return op_CLGR
	case AFCMPO: //ordered
		return op_KDBR
	case AFCMPU: //unordered
		return op_CDBR
	case ACEBR:
		return op_CEBR
	}
	ctxt.Diag("unknown rre opcode %v", obj.Aconv(int(a)))
	return 0
}

func zoprr(ctxt *obj.Link, a int16) uint32 {
	switch a {
	case ACMPW:
		return op_CR
	case ACMPWU:
		return op_CLR
	}
	ctxt.Diag("unknown rr opcode %v", obj.Aconv(int(a)))
	return 0
}

func zopril(ctxt *obj.Link, a int16) uint32 {
	switch a {
	case ACMP:
		return op_CGFI
	case ACMPU:
		return op_CLGFI
	case ACMPW:
		return op_CFI
	case ACMPWU:
		return op_CLFI
	}
	ctxt.Diag("unknown ril opcode %v", obj.Aconv(int(a)))
	return 0
}

// z instructions sizes.
const (
	sizeE    = 2
	sizeI    = 2
	sizeIE   = 4
	sizeMII  = 6
	sizeRI   = 4
	sizeRI1  = 4
	sizeRI2  = 4
	sizeRI3  = 4
	sizeRIE  = 6
	sizeRIE1 = 6
	sizeRIE2 = 6
	sizeRIE3 = 6
	sizeRIE4 = 6
	sizeRIE5 = 6
	sizeRIE6 = 6
	sizeRIL  = 6
	sizeRIL1 = 6
	sizeRIL2 = 6
	sizeRIL3 = 6
	sizeRIS  = 6
	sizeRR   = 2
	sizeRRD  = 4
	sizeRRE  = 4
	sizeRRF  = 4
	sizeRRF1 = 4
	sizeRRF2 = 4
	sizeRRF3 = 4
	sizeRRF4 = 4
	sizeRRF5 = 4
	sizeRRR  = 2
	sizeRRS  = 6
	sizeRS   = 4
	sizeRS1  = 4
	sizeRS2  = 4
	sizeRSI  = 4
	sizeRSL  = 6
	sizeRSY  = 6
	sizeRSY1 = 6
	sizeRSY2 = 6
	sizeRX   = 4
	sizeRX1  = 4
	sizeRX2  = 4
	sizeRXE  = 6
	sizeRXF  = 6
	sizeRXY  = 6
	sizeRXY1 = 6
	sizeRXY2 = 6
	sizeS    = 4
	sizeSI   = 4
	sizeSIL  = 6
	sizeSIY  = 6
	sizeSMI  = 6
	sizeSS   = 6
	sizeSS1  = 6
	sizeSS2  = 6
	sizeSS3  = 6
	sizeSS4  = 6
	sizeSS5  = 6
	sizeSS6  = 6
	sizeSSE  = 6
	sizeSSF  = 6
)

// instruction format variations.
const (
	a = iota
	b
	c
	d
	e
	f
	g
)

func zE(op uint32, asm *[]byte) {
	*asm = append(*asm, uint8(op>>8), uint8(op))
}

func zI(op, i1 uint32, asm *[]byte) {
	*asm = append(*asm, uint8(op>>8), uint8(i1))
}

func zMII(op, m1, ri2, ri3 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(m1)<<4)|uint8((ri2>>8)&0x0F),
		uint8(ri2),
		uint8(ri3>>16),
		uint8(ri3>>8),
		uint8(ri3))
}

func zRI(op, r1_m1, i2_ri2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1_m1)<<4)|(uint8(op)&0x0F),
		uint8(i2_ri2>>8),
		uint8(i2_ri2))
}

// Expected argument values for the instruction formats.
//
// Format     a1    a2     a3     a4     a5  a6  a7
// ------------------------------------
// a         r1,  0,  i2,  0,  0, m3,  0
// b         r1, r2, ri4,  0,  0, m3,  0
// c         r1, m3, ri4,  0,  0,  0, i2
// d         r1, r3,  i2,  0,  0,  0,  0
// e         r1, r3, ri2,  0,  0,  0,  0
// f         r1, r2,   0, i3, i4,  0, i5
// g         r1, m3,  i2,  0,  0,  0,  0
func zRIE(type_, op, r1, r2_m3_r3, i2_ri4_ri2, i3, i4, m3, i2_i5 uint32, asm *[]byte) {
	*asm = append(*asm, uint8(op>>8), uint8(r1)<<4|uint8(r2_m3_r3&0x0F))

	switch type_ {
	default:
		*asm = append(*asm, uint8(i2_ri4_ri2>>8), uint8(i2_ri4_ri2))
	case f:
		*asm = append(*asm, uint8(i3), uint8(i4))
	}

	switch type_ {
	case a, b:
		*asm = append(*asm, uint8(m3)<<4)
	default:
		*asm = append(*asm, uint8(i2_i5))
	}

	*asm = append(*asm, uint8(op))
}

func zRIL(type_, op, r1_m1, i2_ri2 uint32, asm *[]byte) {
	if type_ == a || type_ == b {
		r1_m1 = r1_m1 - obj.RBaseS390X // this is a register base
	}
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1_m1)<<4)|(uint8(op)&0x0F),
		uint8(i2_ri2>>24),
		uint8(i2_ri2>>16),
		uint8(i2_ri2>>8),
		uint8(i2_ri2))
}

func zRIS(op, r1, m3, b4, d4, i2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1)<<4)|uint8(m3&0x0F),
		(uint8(b4)<<4)|(uint8(d4>>8)&0x0F),
		uint8(d4),
		uint8(i2),
		uint8(op))
}

func zRR(op, r1, r2 uint32, asm *[]byte) {
	*asm = append(*asm, uint8(op>>8), (uint8(r1)<<4)|uint8(r2&0x0F))
}

func zRRD(op, r1, r3, r2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(op),
		uint8(r1)<<4,
		(uint8(r3)<<4)|uint8(r2&0x0F))
}

func zRRE(op, r1, r2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(op),
		0,
		(uint8(r1)<<4)|uint8(r2&0x0F))
}

func zRRF(op, r3_m3, m4, r1, r2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(op),
		(uint8(r3_m3)<<4)|uint8(m4&0x0F),
		(uint8(r1)<<4)|uint8(r2&0x0F))
}

func zRRS(op, r1, r2, b4, d4, m3 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1)<<4)|uint8(r2&0x0F),
		(uint8(b4)<<4)|uint8((d4>>8)&0x0F),
		uint8(d4),
		uint8(m3)<<4,
		uint8(op))
}

func zRS(op, r1, r3_m3, b2, d2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1)<<4)|uint8(r3_m3&0x0F),
		(uint8(b2)<<4)|uint8((d2>>8)&0x0F),
		uint8(d2))
}

func zRSI(op, r1, r3, ri2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1)<<4)|uint8(r3&0x0F),
		uint8(ri2>>8),
		uint8(ri2))
}

func zRSL(type_, op, l1, b2, d2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(l1),
		(uint8(b2)<<4)|uint8((d2>>8)&0x0F),
		uint8(d2),
		uint8(op))
}

// (20b) d2 with (12b) dl2 and (8b) dh2.
func zRSY(op, r1, r3_m3, b2, d2 uint32, asm *[]byte) {
	dl2 := uint16(d2) & 0x0FFF
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1)<<4)|uint8(r3_m3&0x0F),
		(uint8(b2)<<4)|(uint8(dl2>>8)&0x0F),
		uint8(dl2),
		uint8(d2>>12),
		uint8(op))
}

func zRX(op, r1_m1, x2, b2, d2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1_m1)<<4)|uint8(x2&0x0F),
		(uint8(b2)<<4)|uint8((d2>>8)&0x0F),
		uint8(d2))
}

func zRXE(op, r1, x2, b2, d2, m3 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1)<<4)|uint8(x2&0x0F),
		(uint8(b2)<<4)|uint8((d2>>8)&0x0F),
		uint8(d2),
		uint8(m3)<<4,
		uint8(op))
}

func zRXF(op, r3, x2, b2, d2, m1 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r3)<<4)|uint8(x2&0x0F),
		(uint8(b2)<<4)|uint8((d2>>8)&0x0F),
		uint8(d2),
		uint8(m1)<<4,
		uint8(op))
}

func zRXY(type_, op, r1_m1, x2, b2, d2 uint32, asm *[]byte) {
	dl2 := uint16(d2) & 0x0FFF
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r1_m1)<<4)|uint8(x2&0x0F),
		(uint8(b2)<<4)|(uint8(dl2>>8)&0x0F),
		uint8(dl2),
		uint8(d2>>12),
		uint8(op))
}

func zS(op, b2, d2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(op),
		(uint8(b2)<<4)|uint8((d2>>8)&0x0F),
		uint8(d2))
}

func zSI(op, i2, b1, d1 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(i2),
		(uint8(b1)<<4)|uint8((d1>>8)&0x0F),
		uint8(d1))
}

func zSIL(op, b1, d1, i2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(op),
		(uint8(b1)<<4)|uint8((d1>>8)&0x0F),
		uint8(d1),
		uint8(i2>>8),
		uint8(i2))
}

func zSIY(op, i2, b1, d1 uint32, asm *[]byte) {
	dl1 := uint16(d1) & 0x0FFF
	*asm = append(*asm,
		uint8(op>>8),
		uint8(i2),
		(uint8(b1)<<4)|(uint8(dl1>>8)&0x0F),
		uint8(dl1),
		uint8(d1>>12),
		uint8(op))
}

func zSMI(op, m1, b3, d3, ri2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(m1)<<4,
		(uint8(b3)<<4)|uint8((d3>>8)&0x0F),
		uint8(d3),
		uint8(ri2>>8),
		uint8(ri2))
}

// Expected argument values for the instruction formats.
//
// Format    a1  a2  a3  a4  a5  a6
// -------------------------------
// a         l1,  0, b1, d1, b2, d2
// b         l1, l2, b1, d1, b2, d2
// c         l1, i3, b1, d1, b2, d2
// d         r1, r3, b1, d1, b2, d2
// e         r1, r3, b2, d2, b4, d4
// f          0, l2, b1, d1, b2, d2
func zSS(type_, op, l1_r1, l2_i3_r3, b1_b2, d1_d2, b2_b4, d2_d4 uint32, asm *[]byte) {
	*asm = append(*asm, uint8(op>>8))

	switch type_ {
	case a:
		*asm = append(*asm, uint8(l1_r1))
	case b, c, d, e:
		*asm = append(*asm, (uint8(l1_r1)<<4)|uint8(l2_i3_r3&0x0F))
	case f:
		*asm = append(*asm, uint8(l2_i3_r3))
	}

	*asm = append(*asm,
		(uint8(b1_b2)<<4)|uint8((d1_d2>>8)&0x0F),
		uint8(d1_d2),
		(uint8(b2_b4)<<4)|uint8((d2_d4>>8)&0x0F),
		uint8(d2_d4))
}

func zSSE(op, b1, d1, b2, d2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(op),
		(uint8(b1)<<4)|uint8((d1>>8)&0x0F),
		uint8(d1),
		(uint8(b2)<<4)|uint8((d2>>8)&0x0F),
		uint8(d2))
}

func zSSF(op, r3, b1, d1, b2, d2 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(r3)<<4)|(uint8(op)&0x0F),
		(uint8(b1)<<4)|uint8((d1>>8)&0x0F),
		uint8(d1),
		(uint8(b2)<<4)|uint8((d2>>8)&0x0F),
		uint8(d2))
}

func rxb(va, vb, vc, vd uint32) uint8 {
	mask := uint8(0)
	if va >= REG_V16 && va <= REG_V31 {
		mask |= 0x8
	}
	if vb >= REG_V16 && vb <= REG_V31 {
		mask |= 0x4
	}
	if vc >= REG_V16 && vc <= REG_V31 {
		mask |= 0x2
	}
	if vd >= REG_V16 && vd <= REG_V31 {
		mask |= 0x1
	}
	return mask
}

func zVRX(op, v1, x2, b2, d2, m3 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(x2)&0xf),
		(uint8(b2)<<4)|(uint8(d2>>8)&0xf),
		uint8(d2),
		(uint8(m3)<<4)|rxb(v1, 0, 0, 0),
		uint8(op))
}

func zVRV(op, v1, v2, b2, d2, m3 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v2)&0xf),
		(uint8(b2)<<4)|(uint8(d2>>8)&0xf),
		uint8(d2),
		(uint8(m3)<<4)|rxb(v1, v2, 0, 0),
		uint8(op))
}

func zVRS(op, v1, v3_r3, b2, d2, m4 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v3_r3)&0xf),
		(uint8(b2)<<4)|(uint8(d2>>8)&0xf),
		uint8(d2),
		(uint8(m4)<<4)|rxb(v1, v3_r3, 0, 0),
		uint8(op))
}

func zVRRa(op, v1, v2, m5, m4, m3 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v2)&0xf),
		0,
		(uint8(m5)<<4)|(uint8(m4)&0xf),
		(uint8(m3)<<4)|rxb(v1, v2, 0, 0),
		uint8(op))
}

func zVRRb(op, v1, v2, v3, m5, m4 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v2)&0xf),
		uint8(v3)<<4,
		uint8(m5)<<4,
		(uint8(m4)<<4)|rxb(v1, v2, v3, 0),
		uint8(op))
}

func zVRRc(op, v1, v2, v3, m6, m5, m4 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v2)&0xf),
		uint8(v3)<<4,
		(uint8(m6)<<4)|(uint8(m5)&0xf),
		(uint8(m4)<<4)|rxb(v1, v2, v3, 0),
		uint8(op))
}

func zVRRd(op, v1, v2, v3, m5, m6, v4 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v2)&0xf),
		(uint8(v3)<<4)|(uint8(m5)&0xf),
		uint8(m6)<<4,
		(uint8(v4)<<4)|rxb(v1, v2, v3, v4),
		uint8(op))
}

func zVRRe(op, v1, v2, v3, m6, m5, v4 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v2)&0xf),
		(uint8(v3)<<4)|(uint8(m6)&0xf),
		uint8(m5),
		(uint8(v4)<<4)|rxb(v1, v2, v3, v4),
		uint8(op))
}

func zVRRf(op, v1, r2, r3 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(r2)&0xf),
		uint8(r3)<<4,
		0,
		rxb(v1, 0, 0, 0),
		uint8(op))
}

func zVRIa(op, v1, i2, m3 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(v1)<<4,
		uint8(i2>>8),
		uint8(i2),
		(uint8(m3)<<4)|rxb(v1, 0, 0, 0),
		uint8(op))
}

func zVRIb(op, v1, i2, i3, m4 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		uint8(v1)<<4,
		uint8(i2),
		uint8(i3),
		(uint8(m4)<<4)|rxb(v1, 0, 0, 0),
		uint8(op))
}

func zVRIc(op, v1, v3, i2, m4 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v3)&0xf),
		uint8(i2>>8),
		uint8(i2),
		(uint8(m4)<<4)|rxb(v1, v3, 0, 0),
		uint8(op))
}

func zVRId(op, v1, v2, v3, i4, m5 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v2)&0xf),
		uint8(v3)<<4,
		uint8(i4),
		(uint8(m5)<<4)|rxb(v1, v2, v3, 0),
		uint8(op))
}

func zVRIe(op, v1, v2, i3, m5, m4 uint32, asm *[]byte) {
	*asm = append(*asm,
		uint8(op>>8),
		(uint8(v1)<<4)|(uint8(v2)&0xf),
		uint8(i3>>4),
		(uint8(i3)<<4)|(uint8(m5)&0xf),
		(uint8(m4)<<4)|rxb(v1, v2, 0, 0),
		uint8(op))
}
