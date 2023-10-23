package main

import (
	"fmt"
	"os"
	"time"

	udp "github.com/mangenotwork/udp_comm"
)

// 保存c端put来的数据
var testFile = "test.txt"

func main() {
	// 初始化 s端
	servers, err := udp.NewServers("0.0.0.0", 12345)
	if err != nil {
		panic(err)
	}

	// 定义put方法
	servers.PutHandleFunc("case1", Case1)
	servers.PutHandleFunc("case2", Case2)
	// 定义get方法
	servers.GetHandleFunc("case3", Case3)

	// 每5秒发送一个通知
	go func() {
		for {
			time.Sleep(5 * time.Second)
			servers.OnLineTable()
			// 发送一个通知 [测试put]
			rse, rseErr := servers.Notice("", "testNotice", []byte("testNotice"),
				servers.SetNoticeRetry(2, 3000))
			if rseErr != nil {
				udp.Error(rseErr)
				udp.Info("[Servers 测试notice] failed")
				continue
			}
			udp.Info("[Servers 测试notice] passed")
			udp.Info(rse)
		}
	}()

	// 启动servers
	servers.Run()
}

func Case1(s *udp.Servers, c *udp.ClientInfo, body []byte) {
	udp.Info("Case1 func --> ")
	udp.Info("收到的数据: ", string(body))
	// 发送get,获取客户端信息
	rse, err := s.Get("getClient", "", []byte("getClient"))
	if err != nil {
		udp.Info("[Servers 测试get] failed")
		return
	}
	udp.Info(string(rse), err)
	udp.Info("[Servers 测试get] passed")
}

func Case2(s *udp.Servers, c *udp.ClientInfo, body []byte) {
	udp.Info("Case2 func --> ", string(body))
	udp.Info("[Client 测试put] passed")
	file, err := os.OpenFile(testFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		udp.Error(err)
	}
	defer func() {
		_ = file.Close()
	}()
	content := []byte(string(body) + "\n")
	_, err = file.Write(content)
	if err != nil {
		panic(err)
	}
}

func Case3(s *udp.Servers, param []byte) (int, []byte) {
	udp.Info("获取到的请求参数  param = ", string(param))
	return 0, []byte(fmt.Sprintf("服务器名称 %s.", s.GetServersName()))
}
