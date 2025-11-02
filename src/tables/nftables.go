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
	if n.cfg.Threads > 1 {
		start := n.cfg.QueueStartNum
		end := n.cfg.QueueStartNum + n.cfg.Threads - 1
		return fmt.Sprintf("queue num %d-%d bypass", start, end)
	}
	return fmt.Sprintf("queue num %d bypass", n.cfg.QueueStartNum)
}

func (n *NFTablesManager) Apply() error {
	if !hasBinary("nft") {
		return fmt.Errorf("nft binary not found")
	}

	log.Tracef("NFTABLES: adding rules")
	loadKernelModules()

	// Create table
	if err := n.createTable(); err != nil {
		return err
	}

	// Create base chains that hook into netfilter
	// Use mangle - 1 priority (typically 149) to match youtubeUnblock
	if err := n.createChain("postrouting", "postrouting", 149, "accept"); err != nil {
		return err
	}

	if err := n.createChain("output", "output", 149, "accept"); err != nil {
		return err
	}

	if err := n.createChain(nftChainName, "", 0, ""); err != nil {
		return err
	}

	markAccept := n.cfg.Mark

	// Add rules
	if err := n.addRuleArgs("postrouting", "jump", nftChainName); err != nil {
		return err
	}

	if err := n.addRuleArgs("output", "meta", "mark", fmt.Sprintf("%d", markAccept), "accept"); err != nil {
		return err
	}

	if err := n.addRuleArgs(nftChainName, "meta", "mark", fmt.Sprintf("%d", markAccept), "return"); err != nil {
		return err
	}

	// TCP rule - match youtubeUnblock exactly
	// Note: youtubeUnblock uses < 20 for nftables but 0:19 for iptables
	tcpRuleArgs := []string{"tcp", "dport", "443", "ct", "original", "packets", "<", "20", "counter"}
	tcpRuleArgs = append(tcpRuleArgs, strings.Fields(n.buildNFQueueAction())...)
	if err := n.addRuleArgs(nftChainName, tcpRuleArgs...); err != nil {
		return err
	}

	// UDP rule - ALL UDP TRAFFIC, not just port 443!
	// This is critical - youtubeUnblock doesn't filter UDP by port
	udpRuleArgs := []string{"meta", "l4proto", "udp", "ct", "original", "packets", "<", "9", "counter"}
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

// Add this helper method
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
