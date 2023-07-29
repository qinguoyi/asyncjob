package v0

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/qinguoyi/asyncjob/app/models"
	"github.com/qinguoyi/asyncjob/app/pkg/base"
	"github.com/qinguoyi/asyncjob/app/pkg/event"
	"github.com/qinguoyi/asyncjob/app/pkg/event/dispatch"
	"github.com/qinguoyi/asyncjob/app/pkg/event/tools"
	"github.com/qinguoyi/asyncjob/app/pkg/utils"
	"github.com/qinguoyi/asyncjob/app/pkg/web"
	"github.com/qinguoyi/asyncjob/bootstrap/plugins"
	"strconv"
	"time"
)

// DelayJobDeliveryHandler 异步任务投递
//
//	@Summary		异步任务(非周期)投递
//	@Description	异步任务(非周期)投递
//	@Tags			任务投递
//	@Accept			application/json
//	@Produce		application/json
//	@Param			RequestBody	body		models.AsyncJob	true	"异步任务投递请求体"
//	@Success		200			{object}	web.Response
//	@Router			/api/async/v0/job/async [post]
func DelayJobDeliveryHandler(c *gin.Context) {
	var asyncJob models.AsyncJob
	if err := c.ShouldBindJSON(&asyncJob); err != nil {
		web.ParamsError(c, fmt.Sprintf("参数解析有误，详情:%s", err))
		return
	}

	if event.NewEventsHandler().GetHandler(asyncJob.Type) == nil {
		web.ParamsError(c, fmt.Sprintf("当前任务类型:%s，没有对应的handler", asyncJob.Type))
		return
	}

	var timeDelay time.Duration
	if asyncJob.StartTime != "" {
		localZone, err := time.LoadLocation("Local")
		if err != nil {
			web.ParamsError(c, fmt.Sprintf("解析本地时区失败，详情：%s", err))
			return
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", asyncJob.StartTime, localZone)
		if err != nil {
			web.ParamsError(c, fmt.Sprintf("异步任务启动时间字符串有误，详情：%s", err))
			return
		}

		now := time.Now()
		if t.Unix() > now.Unix() {
			timeDelay = t.Sub(now)
		}
	}

	// generate id
	uid, _ := base.NewUidGroup()
	id, err := uid["1"].NextId()
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("生成唯一id失败，详情:%s", err))
		web.InternalError(c, fmt.Sprintf("生成唯一id失败，详情:%s", err))
		return
	}
	idStr := strconv.FormatInt(id, 10)
	if err := dispatch.NewAsyncQueue().SendScheduleMsg(dispatch.Job{
		Type:  asyncJob.Type,
		Delay: &timeDelay,
		Body:  asyncJob.Body,
		TTL:   asyncJob.TTL,
		Retry: asyncJob.Retry,
	}, idStr); err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("延时任务投递异常:%s", err))
		web.InternalError(c, fmt.Sprintf("延时任务投递异常:%s", err))
		return
	}
	web.Success(c, idStr)
	return
}

