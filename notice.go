package udp

import "sync"

type NoticeData struct {
	Label    string    // 标签，用于区分当前数据处理的方法
	Id       int64     // 唯一id
	Data     []byte    // 通知内容
	ctxChan  chan bool // 确认接受到消息
	Response []byte    // 返回的数据
	Err      error
}

var NoticeDataMap sync.Map

type ClientNoticeFunc map[string]func(c *Client, data []byte)
