package gopool

import (
	"context"
	"github.com/namejlt/gopool/internal"
	"sync"
	"sync/atomic"
)

var taskPool sync.Pool

func init() {
	taskPool.New = newTask
}

type task struct {
	ctx context.Context
	f   func()

	next *task
}

func (t *task) zero() {
	t.ctx = nil
	t.f = nil
	t.next = nil
}

func (t *task) Recycle() {
	t.zero()
	taskPool.Put(t)
}

func newTask() interface{} {
	return &task{}
}

/**

全局task list

在worker运行满的时候，task临时添加进全局队列
等待worker任意一个就绪，每次worker就绪后，先检查全局队列，然后放入就绪队列

todo 优化全局队列 性能 减少并发处理锁开销 可以一次获取多个放入worker 调整worker结构

*/
// taskList ==========================================================================
type taskList struct {
	lock      sync.Locker
	taskHead  *task
	taskTail  *task
	taskCount int32
}

func newTaskList() *taskList {
	return &taskList{
		lock: internal.NewSpinLock(),
	}
}

func (p *taskList) len() int32 {
	return atomic.LoadInt32(&p.taskCount)
}

func (p *taskList) isEmpty() bool {
	return p.len() == 0
}

func (p *taskList) insert(t *task) { //添加任务到队尾
	p.lock.Lock()
	if p.taskHead == nil {
		p.taskHead = t //只有一个task 是头部
		p.taskTail = t //只有一个task 也是尾部
	} else {
		p.taskTail.next = t //尾部下一个是新的task
		p.taskTail = t      //重置尾部位置
	}
	p.lock.Unlock()
	atomic.AddInt32(&p.taskCount, 1)
}

func (p *taskList) detach() *task { //从队头取出任务
	var t *task
	p.lock.Lock()
	if p.taskHead != nil {
		t = p.taskHead
		p.taskHead = p.taskHead.next
		atomic.AddInt32(&p.taskCount, -1)
	}
	p.lock.Unlock()

	return t
}
