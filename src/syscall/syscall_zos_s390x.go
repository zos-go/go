// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

import (
	"unsafe"
)

const (
	O_CLOEXEC = 0       // Dummy value (not supported).
	AF_LOCAL  = AF_UNIX // AF_LOCAL is an alias for AF_UNIX
)

func sendfile(outfd int, infd int, offset *int64, count int) (written int, err error) {
	// TODO(mundaym): read/write loop?
	panic("sendfile: not implemented")
}

func (d *Dirent) NameString() string {
	if d == nil {
		return ""
	}
	return string(d.Name[:d.Namlen])
}

func (sa *SockaddrInet4) sockaddr() (unsafe.Pointer, _Socklen, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return nil, 0, EINVAL
	}
	sa.raw.Len = SizeofSockaddrInet4
	sa.raw.Family = AF_INET
	p := (*[2]byte)(unsafe.Pointer(&sa.raw.Port))
	p[0] = byte(sa.Port >> 8)
	p[1] = byte(sa.Port)
	for i := 0; i < len(sa.Addr); i++ {
		sa.raw.Addr[i] = sa.Addr[i]
	}
	return unsafe.Pointer(&sa.raw), _Socklen(sa.raw.Len), nil
}

func (sa *SockaddrInet6) sockaddr() (unsafe.Pointer, _Socklen, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return nil, 0, EINVAL
	}
	sa.raw.Len = SizeofSockaddrInet6
	sa.raw.Family = AF_INET6
	p := (*[2]byte)(unsafe.Pointer(&sa.raw.Port))
	p[0] = byte(sa.Port >> 8)
	p[1] = byte(sa.Port)
	sa.raw.Scope_id = sa.ZoneId
	for i := 0; i < len(sa.Addr); i++ {
		sa.raw.Addr[i] = sa.Addr[i]
	}
	return unsafe.Pointer(&sa.raw), _Socklen(sa.raw.Len), nil
}

func (sa *SockaddrUnix) sockaddr() (unsafe.Pointer, _Socklen, error) {
	name := sa.Name
	n := len(name)
	if n >= len(sa.raw.Path) || n == 0 {
		return nil, 0, EINVAL
	}
	sa.raw.Len = byte(3 + n) // 2 for Family, Len; 1 for NUL
	sa.raw.Family = AF_UNIX
	for i := 0; i < n; i++ {
		sa.raw.Path[i] = int8(name[i])
	}
	return unsafe.Pointer(&sa.raw), _Socklen(sa.raw.Len), nil
}

func anyToSockaddr(rsa *RawSockaddrAny) (Sockaddr, error) {
	switch rsa.Addr.Family {
	case AF_UNIX:
		pp := (*RawSockaddrUnix)(unsafe.Pointer(rsa))
		sa := new(SockaddrUnix)
		// For z/OS, only replace NUL with @ when the
		// length is not zero.
		if pp.Len != 0 && pp.Path[0] == 0 {
			// "Abstract" Unix domain socket.
			// Rewrite leading NUL as @ for textual display.
			// (This is the standard convention.)
			// Not friendly to overwrite in place,
			// but the callers below don't care.
			pp.Path[0] = '@'
		}

		// Assume path ends at NUL.
		//
		// For z/OS, the length of the name is a field
		// in the structure. To be on the safe side, we
		// will still scan the name for a NUL but only
		// to the length provided in the structure.
		//
		// This is not technically the Linux semantics for
		// abstract Unix domain sockets--they are supposed
		// to be uninterpreted fixed-size binary blobs--but
		// everyone uses this convention.
		n := 0
		for n < int(pp.Len) && pp.Path[n] != 0 {
			n++
		}
		bytes := (*[10000]byte)(unsafe.Pointer(&pp.Path[0]))[0:n]
		sa.Name = string(bytes)
		return sa, nil

	case AF_INET:
		pp := (*RawSockaddrInet4)(unsafe.Pointer(rsa))
		sa := new(SockaddrInet4)
		p := (*[2]byte)(unsafe.Pointer(&pp.Port))
		sa.Port = int(p[0])<<8 + int(p[1])
		for i := 0; i < len(sa.Addr); i++ {
			sa.Addr[i] = pp.Addr[i]
		}
		return sa, nil

	case AF_INET6:
		pp := (*RawSockaddrInet6)(unsafe.Pointer(rsa))
		sa := new(SockaddrInet6)
		p := (*[2]byte)(unsafe.Pointer(&pp.Port))
		sa.Port = int(p[0])<<8 + int(p[1])
		sa.ZoneId = pp.Scope_id
		for i := 0; i < len(sa.Addr); i++ {
			sa.Addr[i] = pp.Addr[i]
		}
		return sa, nil
	}
	return nil, EAFNOSUPPORT
}

