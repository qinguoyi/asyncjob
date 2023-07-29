package dispatch

import (
	"context"
	"fmt"
	"github.com/qinguoyi/asyncjob/app/pkg/utils"
	"github.com/qinguoyi/asyncjob/bootstrap"
	"sync"
	"time"
)

type TransWorker struct {
	dq *AsyncQueue
	Wg *sync.WaitGroup
}

func NewTransWorker(q *AsyncQueue) *TransWorker {
	return &TransWorker{
		dq: q,
		Wg: &sync.WaitGroup{},
	}
}

func (p *TransWorker) Pending2ReadyWorker() []TransWorker {
	var p2r []TransWorker
	for i := 0; i < utils.BucketNum; i++ {
		p := NewTransWorker(p.dq)
		p2r = append(p2r, *p)
		p.Wg.Add(1)
		go p.Pending2ReadyTimer(taskCtx, fmt.Sprintf(utils.BucketName, i))
	}
	return p2r
}

// Pending2ReadyTimer 生产者,多协程完成对应的流程
func (p *TransWorker) Pending2ReadyTimer(ctx context.Context, bucketName string) {
	timer := time.NewTicker(1 * time.Nanosecond)
	defer p.Wg.Done()
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			err := p.dq.pending2Ready(bucketName)
			if err != nil {
				bootstrap.NewLogger().Logger.Error(fmt.Sprintf("Pending2Ready Timer error: %v\n", err))
			}
		case <-ctx.Done():
			return
		}
		timer.Reset(500 * time.Millisecond)
	}
}

func (p *TransWorker) Running2ReadyWorker() []TransWorker {
	var r2r []TransWorker
	for i := 0; i < utils.BucketNum; i++ {
		p := NewTransWorker(p.dq)
		r2r = append(r2r, *p)
		p.Wg.Add(1)
		go p.Running2ReadyTimer(taskCtx, fmt.Sprintf(utils.BucketName, i))
	}
	return r2r
}

// Running2ReadyTimer 生产者,多协程完成对应的流程
func (p *TransWorker) Running2ReadyTimer(ctx context.Context, bucketName string) {
	timer := time.NewTicker(1 * time.Nanosecond)
	defer p.Wg.Done()
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			err := p.dq.running2Ready(bucketName)
			if err != nil {
				bootstrap.NewLogger().Logger.Error(fmt.Sprintf("Running2Ready Timer error: %v\n", err))
			}
		case <-ctx.Done():
			return
		}
		timer.Reset(500 * time.Millisecond)
	}
}

func (p *TransWorker) Running2PendingWorker() []TransWorker {
	var r2p []TransWorker
	for i := 0; i < utils.BucketNum; i++ {
		p := NewTransWorker(p.dq)
		r2p = append(r2p, *p)
		p.Wg.Add(1)
		go p.Running2PendingTimer(taskCtx, fmt.Sprintf(utils.BucketName, i))
	}
	return r2p
}

func (p *TransWorker) Running2PendingTimer(ctx context.Context, bucketName string) {
	timer := time.NewTicker(1 * time.Nanosecond)
	defer p.Wg.Done()
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			err := p.dq.running2Pending(bucketName)
			if err != nil {
				bootstrap.NewLogger().Logger.Error(fmt.Sprintf("Running2Pending timer error: %v\n", err))
			}
		case <-ctx.Done():
			return
		}
		timer.Reset(500 * time.Millisecond)
	}
}

func (p *TransWorker) Stop() {
	p.Wg.Wait()
}
