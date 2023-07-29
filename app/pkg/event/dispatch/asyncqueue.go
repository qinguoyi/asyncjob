package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/qinguoyi/asyncjob/app/pkg/event/tools"
	"github.com/qinguoyi/asyncjob/app/pkg/utils"
	"github.com/qinguoyi/asyncjob/bootstrap/plugins"
	"strconv"
	"sync"
	"time"
)

type AsyncQueue struct {
	redisClient *redis.Client
}

var (
	asyncQueue *AsyncQueue
	once       sync.Once
)

func NewAsyncQueue() *AsyncQueue {
	if asyncQueue != nil {
		return asyncQueue
	} else {
		once.Do(func() {
			lgRedis := new(plugins.LangGoRedis).NewRedis()
			asyncQueue = &AsyncQueue{
				redisClient: lgRedis,
			}
		})
		return asyncQueue
	}
}

// pending zset
func (q *AsyncQueue) getPendingKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncPendingKey, bucket)
}

// ready list
func (q *AsyncQueue) getReadyKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncReadyKey, bucket)
}

// running zset
func (q *AsyncQueue) getRunningKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncRunningKey, bucket)
}

// getRetryCountKey hash id -> retryCount
func (q *AsyncQueue) getRetryCountKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncRetryKey, bucket)
}

// getTTLKey hash  id -> ttl
func (q *AsyncQueue) getTTLKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncTTLKey, bucket)
}

// getPeriodicKey zset id -> next_time
func (q *AsyncQueue) getPeriodicKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncPeriodKey, bucket)
}

// getBucketKey .
func (q *AsyncQueue) getBucketKey(id string) string {
	return fmt.Sprintf(utils.AsyncBucketKey, id)
}

// getResultKey .
func (q *AsyncQueue) getResultKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncResultKey, bucket)
}

// getTerminateKey .
func (q *AsyncQueue) getTerminateKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncTerminateKey, bucket)
}

// getInfoKey .
func (q *AsyncQueue) getInfoKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncInfoKey, bucket)
}

// getPeriodRetryKey
func (q *AsyncQueue) getPeriodRetryKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncPeriodRetryKey, bucket)
}

// getStatusKey
func (q *AsyncQueue) getStatusKey(bucket string) string {
	return fmt.Sprintf(utils.AsyncStatusKey, bucket)
}

// pending2Ready 就绪的任务转移去排队
func (q *AsyncQueue) pending2Ready(bucketName string) error {
	now := time.Now().Unix()
	ctx := context.Background()
	keys := []string{
		q.getPendingKey(bucketName),
		q.getReadyKey(bucketName),
		q.getRunningKey(bucketName),
		q.getStatusKey(bucketName),
		q.getResultKey(bucketName),
	}
	err := q.redisClient.Eval(ctx, tools.Pending2ReadyScript, keys, now, utils.TaskStatusReady).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("pending2ReadyScript failed: %v", err)
	}
	return nil
}

// ready2Running 排队任务转移去执行
func (q *AsyncQueue) ready2Running(bucketName string) (string, error) {
	ctx := context.Background()
	keys := []string{
		q.getReadyKey(bucketName),
		q.getRunningKey(bucketName),
		q.getTTLKey(bucketName),
		q.getStatusKey(bucketName),
		q.getResultKey(bucketName),
	}

	ret, err := q.redisClient.Eval(ctx, tools.Ready2RunningScript, keys, utils.TaskStatusRunning).Result()
	if err == redis.Nil {
		return "", err
	}
	if err != nil {
		return "", fmt.Errorf("ready2RunningScript failed: %v", err)
	}
	str, ok := ret.(string)
	if !ok {
		return "", fmt.Errorf("illegal result: %#v", ret)
	}
	return str, nil
}

