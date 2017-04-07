// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build s390x,zos

// Hand edited based on ztypes_linux_s390x.go
// TODO: auto-generate.

package syscall

const (
	sizeofPtr      = 0x8
	sizeofShort    = 0x2
	sizeofInt      = 0x4
	sizeofLong     = 0x8
	sizeofLongLong = 0x8
	PathMax        = 0x1000
)

const (
	SizeofSockaddrAny   = 128
	SizeofCmsghdr       = 12
	SizeofIPMreq        = 8
	SizeofIPv6Mreq      = 20
	SizeofICMPv6Filter  = 32
	SizeofIPv6MTUInfo   = 32
	SizeofLinger        = 8
	SizeofSockaddrInet4 = 16
	SizeofSockaddrInet6 = 28
)

type (
	_C_short     int16
	_C_int       int32
	_C_long      int64
	_C_long_long int64
)

type Timespec struct {
	Sec  int64
	_    [4]byte // pad
	Nsec int32
}

type Timeval struct {
	Sec  int64
	_    [4]byte // pad
	Usec int32
}

type Time_t int64

type RawSockaddrInet4 struct {
	Len    uint8
	Family uint8
	Port   uint16
	Addr   [4]byte /* in_addr */
	Zero   [8]uint8
}

type RawSockaddrInet6 struct {
	Len      uint8
	Family   uint8
	Port     uint16
	Flowinfo uint32
	Addr     [16]byte /* in6_addr */
	Scope_id uint32
}

type RawSockaddrUnix struct {
	Len    uint8
	Family uint8
	Path   [108]int8
}

type RawSockaddr struct {
	Len    uint8
	Family uint8
	Data   [14]uint8
}

type RawSockaddrAny struct {
	Addr RawSockaddr
	_    [112]uint8 // pad
}

type _Socklen uint32

type Linger struct {
	Onoff  int32
	Linger int32
}

type Iovec struct {
	Base *byte
	Len  uint64
}

type IPMreq struct {
	Multiaddr [4]byte /* in_addr */
	Interface [4]byte /* in_addr */
}

type IPv6Mreq struct {
	Multiaddr [16]byte /* in6_addr */
	Interface uint32
}

type Msghdr struct {
	Name       *byte
	Iov        *Iovec
	Control    *byte
	Flags      int32
	Namelen    int32
	Iovlen     int32
	Controllen int32
}

type Cmsghdr struct {
	Len   int32
	Level int32
	Type  int32
}

type Inet4Pktinfo struct {
	Addr    [4]byte /* in_addr */
	Ifindex uint32
}

type Inet6Pktinfo struct {
	Addr    [16]byte /* in6_addr */
	Ifindex uint32
}

type IPv6MTUInfo struct {
	Addr RawSockaddrInet6
	Mtu  uint32
}

type ICMPv6Filter struct {
	Data [8]uint32
}

type _Gid_t uint32

type Rusage struct {
	Utime Timeval
	Stime Timeval
}

type Rlimit struct {
	Cur uint64
	Max uint64
}

type Stat_t struct {
	_         [4]byte // eye catcher
	Length    uint16
	Version   uint16
	Mode      uint32 // really an int32
	Ino       uint32
	Dev       uint32
	Nlink     int32
	Uid       uint32
	Gid       uint32
	Size      int64
	Atim31    [4]byte
	Mtim31    [4]byte
	Ctim31    [4]byte
	Rdev      uint32
	Blksize   int32
	Creatim31 [4]byte
	AuditID   [16]byte
	_         [4]byte // rsrvd1
	CharsetID [12]byte
	Blocks    int64
	Genvalue  uint32
	Reftim31  [4]byte
	Fid       [8]byte
	Filefmt   byte
	Fspflag2  byte
	_         [2]byte // rsrvd2
	Ctimemsec int32
	Seclabel  [8]byte
	_         [4]byte // rsrvd3
	_         [4]byte // rsrvd4
	Atim      Time_t
	Mtim      Time_t
	Ctim      Time_t
	Creatim   Time_t
	Reftim    Time_t
	_         [24]byte // rsrvd5
}

type Dirent struct {
	Reclen uint16
	Namlen uint16
	Ino    uint32
	Extra  uintptr
	Name   [256]byte
}

// This struct is packed on z/OS so it can't be used directly.
type Flock_t struct {
	Type   int16
	Whence int16
	Start  int64
	Len    int64
	Pid    int32
}

type Termios struct {
	Iflag uint32
	Oflag uint32
	Cflag uint32
	Lflag uint32
	Cc    [11]uint8
}
