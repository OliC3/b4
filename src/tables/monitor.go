package tables

import (
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
)

// Monitor watches for removed iptables/nftables rules and restores them
type Monitor struct {
	cfg      *config.Config
	stop     chan struct{}
	wg       sync.WaitGroup
	interval time.Duration
	backend  string
}

// NewMonitor creates a new tables monitor
func NewMonitor(cfg *config.Config) *Monitor {
	interval := time.Duration(cfg.Tables.MonitorInterval) * time.Second
	if interval < time.Second {
		interval = 10 * time.Second // Default fallback
	}

	return &Monitor{
		cfg:      cfg,
		stop:     make(chan struct{}),
		interval: interval,
		backend:  detectFirewallBackend(),
	}
}

func (m *Monitor) Start() {
	if m.cfg.Tables.SkipSetup || m.cfg.Tables.MonitorInterval <= 0 {
		log.Infof("Tables monitor disabled")
		return
	}

	m.wg.Add(1)
	go m.monitorLoop()
	log.Infof("Started tables monitor (backend: %s, interval: %v)", m.backend, m.interval)
}

func (m *Monitor) Stop() {
	if m.cfg.Tables.SkipSetup || m.cfg.Tables.MonitorInterval <= 0 {
		return
	}

	close(m.stop)
	m.wg.Wait()
	log.Infof("Stopped tables monitor")
}

func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			if !m.checkRules() {
				log.Warnf("Tables rules missing, restoring...")
				if err := m.restoreRules(); err != nil {
					log.Errorf("Failed to restore tables rules: %v", err)
				} else {
					log.Infof("Tables rules restored successfully")
				}
			}
		}
	}
}

func (m *Monitor) checkRules() bool {
	if m.backend == "nftables" {
		return m.checkNFTablesRules()
	}
	return m.checkIPTablesRules()
}

func (m *Monitor) checkIPTablesRules() bool {
	ipts := []string{}
	if m.cfg.IPv4Enabled && hasBinary("iptables") {
		ipts = append(ipts, "iptables")
	}
	if m.cfg.IPv6Enabled && hasBinary("ip6tables") {
		ipts = append(ipts, "ip6tables")
	}
	if len(ipts) == 0 {
		return true
	}

	for _, ipt := range ipts {
		if _, err := run(ipt, "-w", "-t", "mangle", "-S", "B4"); err != nil {
			return false
		}

		if _, err := run(ipt, "-w", "-t", "mangle", "-C", "POSTROUTING", "-j", "B4"); err != nil {
			return false
		}
	}

	return true
}

func (m *Monitor) checkNFTablesRules() bool {
	nft := NewNFTablesManager(m.cfg)
	return nft.tableExists()
}

func (m *Monitor) restoreRules() error {
	return AddRules(m.cfg)
}

func (m *Monitor) ForceRestore() error {
	log.Infof("Manual rule restoration triggered")
	return m.restoreRules()
}
