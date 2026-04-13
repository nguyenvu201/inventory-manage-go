package calibration

import (
	"inventory-manage/internal/controller"

	"github.com/gin-gonic/gin"
)

// CalibrationRouterGroup groups all calibration-related routes.
type CalibrationRouterGroup struct{}

// InitCalibrationRouter registers calibration endpoints on the given RouterGroup.
func (rg *CalibrationRouterGroup) InitCalibrationRouter(r *gin.RouterGroup, cc *controller.CalibrationController) {
	calibGroup := r.Group("/calibrations")
	{
		calibGroup.POST("", cc.CreateCalibration)
		calibGroup.GET("/:device_id/active", cc.GetActiveCalibration)
		calibGroup.POST("/:device_id/update", cc.UpdateCalibration)
		calibGroup.GET("/:device_id/history", cc.GetAuditHistory)
	}
}
