package tables

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
)

type IPTablesManager struct {
	cfg *config.Config
}

func NewIPTablesManager(cfg *config.Config) *IPTablesManager {
	return &IPTablesManager{cfg: cfg}
}

func (im *IPTablesManager) existsChain(ipt, table, chain string) bool {
	_, err := run(ipt, "-w", "-t", table, "-S", chain)
	return err == nil
}

func (im *IPTablesManager) ensureChain(ipt, table, chain string) {
	if !im.existsChain(ipt, table, chain) {
		_, _ = run(ipt, "-w", "-t", table, "-N", chain)
	}
}

func (im *IPTablesManager) existsRule(ipt, table, chain string, spec []string) bool {
	_, err := run(append([]string{ipt, "-w", "-t", table, "-C", chain}, spec...)...)
	return err == nil
}

func (im *IPTablesManager) delAll(ipt, table, chain string, spec []string) {
	for {
		_, err := run(append([]string{ipt, "-w", "-t", table, "-D", chain}, spec...)...)
		if err != nil {
			break
		}
	}
}

type Rule struct {
	manager *IPTablesManager
	IPT     string
	Table   string
	Chain   string
	Spec    []string
	Action  string
}

func (r Rule) Apply() error {
	if r.manager.existsRule(r.IPT, r.Table, r.Chain, r.Spec) {
		return nil
	}
	op := "-A"
	if strings.ToUpper(r.Action) == "I" {
		op = "-I"
	}
	_, err := run(append([]string{r.IPT, "-w", "-t", r.Table, op, r.Chain}, r.Spec...)...)
	return err
}

func (r Rule) Remove() {
	r.manager.delAll(r.IPT, r.Table, r.Chain, r.Spec)
}

type Chain struct {
	manager *IPTablesManager
	IPT     string
	Table   string
	Name    string
}

func (c Chain) Ensure() {
	c.manager.ensureChain(c.IPT, c.Table, c.Name)
}

func (c Chain) Remove() {
	if c.manager.existsChain(c.IPT, c.Table, c.Name) {
		_, _ = run(c.IPT, "-w", "-t", c.Table, "-F", c.Name)
		_, _ = run(c.IPT, "-w", "-t", c.Table, "-X", c.Name)
	}
}

type SysctlSetting struct {
	Name    string
	Desired string
	Revert  string
}

var sysctlSnapPath = "/tmp/b4_sysctl_snapshot.json"

func loadSysctlSnapshot() map[string]string {
	b, err := os.ReadFile(sysctlSnapPath)
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if json.Unmarshal(b, &m) != nil {
		return map[string]string{}
	}
	return m
}

func saveSysctlSnapshot(m map[string]string) {
	b, _ := json.Marshal(m)
	_ = os.WriteFile(sysctlSnapPath, b, 0600)
}

func (ipt *IPTablesManager) buildNFQSpec(queueStart, threads int) []string {
	if threads > 1 {
		start := strconv.Itoa(queueStart)
		end := strconv.Itoa(queueStart + threads - 1)
		return []string{"-j", "NFQUEUE", "--queue-balance", start + ":" + end, "--queue-bypass"}
	}
	return []string{"-j", "NFQUEUE", "--queue-num", strconv.Itoa(queueStart), "--queue-bypass"}
}

func (s SysctlSetting) Apply() {
	snap := loadSysctlSnapshot()
	if _, ok := snap[s.Name]; !ok {
		snap[s.Name] = getSysctlOrProc(s.Name)
		saveSysctlSnapshot(snap)
	}
	setSysctlOrProc(s.Name, s.Desired)
}

func (s SysctlSetting) RevertBack() {
	snap := loadSysctlSnapshot()
	if v, ok := snap[s.Name]; ok && v != "" {
		setSysctlOrProc(s.Name, v)
		delete(snap, s.Name)
		saveSysctlSnapshot(snap)
		return
	}
	setSysctlOrProc(s.Name, s.Revert)
}

type Manifest struct {
	Chains  []Chain
	Rules   []Rule
	Sysctls []SysctlSetting
}

func (m Manifest) Apply() error {
	for _, c := range m.Chains {
		c.Ensure()
	}
	for _, r := range m.Rules {
		if err := r.Apply(); err != nil {
			return err
		}
	}
	for _, s := range m.Sysctls {
		s.Apply()
	}
	return nil
}

func (m Manifest) RemoveRules() {
	for i := len(m.Rules) - 1; i >= 0; i-- {
		m.Rules[i].Remove()
	}
}

