package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wubbalubbaaa/easyNet"
	"github.com/wubbalubbaaa/easyNet/extension/tls"
)

var (
	tlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
)

func main() {
	var (
		rbuf   []byte
		wbuf   = []byte("hello")
		addr   = "localhost:8888"
		ctx, _ = context.WithTimeout(context.Background(), time.Second)
	)

	g := easyNet.NewGopher(easyNet.Config{})

	done := make(chan int)
	isClient := true
	g.OnOpen(tls.WrapOpen(tlsConfig, isClient, 0, func(c *easyNet.Conn, tlsConn *tls.Conn) {
		log.Println("OnOpen:", c.RemoteAddr().String())
		// tlsConn.Write(wbuf)
	}))
	g.OnClose(tls.WrapClose(func(c *easyNet.Conn, tlsConn *tls.Conn, err error) {
		log.Println("OnClose:", c.RemoteAddr().String())
	}))
	g.OnData(tls.WrapData(func(c *easyNet.Conn, tlsConn *tls.Conn, data []byte) {
		rbuf = append(rbuf, data...)
		if bytes.Equal(wbuf, rbuf) {
			close(done)
		}
	}))

	err := g.Start()
	if err != nil {
		fmt.Printf("Start failed: %v\n", err)
	}
	defer g.Stop()

	// step 1: make a tls.Conn by tls.Dial
	tlsConn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		log.Fatalf("Dial failed: %v\n", err)
	}
	// step 2:
	// add tls.Conn.conn to gopher, and get the easyNet.Conn. the new easyNet.Conn is non-blocking
	nbConn, err := g.Conn(tlsConn.Conn())
	if err != nil {
		log.Fatalf("AddConn failed: %v\n", err)
	}
	// step 3: set tls.Conn and easyNet.Conn to each other, and add easyNet.Conn to the gopher
	nbConn.SetSession(tlsConn)
	nonBlock := true
	readBufferSize := 8192
	tlsConn.ResetConn(nbConn, nonBlock, readBufferSize)
	g.AddConn(nbConn)

	// step 4: write data here or in the OnOpen handler or anywhere
	tlsConn.Write(wbuf)

	select {
	case <-ctx.Done():
		log.Fatal("timeout")
	case <-done:
		log.Println("success")
	}
}
