package v0

import (
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/asyncjob/app/pkg/web"
	"github.com/qinguoyi/asyncjob/bootstrap"
)

var lgLogger *bootstrap.LangGoLogger

// HealthCheckHandler 健康检查
//
//	@Summary		健康检查
//	@Description	健康检查
//	@Tags			检查
//	@Accept			application/json
//	@Produce		application/json
//	@Success		200	{object}	web.Response
//	@Router			/api/async/v0/health [get]
func HealthCheckHandler(c *gin.Context) {
	lgLogger.WithContext(c).Info("Health...")
	web.Success(c, "Health...")
	return
}
