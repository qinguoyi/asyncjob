package dispatch

import (
	"context"
	"fmt"
	_ "github.com/qinguoyi/asyncjob/app/pkg/event/handlers" // 为了执行handlers包里的init 自动注册
	"github.com/qinguoyi/asyncjob/app/pkg/event/tools"
	"github.com/qinguoyi/asyncjob/app/pkg/utils"
	"time"
)

var (
	taskCtx, taskCancel = context.WithCancel(context.Background())
	BucketQueue         = make(chan string)
)

// Job . 数据先存db，再存redis
type Job struct {
	Type     string         `json:"type" msgpack:"1"`  // 任务类型
	Delay    *time.Duration `json:"delay" msgpack:"2"` // 延迟时间, unix时间戳
	Body     interface{}    `json:"body" msgpack:"3"`
	TTL      int64          `json:"ttl" msgpack:"4"` // 任务超时时间, 秒
	Retry    int            `json:"retryCount" msgpack:"5"`
	CronSpec string         `json:"cronSpec" msgpack:"6"`
}

// RunTask 启动任务
func RunTask(q *AsyncQueue) ([]TransWorker, []TransWorker, []TransWorker, []HandleWorker) {
	// round-robin bucket
	go tools.GenerateBucketName(BucketQueue)

	p := NewTransWorker(q)
	// 启动生产者，每个生产者轮询一个zset
	p2r := p.Pending2ReadyWorker()
	r2r := p.Running2ReadyWorker()
	r2p := p.Running2PendingWorker()

	// 启动消费者
	c := NewHandleWorker(q)
	handler := c.HandlerWorker()
	return p2r, r2r, r2p, handler
}

func StopTask(
	p2r []TransWorker,
	r2r []TransWorker,
	r2p []TransWorker,
	consumers []HandleWorker,
) {
	time.Sleep(5 * time.Second)
	taskCancel()
	for i := 0; i < utils.BucketNum; i++ {
		p2r[i].Stop()
		r2r[i].Stop()
		r2p[i].Stop()
	}
	fmt.Println("生产者终止...")
	for i := 0; i < utils.BucketNum; i++ {
		consumers[i].Stop()
	}
	fmt.Println("消费者终止...")
}
