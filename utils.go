package udp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func int64ToBytes(n int64) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	err := binary.Write(buf, binary.BigEndian, n)
	return buf.Bytes(), err
}

func intToBytes(n int) ([]byte, error) {
	return int64ToBytes(int64(n))
}

func bytesToInt(bys []byte) (int, error) {
	i, err := bytesToInt64(bys)
	return int(i), err
}

func bytesToInt64(bys []byte) (int64, error) {
	buf := bytes.NewBuffer(bys)
	var data int64
	err := binary.Read(buf, binary.BigEndian, &data)
	return data, err
}

func formatName(str string) string {
	ln := len(str)
	if ln > 0 && ln <= 7 {
		// 补齐位
		for i := 0; i < 7-ln; i++ {
			str += " "
		}
	}
	return str
}

// LogClose 是否关闭日志
var LogClose bool = true
var std = newStd()

// CloseLog 关闭日志
func CloseLog() {
	LogClose = false
}

type logger struct {
	outFile       bool
	outFileWriter *os.File
}

func newStd() *logger {
	return &logger{}
}

func SetLogFile(name string) {
	std.outFile = true
	std.outFileWriter, _ = os.OpenFile(name+time.Now().Format("-20060102")+".log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
}

type Level int

var LevelMap = map[Level]string{
	1: "[Info]  ",
	4: "[Error] ",
}

func (l *logger) Log(level Level, args string, times int) {
	var buffer bytes.Buffer
	buffer.WriteString(time.Now().Format("2006-01-02 15:04:05.000 "))
	buffer.WriteString(LevelMap[level])
	_, file, line, _ := runtime.Caller(times)
	fileList := strings.Split(file, "/")
	// 最多显示两级路径
	if len(fileList) > 3 {
		fileList = fileList[len(fileList)-3 : len(fileList)]
	}
	buffer.WriteString(strings.Join(fileList, "/"))
	buffer.WriteString(":")
	buffer.WriteString(strconv.Itoa(line))
	buffer.WriteString(" \t| ")
	buffer.WriteString(args)
	buffer.WriteString("\n")
	out := buffer.Bytes()
	if LogClose {
		_, _ = buffer.WriteTo(os.Stdout)
	}
	if l.outFile {
		_, _ = l.outFileWriter.Write(out)
	}
}

func Info(args ...interface{}) {
	std.Log(1, fmt.Sprint(args...), 2)
}

func InfoF(format string, args ...interface{}) {
	std.Log(1, fmt.Sprintf(format, args...), 2)
}

func Error(args ...interface{}) {
	std.Log(4, fmt.Sprint(args...), 2)
}

func ErrorF(format string, args ...interface{}) {
	std.Log(4, fmt.Sprintf(format, args...), 2)
}

type IdWorker struct {
	startTime             int64
	workerIdBits          uint
	datacenterIdBits      uint
	maxWorkerId           int64
	maxDatacenterId       int64
	sequenceBits          uint
	workerIdLeftShift     uint
	datacenterIdLeftShift uint
	timestampLeftShift    uint
	sequenceMask          int64
	workerId              int64
	datacenterId          int64
	sequence              int64
	lastTimestamp         int64
	signMask              int64
	idLock                *sync.Mutex
}

func (idw *IdWorker) InitIdWorker(workerId, datacenterId int64) error {
	var baseValue int64 = -1
	idw.startTime = 1463834116272
	idw.workerIdBits = 5
	idw.datacenterIdBits = 5
	idw.maxWorkerId = baseValue ^ (baseValue << idw.workerIdBits)
	idw.maxDatacenterId = baseValue ^ (baseValue << idw.datacenterIdBits)
	idw.sequenceBits = 12
	idw.workerIdLeftShift = idw.sequenceBits
	idw.datacenterIdLeftShift = idw.workerIdBits + idw.workerIdLeftShift
	idw.timestampLeftShift = idw.datacenterIdBits + idw.datacenterIdLeftShift
	idw.sequenceMask = baseValue ^ (baseValue << idw.sequenceBits)
	idw.sequence = 0
	idw.lastTimestamp = -1
	idw.signMask = ^baseValue + 1
	idw.idLock = &sync.Mutex{}
	if idw.workerId < 0 || idw.workerId > idw.maxWorkerId {
		return fmt.Errorf("workerId[%v] is less than 0 or greater than maxWorkerId[%v].",
			workerId, datacenterId)
	}
	if idw.datacenterId < 0 || idw.datacenterId > idw.maxDatacenterId {
		return fmt.Errorf("datacenterId[%d] is less than 0 or greater than maxDatacenterId[%d].",
			workerId, datacenterId)
	}
	idw.workerId = workerId
	idw.datacenterId = datacenterId
	return nil
}

// NextId 返回一个唯一的 INT64 ID
func (idw *IdWorker) NextId() (int64, error) {
	idw.idLock.Lock()
	timestamp := time.Now().UnixNano()
	if timestamp < idw.lastTimestamp {
		return -1, fmt.Errorf(fmt.Sprintf("Clock moved backwards.  Refusing to generate id for %d milliseconds",
			idw.lastTimestamp-timestamp))
	}
	if timestamp == idw.lastTimestamp {
		idw.sequence = (idw.sequence + 1) & idw.sequenceMask
		if idw.sequence == 0 {
			timestamp = idw.tilNextMillis()
			idw.sequence = 0
		}
	} else {
		idw.sequence = 0
	}
	idw.lastTimestamp = timestamp
	idw.idLock.Unlock()
	id := ((timestamp - idw.startTime) << idw.timestampLeftShift) |
		(idw.datacenterId << idw.datacenterIdLeftShift) |
		(idw.workerId << idw.workerIdLeftShift) |
		idw.sequence
	if id < 0 {
		id = -id
	}
	return id, nil
}

// tilNextMillis
func (idw *IdWorker) tilNextMillis() int64 {
	timestamp := time.Now().UnixNano()
	if timestamp <= idw.lastTimestamp {
		timestamp = time.Now().UnixNano() / int64(time.Millisecond)
	}
	return timestamp
}

func ID64() (int64, error) {
	currWorker := &IdWorker{}
	err := currWorker.InitIdWorker(1000, 2)
	if err != nil {
		return 0, err
	}
	return currWorker.NextId()
}

func id() int64 {
	id, _ := ID64()
	return id
}
