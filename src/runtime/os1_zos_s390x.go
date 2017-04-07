// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

var sigset_all = sigset(^uint64(0))

func sigaddset(mask *sigset, i int) {
	if i > 64 {
		throw("unexpected signal greater than 64")
	}
	*mask |= sigset(uint64(1<<63) >> uint(i))
}

func sigdelset(mask *sigset, i int) {
	if i > 64 {
		throw("unexpected signal greater than 64")
	}
	*mask &^= sigset(uint64(1<<63) >> uint(i))
}

func sigfillset(mask *uint64) {
	*mask = ^uint64(0)
}

func sigcopyset(mask *sigset, m sigmask) {
	for i := 0; i < 32; i++ {
		if m[0]&(1<<uint(i)) != 0 {
			sigaddset(mask, i)
		} else {
			sigdelset(mask, i)
		}
		if m[1]&(1<<uint(i)) != 0 {
			sigaddset(mask, i+32)
		} else {
			sigdelset(mask, i+32)
		}
	}
}
