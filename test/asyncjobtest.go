package main

import (
	"encoding/json"
	"fmt"
	"github.com/qinguoyi/asyncjob/app/pkg/base"
	"io"
	"strings"
	"time"
)

func sendTestMsg(delay int) map[string]interface{} {
	start := time.Now().Add(time.Duration(delay) * time.Second)
	body := map[string]interface{}{
		"type":      "test_msg",
		"startTime": start.Format("2006-01-02 15:04:05"),
		"body":      map[string]interface{}{"id": 1},
		"ttl":       100,
		"retry":     0,
	}
	return body
}

func sendMailMsg(cronSpec string) map[string]interface{} {
	body := map[string]interface{}{
		"type":     "mail_msg",
		"cronSpec": cronSpec,
		"body": map[string]interface{}{
			"toList":  []string{"1532979219@qq.com"},
			"subject": "今晚吃什么？",
			"content": "不吃火锅，就吃烤匠。",
		},
		"ttl":   100,
		"retry": 0,
	}
	return body
}

func main() {
	baseUrl := "http://127.0.0.1:8888"

	// 非周期任务
	urlStr := "/api/async/v0/job/async"
	jsonBytes, err := json.Marshal(sendTestMsg(10))
	if err != nil {
		panic(err)
	}
	req := base.Request{
		Url:    fmt.Sprintf("%s%s", baseUrl, urlStr),
		Body:   io.NopCloser(strings.NewReader(string(jsonBytes))),
		Method: "POST",
		Params: map[string]string{},
	}
	_, _, _, err = base.Ask(req)
	if err != nil {
		panic(err)
	}

	// 周期任务
	urlStr = "/api/async/v0/job/periodic"
	jsonBytes, err = json.Marshal(sendMailMsg("0/5 * * * *"))
	if err != nil {
		panic(err)
	}
	req = base.Request{
		Url:    fmt.Sprintf("%s%s", baseUrl, urlStr),
		Body:   io.NopCloser(strings.NewReader(string(jsonBytes))),
		Method: "POST",
		Params: map[string]string{},
	}
	_, _, _, err = base.Ask(req)
	if err != nil {
		panic(err)
	}
}
