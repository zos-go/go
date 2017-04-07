// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// System calls and other system stuff for zOS s390x; see
// /usr/include/asm-s390/unistd.h for the syscall number definitions.

#include "go_asm.h"
#include "go_tls.h"
#include "textflag.h"

#define LE_exit          0x05A
#define LE_read          0x1B2
#define LE_write         0x1DF
#define LE_open_a        0x6F7 // __open_a -- ASCII to EBCDIC conversion for hardcoded file names (i.e. /dev/urandom)
#define LE_open_o_a      0x78C // _o=Large Files
#define LE_close         0x17E
#define LE_getpid        0x19D
#define LE_kill          0x1A4
#define LE_fcntl         0x18C
#define LE_gettimeofday  0x2F6
#define LE_select        0x390 // __select1
#define LE_mmap          0x295
#define LE_munmap        0x298
#define LE_mprotect      0x296
#define LE_setitimer     0x274
#define LE_sched_yield   0xB32
#define LE_sigaltstack   0x28A
#define LE_raise         0x05E
#define LE___environ_a   0x78F
#define LE_sigaction     0x1BD
#define LE_perror_a      0x712
#define LE_malloc        0x056
#define LE_malloc31      0x7FD
#define LE_calloc        0x058
#define LE_free          0x059
#define LE___errno       0x156
#define LE___err2ad      0x16C
#define LE_setcontext    0x431

#define LE_tmpname_a     0x752
#define LE_lseek         0x1A6
#define LE_unlink_a      0x72E

#define LE_poll          0x380
#define LE_pipe          0x1B0
#define LE_selectex      0x35C

// SUSV3 versions of pthread functions
#define LE_pthread_exit                 0x1E4
#define LE_pthread_create               0xB51 // @@PT3C
#define LE_pthread_attr_init            0xB43 // @@PT3AI
#define LE_pthread_attr_destroy         0xB40 // @@PT3AD
#define LE_pthread_attr_getstacksize    0xB42 // @@PT3AGS
#define LE_pthread_sigmask              0x5F7
#define LE_pthread_cond_init            0xB49 // @@PT3CI
#define LE_pthread_cond_signal          0xB4A // @@PT3CS
#define LE_pthread_cond_timedwait       0xB4B // @@PT3CT
#define LE_pthread_cond_wait            0xB4C // @@PT3CW
#define LE_pthread_kill                 0xC8B // @@PT3KIL
#define LE_pthread_mutex_init           0xB56 // @@PT3MI
#define LE_pthread_mutex_lock           0xB57 // @@PT3ML
#define LE_pthread_mutex_unlock         0xB59 // @@PT3MU

// used for debug only
#define LE_fprintf_a                    0x6FA
#define LE_printf_a                     0x6DD
#define LE_fflush                       0x068

// open option flags (/usr/include/fcntl.h)
#define O_CREAT      0x80
#define O_TRUNC      0x10
#define O_RDWR       0x03

// open mode flags (/usr/include/sys/modes.h)
#define S_IRUSR      0x0100
#define S_IWUSR      0x0080

// lseek positions (/usr/include/unistd.h)
#define SEEK_SET     0

// mmap protections (/usr/include/sys/mman.h)
#define PROT_READ       1             /* page can be read    */
#define PROT_WRITE      2             /* page can be written */
#define PROT_NONE       4             /* can't be accessed   */
#define PROT_EXEC       8             /* page can be executed*/

// mmap flags (/usr/include/sys/mman.h)
#define MAP_PRIVATE     1             /* Changes are private */
#define MAP_SHARED      2             /* Changes are shared  */
#define MAP_FIXED       4             /* place exactly       */

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
#define LE_SVC_WAIT BYTE $0x0A; BYTE $0x01;
#define LE_SVC_POST BYTE $0x0A; BYTE $0x02;

#define TWO_GIG      0x800000000

#define LE_DBG(lca,wrk)   MOVD CAA(lca),wrk; MOVD	EDCHPXV(wrk),wrk; ADD	$(LE_printf_a*16),wrk; LMG	0(wrk),R5,R6; MOVD R3,wrk; BL R7,R6; LE_NOP; MOVD wrk,R3
#define LE_FLUSHALL(lca,wrk) MOVD $0,R1; MOVD CAA(lca),wrk; MOVD	EDCHPXV(wrk),wrk; ADD	$(LE_fflush*16),wrk; LMG	0(wrk),R5,R6; MOVD R3,wrk; BL R7,R6; LE_NOP; MOVD wrk,R3

