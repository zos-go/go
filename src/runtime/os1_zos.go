// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"runtime/internal/atomic"
	"unsafe"
)

//
// z/OS does not have a system call for FUTEX. Use SEMA for
// locking. Use pthread mutex and condition variable to
// implement the SEMA functions:
//
//          semacreate(mp *m)
//          semasleep(ns int64) int32
//          semawakeup(mp *m)
//
//go:nosplit
func semacreate(mp *m) {
}

//go:nosplit
func semasleep(ns int64) int32 {
	if ns >= 0 {
		ns += nanotime()
	}
	_g_ := getg()
	// Poll to see if it is time to wake up.
	for {
		if atomic.Cas(&_g_.m.waitsema, 1, 0) {
			return 0
		}
		if ns >= 0 && nanotime() >= ns {
			// timed out
			return -1
		}
		// TODO(mundaym): sched_yield? something better?
	}
	return 0
}

//go:nosplit
func semawakeup(mp *m) {
	for {
		// TODO(mundaym): not sure if the loop/atomics are actually necessary.
		if atomic.Cas(&mp.waitsema, 0, 1) {
			return
		}
	}
}

func getproccount() int32 {
	// TODO(mundaym): query the operating system for CPU count.
	return 1
}

// May run with m.p==nil, so write barriers are not allowed.
//go:nowritebarrier
func newosproc(mp *m, stk unsafe.Pointer) {
	var attr pthread_attr_t
	pthread_attr_init(&attr)

	// Leave stacklo=0 and set stackhi=size; mstack will do the rest.
	mp.g0.stack.hi = pthread_attr_getstacksize(&attr)

	// Disable signals during clone, so that the new thread starts
	// with signals disabled. It will enable them in minit.
	var isig, osig sigset
	pthread_sigmask(_SIG_SETMASK, &isig, &osig)

	var p pthread_t
	ret := pthread_create(&p, &attr, zosThreadEntry, uintptr(unsafe.Pointer(mp.g0)))

	pthread_sigmask(_SIG_SETMASK, &osig, nil)
	pthread_attr_destroy(&attr)

	if ret < 0 {
		print("runtime: failed to create new OS thread (have ", mcount(), " already; errno=", -ret, ")\n")
		throw("newosproc")
	}
}

var failallocatestack = []byte("runtime: failed to allocate stack for the new OS thread\n")
var failthreadcreate = []byte("runtime: failed to create new OS thread\n")

func osinit() {
	ncpu = getproccount()
}

var urandom_dev = []byte("/dev/urandom\x00")

func getRandomData(r []byte) {
	if startupRandomData != nil {
		n := copy(r, startupRandomData)
		extendRandom(r, n)
		return
	}
	fd := open(&urandom_dev[0], _O_RDONLY, 0)
	n := read(fd, unsafe.Pointer(&r[0]), int32(len(r)))
	closefd(fd)
	extendRandom(r, int(n))
}

func goenvs() {
	// based on goenvs_unix(), which expects the environ array to immediate follow the argv array
	// (argv_index is really generically indexing an array of pointers, not specific to argv)
	environ := __environ()
	n := int32(0)
	for argv_index(environ, n) != nil {
		n++
	}
	envs = make([]string, n)
	for i := int32(0); i < n; i++ {
		envs[i] = gostring(argv_index(environ, i))
	}
}

// Called to do synchronous initialization of Go code built with
// -buildmode=c-archive or -buildmode=c-shared.
// None of the Go runtime is initialized.
//go:nosplit
//go:nowritebarrierrec
func libpreinit() {
	initsig(true)
}

// Called to initialize a new m (including the bootstrap m).
// Called on the parent thread (main thread in case of bootstrap), can allocate memory.
func mpreinit(mp *m) {
	mp.gsignal = malg(2113536) // z/OS wants >= 0x202000 bytes (~2MiB).
	mp.gsignal.m = mp
}

//go:nosplit
func msigsave(mp *m) {
	smask := &mp.sigmask
	pthread_sigmask(_SIG_SETMASK, nil, smask)
}

//go:nosplit
func msigrestore(sigmask sigset) {
	pthread_sigmask(_SIG_SETMASK, &sigmask, nil)
}

//go:nosplit
func sigblock() {
	pthread_sigmask(_SIG_SETMASK, &sigset_all, nil)
}

func gettid() uint64

