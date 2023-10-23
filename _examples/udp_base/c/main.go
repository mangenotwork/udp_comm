package main

import (
	"fmt"
	"time"

	udp "github.com/mangenotwork/udp_comm"
)

func main() {
	// 定义客户端
	client, err := udp.NewClient("127.0.0.1:12345")
	if err != nil {
		panic(err)
	}
	// get方法
	client.GetHandleFunc("getClient", CGetTest)
	// 通知方法
	client.NoticeHandleFunc("testNotice", CNoticeTest)

	// 每两秒发送一些测试数据
	go func() {
		n := 0
		for {
			n++
			time.Sleep(2 * time.Second)

			// put上传数据到服务端的 case2 方法
			client.Put("case1", []byte(fmt.Sprintf("%d | hello : %d", time.Now().UnixNano(), n)))
			udp.Info("n = ", n)

			// put上传数据到服务端的 case2 方法
			client.Put("case2", []byte(fmt.Sprintf("%d | hello : %d", time.Now().UnixNano(), n)))
			udp.Info("n = ", n)

			// get请求服务端的 case3 方法
			rse, err := client.Get("case3", []byte("test"))
			if err != nil {
				udp.Error(err)
				udp.Info("[Client 测试get] failed")
				continue
			}
			udp.Info("get 请求返回 = ", string(rse))
			udp.Info("[Client 测试get] passed")
		}
	}()

	// 运行客户端
	client.Run()
}

func CGetTest(c *udp.Client, param []byte) (int, []byte) {
	udp.Info("获取到的请求参数  param = ", string(param))
	return 0, []byte(fmt.Sprintf("客户端名称 %s.", c.GetName()))
}

func CNoticeTest(c *udp.Client, data []byte) {
	udp.Info("收到来自服务器的通知，开始执行......")
	udp.Info("data = ", string(data))
}
