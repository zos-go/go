// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

// func hasAsm() bool
// returns whether the AES-128, AES-192 and AES-256
// cipher message functions are supported.
TEXT ·hasAsm(SB),NOSPLIT,$16-1
	XOR    R0, R0 // set function code to 0 (query)
	LA     8(R15), R1
	WORD   $0xB92E0024 // KM-Query

	// check if bits 18-20 are set
	MOVD   8(R15), R2
	SRD    $40, R2
	AND    $0x38, R2  // mask bits 18-20 (00111000)
	CMPBNE R2, $0x38, notfound
	MOVBZ  $1, R1
	MOVB   R1, ret+0(FP)
	RET
notfound:
	MOVBZ  R0, ret+0(FP)
	MOVD   $0, 0(R0)
	RET

// func encryptBlockAsm(nr int, xk *uint32, dst, src *byte)
TEXT ·encryptBlockAsm(SB),NOSPLIT,$0-32
	MOVD   nr+0(FP), R7
	MOVD   xk+8(FP), R1
	MOVD   dst+16(FP), R2
	MOVD   src+24(FP), R4
	MOVD   $16, R5
	CMPBEQ R7, $14, aes256
	CMPBEQ R7, $12, aes192
aes128:
	MOVBZ  $18, R0
	BR     enc
aes192:
	MOVBZ  $19, R0
	BR     enc
aes256:
	MOVBZ  $20, R0
enc:
	WORD   $0xB92E0024 // KM-AES
	BVS    enc
	XOR    R0, R0
	RET

// func decryptBlockAsm(nr int, xk *uint32, dst, src *byte)
TEXT ·decryptBlockAsm(SB),NOSPLIT,$0-32
	MOVD   nr+0(FP), R7
	MOVD   xk+8(FP), R1
	MOVD   dst+16(FP), R2
	MOVD   src+24(FP), R4
	MOVD   $16, R5
	CMPBEQ R7, $14, aes256
	CMPBEQ R7, $12, aes192
aes128:
	MOVBZ  $(128+18), R0
	BR     dec
aes192:
	MOVBZ  $(128+19), R0
	BR     dec
aes256:
	MOVBZ  $(128+20), R0
dec:
	WORD   $0xB92E0024 // KM-AES
	BVS    dec
	XOR    R0, R0
	RET

// func expandKeyAsm(nr int, key *byte, enc, dec *uint32)
// We do NOT expand the keys here as the KM command just
// expects the cryptographic key.
// Instead just copy the needed bytes from the key into
// the encryption/decryption expanded keys.
TEXT ·expandKeyAsm(SB),NOSPLIT,$0-32
	MOVD   nr+0(FP), R1
	MOVD   key+8(FP), R2
	MOVD   enc+16(FP), R3
	MOVD   dec+24(FP), R4
	CMPBEQ R1, $14, aes256
	CMPBEQ R1, $12, aes192
aes128:
	MVC    $(128/8), 0(R2), 0(R3)
	MVC    $(128/8), 0(R2), 0(R4)
	RET
aes192:
	MVC    $(192/8), 0(R2), 0(R3)
	MVC    $(192/8), 0(R2), 0(R4)
	RET
aes256:
	MVC    $(256/8), 0(R2), 0(R3)
	MVC    $(256/8), 0(R2), 0(R4)
	RET
