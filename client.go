package udp

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Client struct {
	ServersHost  string           // serversIP:port
	Conn         *net.UDPConn     // 连接对象
	SConn        *net.UDPAddr     // s端连接信息
	name         string           // client的名称
	connectCode  string           // 连接code 是静态的由server端配发
	state        int              // 0:未连接   1:连接成功  2:server端丢失
	sign         string           // 签名
	secretKey    string           // 数据传输加密解密秘钥
	GetHandle    ClientGetFunc    // get方法
	NoticeHandle ClientNoticeFunc // 接收通知的方法
}

type ClientConf struct {
	Name        string
	ConnectCode string
	SecretKey   string // 数据传输加密解密秘钥
}

func SetClientConf(clientName, connectCode, secretKey string) ClientConf {
	return ClientConf{
		Name:        clientName,
		ConnectCode: connectCode,
		SecretKey:   secretKey,
	}
}

func NewClient(host string, conf ...ClientConf) (*Client, error) {
	c := &Client{
		ServersHost:  host,
		state:        0,
		GetHandle:    make(ClientGetFunc),
		NoticeHandle: make(ClientNoticeFunc),
	}
	if len(conf) >= 1 {
		if len(conf[0].ConnectCode) > 0 {
			c.connectCode = conf[0].ConnectCode
		}
		if strings.IndexAny(conf[0].Name, "@") != -1 {
			return nil, ErrClientNameErr
		}
		if len(conf[0].Name) > 7 {
			return nil, ErrNmeLengthAbove
		}
		if len(conf[0].Name) > 0 && len(conf[0].Name) <= 7 {
			c.name = conf[0].Name
		}
		if len(conf[0].SecretKey) != 8 {
			return nil, fmt.Errorf("秘钥的长度只能为8，并且与Servers端统一")
		} else if len(conf[0].SecretKey) == 0 {
			c.secretKey = DefaultSecretKey
		} else {
			c.secretKey = conf[0].SecretKey
		}
	} else {
		c.DefaultClientName()
		c.DefaultConnectCode()
		c.DefaultSecretKey()
	}
	sHost := strings.Split(c.ServersHost, ":")
	sip := net.ParseIP(sHost[0])
	sport, err := strconv.Atoi(sHost[1])
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{IP: sip, Port: sport}
	c.Conn, err = net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		Error(err)
	}
	// 连接服务器
	c.ConnectServers()
	return c, nil
}

func (c *Client) SetClientName(name string) error {
	if strings.IndexAny(name, "@") != -1 {
		return ErrClientNameErr
	}
	if len(name) > 0 && len(name) <= 7 {
		c.name = name
		return nil
	}
	return ErrNmeLengthAbove
}

func (c *Client) SetConnectCode(code string) {
	c.connectCode = code
}

func (c *Client) SetSecretKey(key string) error {
	if len(key) != 8 {
		return ErrClientSecretKey
	}
	c.secretKey = key
	return nil
}

func (c *Client) Run() {
	// 时间轮,心跳维护，动态刷新签名
	c.timeWheel()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		select {
		case s := <-ch:
			Info("Client退出....")
			toUdb() // 将积压的数据持久化
			if i, ok := s.(syscall.Signal); ok {
				os.Exit(int(i))
			} else {
				os.Exit(0)
			}
		}
	}()

	// 启动与servers进行交互
	data := make([]byte, 1024)
	for {
		n, remoteAddr, err := c.Conn.ReadFromUDP(data)
		if err != nil {
			Error(err)
			c.state = 0 // 连接有异常更新连接状态
			continue
		}
		c.SConn = remoteAddr
		// Info("解包....size = ", n)
		packet, err := PacketDecrypt(c.secretKey, data, n)
		if err != nil {
			Error("错误的包 err:", err)
			continue
		}
		go func() {
			switch packet.Command {
			// 来自server端的通知消息
			case CommandNotice:
				notice := &NoticeData{}
				bErr := ByteToObj(packet.Data, &notice)
				if bErr != nil {
					Error("返回的包解析失败， err = ", err)
				}
				// 异步应答这个通知，然后处理执行通知
				go func() {
					notice.Response = []byte("ok")
					b, e := ObjToByte(notice)
					if e != nil {
						Error("ObjToByte err = ", e)
					}
					pack, pErr := PacketEncoder(CommandNotice, c.name, c.sign, c.secretKey, b)
					if pErr != nil {
						Error(pErr)
					}
					c.Write(pack)
				}()
				if fn, ok := c.NoticeHandle[notice.Label]; ok {
					fn(c, notice.Data)
				}

			// 来自server端的get请求
			case CommandGet:
				if c.sign != packet.Sign {
					Info("未知主机认证!")
					return
				}
				getData := &GetData{}
				bErr := ByteToObj(packet.Data, &getData)
				if bErr != nil {
					Error("解析put err :", bErr)
				}
				if fn, ok := c.GetHandle[getData.Label]; ok {
					code, rse := fn(c, getData.Param)
					getData.Response = rse
					gb, gbErr := ObjToByte(getData)
					if gbErr != nil {
						Error("对象转字节错误...")
					}
					c.ReplyGet(getData.Id, code, gb)
				}

			case CommandReply:
				reply := &Reply{}
				bErr := ByteToObj(packet.Data, &reply)
				if bErr != nil {
					Error("返回的包解析失败， err = ", bErr)
				}
				switch CommandCode(reply.Type) {
				case CommandConnect: // 连接包与心跳包的反馈会触发
					// 存储签名
					c.sign = string(reply.Data)
					c.state = 1
					// 将积压的数据进行发送
					c.SendBacklog()
				case CommandPut:
					if c.sign != packet.Sign {
						Error("未知主机认证!")
						return
					}
					if reply.StateCode != 0 {
						// 签名错误
						Error("签名错误")
						break
					}
					// 服务端以确认收到删除对应的数据
					backlogDel(reply.CtxId)

				case CommandGet:
					if c.sign != packet.Sign {
						Error("未知主机认证!")
						return
					}
					getData := &GetData{}
					boErr := ByteToObj(reply.Data, &getData)
					if boErr != nil {
						Error("解析put err :", boErr)
					}
					getF, _ := GetDataMap.Load(getData.Id)
					if getF != nil {
						getF.(*GetData).Response = getData.Response
						getF.(*GetData).ctxChan <- true
					}
				}
			}
		}()
	}
}