func Accept(fd int) (nfd int, sa Sockaddr, err error) {
	var rsa RawSockaddrAny
	var len _Socklen = SizeofSockaddrAny
	nfd, err = accept(fd, &rsa, &len)
	if err != nil {
		return
	}
	sa, err = anyToSockaddr(&rsa)
	if err != nil {
		Close(nfd)
		nfd = 0
	}
	return
}

func (iov *Iovec) SetLen(length int) {
	iov.Len = uint64(length)
}

func (msghdr *Msghdr) SetControllen(length int) {
	msghdr.Controllen = int32(length)
}

func (cmsg *Cmsghdr) SetLen(length int) {
	cmsg.Len = int32(length)
}

//sys   fcntl(fd int, cmd int, arg int) (val int, err error)
//sys	read(fd int, p []byte) (n int, err error)
//sys   readlen(fd int, buf *byte, nbuf int) (n int, err error) = SYS_READ
//sys	write(fd int, p []byte) (n int, err error)

//sys	accept(s int, rsa *RawSockaddrAny, addrlen *_Socklen) (fd int, err error) = SYS___ACCEPT_A
//sys	bind(s int, addr unsafe.Pointer, addrlen _Socklen) (err error) = SYS___BIND_A
//sys	connect(s int, addr unsafe.Pointer, addrlen _Socklen) (err error) = SYS___CONNECT_A
//sysnb	getgroups(n int, list *_Gid_t) (nn int, err error)
//sysnb	setgroups(n int, list *_Gid_t) (err error)
//sys	getsockopt(s int, level int, name int, val unsafe.Pointer, vallen *_Socklen) (err error)
//sys	setsockopt(s int, level int, name int, val unsafe.Pointer, vallen uintptr) (err error)
//sysnb	socket(domain int, typ int, proto int) (fd int, err error)
//sysnb	socketpair(domain int, typ int, proto int, fd *[2]int32) (err error)
//sysnb	getpeername(fd int, rsa *RawSockaddrAny, addrlen *_Socklen) (err error) = SYS___GETPEERNAME_A
//sysnb	getsockname(fd int, rsa *RawSockaddrAny, addrlen *_Socklen) (err error) = SYS___GETSOCKNAME_A
//sys	recvfrom(fd int, p []byte, flags int, from *RawSockaddrAny, fromlen *_Socklen) (n int, err error) = SYS___RECVFROM_A
//sys	sendto(s int, buf []byte, flags int, to unsafe.Pointer, addrlen _Socklen) (err error) = SYS___SENDTO_A
//sys	recvmsg(s int, msg *Msghdr, flags int) (n int, err error) = SYS___RECVMSG_A
//sys	sendmsg(s int, msg *Msghdr, flags int) (n int, err error) = SYS___SENDMSG_A

//sys   Chdir(path string) (err error) = SYS___CHDIR_A
//sys	Chown(path string, uid int, gid int) (err error) = SYS___CHOWN_A
//sys	Chmod(path string, mode uint32) (err error) = SYS___CHMOD_A
//sys	Dup(oldfd int) (fd int, err error)
//sys	Dup2(oldfd int, newfd int) (err error)
//sys	Exit(code int)
//sys	Fchdir(fd int) (err error)
//sys	Fchmod(fd int, mode uint32) (err error)
//sys	Fchown(fd int, uid int, gid int) (err error)
//sys	Fstat(fd int, stat *Stat_t) (err error)
//sys	Fsync(fd int) (err error)
//sys	Ftruncate(fd int, length int64) (err error)

