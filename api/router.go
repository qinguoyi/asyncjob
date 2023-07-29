package api

import (
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/asyncjob/api/v0"
	"github.com/qinguoyi/asyncjob/app/middleware"
	"github.com/qinguoyi/asyncjob/bootstrap"
	"github.com/qinguoyi/asyncjob/config"
	"github.com/qinguoyi/asyncjob/docs"
	gs "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

func NewRouter(
	conf *config.Configuration,
	lgLogger *bootstrap.LangGoLogger,
) *gin.Engine {
	if conf.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// middleware
	corsM := middleware.NewCors()
	traceL := middleware.NewTrace(lgLogger)
	requestL := middleware.NewRequestLog(lgLogger)
	panicRecover := middleware.NewPanicRecover(lgLogger)

	// 跨域 trace-id 日志
	router.Use(corsM.Handler(), traceL.Handler(), requestL.Handler(), panicRecover.Handler())

	// 静态资源
	router.StaticFile("/assets", "../../static/image/back.png")

	// swag docs
	docs.SwaggerInfo.BasePath = "/"
	router.GET("/swagger/*any", gs.WrapHandler(swaggerFiles.Handler))

	// 动态资源 注册 api 分组路由
	setApiGroupRoutes(router)

	return router
}

func setApiGroupRoutes(
	router *gin.Engine,
) *gin.RouterGroup {
	group := router.Group("/api/async/v0")
	{
		//health
		group.GET("/health", v0.HealthCheckHandler)
	}
	{
		// async
		group.GET("/job/:job", v0.GetJobInfoHandler)
		group.POST("/job/async", v0.DelayJobDeliveryHandler)
		group.POST("/job/periodic", v0.PeriodicJobDeliveryHandler)
		group.PATCH("/job/:job/terminate", v0.TerminateJobHandler)
		group.PATCH("/job/:job/recovery", v0.RecoveryJobHandler)
		group.DELETE("/job/:job", v0.DeleteJobHandler)
	}
	return group
}
