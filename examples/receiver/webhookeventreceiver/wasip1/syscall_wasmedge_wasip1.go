//go:build wasip1

package wasip1

// This file contains the definition of host imports compatible with the socket
// extensions from wasmedge v0.12+.

import (
	"encoding/binary"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

const (
	AF_UNSPEC = iota
	AF_INET
	AF_INET6
	AF_UNIX
)

const (
	SOCK_ANY = iota
	SOCK_DGRAM
	SOCK_STREAM
)

const (
	SOL_SOCKET = iota
)

const (
	SO_REUSEADDR = iota
	_
	SO_ERROR
	_
	SO_BROADCAST
)

const (
	AI_PASSIVE = 1 << iota
	_
	AI_NUMERICHOST
	AI_NUMERICSERV
)

const (
	IPPROTO_IP = iota
	IPPROTO_TCP
	IPPROTO_UDP
)

type sockaddr interface {
	sockaddr() (unsafe.Pointer, error)
	sockport() int
}

type sockaddrInet4 struct {
	addr [4]byte
	port uint32
	raw  addressBuffer
}

func (s *sockaddrInet4) sockaddr() (unsafe.Pointer, error) {
	s.raw.bufLen = 4
	s.raw.buf = uintptr32(uintptr(unsafe.Pointer(&s.addr)))
	return unsafe.Pointer(&s.raw), nil
}

func (s *sockaddrInet4) sockport() int {
	return int(s.port)
}

type sockaddrInet6 struct {
	addr [16]byte
	port uint32
	zone uint32
	raw  addressBuffer
}

func (s *sockaddrInet6) sockaddr() (unsafe.Pointer, error) {
	if s.zone != 0 {
		return nil, syscall.ENOTSUP
	}
	s.raw.bufLen = 16
	s.raw.buf = uintptr32(uintptr(unsafe.Pointer(&s.addr)))
	return unsafe.Pointer(&s.raw), nil
}

func (s *sockaddrInet6) sockport() int {
	return int(s.port)
}

type sockaddrUnix struct {
	name string

	raw rawSockaddrAny
	buf addressBuffer
}

func (s *sockaddrUnix) sockaddr() (unsafe.Pointer, error) {
	s.raw.family = AF_UNIX
	if len(s.name) >= len(s.raw.addr)-1 {
		return nil, syscall.EINVAL
	}
	copy(s.raw.addr[:], s.name)
	s.raw.addr[len(s.name)] = 0
	s.buf.bufLen = 128
	s.buf.buf = uintptr32(uintptr(unsafe.Pointer(&s.raw)))
	return unsafe.Pointer(&s.buf), nil
}

func (s *sockaddrUnix) sockport() int {
	return 0
}

type uintptr32 = uint32
type size = uint32

type addressBuffer struct {
	buf    uintptr32
	bufLen size
}

type rawSockaddrAny struct {
	family uint16
	addr   [126]byte
}

//go:wasmimport wasi_snapshot_preview1 sock_open
//go:noescape
func sock_open(af int32, socktype int32, fd unsafe.Pointer) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_bind
//go:noescape
func sock_bind(fd int32, addr unsafe.Pointer, port uint32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_listen
//go:noescape
func sock_listen(fd int32, backlog int32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_connect
//go:noescape
func sock_connect(fd int32, addr unsafe.Pointer, port uint32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getsockopt
//go:noescape
func sock_getsockopt(fd int32, level uint32, name uint32, value unsafe.Pointer, valueLen uint32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_setsockopt
//go:noescape
func sock_setsockopt(fd int32, level uint32, name uint32, value unsafe.Pointer, valueLen uint32) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getlocaladdr
//go:noescape
func sock_getlocaladdr(fd int32, addr unsafe.Pointer, port unsafe.Pointer) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getpeeraddr
//go:noescape
func sock_getpeeraddr(fd int32, addr unsafe.Pointer, port unsafe.Pointer) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_recv_from
//go:noescape
func sock_recv_from(
	fd int32,
	iovs unsafe.Pointer,
	iovsCount int32,
	addr unsafe.Pointer,
	iflags int32,
	port unsafe.Pointer,
	nread unsafe.Pointer,
	oflags unsafe.Pointer,
) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_send_to
//go:noescape
func sock_send_to(
	fd int32,
	iovs unsafe.Pointer,
	iovsCount int32,
	addr unsafe.Pointer,
	port int32,
	flags int32,
	nwritten unsafe.Pointer,
) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_getaddrinfo
//go:noescape
func sock_getaddrinfo(
	node unsafe.Pointer,
	nodeLen uint32,
	service unsafe.Pointer,
	serviceLen uint32,
	hints unsafe.Pointer,
	res unsafe.Pointer,
	maxResLen uint32,
	resLen unsafe.Pointer,
) syscall.Errno

//go:wasmimport wasi_snapshot_preview1 sock_shutdown
func sock_shutdown(fd, how int32) syscall.Errno

func socket(proto, sotype, unused int) (fd int, err error) {
	var newfd int32
	errno := sock_open(int32(proto), int32(sotype), unsafe.Pointer(&newfd))
	if errno != 0 {
		return -1, errno
	}
	return int(newfd), nil
}

func bind(fd int, sa sockaddr) error {
	rawaddr, err := sa.sockaddr()
	if err != nil {
		return err
	}
	errno := sock_bind(int32(fd), rawaddr, uint32(sa.sockport()))
	runtime.KeepAlive(sa)
	if errno != 0 {
		return errno
	}
	return nil
}

func listen(fd int, backlog int) error {
	if errno := sock_listen(int32(fd), int32(backlog)); errno != 0 {
		return errno
	}
	return nil
}

func connect(fd int, sa sockaddr) error {
	rawaddr, err := sa.sockaddr()
	if err != nil {
		return err
	}
	errno := sock_connect(int32(fd), rawaddr, uint32(sa.sockport()))
	runtime.KeepAlive(sa)
	if errno != 0 {
		return errno
	}
	return nil
}

type iovec struct {
	ptr uintptr32
	len uint32
}

func recvfrom(fd int, iovs [][]byte, flags int32) (n int, addr rawSockaddrAny, port, oflags int32, err error) {
	iovsBuf := make([]iovec, 0, 8)
	for _, iov := range iovs {
		iovsBuf = append(iovsBuf, iovec{
			ptr: uintptr32(uintptr(unsafe.Pointer(unsafe.SliceData(iov)))),
			len: uint32(len(iov)),
		})
	}
	addrBuf := addressBuffer{
		buf:    uintptr32(uintptr(unsafe.Pointer(&addr))),
		bufLen: uint32(unsafe.Sizeof(addr)),
	}
	nread := int32(0)
	errno := sock_recv_from(
		int32(fd),
		unsafe.Pointer(unsafe.SliceData(iovsBuf)),
		int32(len(iovsBuf)),
		unsafe.Pointer(&addrBuf),
		flags,
		unsafe.Pointer(&port),
		unsafe.Pointer(&nread),
		unsafe.Pointer(&oflags),
	)
	if errno != 0 {
		return int(nread), addr, port, oflags, errno
	}
	runtime.KeepAlive(addrBuf)
	runtime.KeepAlive(iovsBuf)
	runtime.KeepAlive(iovs)
	return int(nread), addr, port, oflags, nil
}

func sendto(fd int, iovs [][]byte, addr rawSockaddrAny, port, flags int32) (int, error) {
	iovsBuf := make([]iovec, 0, 8)
	for _, iov := range iovs {
		iovsBuf = append(iovsBuf, iovec{
			ptr: uintptr32(uintptr(unsafe.Pointer(unsafe.SliceData(iov)))),
			len: uint32(len(iov)),
		})
	}
	addrBuf := addressBuffer{
		buf:    uintptr32(uintptr(unsafe.Pointer(&addr))),
		bufLen: uint32(unsafe.Sizeof(addr)),
	}
	nwritten := int32(0)
	errno := sock_send_to(
		int32(fd),
		unsafe.Pointer(unsafe.SliceData(iovsBuf)),
		int32(len(iovsBuf)),
		unsafe.Pointer(&addrBuf),
		port,
		flags,
		unsafe.Pointer(&nwritten),
	)
	if errno != 0 {
		return int(nwritten), errno
	}
	runtime.KeepAlive(addrBuf)
	runtime.KeepAlive(iovsBuf)
	runtime.KeepAlive(iovs)
	return int(nwritten), nil
}

func shutdown(fd, how int) error {
	if errno := sock_shutdown(int32(fd), int32(how)); errno != 0 {
		return errno
	}
	return nil
}

func getsockopt(fd, level, opt int) (value int, err error) {
	var n int32
	errno := sock_getsockopt(int32(fd), uint32(level), uint32(opt), unsafe.Pointer(&n), 4)
	if errno != 0 {
		return 0, errno
	}
	return int(n), nil
}

func setsockopt(fd, level, opt int, value int) error {
	var n = int32(value)
	errno := sock_setsockopt(int32(fd), uint32(level), uint32(opt), unsafe.Pointer(&n), 4)
	if errno != 0 {
		return errno
	}
	return nil
}

func getsockname(fd int) (sa sockaddr, err error) {
	var rsa rawSockaddrAny
	buf := addressBuffer{
		buf:    uintptr32(uintptr(unsafe.Pointer(&rsa))),
		bufLen: uint32(unsafe.Sizeof(rsa)),
	}
	var port uint32
	errno := sock_getlocaladdr(int32(fd), unsafe.Pointer(&buf), unsafe.Pointer(&port))
	if errno != 0 {
		return nil, errno
	}
	return anyToSockaddr(&rsa, port)
}

func getpeername(fd int) (sockaddr, error) {
	var rsa rawSockaddrAny
	buf := addressBuffer{
		buf:    uintptr32(uintptr(unsafe.Pointer(&rsa))),
		bufLen: uint32(unsafe.Sizeof(rsa)),
	}
	var port uint32
	errno := sock_getpeeraddr(int32(fd), unsafe.Pointer(&buf), unsafe.Pointer(&port))
	if errno != 0 {
		return nil, errno
	}
	return anyToSockaddr(&rsa, port)
}

func anyToSockaddr(rsa *rawSockaddrAny, port uint32) (sockaddr, error) {
	switch rsa.family {
	case AF_INET:
		addr := sockaddrInet4{port: port}
		copy(addr.addr[:], rsa.addr[:])
		return &addr, nil
	case AF_INET6:
		addr := sockaddrInet6{port: port}
		copy(addr.addr[:], rsa.addr[:])
		return &addr, nil
	case AF_UNIX:
		addr := sockaddrUnix{}
		addr.name = string(rsa.addr[:strlen(rsa.addr[:])])
		return &addr, nil
	default:
		return nil, syscall.ENOTSUP
	}
}

func strlen(b []byte) (n int) {
	for n < len(b) && b[n] != 0 {
		n++
	}
	return n
}

// https://github.com/WasmEdge/WasmEdge/blob/434e1fb4690/thirdparty/wasi/api.hpp#L1885
type sockAddrInfo struct {
	ai_flags        uint16
	ai_family       uint8
	ai_socktype     uint8
	ai_protocol     uint32
	ai_addrlen      uint32
	ai_addr         uintptr32 // *sockAddr
	ai_canonname    uintptr32 // null-terminated string
	ai_canonnamelen uint32
	ai_next         uintptr32 // *sockAddrInfo
}

type sockAddr struct {
	sa_family   uint32
	sa_data_len uint32
	sa_data     uintptr32
	_           [4]byte
}

type addrInfo struct {
	flags      int
	family     int
	socketType int
	protocol   int
	address    sockaddr
	// canonicalName string

	sockAddrInfo
	sockAddr
	sockData  [26]byte
	cannoname [30]byte
	inet4addr sockaddrInet4
	inet6addr sockaddrInet6
}

func getaddrinfo(name, service string, hints *addrInfo, results []addrInfo) (int, error) {
	hints.sockAddrInfo = sockAddrInfo{
		ai_flags:    uint16(hints.flags),
		ai_family:   uint8(hints.family),
		ai_socktype: uint8(hints.socketType),
		ai_protocol: uint32(hints.protocol),
	}
	for i := range results {
		results[i].sockAddr = sockAddr{
			sa_family:   0,
			sa_data_len: uint32(unsafe.Sizeof(addrInfo{}.sockData)),
			sa_data:     uintptr32(uintptr(unsafe.Pointer(&results[i].sockData))),
		}
		results[i].sockAddrInfo = sockAddrInfo{
			ai_flags:        0,
			ai_family:       0,
			ai_socktype:     0,
			ai_protocol:     0,
			ai_addrlen:      uint32(unsafe.Sizeof(sockAddr{})),
			ai_addr:         uintptr32(uintptr(unsafe.Pointer(&results[i].sockAddr))),
			ai_canonname:    uintptr32(uintptr(unsafe.Pointer(&results[i].cannoname))),
			ai_canonnamelen: uint32(unsafe.Sizeof(addrInfo{}.cannoname)),
		}
		if i > 0 {
			results[i-1].sockAddrInfo.ai_next = uintptr32(uintptr(unsafe.Pointer(&results[i].sockAddrInfo)))
		}
	}

	println("name:", name)
	println("service:", service)
	println("hints:", hints)
	println("results:", results)

	resPtr := uintptr32(uintptr(unsafe.Pointer(&results[0].sockAddrInfo)))
	// For compatibility with WasmEdge, make sure strings are null-terminated.
	namePtr, nameLen := nullTerminatedString(name)
	println("namePtr:", unsafe.Pointer(namePtr), "nameLen:", nameLen)
	servPtr, servLen := nullTerminatedString(service)
	println("servPtr:", unsafe.Pointer(servPtr), "servLen:", servLen)

	var n uint32
	println("sock_getaddrinfo", unsafe.Pointer(namePtr), uint32(nameLen), unsafe.Pointer(servPtr), uint32(servLen), unsafe.Pointer(&hints.sockAddrInfo), unsafe.Pointer(&resPtr), uint32(len(results)), unsafe.Pointer(&n))
	println("len(results):", len(results))
	println("hints has numeric host:", hints.flags&AI_NUMERICHOST != 0)
	println("hints has numeric serv:", hints.flags&AI_NUMERICSERV != 0)

	errno := sock_getaddrinfo(
		unsafe.Pointer(namePtr),
		uint32(nameLen),
		unsafe.Pointer(servPtr),
		uint32(servLen),
		unsafe.Pointer(&hints.sockAddrInfo),
		unsafe.Pointer(&resPtr),
		uint32(len(results)),
		unsafe.Pointer(&n),
	)
	if errno != 0 {
		println("sock_getaddrinfo failed")
		return 0, errno
	}

	for i := range results[:n] {
		r := &results[i]
		port := binary.BigEndian.Uint16(results[i].sockData[:2])
		switch results[i].sockAddr.sa_family {
		case AF_INET:
			r.inet4addr.port = uint32(port)
			copy(r.inet4addr.addr[:], results[i].sockData[2:])
			r.address = &r.inet4addr
		case AF_INET6:
			r.inet6addr.port = uint32(port)
			copy(r.inet6addr.addr[:], results[i].sockData[2:])
			r.address = &r.inet6addr
		default:
			r.address = nil
		}
		// TODO: canonical names
	}
	return int(n), nil
}

func nullTerminatedString(s string) (*byte, int) {
	if len(s) == 0 {
		return nil, 0
	}
	if strings.IndexByte(s, 0) >= 0 {
		// String already contains a null terminator
		return (*byte)(unsafe.Pointer(unsafe.StringData(s))), len(s)
	}
	// Allocate a new buffer with space for the null terminator
	buf := make([]byte, len(s)+1)
	copy(buf, s)
	// buf[len(s)] is already 0
	return (*byte)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf)
}