TEXT runtime·exit(SB),NOSPLIT|NOFRAME,$0-4
	MOVW	code+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_exit*16), R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Call function.
	BL	R7, R6
	LE_NOP

	// Shouldn't return.
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	$0, 0(R0)   // Crash.

	RET

TEXT runtime·exit1(SB),NOSPLIT|NOFRAME,$0-4
	MOVW	code+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_exit*16), R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Call function.
	BL	R7, R6
	LE_NOP

	// Shouldn't return.
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	$0, 0(R0)   // Crash.
	RET

// func open(name *byte, mode, perm int32) int32
TEXT runtime·open(SB),NOSPLIT|NOFRAME,$0-20
	MOVD	name+0(FP), R1
	MOVW	mode+8(FP), R2
	MOVW	perm+12(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_open_a*16), R9
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

	MOVW	R3, ret+16(FP)
	RET

// func perror()
TEXT runtime·perror(SB),NOSPLIT|NOFRAME,$0
	MOVD	$0, R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_perror_a*16), R9
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
	RET

// func closefd(fd int32) int32
TEXT runtime·closefd(SB),NOSPLIT|NOFRAME,$0-12
	MOVW	fd+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_close*16), R9
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

	MOVW	R3, ret+8(FP)
	RET

// func write(fd uintptr, p unsafe.Pointer, n int32) int32
TEXT runtime·write(SB),NOSPLIT|NOFRAME,$0-28
	MOVD	fd+0(FP), R1
	MOVD	p+8(FP), R2
	MOVW	n+16(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_write*16), R9
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
	MOVW	R3, ret+24(FP)
	RET

// func read(fd int32, p unsafe.Pointer, n int32) int32
TEXT runtime·read(SB),NOSPLIT|NOFRAME,$0-28
	MOVW	fd+0(FP), R1
	MOVD	p+8(FP), R2
	MOVW	n+16(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_read*16), R9
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

	MOVW	R3, ret+24(FP)
	RET

// func usleep(usec uint32)
TEXT runtime·usleep(SB),NOSPLIT,$16-16
	MOVD	$1000000, R1
	MOVD	$0, R8
	MOVWZ	usec+0(FP), R9
	WORD	$0xB9870081 // DLGR R9=(R8:R9)/R1 R8=(R8:R9)%R1
	MOVD	R8, tv_usec-8(SP)
	MOVD	R9, tv_sec-16(SP)

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_select*16), R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Fill in parameter list.
	MOVD	$0, (2176+24)(R4)
	MOVD	$tv-16(SP), R12
	MOVD	R12, (2176+32)(R4)

	MOVD	$0, R1
	MOVD	$0, R2
	MOVD	$0, R3

	// Call select(0, 0, 0, 0, &tv).
	BL	R7, R6
	LE_NOP
	XOR	R0, R0        // Restore R0 to $0.
	MOVD	R4, 0(R9)     // Save stack pointer.
	// Might get EINTR here. Ignore it.
	RET

// func gettid() uint64
TEXT runtime·gettid(SB),NOSPLIT,$0-8
	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get CEECAATHDID
	MOVD	CAA(R8), R9
	MOVD	0x3D0(R9), R9
	MOVD	R9, ret+0(FP)

	RET

// func raise(sig int32)
// Equivalent to: pthread_kill(pthread_self(), sig).
TEXT runtime·raise(SB),NOSPLIT|NOFRAME,$0-4
	MOVW	sig+0(FP), R2

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get CEECAATHDID
	MOVD	CAA(R8), R9
	MOVD	0x3D0(R9), R1

	// Get function.
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_kill*16), R9
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
	RET

// func raiseproc(sig int32)
TEXT runtime·raiseproc(SB),NOSPLIT|NOFRAME,$0-4
	// raiseproc is equivalent to kill(getpid(), sig).

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// getpid()
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_getpid*16), R9
	LMG	0(R9), R5, R6
	MOVD	$0, R1
	BL	R7, R6
	LE_NOP

	// kill(pid, sig)
	MOVW	R3, R1 // pid
	MOVW	sig+0(FP), R2
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_kill*16), R9
	LMG	0(R9), R5, R6
	BL	R7, R6
	LE_NOP

	// Switch back to Go stack.
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	R4, 0(R9)   // Save stack pointer.
	RET