func Close(fd int) (err error) {
	_, _, e1 := Syscall(SYS_CLOSE, uintptr(fd), 0, 0)
	for i := 0; e1 == EAGAIN && i < 10; i++ {
		_, _, _ = Syscall(SYS_USLEEP, uintptr(10), 0, 0)
		_, _, e1 = Syscall(SYS_CLOSE, uintptr(fd), 0, 0)
	}
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}

//sys   Gethostname(buf []byte) (err error) = SYS___GETHOSTNAME_A
//sysnb	Getegid() (egid int)
//sysnb	Geteuid() (uid int)
//sysnb	Getgid() (gid int)
//sysnb	Getpid() (pid int)
//sysnb	Getppid() (pid int)
//sys	Getpriority(which int, who int) (prio int, err error)
//sysnb	Getrlimit(resource int, rlim *Rlimit) (err error) = SYS_GETRLIMIT
//sysnb	Getuid() (uid int)
//sysnb	Kill(pid int, sig Signal) (err error)
//sys	Lchown(path string, uid int, gid int) (err error) = SYS___LCHOWN_A
//sys	Link(path string, link string) (err error) = SYS___LINK_A
//sys	Listen(s int, n int) (err error)
//sys	Lstat(path string, stat *Stat_t) (err error) = SYS___LSTAT_A
//sys	Mkdir(path string, mode uint32) (err error) = SYS___MKDIR_A
//sys	Mknod(path string, mode uint32, dev uint32) (err error) = SYS___MKNOD_A
//sys	Pread(fd int, p []byte, offset int64) (n int, err error)
//sys	Pwrite(fd int, p []byte, offset int64) (n int, err error)
//sys	Readlink(path string, buf []byte) (n int, err error) = SYS___READLINK_A
//sys	Rename(from string, to string) (err error) = SYS___RENAME_A
//sys	Rmdir(path string) (err error) = SYS___RMDIR_A
//sys   Seek(fd int, offset int64, whence int) (off int64, err error) = SYS_LSEEK
//sys	Setpriority(which int, who int, prio int) (err error)
//sysnb	Setrlimit(resource int, lim *Rlimit) (err error)
//sys	Shutdown(fd int, how int) (err error)
//sys	Stat(path string, stat *Stat_t) (err error) = SYS___STAT_A
//sys	Symlink(path string, link string) (err error) = SYS___SYMLINK_A
//sys	Truncate(path string, length int64) (err error) = SYS___TRUNCATE_A
//sys	Umask(mask int) (oldmask int, err error)
//sys	Unlink(path string) (err error) = SYS___UNLINK_A

//sys	open(path string, mode int, perm uint32) (fd int, err error) = SYS___OPEN_A

func Open(path string, mode int, perm uint32) (fd int, err error) {
	return open(path, mode, perm)
}

//sys	remove(path string) (err error)

func Remove(path string) error {
	return remove(path)
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

const ImplementsGetwd = true

func Getcwd(buf []byte) (n int, err error) {
	var p unsafe.Pointer
	if len(buf) > 0 {
		p = unsafe.Pointer(&buf[0])
	} else {
		p = unsafe.Pointer(&_zero)
	}
	_, _, e := Syscall(SYS___GETCWD_A, uintptr(p), uintptr(len(buf)), 0)
	n = clen(buf) + 1
	if e != 0 {
		err = errnoErr(e)
	}
	return
}

func Getwd() (wd string, err error) {
	var buf [PathMax]byte
	n, err := Getcwd(buf[0:])
	if err != nil {
		return "", err
	}
	// Getcwd returns the number of bytes written to buf, including the NUL.
	if n < 1 || n > len(buf) || buf[n-1] != 0 {
		return "", EINVAL
	}
	return string(buf[0 : n-1]), nil
}

func Getgroups() (gids []int, err error) {
	n, err := getgroups(0, nil)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}

	// Sanity check group count.  Max is 1<<16 on Linux.
	if n < 0 || n > 1<<20 {
		return nil, EINVAL
	}

	a := make([]_Gid_t, n)
	n, err = getgroups(n, &a[0])
	if err != nil {
		return nil, err
	}
	gids = make([]int, n)
	for i, v := range a[0:n] {
		gids[i] = int(v)
	}
	return
}

