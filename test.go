package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"os"
	"runtime"
	metrics2 "github.com/rcrowley/go-metrics"
	"fmt"
)

const (
	maxUint64  = ^uint64(0)
	difficulty = 100000000
)

var (
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	threshold  = new(big.Int).Div(maxUint256, big.NewInt(difficulty))
)

func main() {
	abort := make(chan struct{})
	found := make(chan uint64)
	defer close(found)

	data := argToBytes()
	delta := maxUint64 / uint64(runtime.NumCPU())

	metrics := metrics2.NewMeter()
	for i := 0; i < runtime.NumCPU(); i++ {
		start := uint64(i) * delta
		go mine(start, data, found, abort, metrics)
	}

	select {
	case result := <-found:
		close(abort)
		fmt.Println("Found result", result)
		fmt.Printf("Effective Hashrate is %.2f MH/s\n", metrics.Rate1()/1000000)
	}
}

func mine(start uint64, bytes []byte, found chan<- uint64, abort <-chan struct{}, meter metrics2.Meter) {
	total := len(bytes)
	buffer := make([]byte, total+8)
	copy(buffer, bytes)

	nonce := start
	attempt := int64(0)
	for {
		select {
		case <-abort:
			break

		default:
			binary.LittleEndian.PutUint64(buffer[total:], nonce)
			sum := sha256.Sum256(buffer)
			value := new(big.Int).SetBytes(sum[:])

			if value.Cmp(threshold) <= 0 {
				found <- nonce
				break
			}

			nonce++
			attempt++
			if attempt % (1<<16) == 0 {
				meter.Mark(attempt)
				attempt = 0

			}
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