TEXT runtime·setitimer(SB),NOSPLIT|NOFRAME,$0-24
	MOVW	mode+0(FP), R1
	MOVD	new+8(FP), R2
	MOVD	old+16(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_setitimer*16), R9
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
	CMPBNE	R3, $0, crash
	RET
crash:
	MOVD	$0, 0(R0)
	RET

TEXT runtime·mincore(SB),NOSPLIT|NOFRAME,$0-28
//	Not implemented.
//	MOVD	addr+0(FP), R2
//	MOVD	n+8(FP), R3
//	MOVD	dst+16(FP), R4
//	MOVW	$SYS_mincore, R1
//	SYSCALL
//	MOVW	R2, ret+24(FP)
	RET

// func now() (sec int64, nsec int32)
TEXT time·now(SB),NOSPLIT,$16-12
	MOVD	$tv-16(SP), R1
	MOVD	$0, R2

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_gettimeofday*16), R9
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
	CMPBNE	R3, $0, crash
	MOVD	tv_sec-16(SP), R1
	MOVD	R1, sec+0(FP)
	MOVW	tv_usec-4(SP), R2
	MULLW	$1000, R2
	MOVW	R2, usec+8(FP)
	RET
crash:
	// gettimeofday returned an error.
	MOVD	$0, 0(R0)
	RET

// func nanotime() int64
TEXT runtime·nanotime(SB),NOSPLIT,$16-8
	STCKE	time-16(SP)
	LMG	time-16(SP), R2, R3

	// Put the whole microseconds (us) value into R4.
	SRD	$4, R2, R4

	// Subtract whole us value from R2 to leave fraction.
	SLD	$4, R4, R5
	SUB	R5, R2

	// Convert R4 to from us to nanoseconds (ns).
	WORD	$0xA74D03E8 // MULLD $1000, R4

	// R2:R3 now contains a value less than 1us.
	SRD	$16, R3   // get rid of the programmable field
	SLD	$48, R2   // shift MSBs (in R2) left
	OR	R3, R2    // now have 0x000.xxxxxxxxxx us in R2

	// Divide by (2^52/1000) = (2^49/5^3) to get value in ns.
	WORD	$0xA72D007D // MULLD $125, R2 // multiply by 5^3
	SRD	$49, R2                       // divide by 2^49

	// Add the fractional part to the result.
	ADD	R2, R4

	MOVD	R4, ret+0(FP)
	RET

// func sigaction(sig uintptr, new, old *sigactiont) int32
TEXT runtime·sigaction(SB),NOSPLIT|NOFRAME,$0-28
	MOVW	sig+0(FP), R1
	MOVD	new+8(FP), R2
	MOVD	old+16(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_sigaction*16), R9
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
	MOVW	R3, ret+24(FP)
	RET

TEXT runtime·sigfwd(SB),NOSPLIT,$0-32
	MOVW	sig+8(FP), R2
	MOVD	info+16(FP), R3
	MOVD	ctx+24(FP), R4
	MOVD	fn+0(FP), R11
	BL	R11
	RET

