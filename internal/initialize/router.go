package initialize

import (
	"fmt"

	"inventory-manage/global"
	"inventory-manage/internal/controller"
	"inventory-manage/internal/middlewares"
	"inventory-manage/internal/repository/postgres"
	"inventory-manage/internal/routers"
	"inventory-manage/internal/service/impl"

	"github.com/gin-gonic/gin"
)

// InitRouter builds and returns the Gin engine with all routes registered.
// Service instances are wired manually here (Google Wire auto-generation
// can be added after the initial setup is stable).
func InitRouter() *gin.Engine {
	var r *gin.Engine

	if global.Config.Server.Mode == "dev" {
		gin.SetMode(gin.DebugMode)
		gin.ForceConsoleColor()
		r = gin.Default()
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	}

	// ── Global Middleware ────────────────────────────────────
	r.Use(middlewares.RequestID())
	r.Use(middlewares.ZapLogger())

	// ── Health endpoint ──────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "inventory-manage"})
	})

	// ── Dependency Injection ─────────────────────────────────
	// Repository layer
	deviceRepo := postgres.NewDeviceRepository(global.Pdb)
	calibRepo := postgres.NewCalibrationRepository(global.Pdb)
	inventoryRepo := postgres.NewInventoryRepository(global.Pdb)

	// Service layer
	deviceSvc := impl.NewDeviceService(deviceRepo)
	calibSvc := impl.NewCalibrationService(calibRepo)

	// Controller layer
	deviceCtrl := controller.NewDeviceController(deviceSvc)
	calibCtrl := controller.NewCalibrationController(calibSvc)
	inventoryCtrl := controller.NewInventoryController(inventoryRepo)

	// ── Route Groups ─────────────────────────────────────────
	deviceRouter := routers.RouterGroupApp.Device
	calibRouter := routers.RouterGroupApp.Calibration
	inventoryRouter := routers.RouterGroupApp.Inventory

	v1 := r.Group(fmt.Sprintf("/api/v1"))
	{
		deviceRouter.InitDeviceRouter(v1, deviceCtrl)
		calibRouter.InitCalibrationRouter(v1, calibCtrl)
		inventoryRouter.InitInventoryRouter(v1, inventoryCtrl)
	}

	return r
}
