package threshold

import (
	"inventory-manage/internal/controller"

	"github.com/gin-gonic/gin"
)

type ThresholdRouter struct{}

func (s *ThresholdRouter) InitThresholdRouter(Router *gin.RouterGroup, thresholdController *controller.ThresholdController) {
	thresholdGroup := Router.Group("rules/thresholds")
	{
		thresholdGroup.POST("", thresholdController.CreateRule)
		thresholdGroup.GET("", thresholdController.GetRules)
		thresholdGroup.PUT(":id", thresholdController.UpdateRule)
		thresholdGroup.DELETE(":id", thresholdController.DeleteRule)
	}
}
