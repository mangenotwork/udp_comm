package udp

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/des"
	"encoding/binary"
	"encoding/json"
	"io"
)

/*

Packet 包设计
______________________________________________________________________
|            |              |             |                           |
| 指令(1字节) |  name(7字节)  | 签名(7字节)  |  data(建议小于533字节)...  |
|____________|______________|_____________|___________________________|

指令: 区分是什么数据
name: 主要场景s端指定广播，name对应多个ip(节点)
签名: 用于确保数据安全，签名会更具心跳进行动态签发
data: 传输的数据，不支持分包，建议小于533字节，可以在业务中设计分次传输

包安全: des加密保障数据包不是明文传输
包压缩: 使用Zlib

场景:
1. c -> s ; 必须建立连接
2. s -> c ; 建立好连接下发签名
3. c:put -> s ; c端根据自己的业务进行发包，根据s端的应答确认发送完成
4. s:reply -> c ; s端确认收到 发送ACK

*/

type Packet struct {
	Command CommandCode
	Name    string
	Sign    string
	Data    []byte
}

// PacketEncoder 封包
func PacketEncoder(cmd CommandCode, name, sign, secret string, data []byte) ([]byte, error) {
	var (
		err    error
		stream []byte
		buf    = new(bytes.Buffer)
	)
	_ = binary.Write(buf, binary.LittleEndian, cmd)
	ln := len(name)
	if ln > 0 && ln <= 7 {
		// 补齐位
		for i := 0; i < 7-ln; i++ {
			name += " "
		}
		_ = binary.Write(buf, binary.LittleEndian, []byte(name))
	} else if ln > 7 {
		return nil, ErrNmeLengthAbove
	} else {
		_ = binary.Write(buf, binary.LittleEndian, []byte("0000000"))
	}
	if len(sign) != 7 {
		_ = binary.Write(buf, binary.LittleEndian, []byte("0000000"))
	} else {
		_ = binary.Write(buf, binary.LittleEndian, []byte(sign))
	}
	//Info("源数据 : ", len(data))
	// 压缩数据
	//d := GzipCompress(data)
	dCompress := ZlibCompress(data)
	//Info("压缩后数据长度: ", len(d))

	// 加密数据
	dEncrypt := DesECBEncrypt(dCompress, []byte(secret))
	//Info("加密数据 : ", len(d))

	if len(dEncrypt) > 540 {
		Error(ErrDataLengthAbove)
	}
	err = binary.Write(buf, binary.LittleEndian, dEncrypt)
	if err != nil {
		return stream, err
	}
	stream = buf.Bytes()
	return stream, nil
}

// PacketDecrypt 解包
func PacketDecrypt(secret string, data []byte, n int) (*Packet, error) {
	var err error
	if n < 15 {
		Error("空包")
		return nil, ErrNonePacket
	}
	command := CommandCode(data[0:1][0])
	name := string(data[1:8])
	sign := string(data[8:15])
	b := data[15:n]
	// 解密数据
	bDecrypt := DesECBDecrypt(b, []byte(secret))
	// 解压数据
	//b, err := GzipDecompress(data[15:n])
	bDecompress, err := ZlibDecompress(bDecrypt)
	if err != nil {
		Error("解压数据失败 err: ", err)
		return nil, err
	}
	return &Packet{
		Command: command,
		Name:    name,
		Sign:    sign,
		Data:    bDecompress,
	}, nil
}

func ObjToByte(obj interface{}) ([]byte, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return []byte(""), err
	}
	return b, nil
}

func ByteToObj(data []byte, obj interface{}) error {
	return json.Unmarshal(data, obj)
}

// GzipCompress gzip压缩
func GzipCompress(src []byte) []byte {
	var in bytes.Buffer
	w, err := gzip.NewWriterLevel(&in, gzip.BestCompression)
	_, err = w.Write(src)
	err = w.Close()
	if err != nil {
		Error(err)
	}
	return in.Bytes()
}

// GzipDecompress gzip解压
func GzipDecompress(src []byte) ([]byte, error) {
	reader := bytes.NewReader(src)
	gr, err := gzip.NewReader(reader)
	if err != nil {
		return []byte(""), err
	}
	bf := make([]byte, 0)
	buf := bytes.NewBuffer(bf)
	_, err = io.Copy(buf, gr)
	err = gr.Close()
	return buf.Bytes(), err
}

// ZlibCompress zlib压缩
func ZlibCompress(src []byte) []byte {
	buf := new(bytes.Buffer)
	//根据创建的buffer生成 zlib writer
	writer := zlib.NewWriter(buf)
	//写入数据
	_, err := writer.Write(src)
	err = writer.Close()
	if err != nil {
		Error(err)
	}
	return buf.Bytes()
}

// ZlibDecompress zlib解压
func ZlibDecompress(src []byte) ([]byte, error) {
	reader := bytes.NewReader(src)
	gr, err := zlib.NewReader(reader)
	if err != nil {
		return []byte(""), err
	}
	bf := make([]byte, 0)
	buf := bytes.NewBuffer(bf)
	_, err = io.Copy(buf, gr)
	err = gr.Close()
	return buf.Bytes(), err
}

func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	text := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, text...)
}

func pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}

func DesECBEncrypt(data, key []byte) []byte {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil
	}
	bs := block.BlockSize()
	data = pkcs5Padding(data, bs)
	if len(data)%bs != 0 {
		return nil
	}
	out := make([]byte, len(data))
	dst := out
	for len(data) > 0 {
		block.Encrypt(dst, data[:bs])
		data = data[bs:]
		dst = dst[bs:]
	}
	return out
}

func DesECBDecrypt(data, key []byte) []byte {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	block, err := des.NewCipher(key)
	if err != nil {
		return nil
	}
	bs := block.BlockSize()
	if len(data)%bs != 0 {
		return nil
	}
	out := make([]byte, len(data))
	dst := out
	for len(data) > 0 {
		block.Decrypt(dst, data[:bs])
		data = data[bs:]
		dst = dst[bs:]
	}
	out = pkcs5UnPadding(out)
	return out
}
