package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luongdev/fsagent/pkg/calculator"
	"github.com/luongdev/fsagent/pkg/config"
	"github.com/luongdev/fsagent/pkg/connection"
	"github.com/luongdev/fsagent/pkg/exporter"
	"github.com/luongdev/fsagent/pkg/logger"
	"github.com/luongdev/fsagent/pkg/processor"
	"github.com/luongdev/fsagent/pkg/server"
	"github.com/luongdev/fsagent/pkg/store"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
	version    = flag.Bool("version", false, "Print version and exit")
	help       = flag.Bool("help", false, "Print usage information")
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Print version and exit
	if *version {
		fmt.Println("FSAgent - FreeSWITCH Metrics Collection Agent")
		fmt.Println("Version: 0.1.0")
		os.Exit(0)
	}

	// Print help and exit
	if *help {
		fmt.Println("FSAgent - FreeSWITCH Metrics Collection Agent")
		fmt.Println("Version: 0.1.0")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  fsagent [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  fsagent                              # Use default config.yaml")
		fmt.Println("  fsagent -config /etc/fsagent.yaml    # Use custom config path")
		fmt.Println("  fsagent -version                     # Print version")
		fmt.Println("  fsagent -help                        # Print this help")
		os.Exit(0)
	}

	fmt.Println("FSAgent - FreeSWITCH Metrics Collection Agent")
	fmt.Println("Version: 0.1.0")

	// Initialize configuration from file or environment variables
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration from %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	// Initialize logger with configured level and format
	logLevel := logger.ParseLogLevel(cfg.Logging.Level)
	logger.InitWithFormat(logLevel, cfg.Logging.Format)
	logger.Info("FSAgent starting with log level: %s, format: %s", cfg.Logging.Level, cfg.Logging.Format)

	logger.Info("Loaded configuration with %d FreeSWITCH instance(s)", len(cfg.FreeSwitchInstances))

	// Perform startup validation
	if err := validateStartup(cfg); err != nil {
		logger.Error("Startup validation failed: %v", err)
		os.Exit(1)
	}
	logger.Info("Startup validation completed successfully")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize State Store
	logger.Info("Initializing state store: type=%s", cfg.Storage.Type)

	var stateStore store.StateStore
	if cfg.Storage.Type == "redis" && cfg.Storage.Redis != nil {
		stateStore, err = store.NewStateStore(
			cfg.Storage.Type,
			cfg.Storage.Redis.Host,
			cfg.Storage.Redis.Port,
			cfg.Storage.Redis.Password,
			cfg.Storage.Redis.DB,
		)
	} else {
		// Use memory store (default or when Redis config is missing)
		stateStore, err = store.NewStateStore("memory", "", 0, "", 0)
	}

	if err != nil {
		logger.Error("Failed to initialize state store: %v", err)
		os.Exit(1)
	}
	defer stateStore.Close()
	logger.Info("State store initialized successfully")

	// Initialize RTCP Calculator
	rtcpCalculator := calculator.NewRTCPCalculator(stateStore)
	logger.Info("RTCP calculator initialized")

	// Initialize QoS Calculator
	qosCalculator := calculator.NewQoSCalculator(stateStore)
	logger.Info("QoS calculator initialized")

	// Initialize OpenTelemetry Metrics Exporter
	metricsExporter, err := exporter.NewMetricsExporter(&cfg.OpenTelemetry)
	if err != nil {
		logger.Error("Failed to initialize metrics exporter: %v", err)
		os.Exit(1)
	}
	logger.Info("Metrics exporter initialized: endpoint=%s", cfg.OpenTelemetry.Endpoint)

	// Initialize Event Processor
	eventProcessor := processor.NewEventProcessor(stateStore, rtcpCalculator, qosCalculator, metricsExporter, cfg.Events.RTCP, cfg.Events.QoS)
	logger.Info("Event processor initialized: rtcp=%v, qos=%v", cfg.Events.RTCP, cfg.Events.QoS)
	if err := eventProcessor.Start(ctx); err != nil {
		logger.Error("Failed to start event processor: %v", err)
		os.Exit(1)
	}
	defer eventProcessor.Stop()
	logger.Info("Event processor started")

	// Initialize Connection Manager
	connManager := connection.NewConnectionManager(cfg.FreeSwitchInstances, cfg.Events.RTCP, cfg.Events.QoS)

	// Set event processor as the event forwarder
	if cm, ok := connManager.(*connection.DefaultConnectionManager); ok {
		cm.SetEventForwarder(eventProcessor)
		logger.Info("Event processor connected to connection manager")
	}

	// Start connection manager
	if err := connManager.Start(ctx); err != nil {
		logger.Warn("Some connections failed to start: %v", err)
	}
	defer connManager.Stop()

	// Get connection status
	connStatus := connManager.GetStatus()
	activeConnections := 0
	for _, status := range connStatus {
		if status.Connected {
			activeConnections++
		}
	}

	if activeConnections == 0 {
		logger.Error("No FreeSWITCH connections established")
		os.Exit(1)
	}

	logger.Info("FSAgent started successfully with %d active connection(s)", activeConnections)

	// Initialize and start HTTP server
	httpServer := server.NewHTTPServer(cfg.HTTP.Port, connManager)
	if err := httpServer.Start(ctx); err != nil {
		logger.Error("Failed to start HTTP server: %v", err)
		os.Exit(1)
	}
	defer httpServer.Stop()
	logger.Info("HTTP server started on port %d", cfg.HTTP.Port)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received, initiating graceful shutdown...")

	// Perform graceful shutdown
	gracefulShutdown(ctx, cancel, connManager, eventProcessor, metricsExporter, stateStore, httpServer)

	logger.Info("FSAgent stopped successfully")
}

