package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/validaoxyz/hyperliquid-exporter/internal/alerters"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/validaoxyz/hyperliquid-exporter/internal/config"
	"github.com/validaoxyz/hyperliquid-exporter/internal/exporter"
	"github.com/validaoxyz/hyperliquid-exporter/internal/logger"
	"github.com/validaoxyz/hyperliquid-exporter/internal/metrics"
	"github.com/validaoxyz/hyperliquid-exporter/internal/monitors"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: hl_exporter <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  start    Start the Hyperliquid exporter")
		fmt.Println("\nOptions:")
		fmt.Println("  --log-level        Set the logging level (default: \"debug\")")
		fmt.Println("  --enable-prom      Enable Prometheus endpoint (default: true)")
		fmt.Println("  --disable-prom     Disable Prometheus endpoint")
		fmt.Println("  --enable-otlp      Enable OTLP export (default: false)")
		fmt.Println("  --otlp-endpoint    OTLP endpoint (default: otel.hyperliquid.validao.xyz)")
		fmt.Println("  --node-home        Node home directory (overrides env var)")
		fmt.Println("  --node-binary      Node binary path (overrides env var)")
		fmt.Println("  --alias            Node alias (required when OTLP is enabled)")
		fmt.Println("  --chain            Chain type (required when OTLP is enabled: 'mainnet' or 'testnet')")
		fmt.Println("  --otlp-insecure    Use insecure connection for OTLP (default: false)")
		fmt.Println("  --alert-config     Path to TOML file for alerts config (default: alerts disabled)")
		os.Exit(1)
	}

	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	logLevel := startCmd.String("log-level", "info", "Log level (debug, info, warning, error)")
	enableProm := startCmd.Bool("enable-prom", true, "Enable Prometheus endpoint (default: true)")
	disableProm := startCmd.Bool("disable-prom", false, "Disable Prometheus endpoint")
	enableOTLP := startCmd.Bool("enable-otlp", false, "Enable OTLP export")
	otlpEndpoint := startCmd.String("otlp-endpoint", "otel.hyperliquid.validao.xyz", "OTLP endpoint (default: otel.hyperliquid.validao.xyz)")
	nodeHome := startCmd.String("node-home", "", "Node home directory (overrides env var)")
	nodeBinary := startCmd.String("node-binary", "", "Node binary path (overrides env var)")
	alias := startCmd.String("alias", "", "Node alias (required when OTLP is enabled)")
	chain := startCmd.String("chain", "", "Chain type (required when OTLP is enabled: 'mainnet' or 'testnet')")
	otlpInsecure := startCmd.Bool("otlp-insecure", false, "Use insecure connection for OTLP (default: false)")
	alertConfigPath := startCmd.String("alert-config", "", "Path to TOML file for alerts config (default: alerts disabled)")
	switch os.Args[1] {
	case "start":
		startCmd.Parse(os.Args[2:])
	default:
		fmt.Printf("%q is not a valid command.\n", os.Args[1])
		os.Exit(1)
	}

	if err := logger.SetLogLevel(*logLevel); err != nil {
		fmt.Printf("Error setting log level: %v\n", err)
		os.Exit(1)
	}

	if *chain != "" {
		*chain = strings.ToLower(*chain)
		if *chain != "mainnet" && *chain != "testnet" {
			logger.Error("--chain flag must be either 'mainnet' or 'testnet' (case insensitive)")
			os.Exit(1)
		}
		*chain = strings.ToLower(*chain)
	}

	flags := &config.Flags{
		NodeHome:   *nodeHome,
		NodeBinary: *nodeBinary,
		Chain:      *chain,
	}

	cfg := config.LoadConfig(flags)

	if *enableOTLP {
		if *alias == "" {
			logger.Error("--alias flag is required when OTLP is enabled. This can be whatever you choose and is just an identifier for your node.")
			os.Exit(1)
		}
		if *chain == "" {
			logger.Error("--chain flag is required when OTLP is enabled")
			os.Exit(1)
		}
	}
	var alertConfig alerters.AlerterConfig
	if *alertConfigPath != "" {
		if _, err := toml.DecodeFile(*alertConfigPath, &alertConfig); err != nil {
			log.Fatalf("Error loading configuration from: %s error: %s\n", *alertConfigPath, err)
		}

	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// After loading config, before metrics initialization
	validatorAddress, isValidator := monitors.GetValidatorStatus(cfg.NodeHome)

	// Initialize metrics configuration
	metricsConfig := metrics.MetricsConfig{
		EnablePrometheus: !*disableProm && *enableProm,
		EnableOTLP:       *enableOTLP,
		OTLPEndpoint:     *otlpEndpoint,
		OTLPInsecure:     *otlpInsecure,
		Alias:            *alias,
		Chain:            *chain,
		NodeHome:         cfg.NodeHome,
		ValidatorAddress: validatorAddress,
		IsValidator:      isValidator,
	}

	if err := metrics.InitMetrics(ctx, metricsConfig); err != nil {
		logger.Error("Failed to initialize metrics: %v", err)
		os.Exit(1)
	}

	exporter.Start(ctx, cfg)

	<-ctx.Done()
	logger.Info("Shutting down gracefully")
}
