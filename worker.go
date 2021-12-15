package gopool

import (
	"runtime"
	"time"
)

type worker struct {
	// pool who owns this worker.
	pool *Pool

	// task is a job should be done.
	task chan *task

	// recycleTime will be updated when putting a worker back into queue.
	recycleTime time.Time
}

func (w *worker) run() {
	w.pool.incRunning()
	go func() {
		defer func() {
			w.pool.decRunning()
			w.pool.workerCache.Put(w)
			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker exits from a panic: %v\n", p)
					var buf [4096]byte
					n := runtime.Stack(buf[:], false)
					w.pool.options.Logger.Printf("worker exits from panic: %s\n", string(buf[:n]))
				}
			}
		}()

		for f := range w.task {
			if f == nil { //从chan获取指定nil时  通知go退出
				return
			}
			f.f()
			if ok := w.pool.revertWorker(w); !ok { //进入就绪worker队列成功 继续for循环
				return
			} else {
				if !w.pool.taskList.isEmpty() { //辅助取出执行
					f = w.pool.taskList.detach()
					f.f()
				}
			}
		}
	}()
}
