package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wubbalubbaaa/easyNet"
	"github.com/wubbalubbaaa/easyNet/extension/tls"
	"github.com/wubbalubbaaa/llib/bytes"
	ltls "github.com/wubbalubbaaa/llib/std/crypto/tls"
)

var (
	addr = "localhost:8888"

	tlsConfigs = []*tls.Config{
		// sth wrong with TLS 1.0
		// &tls.Config{
		// 	InsecureSkipVerify: true,
		// 	MaxVersion:         ltls.VersionTLS10,
		// },
		&tls.Config{
			InsecureSkipVerify: true,
			MaxVersion:         ltls.VersionTLS11,
		},
		&tls.Config{
			InsecureSkipVerify: true,
			MaxVersion:         ltls.VersionTLS12,
		},
		&tls.Config{
			InsecureSkipVerify: true,
			MaxVersion:         ltls.VersionTLS13,
		},
		// SSL is not supported
		// &tls.Config{
		// 	InsecureSkipVerify: true,
		// 	MaxVersion:         ltls.VersionSSL30,
		// },
	}
)

// Session .
type Session struct {
	Conn   *tls.Conn
	Buffer *bytes.Buffer
}

// WrapData .
func WrapData(h func(c *easyNet.Conn, tlsConn *tls.Conn, data []byte)) func(c *easyNet.Conn, data []byte) {
	return func(c *easyNet.Conn, data []byte) {

		if isession := c.Session(); isession != nil {
			if session, ok := isession.(*Session); ok {
				session.Conn.Append(data)
				for {
					n, err := session.Conn.Read(session.Conn.ReadBuffer)
					if err != nil {
						c.Close()
						return
					}
					if h != nil && n > 0 {
						h(c, session.Conn, session.Conn.ReadBuffer[:n])
					}
					if n < len(session.Conn.ReadBuffer) {
						return
					}
				}
			}
		}
	}
}

func main() {
	var (
		wg         sync.WaitGroup
		qps        int64
		bufsize    = 64 //1024 * 8
		clientNum  = 128
		totalRead  int64
		totalWrite int64
	)

	g := easyNet.NewGopher(easyNet.Config{})
	g.OnData(WrapData(func(c *easyNet.Conn, tlsConn *tls.Conn, data []byte) {
		session := c.Session().(*Session)
		session.Buffer.Push(data)
		for session.Buffer.Len() >= bufsize {
			buf, _ := session.Buffer.Pop(bufsize)
			tlsConn.Write(buf)
			atomic.AddInt64(&qps, 1)
			atomic.AddInt64(&totalRead, int64(bufsize))
			atomic.AddInt64(&totalWrite, int64(bufsize))
		}
	}))

	err := g.Start()
	if err != nil {
		fmt.Printf("Start failed: %v\n", err)
	}
	defer g.Stop()

	for i := 0; i < clientNum; i++ {
		wg.Add(1)

		tlsConfig := tlsConfigs[i%len(tlsConfigs)]
		go func() {
			data := make([]byte, bufsize)

			tlsConn, err := tls.Dial("tcp", addr, tlsConfig)
			if err != nil {
				log.Fatalf("Dial failed: %v\n", err)
			}

			nbConn, err := g.Conn(tlsConn.Conn())
			if err != nil {
				log.Fatalf("AddConn failed: %v\n", err)
			}

			nbConn.SetSession(&Session{
				Conn:   tlsConn,
				Buffer: bytes.NewBuffer(),
			})
			nonBlock := true
			readBufferSize := 8192
			tlsConn.ResetConn(nbConn, nonBlock, readBufferSize)
			g.AddConn(nbConn)

			tlsConn.Write(data)
			atomic.AddInt64(&totalWrite, int64(len(data)))
		}()
	}

	go func() {
		for {
			time.Sleep(time.Second * 5)
			fmt.Println(g.State().String())
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second)
			fmt.Printf("easyNet tls clients, qps: %v, total read: %.1f M, total write: %.1f M\n", atomic.SwapInt64(&qps, 0), float64(atomic.SwapInt64(&totalRead, 0))/1024/1024, float64(atomic.SwapInt64(&totalWrite, 0))/1024/1024)
		}
	}()

	wg.Wait()
}
