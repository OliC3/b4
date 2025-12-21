package tables

import (
	"bytes"
	"os/exec"
	"strings"
	"sync"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/http/handler"
	"github.com/daniellavrushin/b4/log"
)

var modulesLoaded sync.Once

// AddRulesAuto automatically detects and uses the appropriate firewall backend
func AddRules(cfg *config.Config) error {
	if cfg.System.Tables.SkipSetup {
		return nil
	}

	backend := detectFirewallBackend()
	log.Tracef("Detected firewall backend: %s", backend)
	metrics := handler.GetMetricsCollector()
	metrics.TablesStatus = backend

	if backend == "nftables" {
		nft := NewNFTablesManager(cfg)
		return nft.Apply()
	}

	// Fall back to iptables
	ipt := NewIPTablesManager(cfg)
	return ipt.Apply()
}

// ClearRulesAuto automatically detects and clears the appropriate firewall rules
func ClearRules(cfg *config.Config) error {
	if cfg.System.Tables.SkipSetup {
		return nil
	}

	backend := detectFirewallBackend()

	if backend == "nftables" {
		nft := NewNFTablesManager(cfg)
		return nft.Clear()
	}

	// Fall back to iptables
	ipt := NewIPTablesManager(cfg)
	return ipt.Clear()
}

func run(args ...string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func setSysctlOrProc(name, val string) {
	_, _ = run("sh", "-c", "sysctl -w "+name+"="+val+" || echo "+val+" > /proc/sys/"+strings.ReplaceAll(name, ".", "/"))
}

func getSysctlOrProc(name string) string {
	out, _ := run("sh", "-c", "sysctl -n "+name+" 2>/dev/null || cat /proc/sys/"+strings.ReplaceAll(name, ".", "/"))
	return strings.TrimSpace(out)
}

// detectFirewallBackend determines whether to use iptables or nftables
func detectFirewallBackend() string {
	// First check if nft exists and has rules
	if hasBinary("nft") {
		out, err := run("nft", "list", "tables")
		if err == nil && out != "" {
			// nftables is present and has tables
			return "nftables"
		}
	}

	// Check for iptables
	if hasBinary("iptables") {
		// Check if iptables-legacy or iptables-nft
		out, _ := run("iptables", "--version")
		if strings.Contains(out, "nf_tables") {
			// iptables-nft detected (uses nftables backend)
			return "nftables"
		}
		return "iptables"
	}

	// Default to iptables if nothing found
	return "iptables"
}

func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func loadKernelModules() {
	modulesLoaded.Do(func() {
		_, _ = run("sh", "-c", "modprobe -q nf_conntrack 2>/dev/null || true")
		_, _ = run("sh", "-c", "modprobe -q xt_connbytes 2>/dev/null || true")
		_, _ = run("sh", "-c", "modprobe -q xt_NFQUEUE 2>/dev/null || true")
		_, _ = run("sh", "-c", "modprobe -q nf_tables 2>/dev/null || true")
		_, _ = run("sh", "-c", "modprobe -q nft_queue 2>/dev/null || true")
		_, _ = run("sh", "-c", "modprobe -q nft_ct 2>/dev/null || true")
	})
}
