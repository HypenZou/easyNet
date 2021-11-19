package main

import (
	"fmt"
	"time"

	"github.com/lesismal/easyNet"
)

func onOpen(c *easyNet.Conn) {
	c.SetReadDeadline(time.Now().Add(time.Second * 10))
	fmt.Println("onOpen:", c.RemoteAddr().String(), time.Now().Format("15:04:05.000"))
}

func onClose(c *easyNet.Conn, err error) {
	fmt.Println("onClose:", c.RemoteAddr().String(), time.Now().Format("15:04:05.000"), err)
}

func onData(c *easyNet.Conn, data []byte) {
	c.SetReadDeadline(time.Now().Add(time.Second * 10))
	c.SetWriteDeadline(time.Now().Add(time.Second * 3))
	c.Write(append([]byte{}, data...))
}

func main() {
	g, err := easyNet.NewGopher(easyNet.Config{
		Network: "tcp",
		Addrs:   []string{":8888", ":9999"},
	})
	if err != nil {
		fmt.Printf("easyNet.New failed: %v\n", err)
		return
	}

	g.OnOpen(onOpen)
	g.OnClose(onClose)
	g.OnData(onData)

	err = g.Start()
	if err != nil {
		fmt.Printf("easyNet.Start failed: %v\n", err)
		return
	}

	go func() {
		for {
			time.Sleep(time.Second * 5)
			fmt.Println(g.State().String())
		}
	}()

	g.Wait()
}
