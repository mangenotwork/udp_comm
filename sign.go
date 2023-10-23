package udp

import (
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func createSign() string {
	b := make([]byte, 7)
	for i := range b {
		b[i] = SignLetterBytes[rand.Intn(len(SignLetterBytes))]
	}
	return string(b)
}

var signMap sync.Map

func SignStore(addr, sign string) {
	signMap.Store(addr, sign)
}

func SignCheck(addr, sign string) bool {
	v, ok := signMap.Load(addr)
	if ok && v.(string) == sign {
		return true
	}
	return false
}

func SignGet(addr string) string {
	v, ok := signMap.Load(addr)
	if !ok {
		return ""
	}
	return v.(string)
}
