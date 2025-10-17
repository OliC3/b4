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
	QueueStartNum  int      `json:"queue_start_num" bson:"queue_start_num"`
	Mark           uint     `json:"mark" bson:"mark"`
	ConnBytesLimit int      `json:"conn_bytes_limit" bson:"conn_bytes_limit"`
	Logging        Logging  `json:"logging" bson:"logging"`
	SNIDomains     []string `json:"sni_domains" bson:"sni_domains"`
	Threads        int      `json:"threads" bson:"threads"`
	UseGSO         bool     `json:"use_gso" bson:"use_gso"`
	UseConntrack   bool     `json:"use_conntrack" bson:"use_conntrack"`
	SkipIpTables   bool     `json:"skip_iptables" bson:"skip_iptables"`
	GeoSitePath    string   `json:"geosite_path" bson:"geosite_path"`
	GeoIpPath      string   `json:"geoip_path" bson:"geoip_path"`
	GeoCategories  []string `json:"geo_categories" bson:"geo_categories"`
	Seg2Delay      int      `json:"seg2delay" bson:"seg2delay"`

	FragmentStrategy string
	FragSNIReverse   bool
	FragMiddleSNI    bool
	FragSNIPosition  int

	FakeSNI           bool
	FakeTTL           uint8
	FakeStrategy      string
	FakeSeqOffset     int32
	FakeSNISeqLength  int
	FakeSNIType       int
	FakeCustomPayload string

	UDPMode           string
	UDPFakeSeqLength  int
	UDPFakeLen        int
	UDPFakingStrategy string
	UDPDPortMin       int
	UDPDPortMax       int
	UDPFilterQUIC     string

	WebServer WebServer `json:"web_server" bson:"web_server"`
}

type WebServer struct {
	Port int `json:"port" bson:"port"`
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
	QueueStartNum:  537,
	Mark:           1 << 15,
	Threads:        4,
	ConnBytesLimit: 19,
	UseConntrack:   false,
	UseGSO:         false,
	SkipIpTables:   false,
	GeoSitePath:    "",
	GeoIpPath:      "",
	GeoCategories:  []string{},
	Seg2Delay:      0,

	FragmentStrategy: "tcp",
	FragSNIReverse:   true,
	FragMiddleSNI:    true,
	FragSNIPosition:  1,

	FakeSNI:           true,
	FakeTTL:           8,
	FakeSNISeqLength:  1,
	FakeSNIType:       FakePayloadDefault,
	FakeCustomPayload: "",
	FakeStrategy:      "pastseq",
	FakeSeqOffset:     10000,

	UDPMode:           "drop",
	UDPFakeSeqLength:  6,
	UDPFakeLen:        64,
	UDPFakingStrategy: "none",
	UDPDPortMin:       0,
	UDPDPortMax:       0,
	UDPFilterQUIC:     "parse",

	WebServer: WebServer{
		Port: 0,
	},

	Logging: Logging{
		Level:      log.LevelInfo,
		Instaflush: true,
		Syslog:     false,
	},
}

func (c *Config) SaveToFile(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return log.Errorf("failed to marshal config: %v", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return log.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return log.Errorf("failed to write config file: %v", err)
	}
	return nil
}