func (m Manifest) RemoveChains() {
	for i := len(m.Chains) - 1; i >= 0; i-- {
		m.Chains[i].Remove()
	}
}

func (m Manifest) RevertSysctls() {
	for _, s := range m.Sysctls {
		s.RevertBack()
	}
}

func (manager *IPTablesManager) buildManifest() (Manifest, error) {
	cfg := manager.cfg
	var ipts []string
	if cfg.Queue.IPv4Enabled && hasBinary("iptables") {
		ipts = append(ipts, "iptables")
	}
	if cfg.Queue.IPv6Enabled && hasBinary("ip6tables") {
		ipts = append(ipts, "ip6tables")
	}
	if len(ipts) == 0 {
		return Manifest{}, errors.New("no valid iptables binaries found")
	}
	queueNum := cfg.Queue.StartNum
	threads := cfg.Queue.Threads
	chainName := "B4"
	markAccept := "32768/32768"

	var chains []Chain
	var rules []Rule

	for _, ipt := range ipts {
		ch := Chain{manager: manager, IPT: ipt, Table: "mangle", Name: chainName}
		chains = append(chains, ch)

		rules = append(rules,
			Rule{manager: manager, IPT: ipt, Table: "mangle", Chain: chainName, Action: "I", Spec: []string{"-m", "mark", "--mark", markAccept, "-j", "RETURN"}},
		)

		tcpConnbytesRange := fmt.Sprintf("0:%d", cfg.MainSet.TCP.ConnBytesLimit)
		udpConnbytesRange := fmt.Sprintf("0:%d", cfg.MainSet.UDP.ConnBytesLimit)

		tcpSpec := append(
			[]string{"-p", "tcp", "--dport", "443",
				"-m", "connbytes", "--connbytes-dir", "original",
				"--connbytes-mode", "packets", "--connbytes", tcpConnbytesRange},
			manager.buildNFQSpec(queueNum, threads)...,
		)
		udpSpec := append(
			[]string{"-p", "udp",
				"-m", "connbytes", "--connbytes-dir", "original",
				"--connbytes-mode", "packets", "--connbytes", udpConnbytesRange},
			manager.buildNFQSpec(queueNum, threads)...,
		)

		rules = append(rules,
			Rule{manager: manager, IPT: ipt, Table: "mangle", Chain: chainName, Action: "A", Spec: tcpSpec},
			Rule{manager: manager, IPT: ipt, Table: "mangle", Chain: chainName, Action: "A", Spec: udpSpec},
			Rule{manager: manager, IPT: ipt, Table: "mangle", Chain: "POSTROUTING", Action: "I", Spec: []string{"-j", chainName}},
			Rule{manager: manager, IPT: ipt, Table: "mangle", Chain: "OUTPUT", Action: "I", Spec: []string{"-m", "mark", "--mark", markAccept, "-j", "ACCEPT"}},
		)

	}

	sysctls := []SysctlSetting{
		{Name: "net.netfilter.nf_conntrack_checksum", Desired: "0", Revert: "1"},
		{Name: "net.netfilter.nf_conntrack_tcp_be_liberal", Desired: "1", Revert: "0"},
	}

	return Manifest{Chains: chains, Rules: rules, Sysctls: sysctls}, nil
}

func (ipt *IPTablesManager) Apply() error {

	log.Infof("IPTABLES: adding rules")
	loadKernelModules()
	m, err := ipt.buildManifest()
	if err != nil {
		return err
	}
	result := m.Apply()

	if log.Level(log.CurLevel.Load()) >= log.LevelTrace {
		iptables_trace, _ := run("sh", "-c", "cat /proc/net/netfilter/nfnetlink_queue && iptables -t mangle -vnL --line-numbers")
		log.Tracef("Current iptables mangle table:\n%s", iptables_trace)
	}
	return result
}

func (ipt *IPTablesManager) Clear() error {

	ipts := []string{}
	if hasBinary("iptables") {
		ipts = append(ipts, "iptables")
	}
	if hasBinary("ip6tables") {
		ipts = append(ipts, "ip6tables")
	}
	if len(ipts) == 0 {
		ipts = []string{"iptables"}
	}
	m, err := ipt.buildManifest()
	if err != nil {
		return err
	}
	m.RemoveRules()
	time.Sleep(30 * time.Millisecond)
	m.RemoveChains()
	//m.RevertSysctls()
	return nil
}
