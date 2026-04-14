package controller

import (
	"inventory-manage/global"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
	"inventory-manage/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ReportController struct {
	reportService service.IReportService
}

func NewReportController(reportService service.IReportService) *ReportController {
	return &ReportController{
		reportService: reportService,
	}
}

// GetConsumptionTrend godoc
//
//	@Summary      Get consumption trend report
//	@Description  Returns aggregated historical consumption data points
//	@Tags         reports
//	@Produce      json
//	@Param        sku_code query string true "SKU Code"
//	@Param        from     query string true "Start time (RFC3339)"
//  @Param        to       query string true "End time (RFC3339)"
//  @Param        interval query string true "Aggregation interval (1h, 1d, 1w)"
//  @Param        limit    query int    false "Pagination limit"
//  @Param        cursor   query string false "Pagination cursor"
//	@Success      200  {object}  response.ResponseData
//	@Failure      400  {object}  response.ErrorResponseData
//	@Router       /api/v1/reports/consumption [get]
func (rc *ReportController) GetConsumptionTrend(c *gin.Context) {
	var query model.ConsumptionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ErrorResponseWithHTTP(c, http.StatusBadRequest, response.ErrCodeParamInvalid, err.Error())
		return
	}

	points, nextCursor, err := rc.reportService.GetConsumptionTrend(c.Request.Context(), query)
	if err != nil {
		traceID := c.GetString("trace_id")
		global.Logger.Error("GetConsumptionTrend failed", zap.Error(err), zap.String("trace_id", traceID))
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, map[string]interface{}{
		"points":      points,
		"next_cursor": nextCursor,
	})
}

// GetConsumptionSummary godoc
//
//	@Summary      Get consumption summary
//	@Description  Returns high level summary of total consumption
//	@Tags         reports
//	@Produce      json
//	@Param        sku_code query string true "SKU Code"
//	@Param        from     query string true "Start time (RFC3339)"
//  @Param        to       query string true "End time (RFC3339)"
//  @Param        interval query string true "Aggregation interval (1h, 1d, 1w)"
//	@Success      200  {object}  response.ResponseData
//	@Failure      400  {object}  response.ErrorResponseData
//	@Router       /api/v1/reports/consumption/summary [get]
func (rc *ReportController) GetConsumptionSummary(c *gin.Context) {
	var query model.ConsumptionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ErrorResponseWithHTTP(c, http.StatusBadRequest, response.ErrCodeParamInvalid, err.Error())
		return
	}

	summary, err := rc.reportService.GetConsumptionSummary(c.Request.Context(), query)
	if err != nil {
		traceID := c.GetString("trace_id")
		global.Logger.Error("GetConsumptionSummary failed", zap.Error(err), zap.String("trace_id", traceID))
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, summary)
}
