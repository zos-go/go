// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

#define PSALAA            1208(R0)
#define GTAB64(x)           80(x)
#define LCA64(x)            88(x)
#define CAA(x)               8(x)
#define EDCHPXV(x)        1016(x)       // in the CAA
#define SAVSTACK_ASYNC(x)  336(x)       // in the LCA

// SS_*, where x=SAVSTACK_ASYNC
#define SS_LE(x)             0(x)
#define SS_GO(x)             8(x)
#define SS_ERRNO(x)         16(x)
#define SS_ERRNOJR(x)       20(x)

#define LE_NOP BYTE $0x07; BYTE $0x00;

TEXT ·clearErrno(SB),NOSPLIT,$0-0
	BL	errno<>(SB)
	MOVD	$0, 0(R3)
	RET

// Returns the address of errno in R3.
TEXT errno<>(SB),NOSPLIT|NOFRAME,$0-0
	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get __errno FuncDesc.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(0x156*16), R9
	LMG	0(R9), R5, R6

	// Switch to saved LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Call __errno function.
	BL	R7, R6
	LE_NOP

	// Switch back to Go stack.
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.
	RET

TEXT ·Syscall(SB),NOSPLIT,$0-56
	BL	runtime·entersyscall(SB)
	MOVD	a1+8(FP), R1
	MOVD	a2+16(FP), R2
	MOVD	a3+24(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	MOVD	trap+0(FP), R5
	SLD	$4, R5
	ADD	R5, R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Call function.
	BL	R7, R6
	LE_NOP
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVD	R3, r1+32(FP)
	MOVD	R0, r2+40(FP)
	MOVD	R0, err+48(FP)
	MOVW	R3, R4
	CMP	R4, $-1
	BNE	done
	BL	errno<>(SB)
	MOVWZ	0(R3), R3
	MOVD	R3, err+48(FP)
done:
	BL	runtime·exitsyscall(SB)
	RET

TEXT ·RawSyscall(SB),NOSPLIT,$0-56
	MOVD	a1+8(FP), R1
	MOVD	a2+16(FP), R2
	MOVD	a3+24(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	MOVD	trap+0(FP), R5
	SLD	$4, R5
	ADD	R5, R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Call function.
	BL	R7, R6
	LE_NOP
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVD	R3, r1+32(FP)
	MOVD	R0, r2+40(FP)
	MOVD	R0, err+48(FP)
	MOVW	R3, R4
	CMP	R4, $-1
	BNE	done
	BL	errno<>(SB)
	MOVWZ	0(R3), R3
	MOVD	R3, err+48(FP)
done:
	RET

TEXT ·Syscall6(SB),NOSPLIT,$0-80
	BL	runtime·entersyscall(SB)
	MOVD	a1+8(FP), R1
	MOVD	a2+16(FP), R2
	MOVD	a3+24(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	MOVD	trap+0(FP), R5
	SLD	$4, R5
	ADD	R5, R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Fill in parameter list.
	MOVD	a4+32(FP), R12
	MOVD	R12, (2176+24)(R4)
	MOVD	a5+40(FP), R12
	MOVD	R12, (2176+32)(R4)
	MOVD	a6+48(FP), R12
	MOVD	R12, (2176+40)(R4)

	// Call function.
	BL	R7, R6
	LE_NOP
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVD	R3, r1+56(FP)
	MOVD	R0, r2+64(FP)
	MOVD	R0, err+72(FP)
	MOVW	R3, R4
	CMP	R4, $-1
	BNE	done
	BL	errno<>(SB)
	MOVWZ	0(R3), R3
	MOVD	R3, err+72(FP)
done:
	BL	runtime·exitsyscall(SB)
	RET

TEXT ·RawSyscall6(SB),NOSPLIT,$0-80
	MOVD	a1+8(FP), R1
	MOVD	a2+16(FP), R2
	MOVD	a3+24(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	MOVD	trap+0(FP), R5
	SLD	$4, R5
	ADD	R5, R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Fill in parameter list.
	MOVD	a4+32(FP), R12
	MOVD	R12, (2176+24)(R4)
	MOVD	a5+40(FP), R12
	MOVD	R12, (2176+32)(R4)
	MOVD	a6+48(FP), R12
	MOVD	R12, (2176+40)(R4)

	// Call function.
	BL	R7, R6
	LE_NOP
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVD	R3, r1+56(FP)
	MOVD	R0, r2+64(FP)
	MOVD	R0, err+72(FP)
	MOVW	R3, R4
	CMP	R4, $-1
	BNE	done
	BL	errno<>(SB)
	MOVWZ	0(R3), R3
	MOVD	R3, err+72(FP)
done:
	RET

TEXT ·Syscall9(SB),NOSPLIT,$0-80
	BL	runtime·entersyscall(SB)
	MOVD	a1+8(FP), R1
	MOVD	a2+16(FP), R2
	MOVD	a3+24(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	MOVD	trap+0(FP), R5
	SLD	$4, R5
	ADD	R5, R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Fill in parameter list.
	MOVD	a4+32(FP), R12
	MOVD	R12, (2176+24)(R4)
	MOVD	a5+40(FP), R12
	MOVD	R12, (2176+32)(R4)
	MOVD	a6+48(FP), R12
	MOVD	R12, (2176+40)(R4)
	MOVD	a7+56(FP), R12
	MOVD	R12, (2176+48)(R4)
	MOVD	a8+64(FP), R12
	MOVD	R12, (2176+56)(R4)
	MOVD	a9+72(FP), R12
	MOVD	R12, (2176+64)(R4)

	// Call function.
	BL	R7, R6
	LE_NOP
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVD	R3, r1+80(FP)
	MOVD	R0, r2+88(FP)
	MOVD	R0, err+96(FP)
	MOVW	R3, R4
	CMP	R4, $-1
	BNE	done
	BL	errno<>(SB)
	MOVWZ	0(R3), R3
	MOVD	R3, err+96(FP)
done:
        BL	runtime·exitsyscall(SB)
        RET

TEXT ·RawSyscall9(SB),NOSPLIT,$0-80
	MOVD	a1+8(FP), R1
	MOVD	a2+16(FP), R2
	MOVD	a3+24(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	MOVD	trap+0(FP), R5
	SLD	$4, R5
	ADD	R5, R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Fill in parameter list.
	MOVD	a4+32(FP), R12
	MOVD	R12, (2176+24)(R4)
	MOVD	a5+40(FP), R12
	MOVD	R12, (2176+32)(R4)
	MOVD	a6+48(FP), R12
	MOVD	R12, (2176+40)(R4)
	MOVD	a7+56(FP), R12
	MOVD	R12, (2176+48)(R4)
	MOVD	a8+64(FP), R12
	MOVD	R12, (2176+56)(R4)
	MOVD	a9+72(FP), R12
	MOVD	R12, (2176+64)(R4)

	// Call function.
	BL	R7, R6
	LE_NOP
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVD	R3, r1+80(FP)
	MOVD	R0, r2+88(FP)
	MOVD	R0, err+96(FP)
	MOVW	R3, R4
	CMP	R4, $-1
	BNE	done
	BL	errno<>(SB)
	MOVWZ	0(R3), R3
	MOVD	R3, err+96(FP)
done:
	RET
