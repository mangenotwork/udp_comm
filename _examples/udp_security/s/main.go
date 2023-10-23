package main

import (
	"time"

	udp "github.com/mangenotwork/udp_comm"
)

func main() {
	s, err := udp.NewServers("0.0.0.0", 12346,
		udp.SetServersConf("s1", "123456", "abc12345"))
	if err != nil {
		panic(err)
	}
	// 每5秒发送一个通知
	go func() {
		for {
			time.Sleep(5 * time.Second)
			// 发送一个通知 [测试put]
			s.OnLineTable()
			rse, rseErr := s.Notice("node1", "testNotice", []byte("testNotice"), nil)
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
	s.Run()
}
