package main

import (
	"fmt"

	"github.com/wubbalubbaaa/easyNet"
)

func onData(c *easyNet.Conn, data []byte) {
	c.Write(append([]byte{}, data...))
}

func main() {
	g := easyNet.NewGopher(easyNet.Config{
		Network: "tcp",
		Addrs:   []string{"localhost:8888"},
	})

	g.OnData(onData)

	err := g.Start()
	if err != nil {
		fmt.Printf("easyNet.Start failed: %v\n", err)
		return
	}

	g.Wait()
}