func Setgroups(gids []int) (err error) {
	if len(gids) == 0 {
		return setgroups(0, nil)
	}

	a := make([]_Gid_t, len(gids))
	for i, v := range gids {
		a[i] = _Gid_t(v)
	}
	return setgroups(len(a), &a[0])
}

type WaitStatus uint32

// Wait status is 7 bits at bottom, either 0 (exited),
// 0x7F (stopped), or a signal number that caused an exit.
// The 0x80 bit is whether there was a core dump.
// An extra number (exit code, signal causing a stop)
// is in the high bits.  At least that's the idea.
// There are various irregularities.  For example, the
// "continued" status is 0xFFFF, distinguishing itself
// from stopped via the core dump bit.

const (
	mask    = 0x7F
	core    = 0x80
	exited  = 0x00
	stopped = 0x7F
	shift   = 8
)

func (w WaitStatus) Exited() bool { return w&mask == exited }

func (w WaitStatus) Signaled() bool { return w&mask != stopped && w&mask != exited }

func (w WaitStatus) Stopped() bool { return w&0xFF == stopped }

func (w WaitStatus) Continued() bool { return w == 0xFFFF }

func (w WaitStatus) CoreDump() bool { return w.Signaled() && w&core != 0 }

func (w WaitStatus) ExitStatus() int {
	if !w.Exited() {
		return -1
	}
	return int(w>>shift) & 0xFF
}

func (w WaitStatus) Signal() Signal {
	if !w.Signaled() {
		return -1
	}
	return Signal(w & mask)
}

func (w WaitStatus) StopSignal() Signal {
	if !w.Stopped() {
		return -1
	}
	return Signal(w>>shift) & 0xFF
}

func (w WaitStatus) TrapCause() int { return -1 }

//sys	waitpid(pid int, wstatus *_C_int, options int) (wpid int, err error)

func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error) {
	// TODO(mundaym): z/OS doesn't have wait4. I don't think getrusage does what we want.
	// At the moment rusage will not be touched.
	var status _C_int
	wpid, err = waitpid(pid, &status, options)
	if wstatus != nil {
		*wstatus = WaitStatus(status)
	}
	return
}

func Getpagesize() int { return 4096 }

//sysnb	Gettimeofday(tv *Timeval) (err error)

func Time(t *Time_t) (tt Time_t, err error) {
	var tv Timeval
	err = Gettimeofday(&tv)
	if err != nil {
		return 0, err
	}
	if t != nil {
		*t = Time_t(tv.Sec)
	}
	return Time_t(tv.Sec), nil
}

func TimespecToNsec(ts Timespec) int64 { return int64(ts.Sec)*1e9 + int64(ts.Nsec) }

func NsecToTimespec(nsec int64) (ts Timespec) {
	ts.Sec = nsec / 1e9
	ts.Nsec = int32(nsec % 1e9)
	return
}

func TimevalToNsec(tv Timeval) int64 { return int64(tv.Sec)*1e9 + int64(tv.Usec)*1e3 }

func NsecToTimeval(nsec int64) (tv Timeval) {
	nsec += 999 // round up to microsecond
	tv.Sec = nsec / 1e9
	tv.Usec = int32(nsec % 1e9 / 1e3)
	return
}

//sysnb pipe(p *[2]_C_int) (err error)

func Pipe(p []int) (err error) {
	if len(p) != 2 {
		return EINVAL
	}
	var pp [2]_C_int
	err = pipe(&pp)
	p[0] = int(pp[0])
	p[1] = int(pp[1])
	return
}

//sys	utimes(path string, timeval *[2]Timeval) (err error) = SYS___UTIMES_A

