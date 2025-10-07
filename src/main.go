package main

import (
	"bytes"
	"os"
	"os/signal"
	"syscall"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/iptables"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
)

func main() {
	cfg := config.DefaultConfig
	if _, err := cfg.ParseArgs(os.Args[1:]); err != nil {
		os.Exit(1)
	}
	initLogging(&cfg)
	log.Infof("starting B4...")
	log.Infof("Running with flags: %s", flagsSummary(os.Args[1:]))

	if !cfg.SkipIpTables {
		iptables.ClearRules(&cfg)
		if err := iptables.AddRules(&cfg); err != nil {
			log.Errorf("failed to add iptables rules: %v", err)
			os.Exit(1)
		}
	}

	pool := nfq.NewPool(uint16(cfg.QueueStartNum), cfg.Threads, &cfg)
	if err := pool.Start(); err != nil {
		log.Errorf("nfqueue start failed: %v", err)
		os.Exit(1)
	}
	defer pool.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	if !cfg.SkipIpTables {
		if err := iptables.ClearRules(&cfg); err != nil {
			log.Errorf("failed to clear iptables rules: %v", err)
		}
	}
	log.Infof("bye")
}

func initLogging(cfg *config.Config) error {
	log.Init(os.Stderr, log.Level(cfg.Logging.Level), cfg.Logging.Instaflush)
	if cfg.Logging.Syslog {
		if err := log.EnableSyslog("b4"); err != nil {
			log.Errorf("syslog enable failed: %v", err)
		}
	}
	return nil
}

func flagsSummary(args []string) string {
	var buf bytes.Buffer
	for i, arg := range args {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(arg)
	}
	return buf.String()
}
