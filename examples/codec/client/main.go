package main

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/wubbalubbaaa/easyNet"
)

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
	var (
		message = "hello"
		addr    = "localhost:8888"
	)
	c, _ := net.Dial("tcp", addr)
	data, _ := codecWithVariousLength.Encode(&easyNet.Conn{}, []byte(message))
	for i := 0; i < 1000; i++ {
		fmt.Println(string(data), len(data), len(message))
		c.Write(data)
	}
	c.Close()

}