func (c *Client) Close() {
	if c.Conn == nil {
		return
	}
	err := c.Conn.Close()
	if err != nil {
		Error(err.Error())
	}
}

func (c *Client) Write(data []byte) {
	_, err := c.Conn.Write(data)
	if err != nil {
		ErrorF("error write: %s", err.Error())
	}
}

// Put client put
// 向服务端发送数据，如果服务端未在线数据会被积压，等服务器恢复后积压数据会一并发送
func (c *Client) Put(funcLabel string, data []byte) {
	putData := PutData{
		Label: funcLabel,
		Id:    id(),
		Body:  data,
	}
	// 数据被积压，占时保存
	backlogAdd(putData.Id, putData)
	// 未与servers端确认连接，不发送数据
	if c.state != 1 {
		return
	}
	b, err := ObjToByte(putData)
	if err != nil {
		Error("ObjToByte err = ", err)
	}
	packet, err := PacketEncoder(CommandPut, c.name, c.sign, c.secretKey, b)
	if err != nil {
		Error(err)
	}
	c.Write(packet)
}

// 向服务端获取数据，指定一个超时时间，未应答就超时
func (c *Client) get(timeOut int, funcLabel string, param []byte) ([]byte, error) {
	getData := &GetData{
		Label:    funcLabel,
		Id:       id(),
		Param:    param,
		ctxChan:  make(chan bool),
		Response: make([]byte, 0),
	}
	GetDataMap.Store(getData.Id, getData)
	b, err := ObjToByte(getData)
	if err != nil {
		Error("ObjToByte err = ", err)
	}
	packet, err := PacketEncoder(CommandGet, c.name, c.sign, c.secretKey, b)
	if err != nil {
		Error(err)
	}
	c.Write(packet)
	select {
	case <-getData.ctxChan:
		res := getData.Response
		GetDataMap.Delete(getData.Id)
		return res, nil
	case <-time.After(time.Millisecond * time.Duration(timeOut)):
		GetDataMap.Delete(getData.Id)
		return nil, ErrSGetTimeOut(funcLabel, "servers", c.SConn.String())
	}

}

// ReplyGet 返回put  state:0x0 成功   state:0x1 签名失败  state:2 业务层面的失败
func (c *Client) ReplyGet(id int64, state int, data []byte) {
	reply := &Reply{
		Type:      int(CommandGet),
		CtxId:     id,
		Data:      data,
		StateCode: state,
	}
	b, e := ObjToByte(reply)
	if e != nil {
		Error("打包数据失败, e= ", e)
	}
	data, err := PacketEncoder(CommandReply, c.name, c.sign, c.secretKey, b)
	if err != nil {
		Error(err)
	}
	c.Write(data)
}

func (c *Client) Get(funcLabel string, param []byte) ([]byte, error) {
	return c.get(1000, funcLabel, param)
}

func (c *Client) GetTimeOut(funcLabel string, param []byte, timeOut int) ([]byte, error) {
	return c.get(timeOut, funcLabel, param)
}

func (c *Client) GetHandleFunc(label string, f func(c *Client, param []byte) (int, []byte)) {
	c.GetHandle[label] = f
}

func (c *Client) NoticeHandleFunc(label string, f func(c *Client, data []byte)) {
	c.NoticeHandle[label] = f
}

// ConnectServers 请求连接服务器，获取签名
// 内容是发送 Connect code
func (c *Client) ConnectServers() {
	data, err := PacketEncoder(CommandConnect, c.name, c.sign, c.secretKey, []byte(c.connectCode))
	if err != nil {
		Error(err)
	}
	c.Write(data)
}

func (c *Client) GetName() string {
	return c.name
}

func (c *Client) DefaultClientName() {
	c.name = DefaultClientName
}

func (c *Client) DefaultConnectCode() {
	c.connectCode = DefaultConnectCode
}

func (c *Client) DefaultSecretKey() {
	c.secretKey = DefaultSecretKey
}

// 时间轮，持续制定时间发送心跳包
func (c *Client) timeWheel() {
	go func() {
		tTime := time.Duration(HeartbeatTime) // 时间轮5秒
		for {
			// 5s维护一个心跳，s端收到心跳会返回新的签名
			timer := time.NewTimer(tTime * time.Second)
			select {
			case <-timer.C:
				// 这个时候表示连接不存在
				c.state = 0
				data, err := PacketEncoder(CommandHeartbeat, c.name, c.sign, c.secretKey, []byte(c.connectCode))
				if err != nil {
					Error(err)
				}
				c.Write(data)
			}
		}
	}()
}

// SendBacklog 发送积压的数据，
func (c *Client) SendBacklog() {
	backlog.Range(func(key, value any) bool {
		if value == nil {
			return true
		}
		b, err := ObjToByte(value.(PutData))
		if err != nil {
			Error("ObjToByte err = ", err)
		}
		packet, err := PacketEncoder(CommandPut, c.name, c.sign, c.secretKey, b)
		if err != nil {
			Error(err)
		}
		c.Write(packet)
		return true
	})
	// 如果存在持久化积压数据则进行发送
	BacklogLoad()
}
