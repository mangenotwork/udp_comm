package udp

import "net"

type PutData struct {
	Label string // 标签，用于区分当前数据处理的方法
	Id    int64  // 唯一id
	Body  []byte // 传过来的数据
}

type ServersPutFunc map[string]func(s *Servers, c *ClientInfo, data []byte)

type ClientInfo struct {
	Name        string
	Addr        *net.UDPAddr
	Interactive int64
	PacketSize  int
}

// TODO 给ClientInfo 下发消息，场景是 S端收到C端发来的PUT, S端可以直接进行应答
