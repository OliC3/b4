package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/daniellavrushin/b4/geodat"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/utils"
	"github.com/spf13/cobra"
)

func (c *Config) SaveToFile(path string) error {
	if path == "" {
		log.Tracef("config path is not defined")
		return nil
	}

	if len(c.Sets) == 0 {
		c.Sets = []*SetConfig{&DefaultSetConfig}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return log.Errorf("failed to marshal config: %v", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return log.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return log.Errorf("failed to write config file: %v", err)
	}
	return nil
}

func (c *Config) LoadFromFile(path string) error {
	if path == "" {
		log.Tracef("config path is not defined")
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return log.Errorf("failed to stat config file: %v", err)
	}
	if info.IsDir() {
		return log.Errorf("config path is a directory, not a file: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return log.Errorf("failed to read config file: %v", err)
	}
	err = json.Unmarshal(data, c)
	if err != nil {
		return log.Errorf("failed to parse config file: %v", err)
	}
	if len(c.Sets) == 0 {
		c.Sets = []*SetConfig{&DefaultSetConfig}
	}
	return nil
}

func (c *Config) BindFlags(cmd *cobra.Command) {
	// Config path
	cmd.Flags().StringVar(&c.ConfigPath, "config", c.ConfigPath, "Path to config file")

	// Queue configuration
	cmd.Flags().IntVar(&c.Queue.StartNum, "queue-num", c.Queue.StartNum, "Netfilter queue number")
	cmd.Flags().IntVar(&c.Queue.Threads, "threads", c.Queue.Threads, "Number of worker threads")
	cmd.Flags().UintVar(&c.Queue.Mark, "mark", c.Queue.Mark, "Packet mark value (default 32768)")
	cmd.Flags().BoolVar(&c.Queue.IPv4Enabled, "ipv4", c.Queue.IPv4Enabled, "Enable IPv4 processing")
	cmd.Flags().BoolVar(&c.Queue.IPv6Enabled, "ipv6", c.Queue.IPv6Enabled, "Enable IPv6 processing")

	// TCP bypass configuration
	cmd.Flags().IntVar(&c.MainSet.TCP.ConnBytesLimit, "tcp-connbytes-limit", c.MainSet.TCP.ConnBytesLimit, "TCP connection bytes limit (default 19)")
	cmd.Flags().IntVar(&c.MainSet.TCP.Seg2Delay, "tcp-seg2delay", c.MainSet.TCP.Seg2Delay, "Delay between segments in ms")
	cmd.Flags().BoolVar(&c.MainSet.TCP.SynFake, "tcp-syn-fake", c.MainSet.TCP.SynFake, "Enable SYN fake packets (default false)")
	cmd.Flags().IntVar(&c.MainSet.TCP.SynFakeLen, "tcp-syn-fake-len", c.MainSet.TCP.SynFakeLen, "SYN fake packet size in bytes (default 0)")

	// UDP bypass configuration
	cmd.Flags().StringVar(&c.MainSet.UDP.Mode, "udp-mode", c.MainSet.UDP.Mode, "UDP handling strategy (drop|fake)")
	cmd.Flags().IntVar(&c.MainSet.UDP.FakeSeqLength, "udp-fake-seq-len", c.MainSet.UDP.FakeSeqLength, "UDP fake packet sequence length")
	cmd.Flags().IntVar(&c.MainSet.UDP.FakeLen, "udp-fake-len", c.MainSet.UDP.FakeLen, "UDP fake packet size in bytes")
	cmd.Flags().StringVar(&c.MainSet.UDP.FakingStrategy, "udp-faking-strategy", c.MainSet.UDP.FakingStrategy, "UDP faking strategy (none|ttl|checksum)")
	cmd.Flags().StringVar(&c.MainSet.UDP.DPortFilter, "udp-dport-filter", c.MainSet.UDP.DPortFilter, "UDP destination port filter (comma separated list of ports and port ranges, e.g. '80,443,1000-2000')")
	cmd.Flags().StringVar(&c.MainSet.UDP.FilterQUIC, "udp-filter-quic", c.MainSet.UDP.FilterQUIC, "QUIC filtering mode (disabled|all|parse)")
	cmd.Flags().BoolVar(&c.MainSet.UDP.FilterSTUN, "udp-filter-stun", c.MainSet.UDP.FilterSTUN, "STUN filtering mode (disabled|all|parse)")
	cmd.Flags().IntVar(&c.MainSet.UDP.ConnBytesLimit, "udp-conn-bytes-limit", c.MainSet.UDP.ConnBytesLimit, "UDP connection bytes limit (default 8)")
	cmd.Flags().IntVar(&c.MainSet.UDP.Seg2Delay, "udp-seg2delay", c.MainSet.UDP.Seg2Delay, "Delay between segments in ms (default 0)")

	// Fragmentation configuration
	cmd.Flags().StringVar(&c.MainSet.Fragmentation.Strategy, "frag", c.MainSet.Fragmentation.Strategy, "Fragmentation strategy (tcp|ip|none)")
	cmd.Flags().BoolVar(&c.MainSet.Fragmentation.SNIReverse, "frag-sni-reverse", c.MainSet.Fragmentation.SNIReverse, "Reverse fragment order")
	cmd.Flags().BoolVar(&c.MainSet.Fragmentation.MiddleSNI, "frag-middle-sni", c.MainSet.Fragmentation.MiddleSNI, "Fragment in middle of SNI")
	cmd.Flags().IntVar(&c.MainSet.Fragmentation.SNIPosition, "frag-sni-pos", c.MainSet.Fragmentation.SNIPosition, "SNI fragment position")

	// Faking configuration
	cmd.Flags().StringVar(&c.MainSet.Faking.Strategy, "fake-strategy", c.MainSet.Faking.Strategy, "Faking strategy (ttl|randseq|pastseq|tcp_check|md5sum)")
	cmd.Flags().Uint8Var(&c.MainSet.Faking.TTL, "fake-ttl", c.MainSet.Faking.TTL, "TTL for fake packets")
	cmd.Flags().Int32Var(&c.MainSet.Faking.SeqOffset, "fake-seq-offset", c.MainSet.Faking.SeqOffset, "Sequence offset for fake packets")
	cmd.Flags().BoolVar(&c.MainSet.Faking.SNI, "fake-sni", c.MainSet.Faking.SNI, "Enable fake SNI packets")
	cmd.Flags().IntVar(&c.MainSet.Faking.SNISeqLength, "fake-sni-len", c.MainSet.Faking.SNISeqLength, "Length of fake SNI sequence")
	cmd.Flags().IntVar(&c.MainSet.Faking.SNIType, "fake-sni-type", c.MainSet.Faking.SNIType, "Type of fake SNI payload (0=random, 1=custom, 2=default)")

	// Targets filtering
	cmd.Flags().StringSliceVar(&c.MainSet.Targets.SNIDomains, "sni-domains", c.MainSet.Targets.SNIDomains, "List of SNI domains to match")
	cmd.Flags().StringSliceVar(&c.MainSet.Targets.IPs, "ip", c.MainSet.Targets.IPs, "List of IPs/CIDRs to match")
	cmd.Flags().StringVar(&c.System.Geo.GeoSitePath, "geosite", c.System.Geo.GeoSitePath, "Path to geosite file (e.g., geosite.dat)")
	cmd.Flags().StringVar(&c.System.Geo.GeoIpPath, "geoip", c.System.Geo.GeoIpPath, "Path to geoip file (e.g., geoip.dat)")
	cmd.Flags().StringSliceVar(&c.MainSet.Targets.GeoSiteCategories, "geosite-categories", c.MainSet.Targets.GeoSiteCategories, "Geographic categories to process (e.g., youtube,facebook,amazon)")
	cmd.Flags().StringSliceVar(&c.MainSet.Targets.GeoIpCategories, "geoip-categories", c.MainSet.Targets.GeoIpCategories, "Geographic categories to process (e.g., youtube,facebook,amazon)")

	// System configuration
	cmd.Flags().IntVar(&c.System.Tables.MonitorInterval, "tables-monitor-interval", c.System.Tables.MonitorInterval, "Tables monitor interval in seconds (default 10, 0 to disable)")
	cmd.Flags().BoolVar(&c.System.Tables.SkipSetup, "skip-tables", c.System.Tables.SkipSetup, "Skip iptables/nftables setup on startup")

	cmd.Flags().BoolVarP(&c.System.Logging.Instaflush, "instaflush", "i", c.System.Logging.Instaflush, "Flush logs immediately")
	cmd.Flags().BoolVar(&c.System.Logging.Syslog, "syslog", c.System.Logging.Syslog, "Enable syslog output")

	cmd.Flags().IntVar(&c.System.WebServer.Port, "web-port", c.System.WebServer.Port, "Port for internal web server (0 disables)")
}

func (cfg *Config) ApplyLogLevel(level string) {
	switch level {
	case "debug":
		cfg.System.Logging.Level = log.LevelDebug
	case "trace":
		cfg.System.Logging.Level = log.LevelTrace
	case "info":
		cfg.System.Logging.Level = log.LevelInfo
	case "error":
		cfg.System.Logging.Level = log.LevelError
	case "silent":
		cfg.System.Logging.Level = -1
	default:
		cfg.System.Logging.Level = log.LevelInfo
	}
}

func (c *Config) Validate() error {
	c.System.WebServer.IsEnabled = c.System.WebServer.Port > 0 && c.System.WebServer.Port <= 65535

	if c.MainSet == nil && len(c.Sets) > 0 {
		for _, set := range c.Sets {
			if set.Id == MAIN_SET_ID {
				c.MainSet = set
				break
			}
		}
	} else {
		c.MainSet = &DefaultSetConfig
	}

	if c.MainSet == nil {
		return fmt.Errorf("main set configuration is missing")
	}

	if len(c.MainSet.Targets.GeoSiteCategories) > 0 && c.System.Geo.GeoSitePath == "" {
		return fmt.Errorf("--geosite must be specified when using --geo-categories")
	}

	if len(c.MainSet.Targets.GeoIpCategories) > 0 && c.System.Geo.GeoIpPath == "" {
		return fmt.Errorf("--geoip must be specified when using --geoip-categories")
	}

	if c.Queue.Threads < 1 {
		return fmt.Errorf("threads must be at least 1")
	}

	if c.Queue.StartNum < 0 || c.Queue.StartNum > 65535 {
		return fmt.Errorf("queue-num must be between 0 and 65535")
	}

	if len(c.Sets) > 1 {
		for index, set := range c.Sets {
			if set.Id == "" {
				return fmt.Errorf("each set must have a unique non-empty ID")
			}

			if index > 1 {
				if set.TCP.ConnBytesLimit > c.MainSet.TCP.ConnBytesLimit {
					return fmt.Errorf("set '%s' has TCP ConnBytesLimit greater than main set", set.Name)
				}

				if set.UDP.ConnBytesLimit > c.MainSet.UDP.ConnBytesLimit {
					return fmt.Errorf("set '%s' has UDP ConnBytesLimit greater than main set", set.Name)
				}
			}

			set.UDP.DPortFilter = utils.ValidatePorts(set.UDP.DPortFilter)
		}
	}

	return nil
}

func (c *Config) LogString() string {
	return ""
}

// LoadTargets returns all targets (domains and IPs) from all sets grouped by set name
func (c *Config) LoadTargets() ([]*SetConfig, int, int, error) {
	result := make([]*SetConfig, 0, len(c.Sets))
	totalDomains := 0
	totalIps := 0

	// Process all sets
	for _, set := range c.Sets {
		domains, ips, err := c.GetTargetsForSet(set)
		if err != nil {
			return nil, -1, -1, fmt.Errorf("failed to load domains for set '%s': %w", set.Name, err)
		}
		if len(domains) > 0 {
			totalDomains += len(domains)
		}
		if len(ips) > 0 {
			totalIps += len(ips)
		}
		result = append(result, set)
	}

	return result, totalDomains, totalIps, nil
}

// GetTargetsForSet loads domains for a specific set, combining geosite and manual domains
func (c *Config) GetTargetsForSet(set *SetConfig) ([]string, []string, error) {

	domains := []string{}
	ips := []string{}

	// domains from geosite categories
	if len(set.Targets.GeoSiteCategories) > 0 && c.System.Geo.GeoSitePath != "" {
		geoDomains, err := geodat.LoadDomainsFromCategories(
			c.System.Geo.GeoSitePath,
			set.Targets.GeoSiteCategories,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load geosite domains for set '%s': %w", set.Name, err)
		}

		if len(geoDomains) > 0 {
			domains = append(domains, geoDomains...)
		}
	}

	if len(set.Targets.SNIDomains) > 0 {
		domains = append(domains, set.Targets.SNIDomains...)
	}
	set.Targets.DomainsToMatch = domains

	//	 ips from geoip categories
	if len(set.Targets.GeoIpCategories) > 0 && c.System.Geo.GeoIpPath != "" {
		geoIps, err := geodat.LoadIpsFromCategories(
			c.System.Geo.GeoIpPath,
			set.Targets.GeoIpCategories,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load geosite domains for set '%s': %w", set.Name, err)
		}

		if len(geoIps) > 0 {
			ips = append(ips, geoIps...)
		}
	}

	if len(set.Targets.IPs) > 0 {
		ips = append(ips, set.Targets.IPs...)
	}

	set.Targets.IpsToMatch = ips
	return domains, ips, nil
}

func (c *Config) GetSetById(id string) *SetConfig {
	for _, set := range c.Sets {
		if set.Id == id {
			return set
		}
	}
	return nil
}

func (set *SetConfig) ResetToDefaults() {
	defaultSet := DefaultSetConfig

	// Preserve data
	id := set.Id
	name := set.Name
	targets := set.Targets

	*set = defaultSet

	set.Id = id
	set.Name = name
	set.Targets = targets
}

func (t *TargetsConfig) AppendIP(ip []string) error {
	for _, newIP := range ip {
		exists := false
		for _, existingIP := range t.IPs {
			if existingIP == newIP {
				exists = true
				break
			}
		}
		if !exists {
			t.IPs = append(t.IPs, newIP)
		}
	}

	for _, newIP := range ip {
		exists := false
		for _, existingIP := range t.IpsToMatch {
			if existingIP == newIP {
				exists = true
				break
			}
		}
		if !exists {
			t.IpsToMatch = append(t.IpsToMatch, newIP)
		}
	}

	return nil
}
func (t *TargetsConfig) AppendSNI(sni string) error {

	for _, existingDomain := range t.SNIDomains {
		if existingDomain == sni {
			return log.Errorf("SNI '%s' already exists in the set", sni)
		}
	}
	t.SNIDomains = append(t.SNIDomains, sni)

	for _, existingDomain := range t.DomainsToMatch {
		if existingDomain == sni {
			return log.Errorf("SNI '%s' already exists in the set", sni)
		}
	}
	t.DomainsToMatch = append(t.DomainsToMatch, sni)
	return nil
}
