// Copyright 2016 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"runtime/internal/sys"
	"unsafe"
)

const (
	_PAGE_SIZE = sys.PhysPageSize
	_EACCES    = 13
)

// NOTE: vec must be just 1 byte long here.
// Mincore returns ENOMEM if any of the pages are unmapped,
// but we want to know that all of the pages are unmapped.
// To make these the same, we can only ask about one page
// at a time. See golang.org/issue/7476.
var addrspace_vec [1]byte

func addrspace_free(v unsafe.Pointer, n uintptr) bool {
	var chunk uintptr
	for off := uintptr(0); off < n; off += chunk {
		chunk = _PAGE_SIZE * uintptr(len(addrspace_vec))
		if chunk > (n - off) {
			chunk = n - off
		}
		errval := mincore(unsafe.Pointer(uintptr(v)+off), chunk, &addrspace_vec[0])
		// ENOMEM means unmapped, which is what we want.
		// Anything else we assume means the pages are mapped.
		if errval != -_ENOMEM {
			return false
		}
	}
	return true
}

func mmap_fixed(v unsafe.Pointer, n uintptr, prot, flags, fd int32, offset uint32) unsafe.Pointer {
	p := mmap(v, n, prot, flags, fd, offset)
	// On some systems, mmap ignores v without
	// MAP_FIXED, so retry if the address space is free.
	if p != v && addrspace_free(v, n) {
		if uintptr(p) > 4096 {
			munmap(p, n)
		}
		p = mmap(v, n, prot, flags|_MAP_FIXED, fd, offset)
	}
	return p
}

// Don't split the stack as this method may be invoked without a valid G, which
// prevents us from allocating more stack.
//go:nosplit
func sysAlloc(n uintptr, sysStat *uint64) unsafe.Pointer {
	p := calloc(n)
	if p == nil {
		println("runtime: failed to allocate", n, "bytes")
		exit(2)
	}
	mSysStatInc(sysStat, n)
	return p
}

func sysUnused(v unsafe.Pointer, n uintptr) {
}

func sysUsed(v unsafe.Pointer, n uintptr) {
}

// Don't split the stack as this function may be invoked without a valid G,
// which prevents us from allocating more stack.
//go:nosplit
func sysFree(v unsafe.Pointer, n uintptr, sysStat *uint64) {
	mSysStatDec(sysStat, n)
	// TODO(mundaym): malloc.go says sysFree can be a no-op. Since we
	// don't know if v was allocated by mmap or calloc we can't free
	// it here.
	// free(v)
}

func sysFault(v unsafe.Pointer, n uintptr) {
	mmap(v, n, _PROT_NONE, _MAP_ANON|_MAP_PRIVATE|_MAP_FIXED, -1, 0)
}

func sysReserve(v unsafe.Pointer, n uintptr, reserved *bool) unsafe.Pointer {
	// z/OS currently has no 64-bit (above-the-bar (ATB)) support for mmap,
	// and mmap 31-bit (below-the-bar (BTB)) has limitations due to ESQA usage
	// (see z/OS UNIX System Services MAXMMAPAREA and MAXSHAREAPAGES settings).
	// It's possible to get large areas ATB, but users are restricted in how
	// much they can get (see MVS Storage Management MEMLIMIT setting).
	// Getting large areas guarded and then unguarding as needed is ideal,
	// but requires an authorized IARV64 caller to create private memory objects.
	p := malloc(n)
	if uintptr(p) < 4096 {
		return nil
	}
	*reserved = true
	return p
}

func sysMap(v unsafe.Pointer, n uintptr, reserved bool, sysStat *uint64) {
	mSysStatInc(sysStat, n)

	if !reserved {
		p := mmap_fixed(v, n, _PROT_READ|_PROT_WRITE, _MAP_ANON|_MAP_PRIVATE, -1, 0)
		if uintptr(p) == _ENOMEM {
			throw("runtime: out of memory")
		}
		if p != v {
			print("runtime: address space conflict: map(", v, ") = ", p, "\n")
			throw("runtime: address space conflict")
		}
		return
	}
	// Using malloc, not much we can do here (we could try to allocate again, or... ).
	// Let's just make sure we can write into the start and end of the range.
	*((*byte)(v)) = 0
	*((*byte)(unsafe.Pointer((uintptr)(v) + n - 1))) = 0
	if v == nil {
		throw("runtime: cannot map pages in arena address space with 0 location")
	}
}
