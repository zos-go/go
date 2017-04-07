// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

var sigset_all = sigset(^uint64(0))

func sigaddset(mask *sigset, i int) {
	if i > 64 {
		throw("unexpected signal greater than 64")
	}
	*mask |= 1 << (uint(i) - 1)
}

func sigdelset(mask *sigset, i int) {
	if i > 64 {
		throw("unexpected signal greater than 64")
	}
	*mask &^= 1 << (uint(i) - 1)
}

func sigfillset(mask *uint64) {
	*mask = ^uint64(0)
}

func sigcopyset(mask *sigset, m sigmask) {
	*mask = sigset(uint64(m[0]) | uint64(m[1])<<32)
}
