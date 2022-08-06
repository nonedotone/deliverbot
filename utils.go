package main

import (
	"encoding/binary"
	"math/rand"
	"strconv"
	"time"
)

// BytesConcat concat bytes
func BytesConcat(slices ...[]byte) []byte {
	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}
	tmp := make([]byte, totalLen)
	var i int
	for _, s := range slices {
		i += copy(tmp[i:], s)
	}
	return tmp
}

// Int64ToBytes parse int64 to bytes
func Int64ToBytes(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// BytesToInt64 parse bytes to int64
func BytesToInt64(v []byte) int64 {
	if len(v) == 0 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(v))
}

// Uint64ToBytes parse uint64 to bytes
func Uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// BytesToUint64 parse bytes to uint64
func BytesToUint64(v []byte) uint64 {
	if len(v) == 0 {
		return 0
	}
	return binary.BigEndian.Uint64(v)
}

func PrefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}

	end := make([]byte, len(prefix))
	copy(end, prefix)

	for {
		if end[len(end)-1] != byte(255) {
			end[len(end)-1]++
			break
		}

		end = end[:len(end)-1]

		if len(end) == 0 {
			end = nil
			break
		}
	}

	return end
}

var (
	defaultLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

func RandomString(n int, allowedChars ...[]rune) string {
	rand.Seed(time.Now().UnixNano())
	var letters []rune
	if len(allowedChars) == 0 {
		letters = defaultLetters
	} else {
		letters = allowedChars[0]
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func StringToInt64(str string) int64 {
	i, _ := strconv.ParseInt(str, 10, 64)
	return i

}
