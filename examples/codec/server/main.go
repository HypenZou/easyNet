package main

import (
	"encoding/binary"
	"log"

	"github.com/wubbalubbaaa/easyNet"
)

var cntPacket = 0

var (
	ec = easyNet.EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               4,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}
	dc = easyNet.DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   4,
		LengthAdjustment:    0,
		InitialBytesToStrip: 4,
	}
)

var codecWithFixedLength = easyNet.NewFixedLengthFrameCodec(5)
var codecWithVariousLength = easyNet.NewLengthFieldBasedFrameCodec(ec, dc)

func main() {
	g := easyNet.NewGopher(easyNet.Config{
		Network: "tcp",
		Addrs:   []string{"localhost:8888"},
	})
	g.OnOpen(func(c *easyNet.Conn) {
		c.SetCodec(codecWithVariousLength)
	})
	g.OnData(func(c *easyNet.Conn, data []byte) {
		cntPacket++

	})
	g.OnClose(func(c *easyNet.Conn, err error) {
		log.Println(cntPacket)
	})
	err := g.Start()
	if err != nil {
		log.Printf("easyNet.Start failed: %v\n", err)
		return
	}
	defer g.Stop()

	g.Wait()
}
