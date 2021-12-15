package gopool

import (
	"context"
	"github.com/namejlt/gopool/internal"
	"sync"
	"sync/atomic"
	"time"
)

type Pool struct {
	cap         int32
	running     int32
	lock        sync.Locker
	workers     workerArray
	state       int32
	workerCache sync.Pool
	taskList    *taskList
	options     *Options
}

func NewPool(size int, options ...Option) (*Pool, error) {
	opts := loadOptions(options...)

	if size <= 0 {
		size = -1
	}

	if expiry := opts.ExpiryDuration; expiry < 0 {
		return nil, ErrInvalidPoolExpiry
	} else if expiry == 0 {
		opts.ExpiryDuration = DefaultCleanIntervalTime
	}

	if opts.Logger == nil {
		opts.Logger = defaultLogger
	}

	p := &Pool{
		cap:     int32(size),
		lock:    internal.NewSpinLock(),
		options: opts,
	}
	p.workerCache.New = func() interface{} {
		return &worker{
			pool: p,
			task: make(chan *task, workerChanCap),
		}
	}
	p.workers = newWorkerArray(stackType, 0)
	p.taskList = newTaskList()

	// Start a goroutine to clean up expired workers periodically.
	go p.purgePeriodically()

	return p, nil
}

func (p *Pool) purgePeriodically() {
	heartbeat := time.NewTicker(p.options.ExpiryDuration)
	defer heartbeat.Stop()

	for range heartbeat.C {
		if p.IsClosed() {
			break
		}

		p.lock.Lock()
		expiredWorkers := p.workers.retrieveExpiry(p.options.ExpiryDuration)
		p.lock.Unlock()

		// Notify obsolete workers to stop.
		// This notification must be outside the p.lock, since w.task
		// may be blocking and may consume a lot of time if many workers
		// are located on non-local CPUs.
		for i := range expiredWorkers {
			expiredWorkers[i].task <- nil
			expiredWorkers[i] = nil
		}
	}
}

func (p *Pool) Submit(ctx context.Context, f func()) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}
	var w *worker
	t := taskPool.Get().(*task)
	t.ctx = ctx
	t.f = f
	if w = p.retrieveWorker(); w == nil {
		//worker 已满 task存放全局队列中 异步加载到worker中
		p.taskList.insert(t)
	} else {
		w.task <- t
	}

	return nil
}

// Running returns the amount of the currently running goroutines.
func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

// Free returns the amount of available goroutines to work, -1 indicates this pool is unlimited.
func (p *Pool) Free() int {
	c := p.Cap()
	if c < 0 {
		return -1
	}
	return c - p.Running()
}

// Cap returns the capacity of this pool.
func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.cap))
}

// Tune changes the capacity of this pool, note that it is noneffective to the infinite or pre-allocation pool.
func (p *Pool) Tune(size int) {
	if capacity := p.Cap(); capacity == -1 || size <= 0 || size == capacity || p.options.PreAlloc {
		return
	}
	atomic.StoreInt32(&p.cap, int32(size))
}

// IsClosed indicates whether the pool is closed.
func (p *Pool) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == CLOSED
}

// Release closes this pool and releases the worker queue.
func (p *Pool) Release() {
	atomic.StoreInt32(&p.state, CLOSED)
	p.lock.Lock()
	p.workers.reset()
	p.lock.Unlock()
}

// Reboot reboots a closed pool.
func (p *Pool) Reboot() {
	if atomic.CompareAndSwapInt32(&p.state, CLOSED, OPENED) {
		go p.purgePeriodically()
	}
}

// ---------------------------------------------------------------------------

// incRunning increases the number of the currently running goroutines.
func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

// decRunning decreases the number of the currently running goroutines.
func (p *Pool) decRunning() {
	atomic.AddInt32(&p.running, -1)
}

// retrieveWorker returns an available worker to run the tasks.
func (p *Pool) retrieveWorker() (w *worker) {
	spawnWorker := func() {
		w = p.workerCache.Get().(*worker)
		w.run()
	}

	p.lock.Lock()

	w = p.workers.detach()
	if w != nil { // first try to fetch the worker from the queue
		p.lock.Unlock()
	} else if capacity := p.Cap(); capacity == -1 || capacity > p.Running() {
		// if the worker queue is empty and we don't run out of the pool capacity,
		// then just spawn a new worker goroutine.
		p.lock.Unlock()
		spawnWorker()
	} else {
		p.lock.Unlock()
	}
	return
}

// revertWorker puts a worker back into free pool, recycling the goroutines.
func (p *Pool) revertWorker(worker *worker) bool {
	if capacity := p.Cap(); (capacity > 0 && p.Running() > capacity) || p.IsClosed() {
		return false
	}
	worker.recycleTime = time.Now()
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.IsClosed() {
		return false
	}

	err := p.workers.insert(worker)
	if err != nil {
		return false
	}
	return true
}
