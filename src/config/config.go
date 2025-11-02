// path: src/config/config.go
package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/daniellavrushin/b4/geodat"
	"github.com/daniellavrushin/b4/log"
	"github.com/spf13/cobra"
)

var (
	//go:embed config.json
	configJson string
)

type Config struct {
	ConfigPath     string  `json:"-" bson:"-"`
	QueueStartNum  int     `json:"queue_start_num" bson:"queue_start_num"`
	Mark           uint    `json:"mark" bson:"mark"`
	ConnBytesLimit int     `json:"conn_bytes_limit" bson:"conn_bytes_limit"`
	Logging        Logging `json:"logging" bson:"logging"`
	Threads        int     `json:"threads" bson:"threads"`
	SkipTables     bool    `json:"skip_tables" bson:"skip_tables"`
	Seg2Delay      int     `json:"seg2delay" bson:"seg2delay"`
	IPv4Enabled    bool    `json:"ipv4" bson:"ipv4"`
	IPv6Enabled    bool    `json:"ipv6" bson:"ipv6"`

	Domains       DomainsConfig `json:"domains" bson:"domains"`
	Fragmentation Fragmentation `json:"fragmentation" bson:"fragmentation"`
	Faking        Faking        `json:"faking" bson:"faking"`
	UDP           UDPConfig     `json:"udp" bson:"udp"`

	WebServer WebServer `json:"web_server" bson:"web_server"`
}

type Faking struct {
	SNI           bool   `json:"sni" bson:"sni"`
	TTL           uint8  `json:"ttl" bson:"ttl"`
	Strategy      string `json:"strategy" bson:"strategy"`
	SeqOffset     int32  `json:"seq_offset" bson:"seq_offset"`
	SNISeqLength  int    `json:"sni_seq_length" bson:"sni_seq_length"`
	SNIType       int    `json:"sni_type" bson:"sni_type"`
	CustomPayload string `json:"custom_payload" bson:"custom_payload"`
}

type Fragmentation struct {
	Strategy    string `json:"strategy" bson:"strategy"`
	SNIReverse  bool   `json:"sni_reverse" bson:"sni_reverse"`
	MiddleSNI   bool   `json:"middle_sni" bson:"middle_sni"`
	SNIPosition int    `json:"sni_position" bson:"sni_position"`
}

type UDPConfig struct {
	Mode           string `json:"mode" bson:"mode"`
	FakeSeqLength  int    `json:"fake_seq_length" bson:"fake_seq_length"`
	FakeLen        int    `json:"fake_len" bson:"fake_len"`
	FakingStrategy string `json:"faking_strategy" bson:"faking_strategy"`
	DPortMin       int    `json:"dport_min" bson:"dport_min"`
	DPortMax       int    `json:"dport_max" bson:"dport_max"`
	FilterQUIC     string `json:"filter_quic" bson:"filter_quic"`
	FilterSTUN     bool   `json:"filter_stun" bson:"filter_stun"`
	ConnBytesLimit int    `json:"conn_bytes_limit" bson:"conn_bytes_limit"`
}

type DomainsConfig struct {
	GeoSitePath       string   `json:"geosite_path" bson:"geosite_path"`
	GeoIpPath         string   `json:"geoip_path" bson:"geoip_path"`
	SNIDomains        []string `json:"sni_domains" bson:"sni_domains"`
	GeoSiteCategories []string `json:"geosite_categories" bson:"geosite_categories"`
	GeoIpCategories   []string `json:"geoip_categories" bson:"geoip_categories"`
}

type WebServer struct {
	Port      int  `json:"port" bson:"port"`
	IsEnabled bool `json:"-" bson:"-"`
}

type Logging struct {
	Level      log.Level `json:"level" bson:"level"`
	Instaflush bool      `json:"instaflush" bson:"instaflush"`
	Syslog     bool      `json:"syslog" bson:"syslog"`
}

const (
	FakePayloadRandom = iota
	FakePayloadCustom
	FakePayloadDefault
)

