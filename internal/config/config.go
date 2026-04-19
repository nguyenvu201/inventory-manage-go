package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the service.
// All values are injected via environment variables — never hardcode.
type Config struct {
	// ── Service ──────────────────────────────────────────────────────────────
	ServiceEnv string `env:"SERVICE_ENV" envDefault:"development"`
	ListenAddr string `env:"LISTEN_ADDR" envDefault:":8080"`
	LogLevel   string `env:"LOG_LEVEL"   envDefault:"info"`

	// ── PostgreSQL + TimescaleDB ─────────────────────────────────────────────
	DBHost     string `env:"DB_HOST,required"`
	DBPort     int    `env:"DB_PORT"     envDefault:"5432"`
	DBName     string `env:"DB_NAME,required"`
	DBUser     string `env:"DB_USER,required"`
	DBPassword string `env:"DB_PASSWORD,required"`
	DBSSLMode  string `env:"DB_SSL_MODE"  envDefault:"disable"`

	// ── MQTT / ChirpStack ─────────────────────────────────────────────────────
	MQTTBroker     string `env:"MQTT_BROKER,required"`
	MQTTPort       int    `env:"MQTT_PORT"       envDefault:"1883"`
	MQTTClientID   string `env:"MQTT_CLIENT_ID"  envDefault:"inventory-manage"`
	MQTTUsername   string `env:"MQTT_USERNAME"`
	MQTTPassword   string `env:"MQTT_PASSWORD"`
	// MQTTTLSEnabled: true khi kết nối tới cloud broker dùng TLS (HiveMQ Cloud port 8883)
	// Local Mosquitto: false (plain tcp port 1883)
	MQTTTLSEnabled bool   `env:"MQTT_TLS_ENABLED" envDefault:"false"`

	// ── Migrations ────────────────────────────────────────────────────────────
	MigrationsDir string `env:"MIGRATIONS_DIR" envDefault:"migrations"`

	// ── SMTP (Alert Notifications) ────────────────────────────────────────────
	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUsername string `env:"SMTP_USERNAME"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPFrom     string `env:"SMTP_FROM"`

	// ── Twilio SMS ────────────────────────────────────────────────────────────
	TwilioAccountSID string `env:"TWILIO_ACCOUNT_SID"`
	TwilioAuthToken  string `env:"TWILIO_AUTH_TOKEN"`
	TwilioFromNumber string `env:"TWILIO_FROM_NUMBER"`
}

// DSN returns a PostgreSQL connection string built from config fields.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBName, c.DBUser, c.DBPassword, c.DBSSLMode,
	)
}

// Load reads the .env file (if present) then parses environment variables
// into a Config struct. Returns an error if any required field is missing.
func Load() (*Config, error) {
	// Load .env file if it exists — silently ignore if absent (production uses real env vars)
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}

	return cfg, nil
}
