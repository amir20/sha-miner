package main

import (
	"os"
	"bytes"
	"encoding/binary"
	"crypto/sha256"
	"math/big"
)

func main() {
	bytes := argToBytes()
	total := len(bytes)

	buffer := make([]byte, total+8)
	copy(buffer, bytes)

	threshold := new(big.Int)
	threshold.SetString("00000099999999999999999999999999999999999999999999999999999999999999999999999", 10)

	for i := int64(0); i < 1<<63-1; i++ {
		binary.PutVarint(buffer[total:], i)
		sum := sha256.Sum256(buffer)

		value := new(big.Int)
		value.SetBytes(sum[:])

		if value.Cmp(threshold) < 0 {
			println("Found match", value.String())
			break
		}
	}
}

func argToBytes() []byte {
	var buffer bytes.Buffer
	for i := 1; i < len(os.Args); i++ {
		buffer.WriteString(os.Args[i])
	}

	return buffer.Bytes()
}
