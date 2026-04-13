package initialize

import (
	"inventory-manage/global"
	"inventory-manage/pkg/logger"
)

// InitLogger initialises the global Zap logger from the loaded configuration.
// Must be called after LoadConfig().
func InitLogger() {
	global.Logger = logger.NewLogger(global.Config.Logger)
	global.Logger.Info("Logger initialised successfully")
}
