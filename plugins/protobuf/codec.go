package protobuf

import (
	"encoding/binary"
	"errors"

	"github.com/gobwas/pool/pbytes"
	"github.com/wubbalubbaaa/easyNet"
)

// Message 数据帧定义
type Message struct {
	Len     uint32
	TypeLen uint16
	Type    string
	Data    []byte
}

// Protocol protobuf
type Protocol struct {
}

// New 创建 protobuf Protocol
func New() *Protocol {
	return &Protocol{}
}

// UnPacket ...
func (p *Protocol) Decode(c *easyNet.Conn) (out []byte, err error) {
	cache := c.Cache()
	if len(*cache) > 6 {
		length := int(binary.BigEndian.Uint32((*cache)[:4]))
		if len(*cache) >= length+4 {
			readN(cache, 4)
			typeLen := int(binary.BigEndian.Uint16((*cache)[:2]))
			readN(cache, 2)
			typeByte := pbytes.GetLen(typeLen)
			copy(typeByte, *cache)
			readN(cache, len(typeByte))
			dataLen := length - 2 - typeLen
			data := make([]byte, dataLen)
			copy(data, *cache)
			readN(cache, len(data))
			out = data
			c.SetSession(string(typeByte))
			pbytes.Put(typeByte)
		}
	}

	return
}

// Packet ...
func (p *Protocol) Encode(c *easyNet.Conn, data []byte) ([]byte, error) {
	return data, nil
}
func readN(in *[]byte, n int) (buf []byte, err error) {
	if n == 0 {
		return nil, nil
	}

	if n < 0 {
		return nil, errors.New("negative length is invalid")
	} else if n > len(*in) {
		return nil, errors.New("exceeding buffer length")
	}
	buf = (*in)[:n]
	*in = (*in)[n:]
	return
}
