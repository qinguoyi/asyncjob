package tools

import "github.com/qinguoyi/asyncjob/app/pkg/utils"

var taskIDMapMsg map[int]string

func init() {
	taskIDMapMsg = map[int]string{
		utils.TaskStatusPending:   "Pending",
		utils.TaskStatusReady:     "Ready",
		utils.TaskStatusRunning:   "Running",
		utils.TaskStatusFinish:    "Success",
		utils.TaskStatusTerminate: "Terminate",
		utils.TaskStatusTimeOut:   "Timeout",
		utils.TaskStatusError:     "Error",
	}
}

func GetTaskMsgById(id int) string {
	k, ok := taskIDMapMsg[id]
	if ok {
		return k
	} else {
		return ""
	}
}
