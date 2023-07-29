package dispatch

import (
	"context"
	"fmt"
	"github.com/qinguoyi/asyncjob/app/pkg/event"
	"github.com/qinguoyi/asyncjob/app/pkg/utils"
	"github.com/qinguoyi/asyncjob/bootstrap"
	"strconv"
	"sync"
	"time"
)

type HandleWorker struct {
	dq *AsyncQueue
	Wg *sync.WaitGroup
	ch chan struct{}
}

func NewHandleWorker(q *AsyncQueue) *HandleWorker {
	return &HandleWorker{
		dq: q,
		Wg: &sync.WaitGroup{},
		ch: make(chan struct{}, utils.MaxWorker),
	}
}

func (w *HandleWorker) HandlerWorker() []HandleWorker {
	var consumers []HandleWorker
	for i := 0; i < utils.BucketNum; i++ {
		worker := NewHandleWorker(w.dq)
		consumers = append(consumers, *worker)
		worker.Wg.Add(1)
		go worker.Start(taskCtx, fmt.Sprintf(utils.BucketName, i))
	}
	return consumers
}

func (w *HandleWorker) getTTL(ctx context.Context, bucketName, idStr string) (int64, error) {
	ttlStr, err := w.dq.redisClient.HGet(ctx, w.dq.getTTLKey(bucketName), idStr).Result()
	if err != nil {
		return 0, fmt.Errorf("获取%s的超时时间失败，详情：%s\n", idStr, err.Error())
	}
	ttl, err := strconv.ParseInt(ttlStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("超时时间%s解析失败，详情：%s\n", ttlStr, err.Error())
	}
	return ttl, nil
}

func (w *HandleWorker) terminateMonitor(b, id string, tm, s chan struct{}) {
	ctx := context.Background()
	for {
		select {
		case <-s:
			return
		case <-time.After(500 * time.Millisecond):
			_, err := w.dq.redisClient.HGet(ctx, w.dq.getTerminateKey(b), id).Result()
			if err == nil {
				tm <- struct{}{}
				return
			}
		}
	}
}

func (w *HandleWorker) exec(ctx context.Context, ttl int64, bucketName, idStr string, handler func(i interface{}, c context.Context, t <-chan time.Time, s chan struct{}) error, job *Job) {
	defer func() {
		<-w.ch
	}()

	resChan := make(chan error, 1)
	terminateChan := make(chan struct{}, 1)
	finishChan := make(chan struct{}, 1)
	timeOutChan := time.After(time.Duration(ttl) * time.Second)

	// 任务执行协程
	go func(c context.Context, r chan error, to <-chan time.Time, tm chan struct{}) {
		defer func() {
			if re := recover(); re != nil {
				r <- fmt.Errorf("任务执行异常:%s", re)
			}
		}()
		r <- handler(job.Body, c, to, tm)
	}(ctx, resChan, timeOutChan, terminateChan)

	// 任务终止监控协程
	go w.terminateMonitor(bucketName, idStr, terminateChan, finishChan)

	// 等待信号
	select {
	// 外层有内层还要有，是为了能及时退出
	case <-ctx.Done():
		return
	case <-timeOutChan:
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("任务[%s]执行超时", idStr))
	case <-terminateChan:
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("任务[%s]手动终止", idStr))
		close(terminateChan)
		_ = w.dq.running2Finish(bucketName, idStr, utils.TaskStatusTerminate, "手动终止")
	case err := <-resChan:
		if err == nil {
			_ = w.dq.running2Finish(bucketName, idStr, utils.TaskStatusFinish, "")
		} else {
			_ = w.dq.running2Finish(bucketName, idStr, utils.TaskStatusError, err.Error())
		}
		finishChan <- struct{}{}
		close(finishChan)
		close(resChan)
	}
}

// Start consumer
func (w *HandleWorker) Start(ctx context.Context, bucketName string) {

	defer w.Wg.Done()
	for {
		select {
		default:
			w.ch <- struct{}{}
			<-w.ch
			idStr, err := w.dq.ready2Running(bucketName)
			if err == nil {
				w.ch <- struct{}{}
				ttl, err := w.getTTL(ctx, bucketName, idStr)
				if err != nil {
					_ = w.dq.running2Finish(bucketName, idStr, utils.TaskStatusError, err.Error())
					continue
				}
				job, err := w.dq.GetJob(idStr)
				if err != nil {
					_ = w.dq.running2Finish(bucketName, idStr, utils.TaskStatusError, fmt.Sprintf("获取任务%s详情失败，详情: %s\n", idStr, err.Error()))
					continue
				}

				handler := event.NewEventsHandler().GetHandler(job.Type)
				if handler == nil {
					_ = w.dq.running2Finish(bucketName, idStr, utils.TaskStatusError, fmt.Sprintf("任务[%s]没找到执行handler\n", idStr))
					continue
				} else {
					go w.exec(ctx, ttl, bucketName, idStr, handler, job)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *HandleWorker) Stop() {
	w.Wg.Wait()
}
