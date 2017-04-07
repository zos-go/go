// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build zos

package runtime

import (
	"unsafe"
)

const (
	_POLLRDNORM = 0x0001
	_POLLRDBAND = 0x0002
	_POLLWRNORM = 0x0004
	_POLLWRBAND = 0x0008
	_POLLIN     = (_POLLRDNORM | _POLLRDBAND)
	_POLLPRI    = 0x0010
	_POLLOUT    = _POLLWRNORM

	_POLLERR  = 0x0020
	_POLLHUP  = 0x0040
	_POLLNVAL = 0x0080

	_O_NONBLOCK = 0x04
	_F_SETFL    = 0x4

	_FD_CLOEXEC = 0x01
	_F_SETFD    = 0x2
)

type pollfd struct {
	fd      int32
	events  int16
	revents int16
}
type pipefds struct {
	readfd  int32
	writefd int32
}

func poll(pfd *pollfd, pfd_count uint32, timeout int32) int32
func pipe(fds *pipefds) int32
func fcntl(fd, cmd, arg int32) int32

var (
	pfd      []pollfd        // points to pollfd array
	pdd      []*pollDesc     // points to go pollDesc array
	max_pfd  uint32      = 0 // current size of pfd array
	max_pdd  uint32      = 0 // current size of pdd array
	next_pfd uint32      = 0 // Next entry to use
	pfd_lock mutex           // Lock to serialize updates to variables above
	// These variable are used by netpoll() for nonblocking calls
	pfd2      []pollfd    // points to pollfd array used for poll()
	pdd2      []*pollDesc // points to go pollDesc array used for poll()
	pfd2_lock mutex       // Lock for serailizing netpoll() calls
	// These variable are used by netpoll() for blocking calls
	pfd3      []pollfd    // points to pollfd array used for poll()
	pdd3      []*pollDesc // points to go pollDesc array used for poll()
	pfd3_lock mutex       // Lock for serailizing netpoll() calls
	poll_pipe pipefds     // Pipe used to wakeup blocking poll()

)

func netpollinit() {
	if max_pfd == 0 {
		pfd = make([]pollfd, 1024, 1024)
		if cap(pfd) == 0 {
			throw("make pfd failed")
		}
		pfd2 = make([]pollfd, 1024, 1024)
		if cap(pfd2) == 0 {
			throw("make pfd2 failed")
		}
		pfd3 = make([]pollfd, 1024, 1024)
		if cap(pfd3) == 0 {
			throw("make pfd3 failed")
		}
		max_pfd = 1024
	}
	if max_pdd == 0 {
		pdd = make([]*pollDesc, 1024, 1024)
		if pdd == nil {
			throw("make pdd failed")
		}
		pdd2 = make([]*pollDesc, 1024, 1024)
		if pdd2 == nil {
			throw("make pdd2 failed")
		}
		pdd3 = make([]*pollDesc, 1024, 1024)
		if pdd3 == nil {
			throw("make pdd3 failed")
		}
		max_pdd = 1024
	}
	if poll_pipe.readfd == 0 && poll_pipe.writefd == 0 {
		n := pipe(&poll_pipe)
		if n < 0 {
			e := Errno()
			println("runtime: pipe failed with", e)
			throw("pipe failed")
		}
		pfd[0].fd = poll_pipe.readfd
		next_pfd = 1
		// Set read end of pipe to nonblocking
		if err := fcntl(poll_pipe.readfd, _F_SETFL, _O_NONBLOCK); err != 0 {
			e := Errno()
			println("runtime: fcntl failed with", e)
			throw("fcntl failed")
		}
		// Set read end of pipe to close on exec
		if err := fcntl(poll_pipe.readfd, _F_SETFD, _FD_CLOEXEC); err != 0 {
			e := Errno()
			println("runtime: fcntl failed with", e)
			throw("fcntl failed")
		}
		// Set write end of pipe to nonblocking
		if err := fcntl(poll_pipe.writefd, _F_SETFL, _O_NONBLOCK); err != 0 {
			e := Errno()
			println("runtime: fcntl failed with", e)
			throw("fcntl failed")
		}
		// Set write end of pipe to close on exec
		if err := fcntl(poll_pipe.writefd, _F_SETFD, _FD_CLOEXEC); err != 0 {
			e := Errno()
			println("runtime: fcntl failed with", e)
			throw("fcntl failed")
		}
	}

	return
}

func netpollopen(fd uintptr, pd *pollDesc) int32 {
	// TODO(pathealy): Need to add code for expanding when next_pfd equals max_pfd
	//
	// Handling level trigger when edge trigger is needed:
	// 	Don't ask for events until someone is waiting
	//	Enable event in netpollarm() called by pollwait()
	// 	Disable event when someone is readied in netpoll()
	//
	lock(&pfd_lock)
	if next_pfd == max_pfd {
		unlock(&pfd_lock)
		println("runtime: poll array exceeded capacity")
		throw("poll array exceeded capacity")
	}

	pfd[next_pfd].events = 0
	pfd[next_pfd].revents = 0
	pfd[next_pfd].fd = int32(fd)
	pdd[next_pfd] = pd
	next_pfd++
	unlock(&pfd_lock)

	return 0
}

