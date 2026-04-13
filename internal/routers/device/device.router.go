package device

import (
	"inventory-manage/internal/controller"

	"github.com/gin-gonic/gin"
)

// DeviceRouterGroup groups all device-related routes.
type DeviceRouterGroup struct{}

// InitDeviceRouter registers device endpoints on the given RouterGroup.
// Called from initialize/router.go.
func (rg *DeviceRouterGroup) InitDeviceRouter(r *gin.RouterGroup, dc *controller.DeviceController) {
	deviceGroup := r.Group("/devices")
	{
		deviceGroup.POST("", dc.CreateDevice)
		deviceGroup.GET("", dc.ListDevices)
		deviceGroup.GET("/:id", dc.GetDevice)
		deviceGroup.PUT("/:id", dc.UpdateDevice)
		deviceGroup.DELETE("/:id", dc.DeleteDevice)
	}
}