// Called to initialize a new m (including the bootstrap m).
// Called on the new thread, can not allocate memory.
func minit() {
	// Initialize signal handling.
	_g_ := getg()

	var st sigaltstackt
	sigaltstack(nil, &st)
	if st.ss_flags&_SS_DISABLE != 0 {
		signalstack(&_g_.m.gsignal.stack)
		_g_.m.newSigstack = true
	} else {
		// Use existing signal stack.
		stsp := uintptr(unsafe.Pointer(st.ss_sp))
		_g_.m.gsignal.stack.lo = stsp
		_g_.m.gsignal.stack.hi = stsp + st.ss_size
		_g_.m.gsignal.stackguard0 = stsp + _StackGuard
		_g_.m.gsignal.stackguard1 = stsp + _StackGuard
		_g_.m.gsignal.stackAlloc = st.ss_size
		_g_.m.newSigstack = false
	}

	// for debuggers, in case cgo created the thread
	_g_.m.procid = uint64(gettid())

	// restore signal mask from m.sigmask and unblock essential signals
	nmask := _g_.m.sigmask
	for i := range sigtable {
		if sigtable[i].flags&_SigUnblock != 0 {
			sigdelset(&nmask, i)
		}
	}
	pthread_sigmask(_SIG_SETMASK, &nmask, nil)
}

// Called from dropm to undo the effect of an minit.
//go:nosplit
func unminit() {
	if getg().m.newSigstack {
		signalstack(nil)
	}
}

func memlimit() uintptr {
	return 0
}

func sigreturn()
func sigtramp()

// sigFuncs is a table of env ptr/func ptr pairs for LE.
var sigFuncs [_NSIG + 1][2]uintptr

//go:nosplit
//go:nowritebarrierrec
func setsig(i int32, fn uintptr, restart bool) {
	var sa sigactiont
	memclr(unsafe.Pointer(&sa), unsafe.Sizeof(sa))
	sa.sa_flags = _SA_ONSTACK
	if restart {
		sa.sa_flags |= _SA_RESTART
	}
	sigfillset(&sa.sa_mask)
	if fn == funcPC(sighandler) {
		sa.sa_flags |= _SA_SIGINFO
		fn = funcPC(sigtramp)

		// LE expects a table entry, not just a func ptr.
		sigFuncs[i][1] = fn
		sa.sa_sigaction = uintptr(unsafe.Pointer(&sigFuncs[i]))
	} else if fn == _SIG_DFL || fn == _SIG_IGN {
		sa.sa_handler = fn
	} else {
		// A function set as a signal handler must be ok
		throw("unknown signal handler")
	}

	if sigaction(i, &sa, nil) != 0 {
		perror()
		throw("sigaction failure")
	}
}

//go:nosplit
//go:nowritebarrierrec
func setsigstack(i int32) {
	var sa sigactiont
	if sigaction(i, nil, &sa) != 0 {
		perror()
		throw("sigaction failure")
	}
	// SIG_DFL and SIG_IGN are sa_handler rather than sa_sigaction values.
	// sigtramp is always a sa_sigaction.
	if sa.sa_flags&_SA_SIGINFO == 0 || sa.sa_flags&_SA_ONSTACK != 0 {
		return
	}
	sa.sa_flags |= _SA_ONSTACK
	if sigaction(i, &sa, nil) != 0 {
		perror()
		throw("sigaction failure")
	}
}

//go:nosplit
//go:nowritebarrierrec
func getsig(i int32) uintptr {
	var sa sigactiont

	memclr(unsafe.Pointer(&sa), unsafe.Sizeof(sa))
	if v := sigaction(i, nil, &sa); v != 0 {
		perror()
		throw("sigaction read failure")
	}
	if sa.sa_sigaction == funcPC(sigtramp) {
		return funcPC(sighandler)
	}
	return sa.sa_sigaction
}

//go:nosplit
func signalstack(s *stack) {
	var st sigaltstackt
	if s == nil {
		st.ss_flags = _SS_DISABLE
	} else {
		st.ss_sp = (*byte)(unsafe.Pointer(s.lo))
		st.ss_size = s.hi - s.lo
		st.ss_flags = 0
	}
	sigaltstack(&st, nil)
}

//go:nosplit
//go:nowritebarrierrec
func updatesigmask(m sigmask) {
	var mask sigset
	sigcopyset(&mask, m)
	pthread_sigmask(_SIG_SETMASK, &mask, nil)
}

func unblocksig(sig int32) {
	var mask sigset
	sigaddset(&mask, int(sig))
	pthread_sigmask(_SIG_UNBLOCK, &mask, nil)
}

//go:nosplit
func Errno() uint32 {
	return *__errno()
}

//go:nosplit
func ErrnoJr() uint32 {
	return *__err2ad()
}
