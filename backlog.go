package udp

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

// backlog 积压的数据，所有发送的数据都会到这里，只有服务端确认的数据才会被删除
var backlog sync.Map
var backlogCount int64 = 0
var backlogCountMax int64 = 10000 // 内存中最大积压数据包条数
var backlogCountMin int64 = 5000  // 持久化加载的最小量级
var backlogFile = "%d.udb"

func backlogAdd(putId int64, putData PutData) {
	atomic.AddInt64(&backlogCount, 1)
	backlog.Store(putId, putData)
	backlogStorage()
}

func backlogDel(putId int64) {
	atomic.AddInt64(&backlogCount, -1)
	backlog.Delete(putId)
}

func backlogLen() int64 {
	n := 0
	backlog.Range(func(key, value any) bool {
		n++
		return true
	})
	return int64(n)
}

// backlogStorage 持久化方案: 保护内存不持续增长,尽力保证server掉线后数据不丢失，监听非强制kill把数据持久化
// 只有当积压数据条数大于设定值(backlogCount > max)就将当前所有积压的数据持久化到磁盘，释放内存存放新的数据
// 当积压数据条数小于设定值(backlogCount < min)就把持久化数据写到积压内存
// 当监听到非强制kill把数据持久化
func backlogStorage() {
	if backlogLen() > backlogCountMax {
		Error("触发持久化...... backlogCount = ", backlogCount, " 真实len = ", backlogLen())
		toUdb()
	}
}

func toUdb() {
	if backlogCount < 1 {
		return
	}
	file, err := os.OpenFile(fmt.Sprintf(backlogFile, time.Now().Unix()), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		Error(err)
	}
	defer func() {
		_ = file.Close()
	}()
	backlog.Range(func(key, value any) bool {
		vb, vbErr := ObjToByte(value)
		_, vbErr = file.Write(vb)
		_, vbErr = file.Write([]byte("\n"))
		if vbErr != nil {
			Error(vbErr)
		}
		backlogDel(key.(int64))
		return true
	})
}

// BacklogLoad 加载持久化数据 并消费
func BacklogLoad() {
	if backlogLen() > backlogCountMin {
		Error("当前 队列 大于触发条件不加载 : ", backlogCount)
		return
	}
	files, err := ioutil.ReadDir(".")
	if err != nil {
		Error("error reading directory:", err)
		return
	}
	for _, file := range files {
		extension := path.Ext(file.Name())
		if extension == ".udb" {
			filePath := "./" + file.Name()
			// 删掉没用的文件
			if file.Size() == 0 {
				err := os.Remove(filePath)
				if err != nil {
					Error(err)
					return
				}
			}
			if file.Size() > 0 {
				Info(file.Name())
				fileToBacklog(filePath)
				if backlogCount > backlogCountMin {
					break
				}
			}
		}
	}
}

func fileToBacklog(fName string) {
	f, err := os.Open(fName)
	if err != nil {
		Error(err)
		return
	}
	putDataList1 := make([]PutData, 0)
	putDataList2 := make([]PutData, 0)
	var n int64 = 0
	reader := bufio.NewReader(f)
	for {
		n++
		line, _, linErr := reader.ReadLine()
		if linErr == io.EOF {
			break
		}
		//Info(string(line))
		putData := PutData{}
		err = ByteToObj(line, &putData)
		if err != nil {
			Error(err)
			continue
		}
		//Info("持久化 putData = ", putData)
		if n < int64(backlogCountMax/2)+1 {
			putDataList1 = append(putDataList1, putData)
		} else {
			putDataList2 = append(putDataList2, putData)
		}
		if err != nil {
			Error(err)
			return
		}
	}
	for _, v := range putDataList1 {
		Error("加入 数据到队列 ... ")
		backlogAdd(v.Id, v)
		Error("加入后的count = ", backlogCount)
	}
	_ = f.Close()
	resetBacklogFile(fName, putDataList2)
}

func resetBacklogFile(fName string, putDataList []PutData) {
	err := os.Remove(fName)
	if err != nil {
		Error(err)
		return
	}
	if len(putDataList) < 1 {
		return
	}
	file, err := os.OpenFile(fName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		Error(err)
		return
	}
	defer func() {
		_ = file.Close()
	}()
	for _, v := range putDataList {
		vb, vbErr := ObjToByte(v)
		_, vbErr = file.Write(vb)
		_, vbErr = file.Write([]byte("\n"))
		if vbErr != nil {
			Error(vbErr)
		}
	}
}
