package easyNet

import (
	"log"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

var addr = "localhost:8888"
var gopher *Gopher

func init() {
	addrs := []string{addr}
	g, err := NewGopher(Config{
		Network: "tcp",
		Addrs:   addrs,
		MaxLoad: 10,
	})
	g.maxLoad = 1024 * 100
	if err != nil {
		log.Fatalf("NewGopher failed: %v\n", err)
	}

	g.OnOpen(func(c *Conn) {
		c.SetReadDeadline(time.Now().Add(time.Second * 10))
	})
	g.OnData(func(c *Conn, data []byte) {
		c.Write(append([]byte{}, data...))
	})
	g.OnClose(func(c *Conn, err error) {})

	err = g.Start()
	if err != nil {
		log.Fatalf("Start failed: %v\n", err)
	}

	gopher = g
}

func TestEcho(t *testing.T) {
	var done = make(chan int)
	var clientNum = 2
	var msgSize = 1024
	var total int64 = 0

	g, err := NewGopher(Config{})
	if err != nil {
		log.Fatalf("NewGopher failed: %v\n", err)
	}
	err = g.Start()
	if err != nil {
		log.Fatalf("Start failed: %v\n", err)
	}
	defer g.Stop()

	g.OnOpen(func(c *Conn) {
		c.SetSession(1)
		if c.Session() != 1 {
			log.Fatalf("invalid session: %v", c.Session())
		}
		c.SetLinger(1, 0)
		c.SetNoDelay(true)
		c.SetKeepAlive(true)
		c.SetKeepAlivePeriod(time.Second * 60)
		c.SetDeadline(time.Now().Add(time.Second))
		c.SetReadBuffer(1024 * 4)
		c.SetWriteBuffer(1024 * 4)
		log.Printf("connected, local addr: %v, remote addr: %v", c.LocalAddr(), c.RemoteAddr())
	})
	g.OnData(func(c *Conn, data []byte) {
		recved := atomic.AddInt64(&total, int64(len(data)))
		if recved >= int64(clientNum*msgSize) {
			close(done)
		}
	})

	g.OnMemAlloc(func(c *Conn) []byte {
		return make([]byte, 1024)
	})
	g.OnMemFree(func(c *Conn, b []byte) {

	})

	one := func(n int) {
		c, err := Dial("tcp", addr)
		if err != nil {
			log.Fatalf("Dial failed: %v", err)
		}
		g.AddConn(c)
		if n%2 == 0 {
			c.Write(make([]byte, msgSize))
		} else {
			c.Writev([][]byte{make([]byte, msgSize)})
		}
	}

	for i := 0; i < clientNum; i++ {
		if runtime.GOOS == "linux" {
			one(i)
		} else {
			go one(i)
		}
	}

	<-done
}

func Test10k(t *testing.T) {
	g, err := NewGopher(Config{})
	if err != nil {
		log.Fatalf("NewGopher failed: %v\n", err)
	}
	err = g.Start()
	if err != nil {
		log.Fatalf("Start failed: %v\n", err)
	}
	defer g.Stop()

	var total int64 = 0
	var clientNum int64 = 1024 * 10
	var done = make(chan int)

	if runtime.GOOS == "windows" {
		clientNum = 1024
	}

	t.Log("testing concurrent:", clientNum, "connections")

	g.OnOpen(func(c *Conn) {
		c.Close()
		if atomic.AddInt64(&total, 1) == clientNum {
			close(done)
		}
	})

	one := func() {
		c, err := Dial("tcp", addr)
		if err != nil {
			log.Fatalf("Dial failed: %v", err)
		}
		g.AddConn(c)
	}
	go func() {
		for i := 0; i < int(clientNum); i++ {
			go func() {
				if runtime.GOOS == "linux" {
					one()
				} else {
					go one()
					// 	time.Sleep(time.Second / 1000)
				}
			}()
		}
	}()

	<-done
}

func TestTimeout(t *testing.T) {
	g, err := NewGopher(Config{})
	if err != nil {
		log.Fatalf("NewGopher failed: %v\n", err)
	}
	err = g.Start()
	if err != nil {
		log.Fatalf("Start failed: %v\n", err)
	}
	defer g.Stop()

	var done = make(chan int)
	var begin time.Time
	var timeout = time.Second / 20
	g.OnOpen(func(c *Conn) {
		begin = time.Now()
		c.SetReadDeadline(begin.Add(timeout))
	})
	g.OnClose(func(c *Conn, err error) {
		to := time.Since(begin)
		if to > timeout+time.Second/10 {
			log.Fatalf("timeout: %v, want: %v", to, timeout)
		}
		close(done)
	})

	one := func() {
		c, err := Dial("tcp", addr)
		if err != nil {
			log.Fatalf("Dial failed: %v", err)
		}
		g.AddConn(c)
	}
	one()

	<-done
}

func TestFuzz(t *testing.T) {
	gopher.maxLoad = 10
	for i := 0; i < 100; i++ {
		Dial("tcp", addr)
	}

	listen("", "", 0)
	syscallClose(54321)
}

func TestStop(t *testing.T) {
	log.Println(gopher.State().String())
	gopher.Stop()
}
