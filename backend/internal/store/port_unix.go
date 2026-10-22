//go:build !windows

package store

import "net"

// listenTCPExclusive 在非 Windows 平台即普通绑定（Linux 默认语义已是排他，
// 除非对方显式设置 SO_REUSEADDR，而服务进程不会这么做）。
func listenTCPExclusive(port int) (net.Listener, error) {
	return net.Listen("tcp", fmtAddr(port))
}

// listenUDPExclusive 在非 Windows 平台即普通绑定。
func listenUDPExclusive(addr *net.UDPAddr) (*net.UDPConn, error) {
	return net.ListenUDP("udp", addr)
}
