package main

import (
	"fmt"
	"log"
	"net"

	"github.com/wubbalubbaaa/easyNet"
)

func main() {
	g := easyNet.NewGopher(easyNet.Config{})
	g.OnOpen(func(c *easyNet.Conn) {
		log.Println("OnOpen:", c.RemoteAddr().String())
	})

	g.OnData(func(c *easyNet.Conn, data []byte) {
		c.Write(append([]byte{}, data...))
	})

	err := g.Start()
	if err != nil {
		fmt.Printf("easyNet.Start failed: %v\n", err)
		return
	}
	defer g.Stop()

	ln, err := net.Listen("tcp", "localhost:8888")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept failed:", err)
			continue
		}
		g.AddConn(conn)
	}
}
