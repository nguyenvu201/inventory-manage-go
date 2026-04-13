package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ResponseData is the standard API success envelope.
type ResponseData struct {
	Code    int         `json:"code"`    // business status code (20001, 40001 …)
	Message string      `json:"message"` // human-readable message
	Data    interface{} `json:"data"`    // response payload (nil on error)
}

// ErrorResponseData is the standard API error envelope.
type ErrorResponseData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"` // optional extra context
}

// SuccessResponse writes a 200 response with the INV business code.
func SuccessResponse(c *gin.Context, code int, data interface{}) {
	c.JSON(http.StatusOK, ResponseData{
		Code:    code,
		Message: GetMsg(code),
		Data:    data,
	})
}

// ErrorResponse writes a 200 HTTP response with an INV error code.
// Following the go-ecommerce-api convention: HTTP 200, business code signals error.
func ErrorResponse(c *gin.Context, code int, detail string) {
	message := GetMsg(code)
	c.JSON(http.StatusOK, ErrorResponseData{
		Code:    code,
		Message: message,
		Detail:  detail,
	})
}

// ErrorResponseWithHTTP writes a non-200 HTTP status for hard failures
// (e.g. 404 Not Found, 500 Internal Server Error).
func ErrorResponseWithHTTP(c *gin.Context, httpStatus int, code int, detail string) {
	c.JSON(httpStatus, ErrorResponseData{
		Code:    code,
		Message: GetMsg(code),
		Detail:  detail,
	})
}
