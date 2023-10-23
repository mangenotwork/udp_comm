package udp

// 指令使用1个字节

type CommandCode uint8

const (
	CommandConnect   CommandCode = 0x0 // 首次连接确认身份信息
	CommandPut       CommandCode = 0x1 // 发送消息
	CommandReply     CommandCode = 0x2 // 收到回应， ackType: put, heartbeat, sign
	CommandHeartbeat CommandCode = 0x3 // 发送心跳
	CommandNotice    CommandCode = 0x4 // 下发签名
	CommandGet       CommandCode = 0x5 // 获取消息
)

// CommandPut,CommandGet  必须验证签名，否则不接收， 签名由client主导

// 签名逻辑
// 1. c:CommandConnect 发送请求连接
// 2. s验证请求code
// 3. s:CommandSign  下发签名
// 4. c存储sign
// 5. c:CommandReply   回应
// 6. s:收到回应  更新c对应的sign

// 特殊情况1: 如果s端断线，c端只发心跳包收到回应再发送连接请求，这个时候积压数据包
// 特殊情况2: 如果签名失败，c端就一直请求签名
