package udp

import "sync"

type GetData struct {
	Label    string    // 标签，用于区分当前数据处理的方法
	Id       int64     // 唯一id
	Param    []byte    // 传过来的数据
	ctxChan  chan bool // 确认接受到消息
	Response []byte    // 返回的数据
	Err      error
}

type ServersGetFunc map[string]func(s *Servers, param []byte) (int, []byte)

type ClientGetFunc map[string]func(c *Client, param []byte) (int, []byte)

var GetDataMap sync.Map
