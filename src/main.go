// src/main.go
package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/daniellavrushin/b4/config"
	b4http "github.com/daniellavrushin/b4/http"
	"github.com/daniellavrushin/b4/http/handler"
	"github.com/daniellavrushin/b4/iptables"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
	"github.com/spf13/cobra"
)

var (
	cfg         = config.DefaultConfig
	configPath  string
	verboseFlag string
	showVersion bool
	Version     = "dev"
	Commit      = "none"
	Date        = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "b4",
	Short: "B4 network packet processor",
	Long:  `B4 is a netfilter queue based packet processor for DPI circumvention`,
	RunE:  runB4,
}

func init() {
	// Bind all configuration flags
	cfg.BindFlags(rootCmd)

	// Add verbosity flags separately since they need special handling
	rootCmd.Flags().StringVar(&verboseFlag, "verbose", "info", "Set verbosity level (debug, trace, info, silent), default: info")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version and exit")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", configPath, "Path to a configuration file to save and load")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runB4(cmd *cobra.Command, args []string) error {
	if showVersion {
		fmt.Printf("B4 version: %s (%s) %s\n", Version, Commit, Date)
		return nil
	}

	cfg.ApplyLogLevel(verboseFlag)

	// Initialize logging first thing
	if err := initLogging(&cfg); err != nil {
		return fmt.Errorf("logging initialization failed: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return log.Errorf("invalid configuration: %w", err)
	}

	log.Infof("Starting B4 packet processor")

	cfg.LoadFromFile(configPath)
	cfg.SaveToFile(configPath)
	printConfigDefaults(&cfg)

	// Initialize metrics collector early
	metrics := handler.GetMetricsCollector()
	metrics.RecordEvent("info", "B4 starting up")

	// Start internal web server if configured
	if _, err := b4http.StartServer(&cfg); err != nil {
		metrics.RecordEvent("error", fmt.Sprintf("Failed to start web server: %v", err))
		return log.Errorf("failed to start web server: %w", err)
	}

	if cfg.WebServer.Port > 0 {
		metrics.RecordEvent("info", fmt.Sprintf("Web server started on port %d", cfg.WebServer.Port))
	}

	// Load domains from geodata if specified
	if cfg.GeoSitePath != "" && len(cfg.GeoCategories) > 0 {
		log.Infof("Loading domains from geodata for categories: %v", cfg.GeoCategories)
		domains, err := cfg.LoadDomainsFromGeodata()
		if err != nil {
			metrics.RecordEvent("error", fmt.Sprintf("Failed to load geodata: %v", err))
			return fmt.Errorf("failed to load geodata domains: %w", err)
		}
		log.Infof("Loaded %d domains from geodata", len(domains))
		metrics.RecordEvent("info", fmt.Sprintf("Loaded %d domains from geodata", len(domains)))

		// Merge with existing SNI domains
		cfg.SNIDomains = append(cfg.SNIDomains, domains...)
	}

	if len(cfg.SNIDomains) > 0 {
		log.Infof("Total SNI domains to match: %d", len(cfg.SNIDomains))
	}

	// Setup iptables rules
	if !cfg.SkipIpTables {
		log.Infof("Clearing existing iptables rules")
		iptables.ClearRules(&cfg)

		log.Infof("Adding iptables rules")
		if err := iptables.AddRules(&cfg); err != nil {
			metrics.RecordEvent("error", fmt.Sprintf("Failed to add iptables rules: %v", err))
			return fmt.Errorf("failed to add iptables rules: %w", err)
		}
		metrics.RecordEvent("info", "IPTables rules configured successfully")
		metrics.IPTablesStatus = "active"
	} else {
		log.Infof("Skipping iptables setup (--skip-iptables)")
		metrics.IPTablesStatus = "skipped"
	}

	// Start netfilter queue pool
	log.Infof("Starting netfilter queue pool (queue: %d, threads: %d)", cfg.QueueStartNum, cfg.Threads)
	pool := nfq.NewPool(uint16(cfg.QueueStartNum), cfg.Threads, &cfg)
	if err := pool.Start(); err != nil {
		metrics.RecordEvent("error", fmt.Sprintf("NFQueue start failed: %v", err))
		metrics.NFQueueStatus = "error"
		return fmt.Errorf("netfilter queue start failed: %w", err)
	}

	metrics.RecordEvent("info", fmt.Sprintf("NFQueue started with %d threads", cfg.Threads))
	metrics.NFQueueStatus = "active"

	// Initialize worker status
	workers := make([]handler.WorkerHealth, cfg.Threads)
	for i := 0; i < cfg.Threads; i++ {
		workers[i] = handler.WorkerHealth{
			ID:        i,
			Status:    "active",
			Processed: 0,
		}
	}
	metrics.UpdateWorkerStatus(workers)

	log.Infof("B4 is running. Press Ctrl+C to stop")
	metrics.RecordEvent("info", "B4 is fully operational")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Infof("Received signal: %v, shutting down gracefully", sig)
	metrics.RecordEvent("info", fmt.Sprintf("Shutdown initiated by signal: %v", sig))
	metrics.NFQueueStatus = "stopping"

	pool.Stop()

	// Cleanup iptables rules
	if !cfg.SkipIpTables {
		log.Infof("Clearing iptables rules")
		if err := iptables.ClearRules(&cfg); err != nil {
			log.Errorf("Failed to clear iptables rules: %v", err)
			metrics.RecordEvent("error", fmt.Sprintf("Failed to clear iptables rules: %v", err))
		} else {
			metrics.IPTablesStatus = "inactive"
		}
	}

	log.Infof("B4 stopped successfully")
	metrics.RecordEvent("info", "B4 shutdown complete")
	return nil
}

func initLogging(cfg *config.Config) error {
	// Ensure logging is initialized with stderr output and WebSocket broadcast
	w := io.MultiWriter(os.Stderr, b4http.LogWriter())
	log.Init(w, log.Level(cfg.Logging.Level), cfg.Logging.Instaflush)

	// Log that initialization happened
	fmt.Fprintf(os.Stderr, "[INIT] Logging initialized at level %d\n", cfg.Logging.Level)

	if cfg.Logging.Syslog {
		if err := log.EnableSyslog("b4"); err != nil {
			log.Errorf("Failed to enable syslog: %v", err)
			return err
		}
		log.Infof("Syslog enabled")
	}

	return nil
}

func printConfigDefaults(cfg *config.Config) {
	log.Debugf("Configuration:")
	log.Debugf("  Queue number: %d", cfg.QueueStartNum)
	log.Debugf("  Threads: %d", cfg.Threads)
	log.Debugf("  Mark: %d (0x%x)", cfg.Mark, cfg.Mark)
	log.Debugf("  ConnBytes limit: %d", cfg.ConnBytesLimit)
	log.Debugf("  GSO: %v", cfg.UseGSO)
	log.Debugf("  Conntrack: %v", cfg.UseConntrack)
	log.Debugf("  Skip iptables: %v", cfg.SkipIpTables)
	if cfg.GeoSitePath != "" {
		log.Debugf("  Geo Site path: %s", cfg.GeoSitePath)
	}
	if cfg.GeoIpPath != "" {
		log.Debugf("  Geo IP path: %s", cfg.GeoIpPath)
	}
	if len(cfg.GeoCategories) > 0 {
		log.Debugf("  Geo Categories: %v", cfg.GeoCategories)
	}
	if len(cfg.SNIDomains) > 0 {
		log.Debugf("  SNI Domains: %v", cfg.SNIDomains)
	}
	log.Debugf("  Logging level: %d", cfg.Logging.Level)
	log.Debugf("  Logging instaflush: %v", cfg.Logging.Instaflush)
	log.Debugf("  Logging syslog: %v", cfg.Logging.Syslog)

	log.Debugf("  Fragment Strategy: %s", cfg.FragmentStrategy)
	log.Debugf("  Fragment SNI Reverse: %v", cfg.FragSNIReverse)
	log.Debugf("  Fragment Middle SNI: %v", cfg.FragMiddleSNI)
	log.Debugf("  Fragment SNI Position: %d", cfg.FragSNIPosition)

	log.Debugf("  Fake SNI: %v", cfg.FakeSNI)
	log.Debugf("    Fake TTL: %d", cfg.FakeTTL)
	log.Debugf("    Fake Strategy: %s", cfg.FakeStrategy)
	log.Debugf("    Fake Seq Offset: %d", cfg.FakeSeqOffset)
	log.Debugf("    Fake SNI Type: %d", cfg.FakeSNIType)
	log.Debugf("    Fake Custom Payload: %s", cfg.FakeCustomPayload)

	log.Debugf("  UDP Mode: %s", cfg.UDPMode)
	log.Debugf("    UDP Fake Len: %d", cfg.UDPFakeLen)
	log.Debugf("    UDP Fake Seq Length: %d", cfg.UDPFakeSeqLength)
	log.Debugf("    UDP Faking Strategy: %s", cfg.UDPFakingStrategy)
	log.Debugf("    UDP DPort Min: %d", cfg.UDPDPortMin)
	log.Debugf("    UDP DPort Max: %d", cfg.UDPDPortMax)
	log.Debugf("    UDP Filter QUIC: %s", cfg.UDPFilterQUIC)

	log.Debugf("  Web Server:")
	log.Debugf("    Port: %d", cfg.WebServer.Port)
}
