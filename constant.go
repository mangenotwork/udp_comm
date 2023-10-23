package udp

import (
	"fmt"
)

const (
	SignLetterBytes         = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_+=~!@#$%^&*()<>{},.?~"
	DefaultConnectCode      = "c"
	DefaultServersName      = "servers"
	DefaultClientName       = "client"
	DefaultSecretKey        = "12345678"
	DefaultSGetTimeOut      = 1000 // 单位 ms
	DefaultNoticeMaxRetry   = 10   // 通知消息最大重试次数
	DefaultNoticeRetryTimer = 100  // 重试等待时间 单位ms
	HeartbeatTime           = 5    // 5s
	HeartbeatTimeLast       = 6    // 6s
	ServersTimeWheel        = 2    // 2s servers 时间轮
)

// err
var (
	ErrNmeLengthAbove  = fmt.Errorf("名字不能超过7个长度")
	ErrDataLengthAbove = fmt.Errorf("数据大于 540个字节, 建议拆分")
	ErrNonePacket      = fmt.Errorf("空包")
	ErrSGetTimeOut     = func(label, name, ip string) error {
		return fmt.Errorf("请求客户端 FuncLabel:%s | name:%s | IP:%s 超时", label, name, ip)
	}
	ErrNotFondClient = func(name string) error {
		return fmt.Errorf("未找到客户端 name:%s ", name)
	}
	PanicGetHandleFuncExist = func(label string) {
		panic(fmt.Sprintf("get handle func label:%s is exist.", label))
	}
	PanicPutHandleFuncExist = func(label string) {
		panic(fmt.Sprintf("put handle func label:%s is exist.", label))
	}
	ErrServersSecretKey = fmt.Errorf("秘钥的长度只能为8，并且与Client端统一")
	ErrClientNameErr    = fmt.Errorf("client name 不能含特殊字符 @")
	ErrClientSecretKey  = fmt.Errorf("秘钥的长度只能为8，并且与Servers端统一")
)