func Utimes(path string, tv []Timeval) (err error) {
	if len(tv) != 2 {
		return EINVAL
	}
	return utimes(path, (*[2]Timeval)(unsafe.Pointer(&tv[0])))
}

func UtimesNano(path string, ts []Timespec) error {
	if len(ts) != 2 {
		return EINVAL
	}
	// Not as efficient as it could be because Timespec and
	// Timeval have different types in the different OSes
	tv := [2]Timeval{
		NsecToTimeval(TimespecToNsec(ts[0])),
		NsecToTimeval(TimespecToNsec(ts[1])),
	}
	return utimes(path, (*[2]Timeval)(unsafe.Pointer(&tv[0])))
}

func Getsockname(fd int) (sa Sockaddr, err error) {
	var rsa RawSockaddrAny
	var len _Socklen = SizeofSockaddrAny
	if err = getsockname(fd, &rsa, &len); err != nil {
		return
	}
	return anyToSockaddr(&rsa)
}

func GetsockoptInet4Addr(fd, level, opt int) (value [4]byte, err error) {
	vallen := _Socklen(4)
	err = getsockopt(fd, level, opt, unsafe.Pointer(&value[0]), &vallen)
	return value, err
}

func GetsockoptIPMreq(fd, level, opt int) (*IPMreq, error) {
	var value IPMreq
	vallen := _Socklen(SizeofIPMreq)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptIPv6Mreq(fd, level, opt int) (*IPv6Mreq, error) {
	var value IPv6Mreq
	vallen := _Socklen(SizeofIPv6Mreq)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptIPv6MTUInfo(fd, level, opt int) (*IPv6MTUInfo, error) {
	var value IPv6MTUInfo
	vallen := _Socklen(SizeofIPv6MTUInfo)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func GetsockoptICMPv6Filter(fd, level, opt int) (*ICMPv6Filter, error) {
	var value ICMPv6Filter
	vallen := _Socklen(SizeofICMPv6Filter)
	err := getsockopt(fd, level, opt, unsafe.Pointer(&value), &vallen)
	return &value, err
}

func Recvmsg(fd int, p, oob []byte, flags int) (n, oobn int, recvflags int, from Sockaddr, err error) {
	var msg Msghdr
	var rsa RawSockaddrAny
	msg.Name = (*byte)(unsafe.Pointer(&rsa))
	msg.Namelen = SizeofSockaddrAny
	var iov Iovec
	if len(p) > 0 {
		iov.Base = (*byte)(unsafe.Pointer(&p[0]))
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		// receive at least one normal byte
		if len(p) == 0 {
			iov.Base = &dummy
			iov.SetLen(1)
		}
		msg.Control = (*byte)(unsafe.Pointer(&oob[0]))
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1
	if n, err = recvmsg(fd, &msg, flags); err != nil {
		return
	}
	oobn = int(msg.Controllen)
	recvflags = int(msg.Flags)
	// source address is only specified if the socket is unconnected
	if rsa.Addr.Family != AF_UNSPEC {
		from, err = anyToSockaddr(&rsa)
	}
	return
}

func Sendmsg(fd int, p, oob []byte, to Sockaddr, flags int) (err error) {
	_, err = SendmsgN(fd, p, oob, to, flags)
	return
}

func SendmsgN(fd int, p, oob []byte, to Sockaddr, flags int) (n int, err error) {
	var ptr unsafe.Pointer
	var salen _Socklen
	if to != nil {
		var err error
		ptr, salen, err = to.sockaddr()
		if err != nil {
			return 0, err
		}
	}
	var msg Msghdr
	msg.Name = (*byte)(unsafe.Pointer(ptr))
	msg.Namelen = int32(salen)
	var iov Iovec
	if len(p) > 0 {
		iov.Base = (*byte)(unsafe.Pointer(&p[0]))
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		// send at least one normal byte
		if len(p) == 0 {
			iov.Base = &dummy
			iov.SetLen(1)
		}
		msg.Control = (*byte)(unsafe.Pointer(&oob[0]))
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1
	if n, err = sendmsg(fd, &msg, flags); err != nil {
		return 0, err
	}
	if len(oob) > 0 && len(p) == 0 {
		n = 0
	}
	return n, nil
}

func Opendir(name string) (uintptr, error) {
	p, err := BytePtrFromString(name)
	if err != nil {
		return 0, err
	}
	dir, _, e := Syscall(SYS___OPENDIR_A, uintptr(unsafe.Pointer(p)), 0, 0)
	use(unsafe.Pointer(p))
	if e != 0 {
		err = errnoErr(e)
	}
	return dir, err
}

// clearErrno resets the errno value to 0.
func clearErrno()

func Readdir(dir uintptr) (*Dirent, error) {
	var ent Dirent
	var res uintptr
	// __readdir_r_a returns errno at the end of the directory stream, rather than 0.
	// Therefore to avoid false positives we clear errno before calling it.
	clearErrno() // TODO(mundaym): check pre-emption rules.
	e, _, _ := Syscall(SYS___READDIR_R_A, dir, uintptr(unsafe.Pointer(&ent)), uintptr(unsafe.Pointer(&res)))
	var err error
	if e != 0 {
		err = errnoErr(Errno(e))
	}
	if res == 0 {
		return nil, err
	}
	return &ent, err
}

func Closedir(dir uintptr) error {
	_, _, e := Syscall(SYS_CLOSEDIR, dir, 0, 0)
	if e != 0 {
		return errnoErr(e)
	}
	return nil
}

func Seekdir(dir uintptr, pos int) {
	_, _, _ = Syscall(SYS_SEEKDIR, dir, uintptr(pos), 0)
}

func Telldir(dir uintptr) (int, error) {
	p, _, e := Syscall(SYS_TELLDIR, dir, 0, 0)
	pos := int(p)
	if pos == -1 {
		return pos, errnoErr(e)
	}
	return pos, nil
}

// FcntlFlock performs a fcntl syscall for the F_GETLK, F_SETLK or F_SETLKW command.
func FcntlFlock(fd uintptr, cmd int, lk *Flock_t) error {
	// struct flock is packed on z/OS. We can't emulate that in Go so
	// instead we pack it here.
	var flock [24]byte
	*(*int16)(unsafe.Pointer(&flock[0])) = lk.Type
	*(*int16)(unsafe.Pointer(&flock[2])) = lk.Whence
	*(*int64)(unsafe.Pointer(&flock[4])) = lk.Start
	*(*int64)(unsafe.Pointer(&flock[12])) = lk.Len
	*(*int32)(unsafe.Pointer(&flock[20])) = lk.Pid
	_, _, errno := Syscall(SYS_FCNTL, fd, uintptr(cmd), uintptr(unsafe.Pointer(&flock)))
	lk.Type = *(*int16)(unsafe.Pointer(&flock[0]))
	lk.Whence = *(*int16)(unsafe.Pointer(&flock[2]))
	lk.Start = *(*int64)(unsafe.Pointer(&flock[4]))
	lk.Len = *(*int64)(unsafe.Pointer(&flock[12]))
	lk.Pid = *(*int32)(unsafe.Pointer(&flock[20]))
	if errno == 0 {
		return nil
	}
	return errno
}

func Flock(fd int, how int) error {

	var flock_type int16
	var fcntl_cmd int

	switch how {
	case LOCK_SH | LOCK_NB:
		flock_type = F_RDLCK
		fcntl_cmd = F_SETLK
	case LOCK_EX | LOCK_NB:
		flock_type = F_WRLCK
		fcntl_cmd = F_SETLK
	case LOCK_EX:
		flock_type = F_WRLCK
		fcntl_cmd = F_SETLKW
	case LOCK_UN:
		flock_type = F_UNLCK
		fcntl_cmd = F_SETLKW
	default:
	}

	flock := Flock_t{
		Type:   int16(flock_type),
		Whence: int16(0),
		Start:  int64(0),
		Len:    int64(0),
		Pid:    int32(Getppid()),
	}

	err := FcntlFlock(uintptr(fd), fcntl_cmd, &flock)
	return err
}