// XPLINK linkage, SP is $2048(R4) (not R15) so must setup stack.
TEXT runtime·sigtramp(SB),NOFRAME|NOSPLIT,$0
#define LE_STKSIZE 1024
#define GO_STKSIZE 65536
#define CONTEXT_EYEC 80
	XOR	R0, R0

	// Allocate space on stack for Go and LE.
	// Go takes high space (i.e. the Go stack is a big LE stack
	// variable).
	// TODO(mundaym): give Go the right hi/lo values so
	// that it won't overwrite the LE stack.
	STMG    R4, R15, (2048-LE_STKSIZE-GO_STKSIZE)(R4)
	MOVD	$(-LE_STKSIZE-GO_STKSIZE)(R4), R4

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Save current savstack_async stack and replace it with
	// the LE signal stack.
	// TODO(mundaym): not sure if we should save the old value.
	MOVD	SAVSTACK_ASYNC(R8), R9
	//MOVD	0(R9), R5
	//MOVD	R5, (2048+LE_STKSIZE-8)(R4)
	MOVD	R4, 0(R9)

	// Allocate some space on Go stack.
	MOVD	$(2048+LE_STKSIZE+GO_STKSIZE)(R4), R15
	SUB	$256, R15

	// The g ptr might have been overwritten by LE.
	BL	runtime·load_g(SB)

	// Store the context.
	MOVD	R3, 32(R15)
	// Save eyec from context.
	// The functions that modify the context will increment this field.
	MOVD	CONTEXT_EYEC(R3), R4
	MOVD	R4, 40(R15)

	// Call the signal handler.
	MOVW	R1, 8(R15)
	MOVD	R2, 16(R15)
	MOVD	R3, 24(R15)
	MOVD	$runtime·sigtrampgo(SB), R4
	BL	R4

	// Get the context and the saved eyec value from the Go stack.
	MOVD	32(R15), R1
	MOVD	40(R15), R2

	// Restore the LE stack.
	ADD	$256, R15
	MOVD	$(-2048-LE_STKSIZE-GO_STKSIZE)(R15), R4

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Set savstack_async to 0.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	$0, 0(R9)
	//MOVD    (2048+LE_STKSIZE-8)(R4), R5
	//MOVD	R5, 0(R9)

	//	Compare the saved eyec value with the current eyec value.
	//	If not the same,
	//		The context been has changed.
	//		Call setcontext to resume the altered context.
	//	Else,
	//		The context has not changed.
	//		Return to caller.
	MOVD	CONTEXT_EYEC(R1), R3		// Get eyec value from context
	CMPBEQ	R3, R2, context_unchanged	// Compare to saved value, branch if equal
	MOVD	R2, CONTEXT_EYEC(R1)	// Restore eyec
	// Restore the context.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_setcontext*16), R9
	LMG	0(R9), R5, R6
	BL	R7, R6 // never returns
	LE_NOP

context_unchanged:
	// Return control to LE.
	LMG	2048(R4), R4, R15
	MOVD	$0, R3
	BR	R7
#undef GO_STKSIZE
#undef LE_STKSIZE

GLOBL oneNULLbyte<>(SB), RODATA, $1 // one null byte for write to set length of tmpfile

// NOTES: This service is transitioning from Go to LE. In the transition:
// - R8 will be used for the LCA.
// - R9,R10 will be used for a workregs.
// - R15 is never changed, so can still refer to arguments on (SB) and (FP)
// - Registers will not be saved/restored because in Go the caller does that.
// - Could either use tmpname+open+unlink, or tmpname_a+open_a,unlink_a the latter seems to fit better
//

