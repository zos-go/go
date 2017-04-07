// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"io"
	"syscall"
)

const (
	blockSize = 4096
)

func (f *File) readdirnames(n int) (names []string, err error) {
	// We don't currently use dirinfo to buffer the directory info. Instead
	// we use bufp to store our offset. We need to do this because we can't
	// keep the DIR stream open once we leave this function (to avoid the
	// need for a finalizer), but we need to restart where we left off.
	if f.dirinfo == nil {
		f.dirinfo = new(dirInfo)
	}
	if f.dirinfo.bufp == -1 {
		// We have read all the possible files, return.
		if n > 0 {
			err = io.EOF
		}
		return
	}
	d, err := syscall.Opendir(f.Name())
	if err != nil {
		return
	}
	defer func() {
		err1 := syscall.Closedir(d)
		if err == nil {
			err = err1
		}
	}()
	if f.dirinfo.bufp != 0 {
		// Move to the point where we left off.
		syscall.Seekdir(d, f.dirinfo.bufp)
	}
	for i := 0; n <= 0 || i < n; i++ {
		var ent *syscall.Dirent
		ent, err = syscall.Readdir(d)
		if err != nil {
			break
		}
		if ent == nil {
			if len(names) == 0 && n > 0 {
				err = io.EOF
			}
			// Seekdir wraps around, so we need to explicitly tell
			// future invocations not to return any values.
			f.dirinfo.bufp = -1
			return
		}
		name := ent.NameString()
		if name != "." && name != ".." {
			names = append(names, name)
		} else {
			// Pretend dot and dot-dot don't exist.
			i--
		}
	}
	var terr error
	f.dirinfo.bufp, terr = syscall.Telldir(d)
	if err == nil {
		err = terr
	}
	return
}
