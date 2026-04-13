package routers

import (
	calibrationrouter "inventory-manage/internal/routers/calibration"
	devicerouter "inventory-manage/internal/routers/device"
	inventoryrouter "inventory-manage/internal/routers/inventory"
)

// RouterGroup aggregates all domain route groups.
// New domains (inventory, notification, etc.) are added here.
type RouterGroup struct {
	Device      devicerouter.DeviceRouterGroup
	Calibration calibrationrouter.CalibrationRouterGroup
	Inventory   inventoryrouter.InventoryRouter
}

// RouterGroupApp is the singleton router group used in initialize/router.go.
var RouterGroupApp = new(RouterGroup)