// running2Ready 执行任务超时转移回排队
func (q *AsyncQueue) running2Ready(bucketName string) error {
	ctx := context.Background()
	keys := []string{
		q.getRunningKey(bucketName),
		q.getRetryCountKey(bucketName),
		q.getReadyKey(bucketName),
		q.getTTLKey(bucketName),
		q.getResultKey(bucketName),
		q.getPeriodicKey(bucketName),
		q.getStatusKey(bucketName),
		q.getPeriodRetryKey(bucketName),
	}
	now := time.Now()
	err := q.redisClient.Eval(ctx, tools.Running2ReadyScript,
		keys, now.Unix(), utils.TaskStatusTimeOut, utils.TaskStatusError).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("running to retry script failed: %v", err)
	}
	return nil
}

// running2Pending 周期任务重新入列
func (q *AsyncQueue) running2Pending(bucketName string) error {
	ctx := context.Background()
	now := time.Now().Unix()
	// 从zset中获取到期的任务，然后执行Running2PendingScript，脚本中做原子化
	msgs, err := q.redisClient.ZRangeByScore(ctx, q.getPeriodicKey(bucketName), &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(now, 10),
		Offset: 0,
		Count:  1,
	}).Result()
	if err == redis.Nil {
		return fmt.Errorf("周期任务不存在")
	}
	if err != nil {
		return fmt.Errorf("获取周期任务信息失败")
	}
	if len(msgs) == 0 {
		return nil
	}
	id := msgs[0]

	// 判断任务是否终止
	_, err = q.redisClient.HGet(ctx, q.getTerminateKey(bucketName), id).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("获取任务信息失败")
	}
	if err == nil {
		return nil
	}

	idMsg, err := q.redisClient.HGet(ctx, q.getInfoKey(bucketName), id).Result()
	if err == redis.Nil {
		return fmt.Errorf("任务不存在")
	}
	if err != nil {
		return fmt.Errorf("获取任务信息失败")
	}

	var job Job
	if err := json.Unmarshal([]byte(idMsg), &job); err != nil {
		return fmt.Errorf("转换失败")
	}
	t, err := tools.GetNextTime(job.CronSpec, time.Now())
	if err != nil {
		return fmt.Errorf("获取下次执行时间失败")
	}
	keys := []string{
		q.getPeriodicKey(bucketName),
		q.getPendingKey(bucketName),
	}
	err = q.redisClient.Eval(ctx, tools.Running2PendingScript, keys, now, id,
		float64(t.Unix())).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("running to pending script failed: %v", err)
	}

	return nil
}

func (q *AsyncQueue) running2Finish(bucketName, id string, status int, msg string) error {
	ctx := context.Background()
	keys := []string{
		q.getStatusKey(bucketName),
		q.getResultKey(bucketName),
		q.getRunningKey(bucketName),
		q.getRetryCountKey(bucketName),
		q.getTTLKey(bucketName),
		q.getPeriodicKey(bucketName),
	}
	err := q.redisClient.Eval(ctx, tools.Running2FinishScript, keys, id, status, msg).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("set status and result script failed: %v", err)
	}
	return nil
}

func (q *AsyncQueue) GetJob(id string) (*Job, error) {
	ctx := context.Background()
	bucketName, err := q.redisClient.HGet(ctx, q.getBucketKey(id), id).Result()
	if err == redis.Nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("get bucketName failed: %v", err)
	}
	jobInfo, err := q.redisClient.HGet(ctx, q.getInfoKey(bucketName), id).Result()
	if err == redis.Nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("get job failed: %v", err)
	}
	var job Job
	if err := json.Unmarshal([]byte(jobInfo), &job); err != nil {
		return nil, fmt.Errorf("unmashal job failed: %v", err)
	}
	return &job, nil
}

// SendScheduleMsg .
func (q *AsyncQueue) SendScheduleMsg(job Job, idStr string) error {
	duration := job.Delay
	t := time.Now()
	if duration != nil {
		t = t.Add(*duration)
	}
	next := int64(0)
	if job.CronSpec != "" {
		next = 1
	}
	bucketName := <-BucketQueue
	// parse options
	retryCount := job.Retry
	ttl := job.TTL

	ctx := context.Background()
	b, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal msg failed: %v", err)
	}

	keys := []string{
		q.getInfoKey(bucketName),
		q.getPeriodicKey(bucketName),
		q.getTTLKey(bucketName),
		q.getRetryCountKey(bucketName),
		q.getBucketKey(idStr),
		q.getPendingKey(bucketName),
		q.getPeriodRetryKey(bucketName),
		q.getStatusKey(bucketName),
		q.getTerminateKey(bucketName),
		q.getResultKey(bucketName),
	}
	err = q.redisClient.Eval(ctx, tools.Send2PendingScript, keys, b, next, ttl, retryCount,
		bucketName, float64(t.Unix()), idStr, utils.TaskStatusPending).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("send to pending script failed: %v", err)
	}

	return nil
}

