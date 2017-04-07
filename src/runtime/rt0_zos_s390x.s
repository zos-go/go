// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

//efine EyeCatchLE            $0xD3C5    // LE
#define EyeCatchLE            $-11323    // LE (Go compiler gets confused with big signed immediate
//efine EyeCatchGo            $0xC796    // Go
#define EyeCatchGo            $-14442    // Go
//efine EyeCatchG0            $0xC7F0    // G0 (Go _rt0_s390x_zos)
#define EyeCatchG0            $-14352    // G0
//efine EyeCatchGt            $0xC7A3    // Gt (Go runtime.zosThreadEntry)
#define EyeCatchGt            $-14429    // Gt
#define R1junk            $0x0b110b11    // bll was here

#define PSALAA                 (1208)     // 0x4b8 - Offset in the PSA of the LE Anchor Area (LAA)
#define CEELAA_LCA64             (88)     // 0x058 - Offset in the LAA of the AMODE 64 LCA (Library Communication Area)
#define CEELCA_SAVSTACK_ASYNC   (336)     // 0x150 - Offset in the LCA of the Indirect Save Stack Pointer For Asynch Signals

#define LEbias                 (2048)     // 0x800 - LE AMODE 64 XPLINK stack frame bias
                                          // (LE R4 is analagous to Go R15, except bias allows caller to setup
                                          //  its stack using +displacements before changing the stack pointer.)
// NOTE: These are meant to be sizes
#define LEfixedStack            (128)     // 0x080 - LE AMODE 64 XPLINK stack frame fixed portion size
#define LEinitStack         (65*1024)     // 0x10400 - Initial 65K LE AMODE 64 XPLINK stack frame -- 64K for Go, 1K at the top (low end) for LE
#define LEstackUse           (1*1024)     // 0x00400 -                                            -- 1K at the top for LE to use
#define GOstackUse          (64*1024)     // 0x10000 -                                            -- 64K for Go to use

// NOTE: These are meant to be offsets
#define LEstackGoTran           (256)     // 0x00100 -                                            -- 128 fixed + 128 for args if needed
#define LEsaveStackPtr          (264)     // 0x108 - LE stack pointer offset in LE initial stack
#define GOsaveStackPtr          (272)     // 0x110 - Go stack pointer offset in LE initial stack

TEXT _rt0_s390x_zos(SB),NOSPLIT|NOFRAME,$0
	// In a statically linked binary, the stack contains argc,
	// argv as argc string pointers followed by a NULL, envv as a
	// sequence of string pointers followed by a NULL, and auxv.
	// There is no TLS base pointer.
	//
	// TODO: Support dynamic linking entry point
//	MOVD 0(R15), R2 // argc
//	ADD $8, R15, R3 // argv
//	BR main(SB)

// Save the (possible) arguments in R1-R3 into the argument area (i.e. XPLINK(STOREARGS)),
// it might be useful to have, and also the PPA2 says that we're doing it.
// Note args live in callers stack frame, so STOREARGS callee saves them there.
   STMG R1,R3,LEbias+LEfixedStack(R4)  // args live at +128, after fixed part of stackframe (also must consider bias)

// Set up for LE XPLINK stackframe transitioning to Go stackframe
   STMG R4,R15,LEbias-LEinitStack(R4)  // Save regs into our soon-to-be stack frame (so when we buy it, it is ready to use)
   SUB  $LEinitStack,R4                // Buy initial LE stack, with space for Go system stack (preallocated for Go to use)

//
// We'll carve up this LE initial stack frame as follows:
//
//   +00000 saved registers (96 bytes for R4-R15)
//   ...... rest of fixed portion of the stack frame
//   +00080 arguments (variable size)
//   ...... allow for up to 0x80 bytes of arguments
//   +00100 LE-to-Go-Transition frame eyecatcher LEGoG0LE -- X'D3C5C796C7F0D3C5'
//   +00108 Saved LE stack pointer (CEELCA_SAVSTACK_ASYNC points here)
//   +00110 Saved Go stack pointer (as required)
//   +00118 - +3f8 free space for anything LE needs
//   +003f8 LE-to-Go-Transition frame eyecatcher GoLELEGo -- X'C796D2C5D3C5C796'
//   +00400 - top of the stack area for Go to use (inital Go system stack)
//   ...... - Go stuff ...
//   +103ff - current stack pointer given to Go
//

// First set some eyecatcher stuff

