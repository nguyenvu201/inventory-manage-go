package setting

// Config aggregates all sections of the application configuration.
// Each section maps to a top-level key in config/local.yaml.
type Config struct {
	Server   ServerSetting   `mapstructure:"server"`
	Postgres PostgresSetting `mapstructure:"postgres"`
	Redis    RedisSetting    `mapstructure:"redis"`
	MQTT     MQTTSetting     `mapstructure:"mqtt"`
	Logger   LoggerSetting   `mapstructure:"logger"`
	JWT      JWTSetting      `mapstructure:"jwt"`
}

// ServerSetting controls the HTTP server.
type ServerSetting struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // "dev" | "release"
}

// PostgresSetting contains the TimescaleDB / PostgreSQL connection config.
type PostgresSetting struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxConns        int32  `mapstructure:"maxConns"`
	MinConns        int32  `mapstructure:"minConns"`
	ConnMaxLifetime int    `mapstructure:"connMaxLifetimeSec"`
}

// RedisSetting contains the Redis connection config.
type RedisSetting struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
	PoolSize int    `mapstructure:"poolSize"`
}

// MQTTSetting contains the Mosquitto / ChirpStack broker config.
type MQTTSetting struct {
	Broker   string `mapstructure:"broker"`
	Port     int    `mapstructure:"port"`
	ClientID string `mapstructure:"clientId"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// LoggerSetting controls Zap + Lumberjack log rotation.
type LoggerSetting struct {
	Level       string `mapstructure:"level"`       // debug | info | warn | error
	FileLogName string `mapstructure:"fileLogName"` // path to log file
	MaxSize     int    `mapstructure:"maxSize"`     // MB
	MaxBackups  int    `mapstructure:"maxBackups"`
	MaxAge      int    `mapstructure:"maxAge"` // days
	Compress    bool   `mapstructure:"compress"`
}

// JWTSetting holds JWT signing and expiry config.
type JWTSetting struct {
	SecretKey  string `mapstructure:"secretKey"`
	ExpiryHour int    `mapstructure:"expiryHour"`
}
