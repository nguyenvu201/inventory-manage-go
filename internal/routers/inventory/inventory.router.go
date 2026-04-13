package inventory

import (
	"github.com/gin-gonic/gin"
	
	"inventory-manage/internal/controller"
	"inventory-manage/internal/middlewares"
)

type InventoryRouter struct{}

func (r *InventoryRouter) InitInventoryRouter(Router *gin.RouterGroup, inventoryController *controller.InventoryController) {
	inventoryRouter := Router.Group("inventory").Use(middlewares.RequestID())

	{
		inventoryRouter.GET("current", inventoryController.GetCurrentInventory)
		inventoryRouter.GET(":sku_code/current", inventoryController.GetInventoryBySKU)
	}
}
