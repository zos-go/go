// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

// func block(dig *digest, p []byte)
TEXT ·block(SB),NOSPLIT,$0-32
start:
	// Check that we have the SHA-256 function
	MOVD	·kimdQueryResult(SB), R4
	SRD	$56, R4 // Get the first byte
	AND	$0x20, R4, R5 // Bit 2 for SHA-256
	BNE	hardware
	AND	$0x80, R4, R5 // Bit 0 for Query
	BNE	generic
	MOVD	$·kimdQueryResult(SB), R1
	XOR	R0, R0 // Query function code
	WORD    $0xB93E0006 // KIMD Query (R6 is ignored)
	BR	start

hardware:
	MOVD	dig+0(FP), R1
	MOVD	p_base+8(FP), R2
	MOVD	p_len+16(FP), R3
	MOVBZ	$2, R0 // SHA-256 function code
kimd:
	WORD	$0xB93E0002 // KIMD R2
	BVS	kimd // interrupted -- continue
done:
	XOR	R0, R0 // Restore R0
	RET

generic:
	BR	·blockGeneric(SB)

GLOBL ·kimdQueryResult(SB), NOPTR, $16
