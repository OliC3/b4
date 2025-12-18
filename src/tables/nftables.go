package tables

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
)

const (
	nftTableName = "b4_mangle"
	nftChainName = "b4_chain"
)

// NFTablesManager handles nftables operations
type NFTablesManager struct {
	cfg *config.Config
}

// NewNFTablesManager creates a new nftables manager
func NewNFTablesManager(cfg *config.Config) *NFTablesManager {
	return &NFTablesManager{cfg: cfg}
}

// runNft executes nft command
func (n *NFTablesManager) runNft(args ...string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("nft", args...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// tableExists checks if the b4 table exists
func (n *NFTablesManager) tableExists() bool {
	out, err := n.runNft("list", "tables")
	if err != nil {
		return false
	}
	return strings.Contains(out, nftTableName)
}

// chainExists checks if a chain exists in the table
func (n *NFTablesManager) chainExists(chain string) bool {
	_, err := n.runNft("list", "chain", "inet", nftTableName, chain)
	return err == nil
}

// createTable creates the b4 table if it doesn't exist
func (n *NFTablesManager) createTable() error {
	if n.tableExists() {
		return nil
	}
	_, err := n.runNft("add", "table", "inet", nftTableName)
	if err != nil {
		return fmt.Errorf("failed to create nftables table: %w", err)
	}
	log.Tracef("Created nftables table: %s", nftTableName)
	return nil
}

// createChain creates a chain if it doesn't exist
func (n *NFTablesManager) createChain(chain string, hook string, priority int, policy string) error {
	if n.chainExists(chain) {
		return nil
	}

	var cmd []string
	if hook != "" {
		// Base chain with netfilter hook
		cmd = []string{"add", "chain", "inet", nftTableName, chain,
			fmt.Sprintf("{ type filter hook %s priority %d ; policy %s ; }", hook, priority, policy)}
	} else {
		// Regular chain
		cmd = []string{"add", "chain", "inet", nftTableName, chain}
	}

	_, err := n.runNft(cmd...)
	if err != nil {
		return fmt.Errorf("failed to create chain %s: %w", chain, err)
	}
	log.Tracef("Created nftables chain: %s", chain)
	return nil
}

// buildNFQueueAction builds the nfqueue action string
func (n *NFTablesManager) buildNFQueueAction() string {
	if n.cfg.Queue.Threads > 1 {
		start := n.cfg.Queue.StartNum
		end := n.cfg.Queue.StartNum + n.cfg.Queue.Threads - 1
		return fmt.Sprintf("queue num %d-%d bypass", start, end)
	}
	return fmt.Sprintf("queue num %d bypass", n.cfg.Queue.StartNum)
}

func (n *NFTablesManager) Apply() error {
	cfg := n.cfg
	if !hasBinary("nft") {
		return fmt.Errorf("nft binary not found")
	}

	log.Tracef("NFTABLES: adding rules")
	loadKernelModules()

	// Create table
	if err := n.createTable(); err != nil {
		return err
	}

	if err := n.createChain("output", "output", 149, "accept"); err != nil {
		return err
	}

	if err := n.createChain("prerouting", "prerouting", -150, "accept"); err != nil {
		return err
	}

	if err := n.createChain(nftChainName, "", 0, ""); err != nil {
		return err
	}

	markAccept := fmt.Sprintf("0x%x", n.cfg.Queue.Mark)

	if cfg.Queue.Devices.Enabled && len(cfg.Queue.Devices.Mac) > 0 {
		// Create forward chain
		if err := n.createChain("forward", "forward", 149, "accept"); err != nil {
			return err
		}

		if cfg.Queue.Devices.WhiteIsBlack {
			for _, mac := range cfg.Queue.Devices.Mac {
				mac = strings.ToUpper(strings.TrimSpace(mac))
				if mac == "" {
					continue
				}
				if err := n.addRuleArgs("forward", "ether", "saddr", mac, "return"); err != nil {
					return err
				}
			}
			if err := n.addRuleArgs("forward", "jump", nftChainName); err != nil {
				return err
			}
		} else {
			for _, mac := range cfg.Queue.Devices.Mac {
				mac = strings.ToUpper(strings.TrimSpace(mac))
				if mac == "" {
					continue
				}
				if err := n.addRuleArgs("forward", "ether", "saddr", mac, "jump", nftChainName); err != nil {
					return err
				}
			}
		}
	} else {
		if err := n.createChain("postrouting", "postrouting", 149, "accept"); err != nil {
			return err
		}
		if err := n.addRuleArgs("postrouting", "jump", nftChainName); err != nil {
			return err
		}
	}

	if err := n.addRuleArgs("output", "meta", "mark", markAccept, "accept"); err != nil {
		return err
	}

	if err := n.addRuleArgs(nftChainName, "meta", "mark", markAccept, "return"); err != nil {
		return err
	}

	tcpLimit := fmt.Sprintf("%d", cfg.MainSet.TCP.ConnBytesLimit+1)
	udpLimit := fmt.Sprintf("%d", cfg.MainSet.UDP.ConnBytesLimit+1)

	tcpRuleArgs := []string{"tcp", "dport", "443", "ct", "original", "packets", "<", tcpLimit, "counter"}
	tcpRuleArgs = append(tcpRuleArgs, strings.Fields(n.buildNFQueueAction())...)
	if err := n.addRuleArgs(nftChainName, tcpRuleArgs...); err != nil {
		return err
	}

	dnsRuleArgs := []string{"udp", "dport", "53", "counter"}
	dnsRuleArgs = append(dnsRuleArgs, strings.Fields(n.buildNFQueueAction())...)
	if err := n.addRuleArgs(nftChainName, dnsRuleArgs...); err != nil {
		return err
	}

	dnsResponseArgs := []string{"udp", "sport", "53", "counter"}
	dnsResponseArgs = append(dnsResponseArgs, strings.Fields(n.buildNFQueueAction())...)
	if err := n.addRuleArgs("prerouting", dnsResponseArgs...); err != nil {
		return err
	}

	udpRuleArgs := []string{"meta", "l4proto", "udp", "ct", "original", "packets", "<", udpLimit, "counter"}
	udpRuleArgs = append(udpRuleArgs, strings.Fields(n.buildNFQueueAction())...)
	if err := n.addRuleArgs(nftChainName, udpRuleArgs...); err != nil {
		return err
	}

	// Set sysctls
	setSysctlOrProc("net.netfilter.nf_conntrack_checksum", "0")
	setSysctlOrProc("net.netfilter.nf_conntrack_tcp_be_liberal", "1")

	if log.Level(log.CurLevel.Load()) >= log.LevelTrace {
		nftables_trace, _ := n.runNft("list", "table", "inet", nftTableName)
		log.Tracef("Current nftables rules:\n%s", nftables_trace)
	}

	return nil
}

func (n *NFTablesManager) addRuleArgs(chain string, args ...string) error {
	cmd := append([]string{"add", "rule", "inet", nftTableName, chain}, args...)
	_, err := n.runNft(cmd...)
	if err != nil {
		return fmt.Errorf("failed to add rule: %w", err)
	}
	return nil
}

// Clear removes all nftables rules and tables
func (n *NFTablesManager) Clear() error {
	if !hasBinary("nft") {
		return nil
	}

	log.Tracef("NFTABLES: clearing rules")

	// Flush and delete the table (this removes all chains and rules)
	if n.tableExists() {
		// First flush the table
		if _, err := n.runNft("flush", "table", "inet", nftTableName); err != nil {
			log.Errorf("Failed to flush nftables table: %v", err)
		}

		time.Sleep(30 * time.Millisecond)

		// Then delete the table
		if _, err := n.runNft("delete", "table", "inet", nftTableName); err != nil {
			log.Errorf("Failed to delete nftables table: %v", err)
		}
	}

	return nil
}