// StopJob 终止任务
func (q *AsyncQueue) StopJob(id string) error {
	ctx := context.Background()

	bucketName, err := q.redisClient.HGet(ctx, q.getBucketKey(id), id).Result()
	if err != nil {
		return fmt.Errorf("任务不存在: %v", err)
	}

	status, err := q.redisClient.HGet(ctx, q.getStatusKey(bucketName), id).Result()
	if err != nil {
		return fmt.Errorf("任务不存在: %v", err)
	}
	statusId, err := strconv.ParseInt(status, 10, 64)
	if err != nil {
		return fmt.Errorf("uid参数有误，详情:%s", err)
	}
	if !utils.Contains(int(statusId), []int{
		utils.TaskStatusPending,
		utils.TaskStatusReady,
		utils.TaskStatusRunning,
		utils.TaskStatusTimeOut,
	}) {
		return fmt.Errorf("当前任务已执行完成，无法终止")
	}
	keys := []string{
		q.getPendingKey(bucketName),
		q.getReadyKey(bucketName),
		q.getTerminateKey(bucketName),
	}
	err = q.redisClient.Eval(ctx, tools.TerminateScript, keys, id).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("stop job failed: %v", err)
	}
	return nil
}

// RecoverJob 恢复任务
func (q *AsyncQueue) RecoverJob(id string) error {
	ctx := context.Background()
	bucketName, err := q.redisClient.HGet(ctx, q.getBucketKey(id), id).Result()
	if err == redis.Nil {
		return err
	}
	if err != nil {
		return fmt.Errorf("任务不存在: %v", err)
	}

	// 判断是否是周期任务,	周期任务，等待下一次拉起
	_, err = q.redisClient.HGet(ctx, q.getPeriodicKey(bucketName), id).Result()
	if err == nil {
		// 取消terminate的标志位
		err = q.redisClient.HDel(ctx, q.getBucketKey(id), id).Err()
		if err != nil {
			return fmt.Errorf("任务不存在: %v", err)
		}
		return nil
	}

	// 非周期任务，重新投递
	idMsg, err := q.redisClient.HGet(ctx, q.getInfoKey(bucketName), id).Result()
	if err == redis.Nil {
		return fmt.Errorf("任务不存在")
	}
	if err != nil {
		return fmt.Errorf("获取任务信息失败")
	}
	var job Job
	if err := json.Unmarshal([]byte(idMsg), &job); err != nil {
		return fmt.Errorf("转换失败")
	}
	var delay time.Duration
	job.Delay = &delay
	if err := q.SendScheduleMsg(job, id); err != nil {
		return err
	}
	return nil
}

// DeleteJob 删除任务，终止+删除id信息
func (q *AsyncQueue) DeleteJob(id string) error {
	ctx := context.Background()

	bucketName, err := q.redisClient.HGet(ctx, q.getBucketKey(id), id).Result()
	if err == redis.Nil {
		return err
	}
	if err != nil {
		return fmt.Errorf("get bucketName failed: %v", err)
	}

	keys := []string{
		q.getPendingKey(bucketName),
		q.getRunningKey(bucketName),
		q.getTerminateKey(bucketName),
		q.getInfoKey(bucketName),
		q.getPeriodicKey(bucketName),
		q.getTTLKey(bucketName),
		q.getReadyKey(bucketName),
		q.getBucketKey(id),
		q.getPeriodRetryKey(bucketName),
		q.getRetryCountKey(bucketName),
	}
	err = q.redisClient.Eval(ctx, tools.DeleteScript, keys, id).Err()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("delete job failed: %v", err)
	}
	return nil
}
