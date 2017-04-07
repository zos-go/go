// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"
)

const (
	_PROT_READ  = 0x1
	_PROT_WRITE = 0x2
	_PROT_NONE  = 0x4
	_PROT_EXEC  = 0x8

	_MAP_ANON    = 0x0 // This flag is a placeholder. z/OS does not currently support MAP_ANON.
	_MAP_PRIVATE = 0x1
	_MAP_FIXED   = 0x4

	_ITIMER_REAL    = 0x0
	_ITIMER_VIRTUAL = 0x1
	_ITIMER_PROF    = 0x2
	_ITIMER_MICRO   = 0x0
	_ITIMER_NANO    = 0x4

	_O_RDONLY = 0x2

	_SA_RESTART = 0x8000000
	_SA_ONSTACK = 0x20000000
	_SA_SIGINFO = 0x4000000

	_SIGHUP    = 1
	_SIGINT    = 2
	_SIGABRT   = 3
	_SIGILL    = 4
	_SIGURG    = 6
	_SIGSTOP   = 7
	_SIGFPE    = 8
	_SIGKILL   = 9
	_SIGBUS    = 10
	_SIGSEGV   = 11
	_SIGSYS    = 12
	_SIGPIPE   = 13
	_SIGALRM   = 14
	_SIGUSR1   = 16
	_SIGUSR2   = 17
	_SIGCONT   = 19
	_SIGCHLD   = 20
	_SIGTTIN   = 21
	_SIGTTOU   = 22
	_SIGIO     = 23
	_SIGQUIT   = 24
	_SIGTSTP   = 25
	_SIGTRAP   = 26
	_SIGWINCH  = 28
	_SIGXCPU   = 29
	_SIGXFSZ   = 30
	_SIGVTALRM = 31
	_SIGPROF   = 32

	_FPE_INTDIV = 31
	_FPE_INTOVF = 32
	_FPE_FLTDIV = 33
	_FPE_FLTOVF = 34
	_FPE_FLTUND = 35
	_FPE_FLTRES = 36
	_FPE_FLTINV = 37
	_FPE_FLTSUB = 38

	_EINTR     = 120
	_EAGAIN    = 112
	_ENOMEM    = 132
	_ETIMEDOUT = 1127

	_BUS_ADRALN = 71
	_BUS_ADRERR = 72
	_BUS_OBJERR = 73

	_SEGV_MAPERR    = 51
	_SEGV_ACCERR    = 52
	_SEGV_PROTECT   = 53
	_SEGV_ADDRESS   = 54
	_SEGV_SOFTLIMIT = 10059
)

type timespec struct {
	tv_sec  int64
	_       int32 // pad
	tv_nsec int32
}

func (ts *timespec) set_sec(x int64) {
	ts.tv_sec = x
}

func (ts *timespec) set_nsec(x int32) {
	ts.tv_nsec = x
}

type timeval struct {
	tv_sec  int64
	_       int32 // pad
	tv_usec int32
}

func (tv *timeval) set_usec(x int32) {
	tv.tv_usec = x
}

type sigactiont struct {
	sa_handler   uintptr // only use for SIG_* values
	sa_mask      uint64
	sa_flags     int32
	_            int32 // pad
	sa_sigaction uintptr
}

type siginfo struct {
	si_signo int32
	si_errno int32
	si_code  int32
	_        [7]uint32
	// below here is a union; si_addr is the only field we use
	si_addr uint64
}

type itimerval struct {
	it_interval timeval
	it_value    timeval
}

type sigaltstackt struct {
	ss_sp    *byte
	ss_size  uintptr
	ss_flags int32
	_        int32 // pad
}

type sigcontext struct {
	_        [10]uint64
	eyec     uint64
	_        [5]uint64
	gregs    [16]uint64
	aregs    [16]uint32
	fpregs   [16]uint64
	fpc      uint32
	_        uint32
	_        [13]uint64
	psw_addr uint64
	_        [14]uint64
}

type ucontext struct {
	uc_mcontext sigcontext
	uc_stack    sigaltstackt
	uc_sigmask  uint64
	uc_link     *ucontext
}

// types for pthread_mutex and pthread_cond
type pthread_mutex uint64
type pthread_cond uint64
type pthread_mutexattr uint64
type pthread_condattr uint64
type pthread_t uint64
type pthread_attr_t struct {
	_ [13]uint64
}

func mprotect(addr unsafe.Pointer, n uintptr, prot int32) int32
func __environ() **byte

func calloc(size uintptr) unsafe.Pointer
func malloc(size uintptr) unsafe.Pointer
func free(ptr unsafe.Pointer)

func __errno() *uint32
func __err2ad() *uint32