// func mmap(addr unsafe.Pointer, n uintptr, prot, flags, fd int32, off uint32) unsafe.Pointer
TEXT runtime·mmap(SB),NOSPLIT,$48-40

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Switch to saved LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Get printf function & call for debug
	//MOVD $dbgmmap1<>(SB), R1
	//MOVD	    addr+0(FP), R2
	//MOVD	       n+8(FP), R3
	//LE_DBG(R8,R9)
	//LE_FLUSHALL(R8,R9)

  // check for addr > 2G, don't bother doing all this if so, skip to map_not_anon (will always fail)
	MOVD	addr+0(FP), R1
	MOVD	$TWO_GIG, R2
	CMPUBGE	R1, R2, map_not_anon // address is >= 2GB, so don't bother trying this extra stuff

	// TODO(BLL)
	// check for MAP_ANON, MAP_PRIVATE & FD is NULL, otherwise skip to map_not_anon
	// should to respect _PROT_READ,_PROT_WRITE in the file permissions

	map_anon:

	// Call tmpname_a(filename_addr|NULL)
	// Assume running POSIX(ON), and __POSIX_TMPNAM is unset !!!
	// set up parms
	MOVD	$0, R1 // parm 1 = NULL, so R3 will have thread-safe resultant char*
	// Get tmpname_a FuncDesc
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_tmpname_a*16), R9
	LMG	0(R9), R5, R6
	// Call tmpname_a function, ignore errors, open will fail.
	BL	R7, R6
	LE_NOP
	MOVD	R3,R10 // save tmpname char* for open & unlink

	// Call open_a(pathname,options[,modes])
	// set up parms
	MOVD	R10,R1                        // tmpname filename
	MOVD	$(O_CREAT+O_TRUNC+O_RDWR), R2 // options (O_TRUNC shouldn't be needed since file must be new)
	MOVD	$(S_IRUSR+S_IWUSR), R3        // modes -- only this user can see the file
	// Get open_a FuncDesc
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_open_a*16), R9
	LMG	0(R9), R5, R6
	// Call open_a function, return on error
	BL	R7, R6
	LE_NOP
	CMP	R3, $-1
	BEQ	mmap_ret
	MOVD	R10,R1 // save tmpname filename for unlink
	MOVD	R3,R10 // save open fd# for lseek, write, mmap

	// Call unlink_a(pathname) -- minimize the amount of time this pathname exists
	// set up parms
	//  pathname
	//   already set above after open
	// Get unlink_a FuncDesc
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_unlink_a*16), R9
	LMG	0(R9), R5, R6
	// Call unlink_a function, ignore error
	BL	R7, R6
	LE_NOP

	// Call lseek(fd,offset,pos)
	// set up parms
	//  fd
	MOVD	R10,R1           // fd# saved from open
	//  offset
	MOVD	n+8(FP), R2      // len
	MOVWZ	off+28(FP), R3   // offset
	ADD	R3,R2              // pos last byte
	SUB	$1,R2              // off last byte
	//  pos
	MOVD	$SEEK_SET,R3
	// Get lseek FuncDesc
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_lseek*16), R9
	LMG	0(R9), R5, R6
	// Call lseek function, return on error
	BL	R7, R6
	LE_NOP
	CMP	R3, $-1
	BEQ	mmap_ret

	// Call write(fd,buf,size)
	// set up parms
	//  fd
	MOVD	R10,R1           // fd# saved from open
	//  buf
	MOVD	$oneNULLbyte<>(SB), R2
	//  size
	MOVD	$1,R3
	// Get write FuncDesc
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_write*16), R9
	LMG	0(R9), R5, R6
	// Call write function, return on error
	BL	R7, R6
	LE_NOP
	CMP	R3, $-1
	BEQ	mmap_ret
	BR	mmap_call         // R10 still has fd#

	map_not_anon:

	MOVW	fd+24(FP), R10  // R10 has fd# from caller (was 0 if MAP_ANON)

	mmap_call:

	// Move parms into regs.
	MOVD	addr+0(FP), R1
	MOVD	n+8(FP), R2
	MOVW	prot+16(FP), R3

	// Fill in rest of parameter list.
	MOVW	flags+20(FP), R9
	MOVD	R9, (2176+24)(R4)
	//MOVW	fd+24(FP), R9
	//MOVD	R9, (2176+32)(R4)
	MOVD	R10, (2176+32)(R4)   // R10 will either have original parm or fabricated FD#
	MOVWZ	off+28(FP), R9
	MOVD	R9, (2176+40)(R4)

	// Get mmap FuncDesc
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_mmap*16), R9
	LMG	0(R9), R5, R6

	// Call mmap function.
	BL	R7, R6
	LE_NOP

	mmap_ret:
	// TODO(BLL)
	// If we opened a file & mmap failed, we should probably close it now;
	// can init R10 to 0 and use that FD# as a flag to do so;
	// have to do it here since open may have worked but lseek etc failed

	// Get printf function & call for debug
	//MOVD	$dbgmmap2<>(SB), R1
	//MOVW	    prot+16(FP), R2
	//                     R3 already has return
	//                     R4 arg position should already have flags
	//LE_DBG(R8,R9)
	//LE_FLUSHALL(R8,R9)

	// Save errno/errnojr for PD.
	// Would need to call __errno(), __errno2() (or __err2ad))
	// NOTE: Can do this after switching back to Go stack so as to not have to reload R9 with save stack... but this is just for DBG
	//MOVD	SAVSTACK_ASYNC(R8), R9
	//MOVD	CAA(R8), R10
	//<sprinkle magic here>
	//MVC	$4,ERRNO(R10),SS_ERRNO(R9)
	//MVC	$4,ERRNOJR(R10),SS_ERRNOJR(R9)

	// Get printf function & call for debug
	//MOVD	R3,R10 // ugh, have to save/restore return since using for parm
	//MOVD	$dbgmmap3<>(SB), R1
	//MOVWZ	SS_ERRNO(R9), R2
	//MOVWZ	SS_ERRNOJR(R9), R3
	//LE_DBG(R8,R9)
	//LE_FLUSHALL(R8,R9)
	//MOVD	R10,R3

	// Switch back to Go stack.
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	R4, 0(R9)   // Save stack pointer.

	// Check the result.  Expect an address to be returned.
	//  Return the address, or 0 for failure, to the Go caller.
	MOVD	$-4095, R2
	CMPUBLT	R3, R2, 2(PC)
	NEG	R3, R3
	MOVD	R3, ret+32(FP)
	RET

