package gopool

import (
	"time"
)

type workerArray interface {
	len() int
	isEmpty() bool
	insert(worker *worker) error
	detach() *worker
	retrieveExpiry(duration time.Duration) []*worker
	reset()
}

type arrayType int

const (
	stackType arrayType = 1 << iota
)

func newWorkerArray(aType arrayType, size int) workerArray {
	switch aType {
	case stackType:
		return newWorkerStack(size)
	default:
		return newWorkerStack(size)
	}
}