func (c *Config) LoadFromFile(path string) error {

	if path == "" {
		return log.Errorf("config path is not specified")
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

	// Network configuration
	cmd.Flags().IntVar(&c.QueueStartNum, "queue-num", c.QueueStartNum, "Netfilter queue number")
	cmd.Flags().IntVar(&c.Threads, "threads", c.Threads, "Number of worker threads")
	cmd.Flags().UintVar(&c.Mark, "mark", c.Mark, "Packet mark value")
	cmd.Flags().IntVar(&c.ConnBytesLimit, "connbytes-limit", c.ConnBytesLimit, "Connection bytes limit")
	cmd.Flags().StringSliceVar(&c.SNIDomains, "sni-domains", c.SNIDomains, "List of SNI domains to match")
	cmd.Flags().IntVar(&c.Seg2Delay, "seg2delay", c.Seg2Delay, "Delay between segments in ms")

	// Geodata and site filtering
	cmd.Flags().StringVar(&c.GeoSitePath, "geosite", c.GeoSitePath, "Path to geosite file (e.g., geosite.dat)")
	cmd.Flags().StringVar(&c.GeoIpPath, "geoip", c.GeoIpPath, "Path to geoip file (e.g., geoip.dat)")
	cmd.Flags().StringSliceVar(&c.GeoCategories, "geo-categories", c.GeoCategories, "Geographic categories to process (e.g., youtube,facebook,amazon)")

	// Fake SNI and TTL configuration
	cmd.Flags().StringVar(&c.FragmentStrategy, "frag", "tcp", "Fragmentation strategy (tcp/ip/none)")
	cmd.Flags().BoolVar(&c.FragSNIReverse, "frag-sni-reverse", c.FragSNIReverse, "Reverse fragment order")
	cmd.Flags().BoolVar(&c.FragMiddleSNI, "frag-middle-sni", c.FragMiddleSNI, "Fragment in middle of SNI")
	cmd.Flags().IntVar(&c.FragSNIPosition, "frag-sni-pos", c.FragSNIPosition, "SNI fragment position")

	cmd.Flags().StringVar(&c.FakeStrategy, "fake-strategy", c.FakeStrategy, "Faking strategy (ttl/randseq/pastseq/tcp_check/md5sum)")
	cmd.Flags().Uint8Var(&c.FakeTTL, "fake-ttl", c.FakeTTL, "TTL for fake packets")
	cmd.Flags().Int32Var(&c.FakeSeqOffset, "fake-seq-offset", c.FakeSeqOffset, "Sequence offset for fake packets")
	cmd.Flags().BoolVar(&c.FakeSNI, "fake-sni", c.FakeSNI, "Enable fake SNI packets")
	cmd.Flags().IntVar(&c.FakeSNISeqLength, "fake-sni-len", c.FakeSNISeqLength, "Length of fake SNI sequence")
	cmd.Flags().IntVar(&c.FakeSNIType, "fake-sni-type", c.FakeSNIType, "Type of fake SNI payload (0=random, 1=custom, 2=default)")

	cmd.Flags().StringVar(&c.UDPMode, "udp-mode", c.UDPMode, "UDP handling strategy (drop|fake)")
	cmd.Flags().IntVar(&c.UDPFakeSeqLength, "udp-fake-seq-len", c.UDPFakeSeqLength, "UDP fake packet sequence length")
	cmd.Flags().IntVar(&c.UDPFakeLen, "udp-fake-len", c.UDPFakeLen, "UDP fake packet size in bytes")
	cmd.Flags().StringVar(&c.UDPFakingStrategy, "udp-faking-strategy", c.UDPFakingStrategy, "UDP faking strategy (none|ttl|checksum)")
	cmd.Flags().IntVar(&c.UDPDPortMin, "udp-dport-min", c.UDPDPortMin, "Minimum UDP destination port to handle")
	cmd.Flags().IntVar(&c.UDPDPortMax, "udp-dport-max", c.UDPDPortMax, "Maximum UDP destination port to handle")
	cmd.Flags().StringVar(&c.UDPFilterQUIC, "udp-filter-quic", c.UDPFilterQUIC, "QUIC filtering mode (disabled|all|parse)")

	// Feature flags
	cmd.Flags().BoolVar(&c.UseGSO, "gso", c.UseGSO, "Enable Generic Segmentation Offload")
	cmd.Flags().BoolVar(&c.UseConntrack, "conntrack", c.UseConntrack, "Enable connection tracking")
	cmd.Flags().BoolVar(&c.SkipIpTables, "skip-iptables", c.SkipIpTables, "Skip iptables rules setup")

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
	// If sites are specified, geodata path must be provided
	if len(c.GeoCategories) > 0 && c.GeoSitePath == "" {
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
	return geodat.LoadDomainsFromSites(c.GeoSitePath, c.GeoCategories)
}
