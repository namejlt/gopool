// MIT License

// Copyright (c) 2018 Andy Pan

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package gopool

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024
	MiB // 1048576
	// GiB // 1073741824
	// TiB // 1099511627776             (超过了int32的范围)
	// PiB // 1125899906842624
	// EiB // 1152921504606846976
	// ZiB // 1180591620717411303424    (超过了int64的范围)
	// YiB // 1208925819614629174706176
)

const (
	Param    = 100
	Size     = 1000
	TestSize = 10000
	n        = 100000
)

var curMem uint64

// TestPoolWaitToGetWorkerSimple is used to test waiting to get worker.
func TestPoolWaitToGetWorkerSimple(t *testing.T) {
	var wg sync.WaitGroup
	p, _ := NewPool(Size)
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		fmt.Println("i", i)
		param := i
		_ = p.Submit(nil, func() {
			demoPoolFunc(param)
			wg.Done()
		})
	}
	fmt.Println("wait")
	wg.Wait()
	fmt.Println("wait ok~!!!!!")
	t.Logf("pool, running workers number:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestPoolWaitToGetWorkerPreMalloc(t *testing.T) {
	var wg sync.WaitGroup
	p, _ := NewPool(Size, WithPreAlloc(true))
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Submit(nil, func() {
			demoPoolFunc(Param)
			wg.Done()
		})
	}
	wg.Wait()
	t.Logf("pool, running workers number:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

// TestPoolGetWorkerFromCache is used to test getting worker from sync.Pool.
func TestPoolGetWorkerFromCache(t *testing.T) {
	p, _ := NewPool(TestSize)
	defer p.Release()

	for i := 0; i < Size; i++ {
		_ = p.Submit(nil, demoFunc)
	}
	time.Sleep(2 * DefaultCleanIntervalTime)
	_ = p.Submit(nil, demoFunc)
	t.Logf("pool, running workers number:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

//-------------------------------------------------------------------------------------------
// Contrast between goroutines without a pool and goroutines with ants pool.
//-------------------------------------------------------------------------------------------

func TestNoPool(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			demoFunc()
			wg.Done()
		}()
	}

	wg.Wait()
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestPool(t *testing.T) {
	defer Release()
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = Submit(nil, func() {
			demoFunc()
			wg.Done()
		})
	}
	wg.Wait()

	t.Logf("pool, capacity:%d", Cap())
	t.Logf("pool, running workers number:%d", Running())
	t.Logf("pool, free workers number:%d", Free())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

//-------------------------------------------------------------------------------------------
//-------------------------------------------------------------------------------------------

func TestPanicHandler(t *testing.T) {
	var panicCounter int64
	var wg sync.WaitGroup
	p0, err := NewPool(10, WithPanicHandler(func(p interface{}) {
		defer wg.Done()
		atomic.AddInt64(&panicCounter, 1)
		t.Logf("catch panic with PanicHandler: %v", p)
	}))
	assert.NoErrorf(t, err, "create new pool failed: %v", err)
	defer p0.Release()
	wg.Add(1)
	_ = p0.Submit(nil, func() {
		panic("Oops!")
	})
	wg.Wait()
	c := atomic.LoadInt64(&panicCounter)
	assert.EqualValuesf(t, 1, c, "panic handler didn't work, panicCounter: %d", c)
	assert.EqualValues(t, 0, p0.Running(), "pool should be empty after panic")
}

func TestPanicHandlerPreMalloc(t *testing.T) {
	var panicCounter int64
	var wg sync.WaitGroup
	p0, err := NewPool(10, WithPreAlloc(true), WithPanicHandler(func(p interface{}) {
		defer wg.Done()
		atomic.AddInt64(&panicCounter, 1)
		t.Logf("catch panic with PanicHandler: %v", p)
	}))
	assert.NoErrorf(t, err, "create new pool failed: %v", err)
	defer p0.Release()
	wg.Add(1)
	_ = p0.Submit(nil, func() {
		panic("Oops!")
	})
	wg.Wait()
	c := atomic.LoadInt64(&panicCounter)
	assert.EqualValuesf(t, 1, c, "panic handler didn't work, panicCounter: %d", c)
	assert.EqualValues(t, 0, p0.Running(), "pool should be empty after panic")
}

func TestPoolPanicWithoutHandler(t *testing.T) {
	p0, err := NewPool(10)
	assert.NoErrorf(t, err, "create new pool failed: %v", err)
	defer p0.Release()
	_ = p0.Submit(nil, func() {
		panic("Oops!")
	})
}

func TestPoolPanicWithoutHandlerPreMalloc(t *testing.T) {
	p0, err := NewPool(10, WithPreAlloc(true))
	assert.NoErrorf(t, err, "create new pool failed: %v", err)
	defer p0.Release()
	_ = p0.Submit(nil, func() {
		panic("Oops!")
	})

}

func TestPurge(t *testing.T) {
	p, err := NewPool(10)
	assert.NoErrorf(t, err, "create TimingPool failed: %v", err)
	defer p.Release()
	_ = p.Submit(nil, demoFunc)
	time.Sleep(3 * DefaultCleanIntervalTime)
	assert.EqualValues(t, 0, p.Running(), "all p should be purged")
}

func TestPurgePreMalloc(t *testing.T) {
	p, err := NewPool(10, WithPreAlloc(true))
	assert.NoErrorf(t, err, "create TimingPool failed: %v", err)
	defer p.Release()
	_ = p.Submit(nil, demoFunc)
	time.Sleep(3 * DefaultCleanIntervalTime)
	assert.EqualValues(t, 0, p.Running(), "all p should be purged")
}

func TestRebootDefaultPool(t *testing.T) {
	defer Release()
	Reboot()
	var wg sync.WaitGroup
	wg.Add(1)
	_ = Submit(nil, func() {
		demoFunc()
		wg.Done()
	})
	wg.Wait()
	Release()
	assert.EqualError(t, Submit(nil, nil), ErrPoolClosed.Error(), "pool should be closed")
	Reboot()
	wg.Add(1)
	assert.NoError(t, Submit(nil, func() { wg.Done() }), "pool should be rebooted")
	wg.Wait()
}

func TestRebootNewPool(t *testing.T) {
	var wg sync.WaitGroup
	p, err := NewPool(10)
	assert.NoErrorf(t, err, "create Pool failed: %v", err)
	defer p.Release()
	wg.Add(1)
	_ = p.Submit(nil, func() {
		demoFunc()
		wg.Done()
	})
	wg.Wait()
	p.Release()
	assert.EqualError(t, p.Submit(nil, nil), ErrPoolClosed.Error(), "pool should be closed")
	p.Reboot()
	wg.Add(1)
	assert.NoError(t, p.Submit(nil, func() { wg.Done() }), "pool should be rebooted")
	wg.Wait()

}

func TestInfinitePool(t *testing.T) {
	c := make(chan struct{})
	p, _ := NewPool(-1)
	_ = p.Submit(nil, func() {
		_ = p.Submit(nil, func() {
			<-c
		})
	})
	c <- struct{}{}
	if n := p.Running(); n != 2 {
		t.Errorf("expect 2 workers running, but got %d", n)
	}
	if n := p.Free(); n != -1 {
		t.Errorf("expect -1 of free workers by unlimited pool, but got %d", n)
	}
	p.Tune(10)
	if capacity := p.Cap(); capacity != -1 {
		t.Fatalf("expect capacity: -1 but got %d", capacity)
	}
}
