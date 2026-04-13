package routers

import (
	devicerouter "inventory-manage/internal/routers/device"
	calibrationrouter "inventory-manage/internal/routers/calibration"
)

// RouterGroup aggregates all domain route groups.
// New domains (inventory, notification, etc.) are added here.
type RouterGroup struct {
	Device      devicerouter.DeviceRouterGroup
	Calibration calibrationrouter.CalibrationRouterGroup
}

// RouterGroupApp is the singleton router group used in initialize/router.go.
var RouterGroupApp = new(RouterGroup)
