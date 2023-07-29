package main

import (
	"github.com/qinguoyi/asyncjob/api"
	"github.com/qinguoyi/asyncjob/app"
	"github.com/qinguoyi/asyncjob/app/pkg/base"
	"github.com/qinguoyi/asyncjob/bootstrap"
	"github.com/qinguoyi/asyncjob/bootstrap/plugins"
	"time"
)

//	@title		asyncjob
//	@version	1.0
//	@description
//	@contact.name	qinguoyi
//	@host			127.0.0.1:8888
//	@BasePath		/
func main() {
	// config log
	lgConfig := bootstrap.NewConfig("conf/config.yaml")
	lgLogger := bootstrap.NewLogger()

	// plugins DB Redis Minio
	plugins.NewPlugins()
	defer plugins.ClosePlugins()

	// 初始化发号器
	uid, _ := base.NewUidGroup()
	for {
		if uid == nil {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	// router
	engine := api.NewRouter(lgConfig, lgLogger)
	server := app.NewHttpServer(lgConfig, engine)

	// app run-server
	application := app.NewApp(lgConfig, lgLogger.Logger, server)
	application.RunServer()
}
