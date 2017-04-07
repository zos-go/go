// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// chwan -
// HACK - this macro is required for xlc. xlcdev does not need it.
#define _UNIX03_THREADS
// HACK end
#include <pthread.h>
#include <string.h>
#include <signal.h>
#include "libcgo.h"

static void *threadentry(void*);

void (*x_cgo_inittls)(void **tlsg, void **tlsbase);
static void (*setg_gcc)(void*);

void
x_cgo_init(G *g, void (*setg)(void*), void **tlsbase)
{
	pthread_attr_t attr;
	size_t size;

	setg_gcc = setg;
	pthread_attr_init(&attr);
	pthread_attr_getstacksize(&attr, &size);
	g->stacklo = (uintptr)&attr - size + 4096;
	pthread_attr_destroy(&attr);
}

void
_cgo_sys_thread_start(ThreadStart *ts)
{
	pthread_attr_t attr;
	sigset_t ign, oset;
	pthread_t p;
	size_t size;
	int err;

	sigfillset(&ign);
	pthread_sigmask(SIG_SETMASK, &ign, &oset);

	pthread_attr_init(&attr);
	pthread_attr_getstacksize(&attr, &size);
	// Leave stacklo=0 and set stackhi=size; mstack will do the rest.
	ts->g->stackhi = size;
	err = pthread_create(&p, &attr, threadentry, ts);

	pthread_sigmask(SIG_SETMASK, &oset, nil);

	if (err != 0) {
		fatalf("pthread_create failed: %s", strerror(err));
	}
}

extern void crosscall_s390x(void (*fn)(void), void *g);

static void*
threadentry(void *v)
{
	ThreadStart ts;

	ts = *(ThreadStart*)v;
	free(v);

	// Save g for this thread in C TLS
	// chwan -
	// HACK - we don't know what to do with setg_gcc for z/OS yet
//	setg_gcc((void*)ts.g);
	// HACK end

	crosscall_s390x(ts.fn, (void*)ts.g);
	return nil;
}
