//go:build windows

package store

import (
	"net"
	"syscall"
)

// soExclusiveAddrUse 是 Windows 的 SO_EXCLUSIVEADDRUSE（值为 -5）。
// Go 的 syscall 包未导出该常量，这里按 winsock 文档取字面值。
const soExclusiveAddrUse = 0xFFFFFFFB

// listenTCPExclusive 以 SO_EXCLUSIVEADDRUSE 绑定 TCP 端口。
func listenTCPExclusive(port int) (net.Listener, error) {
	var lc net.ListenConfig
	lc.Control = func(network, address string, c syscall.RawConn) error {
		var opErr error
		if err := c.Control(func(fd uintptr) {
			opErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, soExclusiveAddrUse, 1)
		}); err != nil {
			return err
		}
		return opErr
	}
	return lc.Listen(nil, "tcp", fmtAddr(port))
}

// listenUDPExclusive 以 SO_EXCLUSIVEADDRUSE 绑定 UDP 端口。
func listenUDPExclusive(addr *net.UDPAddr) (*net.UDPConn, error) {
	var lc net.ListenConfig
	lc.Control = func(network, address string, c syscall.RawConn) error {
		var opErr error
		if err := c.Control(func(fd uintptr) {
			opErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, soExclusiveAddrUse, 1)
		}); err != nil {
			return err
		}
		return opErr
	}
	pc, err := lc.ListenPacket(nil, "udp", addr.String())
	if err != nil {
		return nil, err
	}
	return pc.(*net.UDPConn), nil
}
