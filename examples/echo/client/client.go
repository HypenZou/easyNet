package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/wubbalubbaaa/easyNet"
)

func main() {
	var (
		ret    []byte
		buf    = make([]byte, 1024)
		addr   = "localhost:8888"
		ctx, _ = context.WithTimeout(context.Background(), time.Second)
	)

	rand.Read(buf)

	g := easyNet.NewGopher(easyNet.Config{})

	done := make(chan int)
	g.OnData(func(c *easyNet.Conn, data []byte) {
		ret = append(ret, data...)
		if len(ret) == len(buf) {
			if bytes.Equal(buf, ret) {
				close(done)
			}
		}
	})

	err := g.Start()
	if err != nil {
		fmt.Printf("Start failed: %v\n", err)
	}
	defer g.Stop()

	c, err := easyNet.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Dial failed: %v\n", err)
	}
	g.AddConn(c)
	c.Write(buf)

	select {
	case <-ctx.Done():
		//log.Error("timeout")
	case <-done:
		//log.Info("success")
	}
}
