// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

type mOS struct {
	// semaphore for parking on locks
	waitsema uint32
}

//go:noescape
func pthread_cond_init(condaddr *pthread_cond, condattraddr *pthread_condattr) int32

//go:noescape
func pthread_cond_signal(condaddr *pthread_cond) int32

//go:noescape
func pthread_cond_timedwait(condaddr *pthread_cond, mutexaddr *pthread_mutex, timeaddr *timespec) int32

//go:noescape
func pthread_cond_wait(condaddr *pthread_cond, mutexaddr *pthread_mutex) int32

//go:noescape
func pthread_mutex_init(mutexaddr *pthread_mutex, mutexattraddr *pthread_mutexattr) int32

//go:noescape
func pthread_mutex_lock(mutexaddr *pthread_mutex) int32

//go:noescape
func pthread_mutex_unlock(mutexaddr *pthread_mutex) int32

//go:noescape
func pthread_create(p *pthread_t, attr *pthread_attr_t, fn func(uintptr) uintptr, arg uintptr) int32

func zosThreadEntry(g uintptr) uintptr

//go:noescape
func pthread_attr_init(attr *pthread_attr_t) int32

//go:noescape
func pthread_attr_destroy(attr *pthread_attr_t) int32

//go:noescape
func pthread_attr_getstacksize(attr *pthread_attr_t) uintptr

//go:noescape
func pthread_sigmask(how int32, old, new *sigset) int32

//go:noescape
func sigaction(sig int32, new, old *sigactiont) int32

//go:noescape
func sigaltstack(new, old *sigaltstackt)

//go:noescape
func setitimer(mode int32, new, old *itimerval)

func raise(sig int32)
func raiseproc(sig int32)
func osyield()
func perror()
