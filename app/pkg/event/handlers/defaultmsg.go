package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qinguoyi/asyncjob/app/pkg/event"
	"github.com/qinguoyi/asyncjob/app/pkg/utils"
	"github.com/qinguoyi/asyncjob/bootstrap"
	"time"
)

func init() {
	event.NewEventsHandler().RegHandler(utils.TaskTestMsg, handleTest)
}

type MsgTest struct {
	Id int `json:"id"`
}

func handleTest(i interface{}, c context.Context, t <-chan time.Time, s chan struct{}) error {
	var msgTest MsgTest
	msgByte, _ := json.Marshal(i)
	if err := json.Unmarshal(msgByte, &msgTest); err != nil {
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("unmashal job failed: %v", err))
		return fmt.Errorf("unmashal job failed: %v", err)
	}
	bootstrap.NewLogger().Logger.Info(fmt.Sprintf("time:%s, num:%v, unmarshal:%v\n", time.Now(), i, msgTest.Id))
	return nil
}
