package controller

import (
	"inventory-manage/global"
	"inventory-manage/internal/model"
	"inventory-manage/internal/service"
	"inventory-manage/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ThresholdController struct {
	thresholdService service.IThresholdService
}

func NewThresholdController(thresholdService service.IThresholdService) *ThresholdController {
	return &ThresholdController{
		thresholdService: thresholdService,
	}
}

// CreateRule godoc
//
//	@Summary      Create a threshold rule
//	@Description  Creates a new threshold rule for a SKU
//	@Tags         rules
//	@Accept       json
//	@Produce      json
//	@Param        rule body      model.ThresholdRule true "Rule details"
//	@Success      200  {object}  response.ResponseData
//	@Failure      400  {object}  response.ErrorResponseData
//	@Router       /api/v1/rules/thresholds [post]
func (tc *ThresholdController) CreateRule(c *gin.Context) {
	var rule model.ThresholdRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.ErrorResponse(c, response.ErrCodeParamInvalid, err.Error())
		return
	}

	err := tc.thresholdService.CreateRule(c.Request.Context(), &rule)
	if err != nil {
		traceID := c.GetString("trace_id")
		global.Logger.Error("CreateRule failed", zap.Error(err), zap.String("trace_id", traceID))
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, rule)
}

// GetRules godoc
//
//	@Summary      Get threshold rules
//	@Description  Returns all rules matching the query
//	@Tags         rules
//	@Produce      json
//	@Param        sku_code query string false "SKU Code"
//	@Success      200  {object}  response.ResponseData
//	@Failure      500  {object}  response.ErrorResponseData
//	@Router       /api/v1/rules/thresholds [get]
func (tc *ThresholdController) GetRules(c *gin.Context) {
	var query model.ThresholdRuleQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ErrorResponse(c, response.ErrCodeParamInvalid, err.Error())
		return
	}

	rules, err := tc.thresholdService.GetRules(c.Request.Context(), query)
	if err != nil {
		traceID := c.GetString("trace_id")
		global.Logger.Error("GetRules failed", zap.Error(err), zap.String("trace_id", traceID))
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, rules)
}

// UpdateRule godoc
//
//	@Summary      Update a threshold rule
//	@Description  Updates an existing threshold rule
//	@Tags         rules
//	@Accept       json
//	@Produce      json
//	@Param        id   path      string              true "Rule ID"
//	@Param        rule body      model.ThresholdRule true "Rule details"
//	@Success      200  {object}  response.ResponseData
//	@Failure      400  {object}  response.ErrorResponseData
//	@Router       /api/v1/rules/thresholds/{id} [put]
func (tc *ThresholdController) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var rule model.ThresholdRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.ErrorResponse(c, response.ErrCodeParamInvalid, err.Error())
		return
	}

	err := tc.thresholdService.UpdateRule(c.Request.Context(), id, &rule)
	if err != nil {
		traceID := c.GetString("trace_id")
		global.Logger.Error("UpdateRule failed", zap.Error(err), zap.String("trace_id", traceID))
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, nil)
}

// DeleteRule godoc
//
//	@Summary      Delete a threshold rule
//	@Description  Deletes an existing threshold rule
//	@Tags         rules
//	@Produce      json
//	@Param        id   path      string true "Rule ID"
//	@Success      200  {object}  response.ResponseData
//	@Failure      500  {object}  response.ErrorResponseData
//	@Router       /api/v1/rules/thresholds/{id} [delete]
func (tc *ThresholdController) DeleteRule(c *gin.Context) {
	id := c.Param("id")

	err := tc.thresholdService.DeleteRule(c.Request.Context(), id)
	if err != nil {
		traceID := c.GetString("trace_id")
		global.Logger.Error("DeleteRule failed", zap.Error(err), zap.String("trace_id", traceID))
		response.ErrorResponse(c, response.ErrCodeInternalServer, err.Error())
		return
	}

	response.SuccessResponse(c, response.ErrCodeSuccess, nil)
}
