package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"inventory-manage/internal/config"
	"inventory-manage/internal/domain/telemetry"
	inventorymqtt "inventory-manage/internal/platform/mqtt"
	"inventory-manage/internal/worker"
)

func main() {
	// Pretty logging in dev, JSON in production
	if os.Getenv("SERVICE_ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	log.Info().
		Str("env", cfg.ServiceEnv).
		Str("addr", cfg.ListenAddr).
		Msg("inventory-manage service starting")

	// Setup MQTT Client
	mqttClient, err := inventorymqtt.NewClient(
		cfg.MQTTBroker, cfg.MQTTPort, cfg.MQTTClientID, cfg.MQTTUsername, cfg.MQTTPassword,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize MQTT client")
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mqttClient.Connect(startupCtx); err != nil {
		log.Error().Err(err).Msg("MQTT broker unavailable at startup, will rely on auto-reconnect")
	}

	// Start Telemetry Pipeline
	telemetryChan := make(chan telemetry.TelemetryPayload, 1000)
	processor := telemetry.NewProcessor()
	receiver := worker.NewTelemetryReceiver(mqttClient, processor, telemetryChan)

	if err := receiver.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start telemetry receiver")
	}

	// Temporary: consume the channel so it doesn't block until TASK-004
	go func() {
		for payload := range telemetryChan {
			log.Debug().Interface("payload", payload).Msg("Drained payload from channel (TASK-004 stub)")
		}
	}()

	// Setup HTTP Server
	router := http.NewServeMux()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Str("addr", cfg.ListenAddr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	<-quit
	log.Info().Msg("shutdown signal received")
	
	close(telemetryChan)
	mqttClient.Disconnect()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP server forced shutdown")
	}

	log.Info().Msg("service stopped")
}
