* Copyright 2016 The Go Authors.  All rights reserved.
* Use of this source code is governed by a BSD-style
* license that can be found in the LICENSE file.
*
* void crosscall_s390x(void (*fn)(void), void *g)
*
* Calling into the go tool chain, where all registers
* are caller save.
* Called from standard s390x C ABI, where r6-r13, r15,
* and f0, f2, f4 and f6 are callee-save, so they must
* callee-save, so they must be saved explicitly.
*
* chwan -
* This is the z/OS version (HLASM) of crosscall_s390x.
* It is incomplete as it does not know how to call via
* the function pointer "fn".
* And it does not have code to save and restore the
* floating-point registers that the loz version does.
*
CRSCLL#C  CSECT
CRSCLL#C  AMODE 64
CRSCLL#C  RMODE 64
G_CODE64  CATTR RMODE(64)
          DS    0D
          DC    XL7'00C300C500C500'
          DC    XL1'F1'
          DC    XL4'0000002C'
          DC    XL4'000000A0'
crosscall_s390x ALIAS C'crosscall_s390x'
          ENTRY crosscall_s390x
crosscall_s390x DS 0H
crosscall_s390x AMODE 64
crosscall_s390x RMODE ANY
crosscall_s390x XATTR LINKAGE(XPLINK)
          STMG  4,15,(2048-(((160+31)/32)*32))(4)
          STMG  1,3,2176(4)
R4CHG0001 EQU   *-crosscall_s390x
          AGHI  4,-((((160+31)/32)*32))
          BASR  8,0
          USING *,8
          LA    6,CELQ0003_FD
          LMG   5,6,0(6)
          BASR  7,6
          NOPR  0
          LA    6,CELQ0004_FD
          LMG   5,6,0(6)
          BASR  7,6
          NOPR  0
* chwan - this is kind of code calling fn assuming it is
*         XPLINK and takes *g as the argument
*         LMG   5,6,0(1)       * R1 points to fn which is
*			       * is a function descriptor
*         LGR   1,2            * R1 = *g passed to fn?
*         BASR  7,6
PRLGL0001 EQU   *-crosscall_s390x
          LMG   4,15,2048(4)
          BR    7
_cgo_reginit ALIAS C'_cgo_reginit'
_cgo_reginit AMODE 64
_cgo_reginit XATTR LINKAGE(XPLINK),SCOPE(IMPORT),REF(CODE)
CELQ0003_FD  DC RD(_cgo_reginit)
          DC VD(_cgo_reginit)
_cgo_load_g ALIAS C'_cgo_load_g'
_cgo_load_g AMODE 64
_cgo_load_g XATTR LINKAGE(XPLINK),SCOPE(IMPORT),REF(CODE)
CELQ0004_FD  DC RD(_cgo_load_g)
          DC VD(_cgo_load_g)
CODLN0001 EQU   *-crosscall_s390x
XPPA10001 DS    0F
          DC    B'00000010'
          DC    X'CE'
          DC    XL2'0FFF'
          DC    AL4(XPPA2001-XPPA10001)
          DC    B'10000000'      Flag1
          DC    B'00000000'      Flag2
          DC    B'00000000'      Flag3
          DC    B'00000001'      Flag4
          DC    AL2(0)           Parameter Length in words
          DC    AL1(PRLGL0001/2) Length of XPL prolog / 2
          DC    XL.4'0',AL.4(R4CHG0001/2)
          DC    AL4(CODLN0001)   Length of HPL appl code in bytes
          DC    H'15'            Length of Entry Point Name
          DC    CL15'crosscall_s390x'
XPPA2001  DS    0F
          DC    AL1(15)
          DC    AL1(00)                  Sub ID
          DC    AL1(0)                   Member defined
          DC    AL1(4)                   Level of PPAx control blocks
          DC    A(0)                     Offset to CELQSTRT from PPA2
          DC    AL4(0)                   A(Offset to PPA4) FORCED ZERO
          DC    A(CEETIMES-XPPA2001)     A(Offset to Timestamp)
          DC    A(crosscall_s390x-XPPA2001)
          DC    B'10010001'              PPA2 Compilation Flags
          DC    AL3(0)                   Reserved
CEETIMES  DS    0F
          DS    XL20
          END
