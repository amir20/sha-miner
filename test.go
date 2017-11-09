package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"os"
	"time"
	"fmt"
)

func main() {
	bytes := argToBytes()
	total := len(bytes)

	buffer := make([]byte, total+8)
	copy(buffer, bytes)

	var maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	threshold := new(big.Int).Div(maxUint256, big.NewInt(1000000000))

	start := time.Now()
	found := int64(0)
	for i := int64(0); i < 1<<63-1; i++ {
		binary.PutVarint(buffer[total:], i)
		sum := sha256.Sum256(buffer)

		value := new(big.Int).SetBytes(sum[:])

		if value.Cmp(threshold) < 0 {
			println("Found match", value.String())
			found = i
			break
		}
	}

	hashesPerSecond := found / int64(time.Since(start).Seconds())

	fmt.Printf("Found a match with hashrate of %.2f MH/s\n", float32(hashesPerSecond)/float32(1000000))
}

func argToBytes() []byte {
	var buffer bytes.Buffer
	for i := 1; i < len(os.Args); i++ {
		buffer.WriteString(os.Args[i])
	}

	return buffer.Bytes()
}
