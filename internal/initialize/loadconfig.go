package initialize

import (
	"fmt"

	"inventory-manage/global"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// LoadConfig reads the YAML configuration file and unmarshals it into global.Config.
// The config file is resolved from ./config/local.yaml relative to the working directory.
func LoadConfig() {
	v := viper.New()
	v.AddConfigPath("./config/")
	v.SetConfigName("local")
	v.SetConfigType("yaml")
	v.BindEnv("postgres.host", "DB_HOST")
	v.BindEnv("redis.host", "REDIS_HOST")
	v.BindEnv("mqtt.broker", "MQTT_BROKER")

	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("LoadConfig: failed to read config file: %v", err))
	}

	if err := v.Unmarshal(&global.Config); err != nil {
		panic(fmt.Sprintf("LoadConfig: failed to unmarshal config: %v", err))
	}

	// Logger may not be initialised yet — safe to use fmt here.
	fmt.Printf("Config loaded — server.port=%d mode=%s\n",
		global.Config.Server.Port,
		global.Config.Server.Mode,
	)
}

// loggerOrPanic is a helper used by other init functions to log fatal errors
// after the logger has been initialised.
func loggerOrPanic(msg string, err error) {
	if global.Logger != nil {
		global.Logger.Fatal(msg, zap.Error(err))
	} else {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
}