var DefaultConfig = Config{
	ConfigPath:     "",
	QueueStartNum:  537,
	Mark:           1 << 15,
	Threads:        4,
	ConnBytesLimit: 19,
	SkipTables:     false,
	Seg2Delay:      0,
	IPv4Enabled:    true,
	IPv6Enabled:    false,

	Domains: DomainsConfig{
		GeoSitePath:       "",
		GeoIpPath:         "",
		SNIDomains:        []string{},
		GeoSiteCategories: []string{},
		GeoIpCategories:   []string{},
	},

	Fragmentation: Fragmentation{
		Strategy:    "tcp",
		SNIReverse:  true,
		MiddleSNI:   true,
		SNIPosition: 1,
	},

	Faking: Faking{
		SNI:           true,
		TTL:           8,
		SNISeqLength:  1,
		SNIType:       FakePayloadDefault,
		CustomPayload: "",
		Strategy:      "pastseq",
		SeqOffset:     10000,
	},

	UDP: UDPConfig{
		Mode:           "fake",
		FakeSeqLength:  6,
		FakeLen:        64,
		FakingStrategy: "none",
		DPortMin:       0,
		DPortMax:       0,
		FilterQUIC:     "disabled",
		FilterSTUN:     true,
		ConnBytesLimit: 8,
	},

	WebServer: WebServer{
		Port:      0,
		IsEnabled: false,
	},

	Logging: Logging{
		Level:      log.LevelInfo,
		Instaflush: true,
		Syslog:     false,
	},
}

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

func (c *Config) BindFromEmbed() (Config, error) {
	data := []byte(configJson)
	err := json.Unmarshal(data, c)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse embedded config: %v", err)
	}
	return *c, nil
}

func (c *Config) BindFlags(cmd *cobra.Command) {

	_, err := c.BindFromEmbed()
	if err != nil {
		fmt.Printf("Warning: failed to load embedded config: %v\n", err)
	}

	// Config path
	cmd.Flags().StringVar(&c.ConfigPath, "config", c.ConfigPath, "Path to config file")

	// Network configuration
	cmd.Flags().IntVar(&c.QueueStartNum, "queue-num", c.QueueStartNum, "Netfilter queue number")
	cmd.Flags().IntVar(&c.Threads, "threads", c.Threads, "Number of worker threads")
	cmd.Flags().UintVar(&c.Mark, "mark", c.Mark, "Packet mark value (default 32768)")
	cmd.Flags().IntVar(&c.ConnBytesLimit, "connbytes-limit", c.ConnBytesLimit, "Connection bytes limit (default 19)")
	cmd.Flags().IntVar(&c.Seg2Delay, "seg2delay", c.Seg2Delay, "Delay between segments in ms")

	// Geodata and site filtering
	cmd.Flags().StringSliceVar(&c.Domains.SNIDomains, "sni-domains", c.Domains.SNIDomains, "List of SNI domains to match")
	cmd.Flags().StringVar(&c.Domains.GeoSitePath, "geosite", c.Domains.GeoSitePath, "Path to geosite file (e.g., geosite.dat)")
	cmd.Flags().StringVar(&c.Domains.GeoIpPath, "geoip", c.Domains.GeoIpPath, "Path to geoip file (e.g., geoip.dat)")
	cmd.Flags().StringSliceVar(&c.Domains.GeoSiteCategories, "geosite-categories", c.Domains.GeoSiteCategories, "Geographic categories to process (e.g., youtube,facebook,amazon)")
	cmd.Flags().StringSliceVar(&c.Domains.GeoIpCategories, "geoip-categories", c.Domains.GeoIpCategories, "Geographic categories to process (e.g., youtube,facebook,amazon)")

	// Fake SNI and TTL configuration
	cmd.Flags().StringVar(&c.Fragmentation.Strategy, "frag", "tcp", "Fragmentation strategy (tcp|ip|none)")
	cmd.Flags().BoolVar(&c.Fragmentation.SNIReverse, "frag-sni-reverse", c.Fragmentation.SNIReverse, "Reverse fragment order")
	cmd.Flags().BoolVar(&c.Fragmentation.MiddleSNI, "frag-middle-sni", c.Fragmentation.MiddleSNI, "Fragment in middle of SNI")
	cmd.Flags().IntVar(&c.Fragmentation.SNIPosition, "frag-sni-pos", c.Fragmentation.SNIPosition, "SNI fragment position")

	cmd.Flags().StringVar(&c.Faking.Strategy, "fake-strategy", c.Faking.Strategy, "Faking strategy (ttl|randseq|pastseq|tcp_check|md5sum)")
	cmd.Flags().Uint8Var(&c.Faking.TTL, "fake-ttl", c.Faking.TTL, "TTL for fake packets")
	cmd.Flags().Int32Var(&c.Faking.SeqOffset, "fake-seq-offset", c.Faking.SeqOffset, "Sequence offset for fake packets")
	cmd.Flags().BoolVar(&c.Faking.SNI, "fake-sni", c.Faking.SNI, "Enable fake SNI packets")
	cmd.Flags().IntVar(&c.Faking.SNISeqLength, "fake-sni-len", c.Faking.SNISeqLength, "Length of fake SNI sequence")
	cmd.Flags().IntVar(&c.Faking.SNIType, "fake-sni-type", c.Faking.SNIType, "Type of fake SNI payload (0=random, 1=custom, 2=default)")

	cmd.Flags().StringVar(&c.UDP.Mode, "udp-mode", c.UDP.Mode, "UDP handling strategy (drop|fake)")
	cmd.Flags().IntVar(&c.UDP.FakeSeqLength, "udp-fake-seq-len", c.UDP.FakeSeqLength, "UDP fake packet sequence length")
	cmd.Flags().IntVar(&c.UDP.FakeLen, "udp-fake-len", c.UDP.FakeLen, "UDP fake packet size in bytes")
	cmd.Flags().StringVar(&c.UDP.FakingStrategy, "udp-faking-strategy", c.UDP.FakingStrategy, "UDP faking strategy (none|ttl|checksum)")
	cmd.Flags().IntVar(&c.UDP.DPortMin, "udp-dport-min", c.UDP.DPortMin, "Minimum UDP destination port to handle")
	cmd.Flags().IntVar(&c.UDP.DPortMax, "udp-dport-max", c.UDP.DPortMax, "Maximum UDP destination port to handle")
	cmd.Flags().StringVar(&c.UDP.FilterQUIC, "udp-filter-quic", c.UDP.FilterQUIC, "QUIC filtering mode (disabled|all|parse)")
	cmd.Flags().BoolVar(&c.UDP.FilterSTUN, "udp-filter-stun", c.UDP.FilterSTUN, "STUN filtering mode (disabled|all|parse)")
	cmd.Flags().IntVar(&c.UDP.ConnBytesLimit, "udp-conn-bytes-limit", c.UDP.ConnBytesLimit, "UDP connection bytes limit (default 8)")

	// Feature flags
	cmd.Flags().BoolVar(&c.SkipTables, "skip-tables", c.SkipTables, "Skip iptables/nftables rules setup")
	cmd.Flags().BoolVar(&c.IPv4Enabled, "ipv4", c.IPv4Enabled, "Enable IPv4 processing")
	cmd.Flags().BoolVar(&c.IPv6Enabled, "ipv6", c.IPv6Enabled, "Enable IPv6 processing")

	// Logging configuration
	cmd.Flags().BoolVarP(&c.Logging.Instaflush, "instaflush", "i", c.Logging.Instaflush, "Flush logs immediately")
	cmd.Flags().BoolVar(&c.Logging.Syslog, "syslog", c.Logging.Syslog, "Enable syslog output")

	// Web server configuration
	cmd.Flags().IntVar(&c.WebServer.Port, "web-port", c.WebServer.Port, "Port for internal web server (0 disables)")

}

