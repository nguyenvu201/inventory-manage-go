package report

import (
	"github.com/gin-gonic/gin"

	"inventory-manage/internal/controller"
)

type ReportRouter struct{}

func (s *ReportRouter) InitReportRouter(Router *gin.RouterGroup, reportCtrl *controller.ReportController) {
	reportGroup := Router.Group("/reports")
	{
		reportGroup.GET("/consumption", reportCtrl.GetConsumptionTrend)
		reportGroup.GET("/consumption/summary", reportCtrl.GetConsumptionSummary)
	}
}