// This marks where LE will save some important stuff
   MOVH EyeCatchLE,LEbias+LEstackGoTran+0(R4)        // LE......
   MOVH EyeCatchGo,LEbias+LEstackGoTran+2(R4)        // ..Go....
   MOVH EyeCatchG0,LEbias+LEstackGoTran+4(R4)        // ....G0..
   MOVH EyeCatchLE,LEbias+LEstackGoTran+6(R4)        // ......LE

// This marks Go's low-water mark for stack allocation --
// if Go over-allocates and crosses this eye-catcher, bad things will likely happen!
   MOVH EyeCatchGo,LEbias+LEstackUse-8(R4)           // Go......
   MOVH EyeCatchLE,LEbias+LEstackUse-6(R4)           // ..LE....
   MOVH EyeCatchLE,LEbias+LEstackUse-4(R4)           // ....LE..
   MOVH EyeCatchGo,LEbias+LEstackUse-2(R4)           // ......Go

// Save the current stack pointer for the LE save-stack pointer mechanism to use
// (both for recovery, and for us to be able to retrieve on the other side of Go, for system calls).
// We'll save the stack pointer at 0x100 into our stack,
// still leaving 0x80 for arguments (still using the 2048 (0x800) LE XPLINK bias)

// Find the LCA
   MOVW PSALAA(R0),R15                 // Get LE Anchor Area
   MOVD CEELAA_LCA64(R15),R15          // Get LE Library Communication Area

// Save the LE stack pointer using the LCA save-stack indirect pointer, pointing into our stack frame
   MOVD $0,LEbias+GOsaveStackPtr(R4)                 // Show Go stack pointer is nadda right now
   MOVD R4,LEbias+LEsaveStackPtr(R4)                 // First put our saved stack pointer in our stack frame
   MOVD R4,R0                                        // Now save ..
   ADD  $(LEbias+LEsaveStackPtr),R0                  // .. it's location  ..
   MOVD R0,CEELCA_SAVSTACK_ASYNC(R15)                // .. into the save-stack indirect field in the LCA,
                                                     // CEELCA_SAVSTACK_ASYNC - Ptr to saved stack ptr for when async signals are handled

// runtime.rt0_go expects argc & argv in R2 & R3,
// LE has them in the first two args, R1 & R2, so move them over
// NOTE: Nothing can have touched R1 & R2 up until now! This could be done anyplace after STOREARGS but prefer to leave them until required.
   MOVD R2,R3                                        // argv
   MOVD R1,R2                                        // argc
   MOVD R1junk,R1                                    // reset R1 to something we can recognize (BLL)

// Now share the LE stack with Go.  Go will treat as the system stack, as-if it's actually allocating it (even though LE already did)
   ADDC $(LEbias+LEinitStack),R4,R15                 // Put the LE stack point into the Go stack pointer,
                                                     // while adjusting the Go stack ptr with the LE bias, plus the size of the initial LE stack
                                                     // (i.e. so R15 is really pointing to the storage to use, unbiased, and is where R4 was upon entry).
                                                     // NOTE: This G0 stack is not expected to be used after GoRuntime init, it's just for coming up...
                                                     // Go will manage it's own stack later. From LE's POV, this is just free space in our stack frame.

	BR main(SB)


TEXT main(SB),NOSPLIT|NOFRAME,$0
	MOVD	$runtime·rt0_go(SB), R11
	BR	R11

// After LE initializes and gives control to Go runtime, Go should never come back on R14 (its return reg),
// so in case somehow that happens, we'll make it clear that it did happen.
// The termination flow should be that Go eventually calls the exit system service.
//
// LE ABEND U4091 reason 40
//  U4091 is "An unexpected condition occurred during the running of Language Environment condition management."
//  Reason 0x40 does not currently exist.

// HObyte, bit x'80' says to dump
// HObyte, bit x'04' says a reason code is specified
// Remaining 3 bytes is abend code 0xFFB = 4091
  MOVD $0x04000FFB,R1

// reason X0x40
  MOVD $0x40,R15

//SYSCALL $3                           // SVC 03 is EXIT
  SYSCALL $13                          // SVC 0D is ABEND


// zosThreadEntry is called by pthread_create, therefore XPLINK linkage.
// Transition to Go linkage.
// func zosThreadEntry(g uintptr) uintptr
TEXT runtime·zosThreadEntry(SB),NOSPLIT|NOFRAME,$0
// Set up for LE XPLINK stackframe transitioning to Go stackframe

// Save the (possible) arguments in R1-R3 into the argument area (i.e. XPLINK(STOREARGS)),
// it might be useful to have, and also the PPA2 says that we're doing it.
// Note args live in callers stack frame, so STOREARGS callee saves them there.
   STMG R1,R3,LEbias+LEfixedStack(R4)  // args live at +128, after fixed part of stackframe (also must consider bias)