func (cfg *Config) ApplyLogLevel(level string) {
	switch level {
	case "debug":
		cfg.Logging.Level = log.LevelDebug
	case "trace":
		cfg.Logging.Level = log.LevelTrace
	case "info":
		cfg.Logging.Level = log.LevelInfo
	case "error":
		cfg.Logging.Level = log.LevelError
	case "silent":
		cfg.Logging.Level = -1
	default:
		cfg.Logging.Level = log.LevelInfo
	}
}

func (c *Config) Validate() error {

	c.WebServer.IsEnabled = c.WebServer.Port > 0 && c.WebServer.Port <= 65535

	// If sites are specified, geodata path must be provided
	if len(c.Domains.GeoSiteCategories) > 0 && c.Domains.GeoSitePath == "" {
		return fmt.Errorf("--geosite must be specified when using --geo-categories")
	}

	if c.Threads < 1 {
		return fmt.Errorf("threads must be at least 1")
	}

	if c.QueueStartNum < 0 || c.QueueStartNum > 65535 {
		return fmt.Errorf("queue-num must be between 0 and 65535")
	}

	return nil
}

func (c *Config) LogString() string {
	return ""
}

// LoadDomainsFromGeodata loads domains from geodata file for specified sites
// and returns them as a slice
func (c *Config) LoadDomainsFromGeodata() ([]string, error) {
	return geodat.LoadDomainsFromSites(c.Domains.GeoSitePath, c.Domains.GeoSiteCategories)
}
