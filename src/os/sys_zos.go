// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build zos

package os

import (
	"syscall"
)

// supportsCloseOnExec reports whether the platform supports the
// O_CLOEXEC flag.
const supportsCloseOnExec = false

func hostname() (name string, err error) {
	var buffer [256]byte // Names are limited to 255 bytes.
	err = syscall.Gethostname(buffer[:])
	if err == nil {
		len := 0
		for i, c := range buffer[:] {
			if c == 0 {
				len = i
				break
			}
		}
		name = string(buffer[:len])
	}
	return
}
