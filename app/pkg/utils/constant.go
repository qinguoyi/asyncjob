package utils

// 任务类型
const (
	TaskMailMsg = "mail_msg"
	TaskTestMsg = "test_msg"
)

// 任务状态
const (
	TaskStatusPending   = 0
	TaskStatusReady     = 1
	TaskStatusRunning   = 2
	TaskStatusFinish    = 3
	TaskStatusTerminate = 97
	TaskStatusTimeOut   = 98
	TaskStatusError     = 99
)

// MaxWorker worker
const (
	MaxWorker = 2000
)

// bucket
const (
	BucketName = "bucket_%d"
	BucketNum  = 5
)

// async
const (
	AsyncPendingKey     = "async:%s:pending"      // zSet
	AsyncReadyKey       = "async:%s:ready"        // list
	AsyncRunningKey     = "async:%s:running"      // zSet
	AsyncRetryKey       = "async:%s:retry"        // hSet
	AsyncTTLKey         = "async:%s:ttl"          // hSet
	AsyncPeriodKey      = "async:%s:period"       // zSet
	AsyncBucketKey      = "async:%s:bucket"       // hSet
	AsyncResultKey      = "async:%s:result"       // hSet
	AsyncTerminateKey   = "async:%s:terminate"    // hSet
	AsyncInfoKey        = "async:%s:info"         // hSet
	AsyncPeriodRetryKey = "async:%s:period:retry" // hSet
	AsyncStatusKey      = "async:%s:status"       // hSet
)
