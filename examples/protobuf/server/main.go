package main

import (
	"log"

	"github.com/wubbalubbaaa/easyNet"
	pb "github.com/wubbalubbaaa/easyNet/examples/protobuf/proto"
	"github.com/wubbalubbaaa/easyNet/plugins/protobuf"
	"google.golang.org/protobuf/proto"
)

func main() {
	g := easyNet.NewGopher(easyNet.Config{
		Network: "tcp",
		Addrs:   []string{"localhost:8888"},
	})
	g.OnOpen(func(c *easyNet.Conn) {
		codec := protobuf.New()
		c.SetCodec(codec)
	})
	g.OnData(func(c *easyNet.Conn, data []byte) {
		msgType := c.Session().(string)
		switch msgType {
		case "msg1":
			msg := &pb.Msg1{}
			if err := proto.Unmarshal(data, msg); err != nil {
				log.Println(err)
			}
			log.Println(msgType, msg)
		case "msg2":
			msg := &pb.Msg2{}
			if err := proto.Unmarshal(data, msg); err != nil {
				log.Println(err)
			}
			log.Println(msgType, msg)
		default:
			log.Println("unknown msg type")
		}

	})
	g.OnClose(func(c *easyNet.Conn, err error) {
	})
	err := g.Start()
	if err != nil {
		log.Printf("easyNet.Start failed: %v\n", err)
		return
	}
	defer g.Stop()

	g.Wait()
}