// Set up for LE XPLINK stackframe transitioning to Go stackframe
   STMG R4,R15,LEbias-LEinitStack(R4)  // Save regs into our soon-to-be stack frame (so when we buy it, it is ready to use)
   SUB  $LEinitStack,R4                // Buy initial LE stack, with space for Go system stack (preallocated for Go to use)

//
// We'll carve up this LE initial stack frame as follows:
//
//   +00000 saved registers (96 bytes for R4-R15)
//   ...... rest of fixed portion of the stack frame
//   +00080 arguments (variable size)
//   ...... allow for up to 0x80 bytes of arguments
//   +00100 LE-to-Go-Transition frame eyecatcher LEGoGtLE -- X'D3C5C796C7A3D3C5'
//   +00108 Saved LE stack pointer (CEELCA_SAVSTACK_ASYNC points here)
//   +00110 Saved Go stack pointer (as required)
//   +00118 - +3f8 free space for anything LE needs
//   +003f8 LE-to-Go-Transition frame eyecatcher GoLELEGo -- X'C796D2C5D3C5C796'
//   +00400 - top of the stack area for Go to use (inital Go system stack)
//   ...... - Go stuff ...
//   +103ff - current stack pointer given to Go
//

// First set some eyecatcher stuff

// This marks where LE will save some important stuff
   MOVH EyeCatchLE,LEbias+LEstackGoTran+0(R4)        // LE......
   MOVH EyeCatchGo,LEbias+LEstackGoTran+2(R4)        // ..Go....
   MOVH EyeCatchGt,LEbias+LEstackGoTran+4(R4)        // ....Gt..
   MOVH EyeCatchLE,LEbias+LEstackGoTran+6(R4)        // ......LE

// This marks Go's low-water mark for stack allocation --
// if Go over-allocates and crosses this eye-catcher, bad things will likely happen!
   MOVH EyeCatchGo,LEbias+LEstackUse-8(R4)           // Go......
   MOVH EyeCatchLE,LEbias+LEstackUse-6(R4)           // ..LE....
   MOVH EyeCatchLE,LEbias+LEstackUse-4(R4)           // ....LE..
   MOVH EyeCatchGo,LEbias+LEstackUse-2(R4)           // ......Go

// Save the current stack pointer for the LE save-stack pointer mechanism to use
// (both for recovery, and for us to be able to retrieve on the other side of Go, for system calls).
// We'll save the stack pointer at 0x100 into our stack,
// still leaving 0x80 for arguments (still using the 2048 (0x800) LE XPLINK bias)

// Find the LCA
   MOVW PSALAA(R0),R15                 // Get LE Anchor Area
   MOVD CEELAA_LCA64(R15),R15          // Get LE Library Communication Area

// Save the LE stack pointer using the LCA save-stack indirect pointer, pointing into our stack frame
   MOVD $0,LEbias+GOsaveStackPtr(R4)                 // Show Go stack pointer is nadda right now
   MOVD R4,LEbias+LEsaveStackPtr(R4)                 // First put our saved stack pointer in our stack frame
   MOVD R4,R0                                        // Now save ..
   ADD  $(LEbias+LEsaveStackPtr),R0                  // .. it's location  ..
   MOVD R0,CEELCA_SAVSTACK_ASYNC(R15)                // .. into the save-stack indirect field in the LCA,
                                                     // CEELCA_SAVSTACK_ASYNC - Ptr to saved stack ptr for when async signals are handled

// Now share the LE stack with Go.  Go will treat as the system stack, as-if it's actually allocating it (even though LE already did)
   ADDC $(LEbias+LEinitStack),R4,R15                 // Put the LE stack point into the Go stack pointer,
                                                     // while adjusting the Go stack ptr with the LE bias, plus the size of the initial LE stack
                                                     // (i.e. so R15 is really pointing to the storage to use, unbiased, and is where R4 was upon entry).
                                                     // NOTE: This G0 stack is not expected to be used after GoRuntime init, it's just for coming up...
                                                     // Go will manage it's own stack later. From LE's POV, this is just free space in our stack frame.

    // Move g pointer (1st and only argument) into g (R13)
    MOVD R1, g

    // Initialise other registers.
    XOR  R0, R0

    BL   runtime·mstart(SB)
    MOVD R0, 0(R0) // crash - need to figure out how to return to XPLINK caller if we ever get here
    RET
