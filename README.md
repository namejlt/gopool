# gopool

goroutine pool

## 协程池说明

```

支持功能：
初始化池子
添加任务不阻塞
并发处理任务
并发协程空闲数保持
lru淘汰
优雅退出

对象：
pool    协程池
worker  协程
task    任务

核心：
池子中包含就绪worker，就绪worker定期进行清理
worker中包含task channel，添加task和处理task
池子中预留chan阻塞时，全局task链表，控制协程异步添加到worker中

```

### 参考

1. [https://github.com/panjf2000/ants](https://github.com/panjf2000/ants)
2. [https://github.com/valyala/fasthttp](https://github.com/valyala/fasthttp)
3. [https://github.com/bytedance/gopkg/tree/develop/util/gopool](https://github.com/bytedance/gopkg/tree/develop/util/gopool)