TEXT runtime·munmap(SB),NOSPLIT|NOFRAME,$0
	MOVD	addr+0(FP), R1
	MOVD	n+8(FP), R2

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_munmap*16), R9
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

	MOVD	$-4095, R1
	CMPUBLT	R3, R1, 2(PC)
	MOVD	R0, 0(R0) // crash
	RET

// func mprotect(addr unsafe.Pointer, n uintptr, prot int32) int32
TEXT runtime·mprotect(SB),NOSPLIT|NOFRAME,$0-28
	MOVD	addr+0(FP), R1
	MOVD	n+8(FP), R2
	MOVW	prot+16(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get mprotect FuncDesc.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_mprotect*16), R9
	LMG	0(R9), R5, R6

	// Switch to saved LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Call mprotect function.
	BL	R7, R6
	LE_NOP

	// Switch back to Go stack.
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVW	R3, ret+24(FP) // Return result
	RET

// func sigaltstack(new, old *sigaltstackt)
TEXT runtime·sigaltstack(SB),NOSPLIT|NOFRAME,$0-16
	MOVD	new+0(FP), R1
	MOVD	old+8(FP), R2

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_sigaltstack*16), R9
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

	CMPBNE	R3, $0, crash
	RET
crash:
	BL	runtime·perror(SB)
	MOVD	$0, 0(R0)
	RET

TEXT runtime·osyield(SB),NOSPLIT|NOFRAME,$0
	MOVD	$0, R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_sched_yield*16), R9
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

	CMPBNE	R3, $0, crash
	RET
crash:
	MOVD	$0, 0(R0)
	RET

// func pthread_cond_init(condaddr *pthread_cond, condattraddr *pthread_condattr) int32
TEXT runtime·pthread_cond_init(SB),NOSPLIT|NOFRAME,$0-20
	MOVD	condaddr+0(FP), R1
	MOVD	condattraddr+8(FP), R2

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_cond_init*16), R9
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

	MOVW	R3, ret+16(FP)
	RET

// func pthread_cond_signal(condaddr *pthread_cond) int32
TEXT runtime·pthread_cond_signal(SB),NOSPLIT|NOFRAME,$0-12
	MOVD	condaddr+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_cond_signal*16), R9
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

	MOVW	R3, ret+8(FP)
	RET

// func pthread_cond_timedwait(condaddr *pthread_cond, mutexaddr *pthread_mutex, timeaddr *timespec) int32
TEXT runtime·pthread_cond_timedwait(SB),NOSPLIT|NOFRAME,$0-28
	MOVD	condaddr+0(FP), R1
	MOVD	mutexaddr+8(FP), R2
	MOVD	timeaddr+16(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_cond_timedwait*16), R9
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

	MOVW	R3, ret+24(FP)
	RET

// func pthread_cond_wait(condaddr *pthread_cond, mutexaddr *pthread_mutex) int32
TEXT runtime·pthread_cond_wait(SB),NOSPLIT|NOFRAME,$0-20
	MOVD	condaddr+0(FP), R1
	MOVD	mutexaddr+8(FP), R2

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_cond_wait*16), R9
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

	MOVW	R3, ret+16(FP)
	RET

// func pthread_mutex_init(mutexaddr *pthread_mutex, mutexattraddr *pthread_mutexattr) int32
TEXT runtime·pthread_mutex_init(SB),NOSPLIT|NOFRAME,$0-20
	MOVD	mutexaddr+0(FP), R1
	MOVD	mutexattraddr+8(FP), R2

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_mutex_init*16), R9
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

	MOVW	R3, ret+16(FP)
	RET

