// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build zos

package runtime

type sigTabT struct {
	flags int32
	name  string
}

var sigtable = [...]sigTabT{
	/* 0 */ {0, "SIGNONE: no trap"},
	/* 1 */ {_SigNotify + _SigKill, "SIGHUP: terminal line hangup"},
	/* 2 */ {_SigNotify + _SigKill, "SIGINT: interrupt"},
	/* 3 */ {_SigNotify + _SigThrow, "SIGABRT: abort"},
	/* 4 */ {_SigThrow + _SigUnblock, "SIGILL: illegal instruction"},
	/* 5 */ {_SigNotify, "SIGPOLL: pollable event"},
	/* 6 */ {_SigNotify, "SIGURG: urgent condition on socket"},
	/* 7 */ {0, "SIGSTOP: stop, unblockable"},
	/* 8 */ {_SigPanic + _SigUnblock, "SIGFPE: floating-point exception"},
	/* 9 */ {0, "SIGKILL: kill"},
	/* 10 */ {_SigPanic + _SigUnblock, "SIGBUS: bus error"},
	/* 11 */ {_SigPanic + _SigUnblock, "SIGSEGV: segmentation violation"},
	/* 12 */ {_SigNotify, "SIGSYS: bad system call"},
	/* 13 */ {_SigNotify, "SIGPIPE: write to broken pipe"},
	/* 14 */ {_SigNotify, "SIGALRM: alarm clock"},
	/* 15 */ {_SigNotify + _SigKill, "SIGTERM: termination"},
	/* 16 */ {_SigNotify, "SIGUSR1: user-defined signal 1"},
	/* 17 */ {_SigNotify, "SIGUSR2: user-defined signal 2"},
	/* 18 */ {0, "SIGABND: abnormal end"},
	/* 19 */ {_SigNotify + _SigDefault, "SIGCONT: continue"},
	/* 20 */ {_SigNotify + _SigUnblock, "SIGCHLD: child status has changed"},
	/* 21 */ {_SigNotify + _SigDefault, "SIGTTIN: background read from tty"},
	/* 22 */ {_SigNotify + _SigDefault, "SIGTTOU: background write to tty"},
	/* 23 */ {_SigNotify, "SIGIO: i/o now possible"},
	/* 24 */ {_SigNotify + _SigThrow, "SIGQUIT: quit"},
	/* 25 */ {_SigNotify + _SigDefault, "SIGTSTP: keyboard stop"},
	/* 26 */ {_SigThrow + _SigUnblock, "SIGTRAP: trace trap"},
	/* 27 */ {_SigThrow + _SigUnblock, "SIGIOERR: I/O error"},
	/* 28 */ {_SigNotify, "SIGWINCH: window size change"},
	/* 29 */ {_SigNotify, "SIGXCPU: cpu limit exceeded"},
	/* 30 */ {_SigNotify, "SIGXFSZ: file size limit exceeded"},
	/* 31 */ {_SigNotify, "SIGVTALRM: virtual alarm clock"},
	/* 32 */ {_SigNotify + _SigUnblock, "SIGPROF: profiling alarm clock"},
}
