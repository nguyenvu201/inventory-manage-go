package main

import (
	"context"
	"errors"
	"fmt"
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
	"inventory-manage/internal/repository/postgres"
	"inventory-manage/internal/worker"

	"github.com/jackc/pgx/v5/pgxpool"
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
	validator := telemetry.NewValidator()
	receiver := worker.NewTelemetryReceiver(mqttClient, processor, validator, telemetryChan)

	if err := receiver.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start telemetry receiver")
	}

	// Setup Database Connection
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	
	pgxConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse DB connection string")
	}

	dbPool, err := pgxpool.NewWithConfig(context.Background(), pgxConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("database ping failed")
	}
	log.Info().Msg("PostgreSQL connected successfully")

	// Setup Repositories
	telemetryRepo := postgres.NewTelemetryRepository(dbPool)

	// Start Storage Worker
	storageCtx, cancelStorage := context.WithCancel(context.Background())
	storageWorker := worker.NewStorageWorker(telemetryRepo, telemetryChan)
	go func() {
		log.Info().Msg("starting storage worker")
		if err := storageWorker.Start(storageCtx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("storage worker exited with error")
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
	cancelStorage()
	
	mqttClient.Disconnect()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP server forced shutdown")
	}

	log.Info().Msg("service stopped")
}
