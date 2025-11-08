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
	cmd.Flags().IntVar(&c.MainSet.TCP.ConnBytesLimit, "connbytes-limit", c.MainSet.TCP.ConnBytesLimit, "TCP connection bytes limit (default 19)")
	cmd.Flags().IntVar(&c.MainSet.TCP.Seg2Delay, "seg2delay", c.MainSet.TCP.Seg2Delay, "Delay between segments in ms")

	// UDP bypass configuration
	cmd.Flags().StringVar(&c.MainSet.UDP.Mode, "udp-mode", c.MainSet.UDP.Mode, "UDP handling strategy (drop|fake)")
	cmd.Flags().IntVar(&c.MainSet.UDP.FakeSeqLength, "udp-fake-seq-len", c.MainSet.UDP.FakeSeqLength, "UDP fake packet sequence length")
	cmd.Flags().IntVar(&c.MainSet.UDP.FakeLen, "udp-fake-len", c.MainSet.UDP.FakeLen, "UDP fake packet size in bytes")
	cmd.Flags().StringVar(&c.MainSet.UDP.FakingStrategy, "udp-faking-strategy", c.MainSet.UDP.FakingStrategy, "UDP faking strategy (none|ttl|checksum)")
	cmd.Flags().IntVar(&c.MainSet.UDP.DPortMin, "udp-dport-min", c.MainSet.UDP.DPortMin, "Minimum UDP destination port to handle")
	cmd.Flags().IntVar(&c.MainSet.UDP.DPortMax, "udp-dport-max", c.MainSet.UDP.DPortMax, "Maximum UDP destination port to handle")
	cmd.Flags().StringVar(&c.MainSet.UDP.FilterQUIC, "udp-filter-quic", c.MainSet.UDP.FilterQUIC, "QUIC filtering mode (disabled|all|parse)")
	cmd.Flags().BoolVar(&c.MainSet.UDP.FilterSTUN, "udp-filter-stun", c.MainSet.UDP.FilterSTUN, "STUN filtering mode (disabled|all|parse)")
	cmd.Flags().IntVar(&c.MainSet.UDP.ConnBytesLimit, "udp-conn-bytes-limit", c.MainSet.UDP.ConnBytesLimit, "UDP connection bytes limit (default 8)")

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

	// Domain filtering
	cmd.Flags().StringSliceVar(&c.MainSet.Domains.SNIDomains, "sni-domains", c.MainSet.Domains.SNIDomains, "List of SNI domains to match")
	cmd.Flags().StringVar(&c.System.Geo.GeoSitePath, "geosite", c.System.Geo.GeoSitePath, "Path to geosite file (e.g., geosite.dat)")
	cmd.Flags().StringVar(&c.System.Geo.GeoIpPath, "geoip", c.System.Geo.GeoIpPath, "Path to geoip file (e.g., geoip.dat)")
	cmd.Flags().StringSliceVar(&c.MainSet.Domains.GeoSiteCategories, "geosite-categories", c.MainSet.Domains.GeoSiteCategories, "Geographic categories to process (e.g., youtube,facebook,amazon)")
	cmd.Flags().StringSliceVar(&c.MainSet.Domains.GeoIpCategories, "geoip-categories", c.MainSet.Domains.GeoIpCategories, "Geographic categories to process (e.g., youtube,facebook,amazon)")

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

	if len(c.MainSet.Domains.GeoSiteCategories) > 0 && c.System.Geo.GeoSitePath == "" {
		return fmt.Errorf("--geosite must be specified when using --geo-categories")
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
		}
	}

	return nil
}

func (c *Config) LogString() string {
	return ""
}

// LoadDomains returns all domains from all sets grouped by set name
func (c *Config) LoadDomains() ([]*SetConfig, int, error) {
	result := make([]*SetConfig, 0, len(c.Sets))
	totalDomains := 0

	// Process all sets
	for _, set := range c.Sets {
		domains, err := c.GetDomainsForSet(set)
		if err != nil {
			return nil, -1, fmt.Errorf("failed to load domains for set '%s': %w", set.Name, err)
		}
		if len(domains) > 0 {
			result = append(result, set)
			totalDomains += len(domains)
		}
	}

	// If no sets have domains, use MainSet as fallback
	if len(result) == 0 && c.MainSet != nil {
		domains, err := c.GetDomainsForSet(c.MainSet)
		if err != nil {
			return nil, -1, fmt.Errorf("failed to load domains for main set: %w", err)
		}
		if len(domains) > 0 {
			result = append(result, c.MainSet)
			totalDomains += len(domains)
		}
	}

	return result, totalDomains, nil
}

// GetDomainsForSet loads domains for a specific set, combining geosite and manual domains
func (c *Config) GetDomainsForSet(set *SetConfig) ([]string, error) {

	domains := []string{}

	if len(set.Domains.GeoSiteCategories) > 0 && c.System.Geo.GeoSitePath != "" {
		geoDomains, err := geodat.LoadDomainsFromCategories(
			c.System.Geo.GeoSitePath,
			set.Domains.GeoSiteCategories,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load geosite domains for set '%s': %w", set.Name, err)
		}

		if len(geoDomains) > 0 {
			domains = append(domains, geoDomains...)
		}
	}

	if len(set.Domains.SNIDomains) > 0 {
		domains = append(domains, set.Domains.SNIDomains...)
	}

	if len(domains) > 0 {
		set.Domains.DomainsToMatch = utils.FilterUniqueStrings(domains)
		return set.Domains.DomainsToMatch, nil
	}

	return nil, nil
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
	domains := set.Domains

	*set = defaultSet

	set.Id = id
	set.Name = name
	set.Domains = domains
}
