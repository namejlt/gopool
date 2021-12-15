package gopool

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	"runtime"
	"time"
)

const (
	// DefaultPoolSize is the default capacity for a default goroutine pool.
	DefaultPoolSize = math.MaxInt32

	// DefaultCleanIntervalTime is the interval time to clean up goroutines.
	DefaultCleanIntervalTime = time.Second
)

const (
	// OPENED represents that the pool is opened.
	OPENED = iota

	// CLOSED represents that the pool is closed.
	CLOSED
)

var (
	//
	//--------------------------Error types for the  API------------------------------

	// ErrInvalidPoolExpiry will be returned when setting a negative number as the periodic duration to purge goroutines.
	ErrInvalidPoolExpiry = errors.New("invalid expiry for pool")

	// ErrPoolClosed will be returned when submitting task to a closed pool.
	ErrPoolClosed = errors.New("this pool has been closed")

	//---------------------------------------------------------------------------

	// workerChanCap determines whether the channel of a worker should be a buffered channel
	// to get the best performance. Inspired by fasthttp at
	// https://github.com/valyala/fasthttp/blob/master/workerpool.go#L139
	workerChanCap = func() int {
		// Use blocking channel if GOMAXPROCS=1.
		// This switches context from sender to receiver immediately,
		// which results in higher performance (under go1.5 at least).
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}

		// Use non-blocking workerChan if GOMAXPROCS>1,
		// since otherwise the sender might be dragged down if the receiver is CPU-bound.
		return 1
	}()

	defaultLogger = Logger(log.New(os.Stderr, "", log.LstdFlags))

	// Init an instance pool when importing ants.
	defaultPool, _ = NewPool(DefaultPoolSize)
)

// Submit submits a task to pool.
func Submit(ctx context.Context, f func()) error {
	return defaultPool.Submit(ctx, f)
}

// Running returns the number of the currently running goroutines.
func Running() int {
	return defaultPool.Running()
}

// Cap returns the capacity of this default pool.
func Cap() int {
	return defaultPool.Cap()
}

// Free returns the available goroutines to work.
func Free() int {
	return defaultPool.Free()
}

// Release Closes the default pool.
func Release() {
	defaultPool.Release()
}

// Reboot reboots the default pool.
func Reboot() {
	defaultPool.Reboot()
}
