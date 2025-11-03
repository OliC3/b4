package tables

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/http/handler"
	"github.com/daniellavrushin/b4/log"
)

// AddRulesAuto automatically detects and uses the appropriate firewall backend
func AddRules(cfg *config.Config) error {
	if cfg.Tables.SkipSetup {
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
	if cfg.Tables.SkipSetup {
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
	// Common netfilter modules
	_, _ = run("sh", "-c", "modprobe nf_conntrack >/dev/null 2>&1 || true")

	// iptables specific
	_, _ = run("sh", "-c", "modprobe xt_connbytes --first-time >/dev/null 2>&1 || true")
	_, _ = run("sh", "-c", "modprobe xt_NFQUEUE --first-time >/dev/null 2>&1 || true")

	// nftables specific (will fail silently if not needed)
	_, _ = run("sh", "-c", "modprobe nf_tables >/dev/null 2>&1 || true")
	_, _ = run("sh", "-c", "modprobe nft_queue >/dev/null 2>&1 || true")
	_, _ = run("sh", "-c", "modprobe nft_ct >/dev/null 2>&1 || true")
}
