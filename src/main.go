// src/main.go
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/daniellavrushin/b4/config"
	b4http "github.com/daniellavrushin/b4/http"
	"github.com/daniellavrushin/b4/http/handler"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
	"github.com/daniellavrushin/b4/tables"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	cfg         = config.DefaultConfig
	verboseFlag string
	showVersion bool
	clearTables bool
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
	rootCmd.Flags().BoolVar(&clearTables, "clear-tables", false, "Perform only iptables/nftables cleanup and exit")
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

	if clearTables {
		// Initialize logging first thing
		if err := initLogging(&cfg); err != nil {
			return fmt.Errorf("logging initialization failed: %w", err)
		}

		log.Infof("Clearing iptables rules as requested (--clear-iptables)")
		tables.ClearRules(&cfg)
		log.Infof("IPTables rules cleared successfully")
		return nil
	}

	log.Infof("Starting B4 packet processor")

	cfg.LoadWithMigration(cfg.ConfigPath)
	cfg.SaveToFile(cfg.ConfigPath)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return log.Errorf("invalid configuration: %w", err)
	}

	printConfigDefaults(cmd)

	// Initialize metrics collector early
	metrics := handler.GetMetricsCollector()
	metrics.RecordEvent("info", "B4 starting up")

	if cfg.System.WebServer.Port > 0 {
		metrics.RecordEvent("info", fmt.Sprintf("Web server started on port %d", cfg.System.WebServer.Port))
	}

	// Load domains
	_, totalDomains, err := cfg.LoadDomains()
	if err != nil {
		metrics.RecordEvent("error", fmt.Sprintf("Failed to load domains: %v", err))
		return fmt.Errorf("failed to load domains: %w", err)
	}

	if totalDomains == 0 {
		log.Warnf("No domains configured in any set, B4 will not process any traffic")
	} else {
		for _, set := range cfg.Sets {
			log.Infof("Set '%s': %d domains", set.Name, len(set.Domains.DomainsToMatch))
		}
		log.Infof("Total: %d domains across %d sets", totalDomains, len(cfg.Sets))
	}

	// Setup iptables/nftables rules
	if !cfg.System.Tables.SkipSetup {
		log.Tracef("Clearing existing iptables/nftables rules")
		tables.ClearRules(&cfg)

		log.Tracef("Adding tables rules")
		if err := tables.AddRules(&cfg); err != nil {
			metrics.RecordEvent("error", fmt.Sprintf("Failed to add tables rules: %v", err))
			return fmt.Errorf("failed to add tables rules: %w", err)
		}
		metrics.RecordEvent("info", "Tables rules configured successfully")
	} else {
		log.Infof("Skipping tables setup (--skip-tables)")
		metrics.TablesStatus = "skipped"
	}

	// Start netfilter queue pool
	log.Infof("Starting netfilter queue pool (queue: %d, threads: %d)", cfg.Queue.StartNum, cfg.Queue.Threads)
	pool := nfq.NewPool(&cfg)
	if err := pool.Start(); err != nil {
		metrics.RecordEvent("error", fmt.Sprintf("NFQueue start failed: %v", err))
		metrics.NFQueueStatus = "error"
		return fmt.Errorf("netfilter queue start failed: %w", err)
	}

	metrics.RecordEvent("info", fmt.Sprintf("NFQueue started with %d threads", cfg.Queue.Threads))
	metrics.NFQueueStatus = "active"

	// Start tables monitor to handle rule restoration if system wipes them
	var tablesMonitor *tables.Monitor
	if !cfg.System.Tables.SkipSetup && cfg.System.Tables.MonitorInterval > 0 {
		tablesMonitor = tables.NewMonitor(&cfg)
		tablesMonitor.Start()
	}

	// Start internal web server if configured
	httpServer, err := b4http.StartServer(&cfg, pool)
	if err != nil {
		metrics.RecordEvent("error", fmt.Sprintf("Failed to start web server: %v", err))
		return log.Errorf("failed to start web server: %w", err)
	}

	log.Infof("B4 is running. Press Ctrl+C to stop")
	metrics.RecordEvent("info", "B4 is fully operational")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	log.Infof("Received signal: %v, shutting down gracefully", sig)
	metrics.RecordEvent("info", fmt.Sprintf("Shutdown initiated by signal: %v", sig))

	// Perform graceful shutdown with timeout
	return gracefulShutdown(&cfg, pool, httpServer, metrics)
}

