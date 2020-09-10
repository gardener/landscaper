package crypt

import (
	"math/rand"
	"time"
)

func GenerateSALT(length int) []byte {
	rand.Seed(time.Now().UnixNano())
	s := ""
	for i := 0; i < length; i++ {
		s += string(itoa64[rand.Uint64()%64])
	}
	return []byte(s)
}
