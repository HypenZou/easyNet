// +build linux

package easyNet

import (
	"errors"
	"io"
	"log"
	"net"
	"sync/atomic"
	"syscall"
)

type poller struct {
	g *Gopher

	epfd int

	index int

	currLoad int64

	readBuffer []byte

	pollType   string
	isListener bool
	shutdown   bool
}

func (p *poller) online() int64 {
	return atomic.LoadInt64(&p.currLoad)
}

func (p *poller) increase() {
	atomic.AddInt64(&p.currLoad, 1)
}

func (p *poller) decrease() {
	atomic.AddInt64(&p.currLoad, -1)
}

func (p *poller) accept(lfd int) error {
	fd, saddr, err := syscall.Accept(lfd)
	if err != nil {
		return err
	}

	if !p.acceptable(fd) {
		syscallClose(fd)
		return nil
	}

	err = syscall.SetNonblock(fd, true)
	if err != nil {
		syscallClose(fd)
		return nil
	}

	laddr, err := syscall.Getsockname(fd)
	if err != nil {
		syscallClose(fd)
		return nil
	}

	c := newConn(int(fd), sockaddrToAddr(laddr), sockaddrToAddr(saddr))
	o := p.g.pollers[int(fd)%len(p.g.pollers)]
	o.addConn(c)

	return nil
}

func (p *poller) acceptable(fd int) bool {
	if fd < 0 {
		return false
	}
	if fd >= len(p.g.connsLinux) {
		p.g.connsLinux = append(p.g.connsLinux, make([]*Conn, 1024)...)
	}
	if atomic.AddInt64(&p.g.currLoad, 1) > p.g.maxLoad {
		atomic.AddInt64(&p.g.currLoad, -1)
		return false
	}

	return true
}

func (p *poller) addConn(c *Conn) error {
	p.g.onOpen(c)

	fd := c.fd
	c.g = p.g
	err := p.setRead(fd)
	if err == nil {
		p.g.connsLinux[fd] = c
		p.increase()
	}

	return err
}

func (p *poller) getConn(fd int) (*Conn, bool) {
	c := p.g.connsLinux[fd]
	return c, c != nil
}

func (p *poller) deleteConn(c *Conn) {
	p.g.connsLinux[c.fd] = nil
	p.decrease()
	p.g.decrease()
	p.g.onClose(c, c.closeErr)
}

func (p *poller) stop() {
	log.Printf("poller[%v] stop...", p.index)
	p.shutdown = true
	syscallClose(p.epfd)
}

func (p *poller) start() {
	defer p.g.Done()

	log.Printf("%v[%v] start", p.pollType, p.index)
	defer log.Printf("%v[%v] stopped", p.pollType, p.index)
	p.shutdown = false

	msec := -1
	events := make([]syscall.EpollEvent, 128)
	if p.isListener {
		for !p.shutdown {
			n, err := syscall.EpollWait(p.epfd, events, msec)
			if err != nil && err != syscall.EINTR {
				return
			}

			// if n <= 0 {
			//	msec = -1
			//	runtime.Gosched()
			//	continue
			// }
			// msec = 0

			for i := 0; i < n; i++ {
				err = p.accept(int(events[i].Fd))
				if err != nil && err != syscall.EAGAIN {
					return
				}
			}
		}
	} else {
		for !p.shutdown {
			n, err := syscall.EpollWait(p.epfd, events, msec)
			if err != nil && err != syscall.EINTR {
				return
			}

			// if n <= 0 {
			//	msec = -1
			//	runtime.Gosched()
			//	continue
			// }
			// msec = 0

			for i := 0; i < n; i++ {
				p.readWrite(&events[i])
			}
		}
	}
}

func (p *poller) setRead(fd int) error {
	return syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{Fd: int32(fd), Events: syscall.EPOLLRDHUP | syscall.EPOLLIN})
}

func (p *poller) setReadWrite(fd int) error {
	return syscall.EpollCtl(p.epfd, syscall.EPOLL_CTL_MOD, fd, &syscall.EpollEvent{Fd: int32(fd), Events: syscall.EPOLLRDHUP | syscall.EPOLLIN | syscall.EPOLLOUT})
}

func (p *poller) readWrite(ev *syscall.EpollEvent) {
	fd := int(ev.Fd)
	if c, ok := p.getConn(fd); ok {
		if ev.Events&(syscall.EPOLLERR|syscall.EPOLLHUP|syscall.EPOLLRDHUP) != 0 {
			c.closeWithError(io.EOF)
			return
		}
		if ev.Events&syscall.EPOLLIN != 0 {
			buffer := p.g.borrow(c)
			n, err := c.Read(buffer)
			if err == nil {
				p.g.onData(c, buffer[:n])
			} else {
				if err != nil && err != syscall.EINTR && err != syscall.EAGAIN {
					c.closeWithError(err)
					return
				}
			}
			p.g.payback(c, buffer)
		}

		if ev.Events&syscall.EPOLLOUT != 0 {
			c.flush()
		}
	}
}

func newPoller(g *Gopher, isListener bool, index int) (*poller, error) {
	if g == nil {
		panic("invalid gopher")
	}

	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	if isListener {
		if len(g.lfds) > 0 {
			for _, lfd := range g.lfds {
				// EPOLLEXCLUSIVE := (1 << 28)
				if err := syscall.EpollCtl(fd, syscall.EPOLL_CTL_ADD, lfd, &syscall.EpollEvent{Fd: int32(lfd), Events: syscall.EPOLLIN | (1 << 28)}); err != nil {
					syscallClose(fd)
					return nil, err
				}
			}
		} else {
			panic("invalid listener num")
		}
	}

	p := &poller{
		g:          g,
		index:      index,
		epfd:       fd,
		isListener: isListener,
	}

	if isListener {
		p.pollType = "listener"
	} else {
		p.pollType = "poller"
	}

	return p, nil
}

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

	return nil, -1, errors.New("unsupported protocol")
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

	return "", errors.New("unsupported protocol")
}

func listen(network, address string, backlogNum int64) (int, error) {
	var (
		err        error
		soType, fd int
		sockaddr   syscall.Sockaddr
	)

	if sockaddr, soType, err = getSockaddr(network, address); err != nil {
		return -1, err
	}

	syscall.ForkLock.RLock()
	defer syscall.ForkLock.RUnlock()
	if fd, err = syscall.Socket(soType, syscall.SOCK_STREAM, syscall.IPPROTO_TCP); err != nil {
		return -1, err
	}

	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscallClose(fd)
		return -1, err
	}

	socketOptReusePort := 0x0F
	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, socketOptReusePort, 1); err != nil {
		syscallClose(fd)
		return -1, err
	}

	if err = syscall.Bind(fd, sockaddr); err != nil {
		syscallClose(fd)
		return -1, err
	}

	n := int(backlogNum)
	if backlogNum <= 0 {
		n = syscall.SOMAXCONN
	}
	if err = syscall.Listen(fd, n); err != nil {
		syscallClose(fd)
		return -1, err
	}

	if err = syscall.SetNonblock(fd, true); err != nil {
		syscallClose(fd)
		return -1, err
	}

	return fd, nil
}
