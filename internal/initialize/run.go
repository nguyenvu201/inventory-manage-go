package initialize

import (
	"context"

	"inventory-manage/global"
	"inventory-manage/internal/domain/telemetry"
	"inventory-manage/internal/model"
	"inventory-manage/internal/repository/postgres"
	"inventory-manage/internal/worker"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Run is the single entry point called from main.go.
// It initialises all infrastructure components in the correct order,
// starts background workers, and returns the Gin engine.
func Run() *gin.Engine {
	// Phase 1: Configuration must load first
	LoadConfig()

	// Phase 2: Logger depends on Config
	InitLogger()

	// Phase 3: Infrastructure (DB, Redis, MQTT)
	InitPostgres()
	InitRedis()
	InitMQTT()

	// Phase 4: Start telemetry pipeline workers
	startWorkers()

	// Phase 5: HTTP router (last — depends on all above)
	r := InitRouter()

	global.Logger.Info("Inventory Manage service started successfully",
		zap.Int("port", global.Config.Server.Port),
		zap.String("mode", global.Config.Server.Mode),
	)

	return r
}

// startWorkers initialises and launches the background MQTT telemetry pipeline.
// The pipeline runs in goroutines and is separate from the HTTP server lifecycle.
func startWorkers() {
	if MQTTClient == nil {
		global.Logger.Warn("MQTT client not initialised — skipping telemetry workers")
		return
	}

	// Channel pipeline: MQTT receiver → storage worker
	telemetryChan := make(chan model.TelemetryPayload, 1000)

	processor := telemetry.NewProcessor()
	validator := telemetry.NewValidator()

	receiver := worker.NewTelemetryReceiver(MQTTClient, processor, validator, telemetryChan)
	if err := receiver.Start(); err != nil {
		global.Logger.Error("failed to start telemetry MQTT receiver", zap.Error(err))
		return
	}

	telemetryRepo := postgres.NewTelemetryRepository(global.Pdb)
	storageWorker := worker.NewStorageWorker(telemetryRepo, telemetryChan)

	// Storage worker runs until context cancels — context is passed from main via shutdown hook
	go func() {
		global.Logger.Info("Starting storage worker")
		if err := storageWorker.Start(context.Background()); err != nil {
			global.Logger.Error("storage worker exited with error", zap.Error(err))
		}
	}()

	global.Logger.Info("Telemetry pipeline workers started")
}