// PeriodicJobDeliveryHandler 周期任务投递
//
//	@Summary		周期任务投递，支持cron语法
//	@Description	周期任务投递，支持cron语法
//	@Tags			任务投递
//	@Accept			application/json
//	@Produce		application/json
//	@Param			RequestBody	body		models.PeriodicJob	true	"周期任务投递请求体"
//	@Success		200			{object}	web.Response
//	@Router			/api/async/v0/job/periodic [post]
func PeriodicJobDeliveryHandler(c *gin.Context) {
	var periodicJob models.PeriodicJob
	if err := c.ShouldBindJSON(&periodicJob); err != nil {
		web.ParamsError(c, fmt.Sprintf("参数解析有误，详情:%s", err))
		return
	}

	if event.NewEventsHandler().GetHandler(periodicJob.Type) == nil {
		web.ParamsError(c, fmt.Sprintf("当前任务类型:%s，没有对应的handler", periodicJob.Type))
		return
	}

	var timeDelay time.Duration
	t, err := tools.GetNextTime(periodicJob.CronSpec, time.Now())
	if err != nil {
		web.ParamsError(c, fmt.Sprintf("cron语法有误，详情:%s", err))
		return
	}

	now := time.Now()
	if t.Unix() <= now.Unix() {
		web.ParamsError(c, fmt.Sprintf("cron语法有误，详情:%s", err))
		return
	} else {
		timeDelay = t.Sub(now)
	}
	// 时间间隔不能小于ttl
	t1, err := tools.GetNextTime(periodicJob.CronSpec, t.Add(5*time.Second))
	if err != nil {
		web.ParamsError(c, fmt.Sprintf("cron语法有误，详情:%s", err))
		return
	}
	if t1.Unix()-t.Unix() < periodicJob.TTL {
		web.ParamsError(c, fmt.Sprintf("异步任务的超时时间必须小于两次任务时间间隔"))
		return
	}
	// generate id
	uid, _ := base.NewUidGroup()
	id, err := uid["1"].NextId()
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("生成唯一id失败，详情:%s", err))
		web.InternalError(c, fmt.Sprintf("生成唯一id失败，详情:%s", err))
		return
	}
	idStr := strconv.FormatInt(id, 10)
	if err := dispatch.NewAsyncQueue().SendScheduleMsg(dispatch.Job{
		Type:     periodicJob.Type,
		Delay:    &timeDelay,
		Body:     periodicJob.Body,
		TTL:      periodicJob.TTL,
		Retry:    periodicJob.Retry,
		CronSpec: periodicJob.CronSpec,
	}, idStr); err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("周期任务投递异常:%s", err))
		web.InternalError(c, fmt.Sprintf("周期任务投递异常:%s", err))
		return
	}

	web.Success(c, idStr)
	return
}

// GetJobInfoHandler 查询任务信息
//
//	@Summary		查询任务信息
//	@Description	查询任务信息
//	@Tags			任务查询
//	@Accept			application/json
//	@Produce		application/json
//	@Param			job	path		string	true	"任务id"
//	@Success		200	{object}	web.Response{data=models.JobInfo}
//	@Router			/api/async/v0/job/{job} [get]
func GetJobInfoHandler(c *gin.Context) {
	job := c.Param("job")
	if job == "" {
		web.ParamsError(c, "job参数有误")
		return
	}

	// bucketName
	ctx := context.Background()
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	bucketName, err := lgRedis.HGet(ctx, fmt.Sprintf(utils.AsyncBucketKey, job), job).Result()
	if err == redis.Nil {
		web.ParamsError(c, "任务不存在")
		return
	}
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("查询任务[%s]的bucket报错", job))
		web.InternalError(c, "查询任务的bucket报错")
		return
	}
	jobInfo := models.JobInfo{UUID: job}

	// status
	status, err := lgRedis.HGet(ctx, fmt.Sprintf(utils.AsyncStatusKey, bucketName), job).Result()
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("查询任务[%s]的状态报错", job))
		web.InternalError(c, fmt.Sprintf("查询任务[%s]的状态报错", job))
		return
	}
	statusId, err := strconv.ParseInt(status, 10, 64)
	if err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("查询任务[%s]状态字段有误，详情:%s", job, err))
		web.InternalError(c, fmt.Sprintf("查询任务[%s]状态字段有误，详情:%s", job, err))
		return
	}
	jobInfo.Status = tools.GetTaskMsgById(int(statusId))

	// result
	res, err := lgRedis.HGet(ctx, fmt.Sprintf(utils.AsyncResultKey, bucketName), job).Result()
	if err != nil && err != redis.Nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("获取任务[%s]结果失败，详情:%s", job, err))
		web.InternalError(c, fmt.Sprintf("获取任务[%s]结果失败，详情:%s", job, err))
		return
	}
	jobInfo.Error = res

	// 判断是否是周期任务
	_, err = lgRedis.ZScore(ctx, fmt.Sprintf(utils.AsyncPeriodKey, bucketName), job).Result()
	if err == redis.Nil {
		jobInfo.Type = "Non-Period"
	} else if err == nil {
		jobInfo.Type = "Period"
	} else {
		lgLogger.WithContext(c).Error(fmt.Sprintf("查询任务[%s]是否周期任务报错", job))
		web.InternalError(c, fmt.Sprintf("查询任务[%s]是否周期任务报错", job))
		return
	}

	// 超时重试 或 ready
	if statusId == utils.TaskStatusReady || statusId == utils.TaskStatusTimeOut {
		exist, err := lgRedis.LPos(ctx, fmt.Sprintf(utils.AsyncReadyKey, bucketName), job, redis.LPosArgs{}).Result()
		if err != nil || exist == -1 {
			lgLogger.WithContext(c).Error(fmt.Sprintf("任务[%s]状态为ready，但在ready队列中不存在", job))
			web.InternalError(c, fmt.Sprintf("任务[%s]状态为ready，但在ready队列中不存在", job))
			return
		}
		lLen, err := lgRedis.LLen(ctx, fmt.Sprintf(utils.AsyncReadyKey, bucketName)).Result()
		if err != nil {
			lgLogger.WithContext(c).Error(fmt.Sprintf("查询任务[%s]ready中队列长度报错", job))
			web.InternalError(c, fmt.Sprintf("查询任务[%s]ready中队列长度报错", job))
			return
		}
		jobInfo.QueueInfo = fmt.Sprintf("%d/%d", lLen-exist, lLen)
	}

	// 判断pending中是否存在
	startTime, err := lgRedis.ZScore(ctx, fmt.Sprintf(utils.AsyncPendingKey, bucketName), job).Result()
	if err == nil {
		tm := time.Unix(int64(startTime), 0)
		// 格式化时间为字符串，使用具体的日期时间格式
		timeStr := tm.Format("2006-01-02 15:04:05")
		jobInfo.StartTime = timeStr
	}

	web.Success(c, jobInfo)
	return
}

