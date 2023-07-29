package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qinguoyi/asyncjob/app/pkg/base"
	"github.com/qinguoyi/asyncjob/app/pkg/event"
	"github.com/qinguoyi/asyncjob/app/pkg/utils"
	"github.com/qinguoyi/asyncjob/bootstrap"
	"time"
)

func init() {
	event.NewEventsHandler().RegHandler(utils.TaskMailMsg, handleMail)
}

type MailMsg struct {
	ToList  []string `json:"toList"`
	Subject string   `json:"subject"`
	Content string   `json:"content"`
}

func handleMail(i interface{}, c context.Context, t <-chan time.Time, s chan struct{}) error {
	var msgMail MailMsg
	msgByte, _ := json.Marshal(i)
	if err := json.Unmarshal(msgByte, &msgMail); err != nil {
		bootstrap.NewLogger().Logger.Info(fmt.Sprintf("unmashal job failed: %v", err))
		return fmt.Errorf("unmashal job failed: %v", err)
	}
	if len(msgMail.ToList) == 0 || msgMail.Subject == "" || msgMail.Content == "" {
		return fmt.Errorf("参数有误")
	}
	if err := base.NewMailService().Send(msgMail.ToList, msgMail.Subject, msgMail.Content); err != nil {
		return err
	}
	return nil
}
