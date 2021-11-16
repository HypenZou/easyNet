package easyNet

import (
	"errors"
	"net"
	"syscall"
)

const _REUSE_PORT = 0x0F

var errUnsupportedProtocol = errors.New("unsupported protocol")

func sockaddrToAddr(sa syscall.Sockaddr) net.Addr {
	var a net.Addr
	switch sa := sa.(type) {
	case *syscall.SockaddrInet4:
		a = &net.TCPAddr{
			IP:   append([]byte{}, sa.Addr[:]...),
			Port: sa.Port,
		}
	case *syscall.SockaddrInet6:
		var zone string
		if sa.ZoneId != 0 {
			if ifi, err := net.InterfaceByIndex(int(sa.ZoneId)); err == nil {
				zone = ifi.Name
			}
		}
		a = &net.TCPAddr{
			IP:   append([]byte{}, sa.Addr[:]...),
			Port: sa.Port,
			Zone: zone,
		}
	case *syscall.SockaddrUnix:
		a = &net.UnixAddr{Net: "unix", Name: sa.Name}
	}
	return a
}

func syscallClose(fd int) error {
	return syscall.Close(fd)
}

var (
	listenerBacklogMaxSize    = 1024 * 8
	errUnsupportedTCPProtocol = errors.New("only tcp, tcp4, tcp6 are supported")
)

func getSockaddr(proto, addr string) (sa syscall.Sockaddr, soType int, err error) {
	var tcp *net.TCPAddr

	tcp, err = net.ResolveTCPAddr(proto, addr)
	if err != nil && tcp.IP != nil {
		return nil, -1, err
	}

	tcpVersion, err := determineTCPProto(proto, tcp)
	if err != nil {
		return nil, -1, err
	}

	switch tcpVersion {
	case "tcp":
		return &syscall.SockaddrInet4{Port: tcp.Port}, syscall.AF_INET, nil
	case "tcp4":
		sa := &syscall.SockaddrInet4{Port: tcp.Port}

		if tcp.IP != nil {
			copy(sa.Addr[:], tcp.IP[12:16])
		}

		return sa, syscall.AF_INET, nil
	case "tcp6":
		sa := &syscall.SockaddrInet6{Port: tcp.Port}

		if tcp.IP != nil {
			copy(sa.Addr[:], tcp.IP)
		}

		if tcp.Zone != "" {
			iface, err := net.InterfaceByName(tcp.Zone)
			if err != nil {
				return nil, -1, err
			}

			sa.ZoneId = uint32(iface.Index)
		}

		return sa, syscall.AF_INET6, nil
	}

	return nil, -1, errUnsupportedProtocol
}

func determineTCPProto(proto string, ip *net.TCPAddr) (string, error) {
	if ip.IP.To4() != nil {
		return "tcp4", nil
	}

	if ip.IP.To16() != nil {
		return "tcp6", nil
	}

	switch proto {
	case "tcp", "tcp4", "tcp6":
		return proto, nil
	}

	return "", errUnsupportedProtocol
}

// Listen on network address
func Listen(network, address string) (int, net.Addr, error) {
	var (
		err        error
		soType, fd int
		netAddr    net.Addr
		sockaddr   syscall.Sockaddr
	)

	if sockaddr, soType, err = getSockaddr(network, address); err != nil {
		return -1, netAddr, err
	}

	netAddr = sockaddrToAddr(sockaddr)

	syscall.ForkLock.RLock()
	defer syscall.ForkLock.RUnlock()
	if fd, err = syscall.Socket(soType, syscall.SOCK_STREAM, syscall.IPPROTO_TCP); err != nil {
		return -1, netAddr, err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscallClose(fd)
		return -1, netAddr, err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, _REUSE_PORT, 1); err != nil {
		syscallClose(fd)
		return -1, netAddr, err
	}

	if err = syscall.Bind(fd, sockaddr); err != nil {
		syscallClose(fd)
		return -1, netAddr, err
	}

	if err = syscall.Listen(fd, listenerBacklogMaxSize); err != nil {
		syscallClose(fd)
		return -1, netAddr, err
	}

	if err = syscall.SetNonblock(fd, true); err != nil {
		syscallClose(fd)
		return -1, netAddr, err
	}

	return fd, netAddr, nil
}
