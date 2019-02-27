// Copyright 2017 Amir Raminfar

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	metrics2 "github.com/rcrowley/go-metrics"
	flag "github.com/spf13/pflag"
	"math/big"
	"runtime"
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
	// read the values from command line
	difficulty := flag.Uint64P("difficulty", "d", defaultDifficulty, "Difficulty value to use for mining")
	message := flag.StringP("message", "m", defaultMessage, "Message to compute the hash")
	threads := flag.IntP("threads", "t", runtime.NumCPU(), "Total number of threads to use. Defaults to number of CPUs")
	flag.Parse()

	// total max uint divided by the difficulty determines the threshold. Bigger difficulty means smaller threshold
	threshold = new(big.Int).Div(maxUint256, big.NewInt(int64(*difficulty)))
	data := []byte(*message)

	// create channels needed for results and abort
	abort := make(chan struct{})
	found := make(chan uint64)
	defer close(found)

	// split the total searchable space for nonce into equal chunks
	delta := maxUint64 / uint64(*threads)
	meter := metrics2.NewMeter()

	for i := 0; i < *threads; i++ {

		// start is the beginning of attempts for this thread which be 0, delta, 2 x delta, ...
		start := uint64(i) * delta
		go mine(start, data, found, abort, meter)
	}

	// create a new status thread
	go status(abort, meter)

	select {
	case result := <-found:
		close(abort)
		fmt.Printf("Found nonce %d with hashrate of %.2f MH/s\n", result, meter.Rate1()/1000000)
	}
}

// mine is a worker routine that does the actual work for each thread
func mine(start uint64, bytes []byte, found chan<- uint64, abort <-chan struct{}, meter metrics2.Meter) {

	// create a copy of the bytes and add extra padding for int64 (8 bytes)
	total := len(bytes)
	buffer := make([]byte, total+8)
	copy(buffer, bytes)

	nonce := start
	attempt := int64(0)

	for {
		select {
		case <-abort:
			// stop the worker and return
			return

		default:
			// convert the nonce candidate to bytes and append to the end of buffer
			// note that the buffer is reused here between each attempt by always overwriting the last 8 bytes
			binary.LittleEndian.PutUint64(buffer[total:], nonce)

			// compute the sha
			sum := sha256.Sum256(buffer)

			// convert the sha to big integer
			value := new(big.Int).SetBytes(sum[:])

			// if the integer value of the sha is less than the threshold then we have found a match
			if value.Cmp(threshold) <= 0 {
				found <- nonce
				return
			}

			nonce++
			attempt++
			// update the meter every 2^16 for better performance. This is purely optimization I saw in geth
			if attempt%(1<<16) == 0 {
				meter.Mark(attempt)
				attempt = 0
			}
		}
	}
}

// status is a simple worker thread that prints rate of the meter every 5 seconds
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
