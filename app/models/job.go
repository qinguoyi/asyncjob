package models

// AsyncJob 异步任务(非周期)
type AsyncJob struct {
	Type      string      `json:"type" msgpack:"1" binding:"required"` // 任务类型
	StartTime string      `json:"startTime" msgpack:"2"`               // 启动时间
	Body      interface{} `json:"body" msgpack:"3" binding:"required"` // 任务消息体
	TTL       int64       `json:"ttl" msgpack:"4" binding:"required"`  // 任务超时时间, 秒
	Retry     int         `json:"retryCount" msgpack:"5"`              // 任务重试次数
}

// PeriodicJob 周期任务
type PeriodicJob struct {
	Type     string      `json:"type" msgpack:"1" binding:"required"`     // 任务类型
	CronSpec string      `json:"cronSpec" msgpack:"2" binding:"required"` // cron语法
	Body     interface{} `json:"body" msgpack:"3" binding:"required"`     // 任务消息体
	TTL      int64       `json:"ttl" msgpack:"4" binding:"required"`      // 任务超时时间, 秒
	Retry    int         `json:"retryCount" msgpack:"5"`                  // 任务重试次数
}

// JobInfo 任务详情
type JobInfo struct {
	UUID      string // 任务ID
	Type      string // 周期/非周期任务
	Status    string // 任务状态
	Error     string // 错误信息
	StartTime string // 下次启动时间
	QueueInfo string // 当前排队号/总排队号
}
