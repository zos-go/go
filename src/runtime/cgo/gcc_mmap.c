// Copyright 2015-2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build cgo

// +build linux,amd64 zos

#include <errno.h>
#include <stdint.h>
// chwan -
// HACK this macro is required for xlc. xlcdev does not need it
#define __SUSV3_XSI
// HACK end
#include <sys/mman.h>

void *
x_cgo_mmap(void *addr, uintptr_t length, int32_t prot, int32_t flags, int32_t fd, uint32_t offset) {
	void *p;

	p = mmap(addr, length, prot, flags, fd, offset);
	if (p == MAP_FAILED) {
		/* This is what the Go code expects on failure.  */
		p = (void *) (uintptr_t) errno;
	}
	return p;
}
