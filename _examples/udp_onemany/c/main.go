package main

import (
	"fmt"
	"os"
	"time"

	udp "github.com/mangenotwork/udp_comm"
)

func main() {
	udp.Info(os.Args)
	if len(os.Args) != 3 {
		panic("参数不正确 第一个是servers地址 第二个是客户端昵称")
	}
	// 定义客户端
	client, err := udp.NewClient(os.Args[1])
	if err != nil {
		panic(err)
	}
	_ = client.SetClientName(os.Args[2])

	// get方法
	client.GetHandleFunc("get", Get)
	// 通知方法
	client.NoticeHandleFunc("notice", Notice)

	// 每两秒发送一些测试数据
	go func() {
		n := 0
		for {
			n++
			time.Sleep(10 * time.Millisecond)

			// put上传数据到服务端的 case2 方法
			client.Put("putCase",
				[]byte(fmt.Sprintf("%d | hello %s : %d", time.Now().UnixNano(), os.Args[2], n)))
			udp.Info("n = ", n)

			// get请求服务端的 case3 方法
			rse, err := client.Get("getCase", []byte("test"))
			if err != nil {
				udp.Error(err)
				continue
			}
			udp.Info("get 请求返回 = ", string(rse))
		}
	}()

	// 运行客户端
	client.Run()
}

func Get(c *udp.Client, param []byte) (int, []byte) {
	udp.Info("获取到的请求参数  param = ", string(param))
	return 0, []byte(fmt.Sprintf("客户端名称 %s.", c.GetName()))
}

func Notice(c *udp.Client, data []byte) {
	udp.Info("收到来自服务器的通知，开始执行......")
	udp.Info("data = ", string(data))
}
