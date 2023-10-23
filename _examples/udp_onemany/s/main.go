package main

import (
	"fmt"
	"os"

	udp "github.com/mangenotwork/udp_comm"
)

// 保存c端put来的数据
var testFile = "test.txt"

func main() {
	servers, err := udp.NewServers("0.0.0.0", 12345)
	if err != nil {
		panic(err)
	}
	servers.PutHandleFunc("putCase", putCase)
	servers.GetHandleFunc("getCase", getCase)
	// 启动servers
	go func() {
		for {
			//time.Sleep(1 * time.Millisecond)
			servers.OnLineTable()
			// 发送get,获取客户端信息
			rse, err := servers.Get("get", "node1", []byte("getClient"))
			rse, err = servers.Get("get", "node2", []byte("getClient"))
			rse, err = servers.Get("get", "node3", []byte("getClient"))
			rse, err = servers.Get("get", "node4", []byte("getClient"))
			rse, err = servers.Get("get", "node5", []byte("getClient"))
			if err != nil {
				udp.Info("[Servers 测试get] failed. err = ", err)
				continue
			}
			udp.Info("Get 客户端结果 ... ", string(rse))

			// 发送一个通知 [测试put]
			rseN, rseErr := servers.Notice("node2", "notice", []byte("testNotice"), nil)
			if rseErr != nil {
				udp.Error(rseErr)
				udp.Info("[Servers 测试notice] failed")
				continue
			}
			udp.Info("[Servers 测试notice] passed")
			udp.Info(rseN)

			servers.NoticeAll("notice", []byte("testNotice"), nil)

		}
	}()
	servers.Run()
}

func putCase(s *udp.Servers, c *udp.ClientInfo, body []byte) {
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

func getCase(s *udp.Servers, param []byte) (int, []byte) {
	udp.Info("获取到的请求参数  param = ", string(param))
	return 0, []byte(fmt.Sprintf("服务器名称 %s.", s.GetServersName()))
}