// TerminateJobHandler 终止任务
//
//	@Summary		终止任务
//	@Description	终止任务
//	@Tags			任务终止
//	@Accept			application/json
//	@Produce		application/json
//	@Param			job	path		string	true	"任务id"
//	@Success		200	{object}	web.Response
//	@Router			/api/async/v0/job/{job}/terminate [patch]
func TerminateJobHandler(c *gin.Context) {
	job := c.Param("job")
	if job == "" {
		web.ParamsError(c, "job参数有误")
		return
	}
	if err := dispatch.NewAsyncQueue().StopJob(job); err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("终止任务[%s]异常: %s", job, err.Error()))
		web.InternalError(c, fmt.Sprintf("终止任务[%s]异常: %s", job, err.Error()))
		return
	}
	web.Success(c, "终止任务成功...")
	return
}

// RecoveryJobHandler 恢复任务
//
//	@Summary		恢复任务
//	@Description	恢复任务
//	@Tags			任务恢复
//	@Accept			application/json
//	@Produce		application/json
//	@Param			job	path		string	true	"任务id"
//	@Success		200	{object}	web.Response
//	@Router			/api/async/v0/job/{job}/recovery [patch]
func RecoveryJobHandler(c *gin.Context) {
	job := c.Param("job")
	if job == "" {
		web.ParamsError(c, "job参数有误")
		return
	}
	if err := dispatch.NewAsyncQueue().RecoverJob(job); err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("恢复任务[%s]异常: %s", job, err.Error()))
		web.InternalError(c, fmt.Sprintf("恢复任务[%s]异常: %s", job, err.Error()))
		return
	}
	web.Success(c, "恢复任务成功...")
	return
}

// DeleteJobHandler 删除任务
//
//	@Summary		删除任务
//	@Description	删除任务
//	@Tags			任务删除
//	@Accept			application/json
//	@Produce		application/json
//	@Param			job	path		string	true	"任务id"
//	@Success		200	{object}	web.Response
//	@Router			/api/async/v0/job/{job} [delete]
func DeleteJobHandler(c *gin.Context) {
	job := c.Param("job")
	if job == "" {
		web.ParamsError(c, "job参数有误")
		return
	}
	if err := dispatch.NewAsyncQueue().DeleteJob(job); err != nil {
		lgLogger.WithContext(c).Error(fmt.Sprintf("删除任务[%s]异常: %s", job, err.Error()))
		web.InternalError(c, fmt.Sprintf("删除任务[%s]异常: %s", job, err.Error()))
		return
	}
	web.Success(c, "删除任务成功...")
	return
}
