# easyNet - NON BLOCKING IO

[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Build Status][5]][6] [![Go Report Card][7]][8] [![Coverage Statusd][9]][10]

[1]: https://godoc.org/github.com/wubbalubbaaa/easyNet?status.svg
[2]: https://godoc.org/github.com/wubbalubbaaa/easyNet
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://travis-ci.org/wubbalubbaaa/easyNet.svg?branch=master
[6]: https://travis-ci.org/wubbalubbaaa/easyNet
[7]: https://goreportcard.com/badge/github.com/wubbalubbaaa/easyNet
[8]: https://goreportcard.com/report/github.com/wubbalubbaaa/easyNet
[9]: https://codecov.io/gh/wubbalubbaaa/easyNet/branch/master/graph/badge.svg
[10]: https://codecov.io/gh/wubbalubbaaa/easyNet

## Examples

- [echo-server](https://github.com/wubbalubbaaa/easyNet/blob/master/examples/echo/server.go)

```golang
package main

import (
	"fmt"
	"github.com/wubbalubbaaa/easyNet"
	"time"
)

func onOpen(c *easyNet.Conn) {
	c.SetReadDeadline(time.Now().Add(time.Second * 13))
	fmt.Println("onOpen:", c.RemoteAddr().String(), time.Now().Format("15:04:05.000"))
}

func onClose(c *easyNet.Conn) {
	fmt.Println("onClose:", c.RemoteAddr().String(), time.Now().Format("15:04:05.000"))
}

func onData(c *easyNet.Conn, data []byte) {
	c.SetReadDeadline(time.Now().Add(time.Second * 10))
	c.SetWriteDeadline(time.Now().Add(time.Second * 3))
	c.Write(append([]byte{}, data...))
}

func main() {
	g, err := easyNet.NewGopher(easyNet.Config{
		Network:      "tcp",
		Address:      ":8888",
		NPoller:      4,
		NWorker:      8,
		QueueSize:    1024,
		BufferSize:   1024 * 64,
		BufferNum:    1024 * 2,
		PollInterval: 200,       //ms
		MaxTimeout:   10 * 1000, //ms
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
```