func netpollclose(fd uintptr) int32 {
	// Do we need to return an error if fd is not found?
	// Does fd only appear once in list?

	lock(&pfd_lock)
	for i := uint32(0); i < next_pfd; i++ {
		if pfd[i].fd == int32(fd) {
			// Write to pipe to wakeup any blocked poll()
			write(uintptr(poll_pipe.writefd), unsafe.Pointer(&pdd[i]), int32(unsafe.Sizeof(pdd[0])))
			osyield() // Yield so any blocked poller wakes up before close is done
			// Decrement count of entries in tables
			next_pfd--
			// If not deleting the last entry, compress the tables
			if i < next_pfd {
				memmove(unsafe.Pointer(&pfd[i]),
					unsafe.Pointer(&pfd[i+1]),
					uintptr(next_pfd-i)*unsafe.Sizeof(pfd[0]))
				memmove(unsafe.Pointer(&pdd[i]),
					unsafe.Pointer(&pdd[i+1]),
					uintptr(next_pfd-i)*unsafe.Sizeof(pdd[0]))
			}
		}
	}
	unlock(&pfd_lock)
	return 0
}

func netpollarm(pd *pollDesc, mode int) {
	switch mode {
	case 'r':
		netpollupdate(pd, _POLLIN, 0)
	case 'w':
		netpollupdate(pd, _POLLOUT, 0)
	default:
		throw("netpollarm: bad mode")
	}
}

func netpollupdate(pd *pollDesc, set, clear uint32) {
	// Can pd occur more than once?
	lock(&pfd_lock)

	for i := uint32(0); i < next_pfd; i++ {
		if pdd[i] == pd {
			old := pfd[i].events
			pfd[i].events = (old & ^int16(clear)) | int16(set)
		}
	}

	// Write to pipe to wakeup any blocked poll()
	write(uintptr(poll_pipe.writefd), unsafe.Pointer(&pd), int32(unsafe.Sizeof(pd)))

	unlock(&pfd_lock)
	return
}

// polls for ready network connections
// returns list of goroutines that become runnable
func netpoll(block bool) *g {

	var (
		gp            guintptr
		waitms        int32
		local_mutex_p *mutex      // point to mutex used to serialize with other netpoll() calls
		local_pfd     []pollfd    // pfd slice to use for poll()
		local_pdd     []*pollDesc // pdd slice to use for poll()
	)

	if !block {
		waitms = 0
		local_mutex_p = &pfd2_lock // Use lock for nonblocking netpoll() calls
		local_pfd = pfd2
		local_pdd = pdd2
	} else {
		waitms = -1
		local_mutex_p = &pfd3_lock // Use lock for blocking netpoll() calls
		local_pfd = pfd3
		local_pdd = pdd3
	}
retry:
	lock(local_mutex_p) // Serialize with other netpoll() calls
	lock(&pfd_lock)     // Serialize with updates to tables

	if next_pfd == 0 { // unlock and exit if table is empty
		unlock(&pfd_lock)
		unlock(local_mutex_p)
		return gp.ptr()
	}

	// Copy pfd table to private copy for calling poll()
	memmove(unsafe.Pointer(&local_pfd[0]), unsafe.Pointer(&pfd[0]), uintptr(next_pfd)*unsafe.Sizeof(pfd[0]))

	next_pfd2 := next_pfd // Save number of inuse entries in tables

	// If blocking, set up to poll the read end of poll_pipe and empty the pipe.
	if block {
		local_pfd[0].events = _POLLIN

		// Clear out any data in the poll_pipe. Use local pdd table as a buffer.
		for read_cnt := int32(int32(len(local_pdd)) * int32(unsafe.Sizeof(local_pdd[0]))); read_cnt == int32(len(local_pdd))*int32(unsafe.Sizeof(local_pdd[0])); read_cnt = read(poll_pipe.readfd,
			unsafe.Pointer(&local_pdd[0]),
			int32(len(local_pdd))*int32(unsafe.Sizeof(local_pdd[0]))) {
		}
	}

	// Copy pdd table to private copy for calling poll()
	memmove(unsafe.Pointer(&local_pdd[0]), unsafe.Pointer(&pdd[0]), uintptr(next_pfd)*unsafe.Sizeof(pdd[0]))

	unlock(&pfd_lock)

	// poll with only file descriptors, no message queues
	n := poll(&local_pfd[0], next_pfd2, waitms)
	if n < 0 {
		if e := Errno(); e != _EINTR {
			unlock(local_mutex_p)
			println("runtime: poll failed with", e)
			throw("poll failed")
		}
		unlock(local_mutex_p)
		goto retry
	} else if (n == 0) || (block && (n == 1) && local_pfd[0].revents != 0) {
		// Nothing to process
	} else {
		// Loop through pollfd array. Stop when all n events are processed.
		//
		// Skip the zero entry which is the pipe used to wakeup a blocked
		// poll() when the list of events to poll has been updated.
		for i := uint32(1); i < next_pfd2; i++ {
			ev := &local_pfd[i]
			if ev.revents == 0 {
				continue
			}
			var mode, clear int32
			if ev.revents&(_POLLIN|_POLLHUP|_POLLERR) != 0 {
				mode += 'r'
				clear |= _POLLIN
			}
			if ev.revents&(_POLLOUT|_POLLHUP|_POLLERR) != 0 {
				mode += 'w'
				clear |= _POLLOUT
			}
			if mode != 0 {
				pd := local_pdd[i]
				netpollupdate(pd, 0, uint32(clear))
				netpollready(&gp, pd, mode)
				n--
				if n <= 0 {
					break
				}
			}
		}
	}

	if block && gp == 0 {
		unlock(local_mutex_p)
		goto retry
	}
	unlock(local_mutex_p)

	return gp.ptr()
}
