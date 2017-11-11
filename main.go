package main

import (
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"runtime"
	metrics2 "github.com/rcrowley/go-metrics"
	"fmt"
	flag "github.com/spf13/pflag"
	"time"
)

const (
	maxUint64         = ^uint64(0)
	defaultDifficulty = 100000000
	defaultMessage    = "Hello world."
)

var (
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	threshold  *big.Int
)

func main() {
	difficulty := flag.Uint64P("difficulty", "d", defaultDifficulty, "Difficulty value to use for mining")
	message := flag.StringP("message", "m", defaultMessage, "Message to compute the hash")
	threads := flag.IntP("threads", "t", runtime.NumCPU(), "Total number of threads to use. Defaults to number of CPUs")

	flag.Parse()

	threshold = new(big.Int).Div(maxUint256, big.NewInt(int64(*difficulty)))
	data := []byte(*message)

	abort := make(chan struct{})
	found := make(chan uint64)
	defer close(found)

	delta := maxUint64 / uint64(*threads)
	meter := metrics2.NewMeter()
	for i := 0; i < *threads; i++ {
		start := uint64(i) * delta
		go mine(start, data, found, abort, meter)
	}

	go status(abort, meter)

	select {
	case result := <-found:
		close(abort)
		fmt.Printf("Found nonce %d with hashrate of %.2f MH/s\n", result, meter.Rate1()/1000000)
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
			if attempt%(1<<16) == 0 {
				meter.Mark(attempt)
				attempt = 0

			}
		}
	}
}

func status(abort <-chan struct{}, meter metrics2.Meter) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			fmt.Printf("Effective Hashrate is %.2f MH/s\n", meter.Rate1()/1000000)
		case <-abort:
			ticker.Stop()
			return
		}
	}
}
