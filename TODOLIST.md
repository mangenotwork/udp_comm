- &#9745; [udp]  连接机制    
- &#9745; [udp] 下发签名    
- &#9745; [udp] 签名认证    
- &#9745; [udp] des加密解密 
- &#9745; [udp] gzip压缩数据  
- &#9745; [udp] 提供交互方法 
- &#9745; [udp] 心跳与时间轮机制
- &#9745; [udp] 数据传输确认交互 
- &#9745; [udp] s端确认c端是否在线
- &#9745; [udp] 数据积压机制
- &#9745; [udp] 数据重传机制 
- &#9745; [udp] 积压数据持久化 
- &#9745; [udp] client put场景设计 
- &#9745; [udp] client get场景设计 
- &#9745; [udp] servers notice场景设计 
- &#9745; [udp] servers get场景设计 
- &#9745; [udp] 移除三方包，自定义日志 
- &#9745; [udp] 测试
- &#9745; [udp] 文档
- &#9745; [整体] 打包 v0.0.1
- &#9745; [udp] 实例编写
- &#9745; [udp] 实际应用 -> https://github.com/mangenotwork/website-monitor
- &#9745; [udp] S端PUT方法增加一个ClientInfo,用于PUT可知client
- &#9745; [整体] 打包 v0.0.2
- &#9744; [udp] S端设计一个Set应答，场景如收到C端的PUT可直接Set(作用于get,notice)
- &#9744; [udp] S端Get可以直接针对ClientInfo下发数据
- &#9744; [udp] Ping包设计，该Ping工具并不向主机发送ICMP请求，而是向服务器发送一个空udp请求,然后获得反馈


其他设计
