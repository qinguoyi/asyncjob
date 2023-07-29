package base

import (
	"errors"
	"fmt"
	"github.com/qinguoyi/asyncjob/bootstrap"
	"net/smtp"
	"strings"
)

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

// Start 实现loginAuth的接口
func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown fromServer")
		}
	}
	return nil, nil
}

type mailService struct{}

func NewMailService() *mailService {
	return &mailService{}
}

func (m *mailService) Send(toList []string, subject, content string) error {
	smtpHost := bootstrap.NewConfig("").SmtpServer.Host
	smtpPort := bootstrap.NewConfig("").SmtpServer.Port
	from := bootstrap.NewConfig("").SmtpServer.User
	password := bootstrap.NewConfig("").SmtpServer.Password

	to := strings.Join(toList, ",")
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, content)

	// 连接 SMTP 服务器
	auth := LoginAuth(from, password)
	err := smtp.SendMail(fmt.Sprintf("%s:%s", smtpHost, smtpPort), auth, from,
		toList, []byte(message))
	if err != nil {
		return err
	}
	return nil
}