func gracefulShutdown(cfg *config.Config, pool *nfq.Pool, httpServer *http.Server, metrics *handler.MetricsCollector) error {
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create wait group for parallel shutdown
	var wg sync.WaitGroup
	shutdownErrors := make(chan error, 3)

	// Shutdown WebSocket connections
	log.Infof("Shutting down WebSocket connections...")
	b4http.Shutdown()

	// Shutdown HTTP server
	if httpServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Infof("Shutting down HTTP server...")
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				log.Errorf("HTTP server shutdown error: %v", err)
				shutdownErrors <- fmt.Errorf("HTTP shutdown: %w", err)
			} else {
				log.Infof("HTTP server stopped")
			}
		}()
	}

	// Stop NFQueue pool
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Infof("Stopping netfilter queue pool...")
		metrics.NFQueueStatus = "stopping"

		// Use a goroutine with timeout for pool.Stop()
		stopDone := make(chan struct{})
		go func() {
			pool.Stop()
			close(stopDone)
		}()

		select {
		case <-stopDone:
			log.Infof("Netfilter queue pool stopped")
		case <-shutdownCtx.Done():
			log.Errorf("Netfilter queue pool stop timed out")
			shutdownErrors <- fmt.Errorf("NFQueue stop timeout")
		}
	}()

	// Clean up iptables/nftables rules
	if !cfg.System.Tables.SkipSetup {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Infof("Clearing iptables/nftables rules...")
			if err := tables.ClearRules(cfg); err != nil {
				log.Errorf("Failed to clear tables rules: %v", err)
				metrics.RecordEvent("error", fmt.Sprintf("Failed to clear tables rules: %v", err))
				shutdownErrors <- fmt.Errorf("tables cleanup: %w", err)
			} else {
				log.Infof("Tables rules cleared")
				metrics.TablesStatus = "inactive"
			}
		}()
	}

	// Wait for all shutdown tasks or timeout
	shutdownDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		// All tasks completed
		close(shutdownErrors)

		// Check for any errors
		var errs []error
		for err := range shutdownErrors {
			errs = append(errs, err)
		}

		if len(errs) > 0 {
			log.Errorf("Shutdown completed with %d errors", len(errs))
			for _, err := range errs {
				log.Errorf("  - %v", err)
			}
			metrics.RecordEvent("warning", fmt.Sprintf("B4 shutdown with %d errors", len(errs)))
		} else {
			log.Infof("B4 stopped successfully")
			metrics.RecordEvent("info", "B4 shutdown complete")
		}

	case <-shutdownCtx.Done():
		log.Errorf("Shutdown timeout reached, forcing exit")
		metrics.RecordEvent("error", "Forced shutdown due to timeout")

		log.Flush()
		time.Sleep(100 * time.Millisecond)

		os.Exit(1)
	}

	log.Flush()
	return nil
}

func initLogging(cfg *config.Config) error {
	w := io.MultiWriter(os.Stderr, b4http.LogWriter())
	log.Init(w, log.Level(cfg.System.Logging.Level), cfg.System.Logging.Instaflush)

	fmt.Fprintf(os.Stderr, "[INIT] Logging initialized at level %d\n", cfg.System.Logging.Level)

	if cfg.System.Logging.Syslog {
		if err := log.EnableSyslog("b4"); err != nil {
			log.Errorf("Failed to enable syslog: %v", err)
			return err
		}
		log.Infof("Syslog enabled")
	}

	return nil
}

func printConfigDefaults(cmd *cobra.Command) {
	var all []*pflag.Flag
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) { all = append(all, f) })
	cmd.Flags().VisitAll(func(f *pflag.Flag) { all = append(all, f) })
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })

	log.Infof("Effective CLI flags:")
	line := ""
	for _, f := range all {
		if line == "" {
			line = fmt.Sprintf("--%s=%s", f.Name, f.Value.String())
		} else {
			line += " " + fmt.Sprintf("--%s=%s", f.Name, f.Value.String())
		}
	}
	log.Infof("  %s", line)
}
