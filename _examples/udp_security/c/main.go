package main

import udp "github.com/mangenotwork/udp_comm"

func main() {
	c, err := udp.NewClient("127.0.0.1:12346",
		udp.SetClientConf("node1", "123456", "abc12345"))
	if err != nil {
		panic(err)
	}
	c.NoticeHandleFunc("testNotice", CNoticeTest)
	c.Run()
}

func CNoticeTest(c *udp.Client, data []byte) {
	udp.Info("收到来自服务器的通知，开始执行......")
	udp.Info("data = ", string(data))
}