// validateStartup performs startup validation of all components
func validateStartup(cfg *config.Config) error {
	logger.Info("Performing startup validation...")

	// Validate State Store connection
	logger.Info("Validating state store connection...")
	if err := validateStateStore(cfg); err != nil {
		return fmt.Errorf("state store validation failed: %w", err)
	}
	logger.Info("State store validation successful")

	// Validate OTel endpoint connectivity
	logger.Info("Validating OpenTelemetry endpoint connectivity...")
	if err := validateOTelEndpoint(cfg); err != nil {
		return fmt.Errorf("OTel endpoint validation failed: %w", err)
	}
	logger.Info("OpenTelemetry endpoint validation successful")

	return nil
}

// validateStateStore validates the state store connection
func validateStateStore(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stateStore store.StateStore
	var err error

	if cfg.Storage.Type == "redis" && cfg.Storage.Redis != nil {
		stateStore, err = store.NewStateStore(
			cfg.Storage.Type,
			cfg.Storage.Redis.Host,
			cfg.Storage.Redis.Port,
			cfg.Storage.Redis.Password,
			cfg.Storage.Redis.DB,
		)
	} else {
		// Memory store always succeeds
		stateStore, err = store.NewStateStore("memory", "", 0, "", 0)
	}

	if err != nil {
		return fmt.Errorf("failed to create state store: %w", err)
	}
	defer stateStore.Close()

	// Test basic operations
	testState := &store.ChannelState{
		ChannelID:     "test-validation-channel",
		CorrelationID: "test-correlation-id",
		DomainName:    "test.domain",
		InstanceName:  "validation",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Test Set operation
	if err := stateStore.Set(ctx, testState.ChannelID, testState, 10*time.Second); err != nil {
		return fmt.Errorf("failed to set test state: %w", err)
	}

	// Test Get operation
	if _, err := stateStore.Get(ctx, testState.ChannelID); err != nil {
		return fmt.Errorf("failed to get test state: %w", err)
	}

	// Test Delete operation
	if err := stateStore.Delete(ctx, testState.ChannelID); err != nil {
		return fmt.Errorf("failed to delete test state: %w", err)
	}

	return nil
}

// validateOTelEndpoint validates OpenTelemetry endpoint connectivity
func validateOTelEndpoint(cfg *config.Config) error {
	// Create a temporary exporter to test connectivity
	testExporter, err := exporter.NewMetricsExporter(&cfg.OpenTelemetry)
	if err != nil {
		return fmt.Errorf("failed to create test exporter: %w", err)
	}

	// Start the exporter
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := testExporter.Start(ctx); err != nil {
		return fmt.Errorf("failed to start test exporter: %w", err)
	}

	// Stop the exporter
	if err := testExporter.Stop(ctx); err != nil {
		logger.Warn("Failed to stop test exporter cleanly: %v", err)
	}

	return nil
}

// gracefulShutdown performs graceful shutdown of all components
func gracefulShutdown(
	ctx context.Context,
	cancel context.CancelFunc,
	connManager connection.ConnectionManager,
	eventProcessor processor.EventProcessor,
	metricsExporter exporter.MetricsExporter,
	stateStore store.StateStore,
	httpServer *server.HTTPServer,
) {
	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel main context to signal all components
	cancel()

	// Step 1: Stop HTTP server (stop accepting new requests)
	logger.Info("Stopping HTTP server...")
	if err := httpServer.Stop(); err != nil {
		logger.Error("Error stopping HTTP server: %v", err)
	} else {
		logger.Info("HTTP server stopped")
	}

	// Step 2: Stop Connection Manager (stop receiving new events)
	logger.Info("Stopping connection manager...")
	if err := connManager.Stop(); err != nil {
		logger.Error("Error stopping connection manager: %v", err)
	} else {
		logger.Info("Connection manager stopped")
	}

	// Step 3: Stop Event Processor (finish processing queued events)
	logger.Info("Stopping event processor...")
	if err := eventProcessor.Stop(); err != nil {
		logger.Error("Error stopping event processor: %v", err)
	} else {
		logger.Info("Event processor stopped")
	}

	// Step 4: Flush and stop Metrics Exporter
	logger.Info("Flushing and stopping metrics exporter...")
	if err := metricsExporter.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping metrics exporter: %v", err)
	} else {
		logger.Info("Metrics exporter stopped")
	}

	// Step 5: Close State Store
	logger.Info("Closing state store...")
	if err := stateStore.Close(); err != nil {
		logger.Error("Error closing state store: %v", err)
	} else {
		logger.Info("State store closed")
	}

	logger.Info("Graceful shutdown completed")
}
