# beacon-tower
基于传输层UDP设计和封装的应用层网络传输协议，目的是为高效开发CS架构项目的数据传输功能提供支持。

## 基于UDP的网络传输协议
udp对要求实时性业务场景很友好，对udp在进一步设计可以有效避免其缺点，保障了完全性，可靠性，完整性；
使用本协议可以高效开发实时性场景，分布式场景，低功耗交互式场景的应用，本协议也在实际场景中得到了验证。


### 设计:

数据包:
```
Packet 包设计
______________________________________________________________________
|            |              |             |                           |
| 指令(1字节) |  name(7字节)  | 签名(7字节)  |  data(建议小于533字节)...  |
|____________|______________|_____________|___________________________|

指令: 区分是什么数据 Connect,Put,Reply,Heartbeat,Notice,Get
name: 主要场景s端指定广播，name对应多个ip(节点)
签名: 用于确保数据安全，签名会更具心跳进行动态签发
data: 传输的数据，不支持分包，建议小于533字节，可以在业务中设计分次传输

封包 : 装载数据 -> 压缩 -> 加密  
解包 : 解密 -> 解压 -> 匹配指令 -> 验证签名

```

是如何提升安全性?
1. 采用连接认证机制  
2. IP黑白名单
3. 动态签名机制   
4. 数据加解密 

是如何提升可靠性?
- 心跳与时间轮机制
- 数据传输确认机制
- 数据积压机制
- 数据重传机制

其他?
- 数据压缩


限制: 
数据包应小而独立，大数据包应在业务层进行拆分

### 基础
#### S 端有 Notice(通知), Get(获取) 两种通讯方法

Notice
1. 一对多发送通知
2. 支持重传
3. 指定节点发送通知

Get
1. 获取C端数据
2. 超时报错
3. 存储C端的连接信息 一个name对应多个连接地址
4. 最佳场景是设置每个C端独立名称对应一个连接地址

#### C 端有 Put(发送), Get(获取) 两种通讯方法

Put
1. 发送数据包
2. 积压模式: 每个数据包都会被积压，只有当s端确认接收后清除，当心跳包确认后触发积压数据重传
3. 积压数据持久化: 积压数据包到达一定量被持久化到磁盘，重传时积压数据小于指定值读取持久化数据一半的数据量
4. C端收到信号量 SIGTERM, SIGINT, SIGKILL, SIGHUP, SIGQUIT 当前积压数据包全部持久化

Get
1. 获取C端数据
2. 超时报错


### 安全

1. 使用 DES ECB 对数据包加解密，保障数据被抓包并非明文
2. 连接Code用于确保两端下发签名的识别
3. 每次收到心跳包重新颁发签名
4. 除连接包和心跳包都会确认签名

### 如何在弱网环境下保障数据的传输可靠性
重传:
S端采用通知的方式广播数据包,在此上设计了确认机制，如果在指定时间内收不到C端的确认包就会触发重传，重传是可配置的;

积压:
C端采用Put方式上传数据到S端，在此上设计了数据包积压机制，只有当收到S端对应数据包id的确认包到才会将此条数据包移除,
在确认连接成功后触发积压包重传，心跳包的时间节点维护积压数据包的持久化;


### 例子
servers
```go
package main

import (
	"github.com/mangenotwork/beacon-tower/udp"
	"fmt"
	"os"
	"time"
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
			// 发送一个通知
			rse, rseErr := servers.Notice("", "testNotice", []byte("testNotice"),
				servers.SetNoticeRetry(2, 3000))
			if rseErr != nil {
				udp.Error(rseErr)
				continue
			}
			udp.Info(rse)
		}
	}()
	// 启动servers
	servers.Run()
}

func Case1(s *udp.Servers, body []byte) {
	udp.Info("收到的数据: ", string(body))
	// 发送get,获取客户端信息
	rse, err := s.Get("getClient", "", []byte("getClient"))
	if err != nil {
		return
	}
	udp.Info(string(rse), err)
}

func Case2(s *udp.Servers, body []byte) {
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

```

client端
```go
package main

import (
	"github.com/mangenotwork/beacon-tower/udp"
	"fmt"
	"time"
)

func main() {
	// 定义客户端
	client, err := udp.NewClient("192.168.3.86:12345")
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
			client.Put("case2", []byte(fmt.Sprintf("%d | hello : %d", time.Now().UnixNano(), n)))
			udp.Info("n = ", n)
			// get请求服务端的 case3 方法
			rse, err := client.Get("case3", []byte("test"))
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

func CGetTest(c *udp.Client, param []byte) (int, []byte) {
	udp.Info("获取到的请求参数  param = ", string(param))
	return 0, []byte(fmt.Sprintf("客户端名称 %s.", c.DefaultClientName))
}

func CNoticeTest(c *udp.Client, data []byte) {
	udp.Info("收到来自服务器的通知，开始执行......")
	udp.Info("data = ", string(data))
}
```

#### 更多例子

- 基础例子:  _examples/udp_base
- 一对多: _examples/udp_onemany
- 安全配置: _examples/udp_security


## 版本

v0.0.1
- 基于udp传输协议的基础设计和实现
- udp传输协议的实例

v0.0.2