// func pthread_mutex_lock(mutexaddr *pthread_mutex) int32
TEXT runtime·pthread_mutex_lock(SB),NOSPLIT|NOFRAME,$0-12
	MOVD	mutexaddr+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_mutex_lock*16), R9
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

	MOVW	R3, ret+8(FP)
	RET

// func pthread_mutex_unlock(mutexaddr *pthread_mutex) int32
TEXT runtime·pthread_mutex_unlock(SB),NOSPLIT|NOFRAME,$0-12
	MOVD	mutexaddr+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_mutex_unlock*16), R9
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

	MOVW	R3, ret+8(FP)
	RET

// func pthread_attr_init(attr *pthread_attr_t) int32
TEXT runtime·pthread_attr_init(SB),NOSPLIT|NOFRAME,$0-12
	MOVD	attr+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_attr_init*16), R9
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

	MOVW	R3, ret+8(FP)
	RET

// func pthread_attr_destroy(attr *pthread_attr_t) int32
TEXT runtime·pthread_attr_destroy(SB),NOSPLIT|NOFRAME,$0-12
	MOVD	attr+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_attr_destroy*16), R9
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

	MOVW	R3, ret+8(FP)
	RET

// func pthread_attr_getstacksize(attr *pthread_attr_t) uintptr
TEXT runtime·pthread_attr_getstacksize(SB),NOSPLIT|NOFRAME,$0-16
	MOVD	attr+0(FP), R1
	MOVD	$ret+8(FP), R2

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_attr_getstacksize*16), R9
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

	RET

// func pthread_sigmask(how int32, old, new *sigset) int32
TEXT runtime·pthread_sigmask(SB),NOSPLIT|NOFRAME,$0-28
	MOVW	how+0(FP), R1
	MOVD	old+8(FP), R2
	MOVD	new+16(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_sigmask*16), R9
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

	MOVW	R3, ret+24(FP)
	RET

// func pthread_create(p *phread_t, attr *pthread_attr_t, fn func(uintptr) uintptr, arg uinptr) int32
TEXT runtime·pthread_create(SB),NOSPLIT|NOFRAME,$0-36
	MOVD	p+0(FP), R1
	MOVD	attr+8(FP), R2
	MOVD	fn+16(FP), R3

	// LE jumps to fn+8 (over ENV pointer).
	SUB	$8, R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pthread_create*16), R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Fill in parameter list.
	MOVD	arg+24(FP), R8
	MOVD	R8, (2176+24)(R4)

	// Call function.
	BL	R7, R6
	LE_NOP
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVW	R3, ret+32(FP)
	RET

// func __environ() unsafe.Pointer
TEXT runtime·__environ(SB),NOSPLIT|NOFRAME,$0-8

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get __environ_a FuncDesc.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE___environ_a*16), R9
	LMG	0(R9), R5, R6

	// Switch to saved LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Call __environ_a function.
	BL	R7, R6
	LE_NOP

	// Switch back to Go stack.
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVD	0(R3), R3
	MOVD	R3, ret+0(FP) // Return char** environ

	RET

// func malloc(size uintptr) unsafe.Pointer
TEXT runtime·malloc(SB),NOSPLIT|NOFRAME,$0-16
	MOVD	size+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_malloc*16), R9
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
	MOVD	R3, ret+8(FP)
	RET

// func malloc31(size uintptr) unsafe.Pointer
TEXT runtime·malloc31(SB),NOSPLIT|NOFRAME,$0-16
	MOVD	size+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_malloc31*16), R9
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
	MOVD	R3, ret+8(FP)
	RET

// func calloc(size uintptr) unsafe.Pointer
TEXT runtime·calloc(SB),NOSPLIT|NOFRAME,$0-16
	MOVD	size+0(FP), R1
	MOVD	$1, R2      // really calloc(num,size)

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_calloc*16), R9
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
	MOVD	R3, ret+8(FP)
	RET

// func free(ptr unsafe.Pointer)
TEXT runtime·free(SB),NOSPLIT|NOFRAME,$0-8
	MOVD	ptr+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_free*16), R9
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
	RET

// func __errno() unsafe.Pointer
// returns the address of errno
TEXT runtime·__errno(SB),NOSPLIT|NOFRAME,$0-8
	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get __errno FuncDesc.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE___errno*16), R9
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

	MOVD	R3, ret+0(FP) // Return result
	RET

// func __err2ad() unsafe.Pointer
// returns the address of errnojr (residual)
TEXT runtime·__err2ad(SB),NOSPLIT|NOFRAME,$0-8
	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get __err2ad FuncDesc.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE___err2ad*16), R9
	LMG	0(R9), R5, R6

	// Switch to saved LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Call __err2ad function.
	BL	R7, R6
	LE_NOP

	// Switch back to Go stack.
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVD	R3, ret+0(FP) // Return result
	RET

// func poll(pfd *pollfd pfd_count uint32 timeout int32) int32
TEXT runtime·poll(SB),NOSPLIT|NOFRAME,$0-20
	MOVD	pfd+0(FP), R1
	MOVW	pfd_count+8(FP), R2
	MOVW	timeout+12(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_poll*16), R9
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

	MOVW	R3, ret+16(FP)
	RET
// func pipe(fdsbuffer *int32) int32
TEXT runtime·pipe(SB),NOSPLIT|NOFRAME,$0-12
	MOVD	fdsbuffer+0(FP), R1

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_pipe*16), R9
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

	MOVW	R3, ret+8(FP)
	RET

// func selectex(numfds int32, readfds *int32, writefds *int32, excepfds *int32, timeout *Timeval, ecbptr *int32) int32
TEXT runtime·selectex(SB),NOSPLIT|NOFRAME,$0-52
	MOVD	numfds+0(FP), R1
	MOVD	readfds+8(FP), R2
	MOVD	writefds+16(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_selectex*16), R9
	LMG	0(R9), R5, R6

	// Restore LE stack.
	MOVD	SAVSTACK_ASYNC(R8), R9
	MOVD	0(R9), R4
	MOVD	$0, 0(R9)

	// Fill in parameter list.
	MOVD	excepfds+24(FP), R8
	MOVD	R8, (2176+24)(R4)

	MOVD	timeout+32(FP), R8
	MOVD	R8, (2176+32)(R4)

	MOVD	excepfds+40(FP), R8
	MOVD	R8, (2176+40)(R4)

	// Call function.
	BL	R7, R6
	LE_NOP
	XOR	R0, R0      // Restore R0 to $0.
	MOVD	R4, 0(R9)   // Save stack pointer.

	MOVW	R3, ret+48(FP)
	RET
// func post(ecbptr *int32, postcode int32)
TEXT runtime·post(SB),NOSPLIT|NOFRAME,$0-12
	MOVD	ecbptr+0(FP), R1
	MOVW	postcode+8(FP), R0

	// Save registers 14 and 15
	MOVD	R14, R8
	MOVD	R15, R9

	// Next instruction destroys registers 0, 1, 14 and 15
	LE_SVC_POST

	// Restore registers 14 and 15
	MOVD	R8, R14
	MOVD	R9, R15

	XOR	R0, R0      // Restore R0 to $0.

	RET
// func wait(ecbptr *int32)
TEXT runtime·wait(SB),NOSPLIT|NOFRAME,$0-8
	MOVD	ecbptr+0(FP), R1
	MOVD	$1, R0

	// Save registers 14 and 15
	MOVD	R14, R8
	MOVD	R15, R9

	// Next instruction destroys registers 0, 1, 14 and 15
	LE_SVC_WAIT

	// Restore registers 14 and 15
	MOVD	R8, R14
	MOVD	R9, R15

	XOR	R0, R0      // Restore R0 to $0.

	RET

// func fcntl(fd, cmd, arg int32) int32
TEXT runtime·fcntl(SB),NOSPLIT|NOFRAME,$0-16
	MOVW	fd+0(FP), R1
	MOVW	cmd+4(FP), R2
	MOVW	arg+8(FP), R3

	// Get library control area (LCA).
	MOVW	PSALAA, R8
	MOVD	LCA64(R8), R8

	// Get function.
	MOVD	CAA(R8), R9
	MOVD	EDCHPXV(R9), R9
	ADD	$(LE_fcntl*16), R9
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

	MOVW	R3, ret+12(FP)
	RET
