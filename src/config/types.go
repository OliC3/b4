package config

import "github.com/daniellavrushin/b4/log"

const (
	ConfigOff = "off"

	FakePayloadRandom = iota
	FakePayloadCustom
	FakePayloadDefault1
	FakePayloadDefault2
	FakePayloadCapture
)

type ApiConfig struct {
	IPInfoToken string `json:"ipinfo_token" bson:"ipinfo_token"`
}

type QueueConfig struct {
	StartNum    int           `json:"start_num" bson:"start_num"`
	Threads     int           `json:"threads" bson:"threads"`
	Mark        uint          `json:"mark" bson:"mark"`
	IPv4Enabled bool          `json:"ipv4" bson:"ipv4"`
	IPv6Enabled bool          `json:"ipv6" bson:"ipv6"`
	Interfaces  []string      `json:"interfaces" bson:"interfaces"`
	Devices     DevicesConfig `json:"devices" bson:"devices"`
}

type DevicesConfig struct {
	Enabled      bool     `json:"enabled" bson:"enabled"`
	WhiteIsBlack bool     `json:"wisb" bson:"wisb"`
	Mac          []string `json:"mac" bson:"mac"`
}

type TCPConfig struct {
	ConnBytesLimit int   `json:"conn_bytes_limit" bson:"conn_bytes_limit"`
	Seg2Delay      int   `json:"seg2delay" bson:"seg2delay"`
	SynFake        bool  `json:"syn_fake" bson:"syn_fake"`
	SynFakeLen     int   `json:"syn_fake_len" bson:"syn_fake_len"`
	SynTTL         uint8 `json:"syn_ttl" bson:"syn_ttl"`
	DropSACK       bool  `json:"drop_sack" bson:"drop_sack"`

	WinMode   string `json:"win_mode" bson:"win_mode"`     // "off", "oscillate", "zero", "random", "escalate"
	WinValues []int  `json:"win_values" bson:"win_values"` // Custom window values

	DesyncMode  string `json:"desync_mode" bson:"desync_mode"`   // "off" "rst", "fin", "ack", "combo", "full"
	DesyncTTL   uint8  `json:"desync_ttl" bson:"desync_ttl"`     // TTL for desync packets
	DesyncCount int    `json:"desync_count" bson:"desync_count"` // Number of desync packets

}

type UDPConfig struct {
	Mode           string `json:"mode" bson:"mode"`
	FakeSeqLength  int    `json:"fake_seq_length" bson:"fake_seq_length"`
	FakeLen        int    `json:"fake_len" bson:"fake_len"`
	FakingStrategy string `json:"faking_strategy" bson:"faking_strategy"`
	DPortFilter    string `json:"dport_filter" bson:"dport_filter"` // can be a comma separated list of ports and port ranges, e.g. "80,443,1000-2000"
	FilterQUIC     string `json:"filter_quic" bson:"filter_quic"`
	FilterSTUN     bool   `json:"filter_stun" bson:"filter_stun"`
	ConnBytesLimit int    `json:"conn_bytes_limit" bson:"conn_bytes_limit"`
	Seg2Delay      int    `json:"seg2delay" bson:"seg2delay"`
}

type FragmentationConfig struct {
	Strategy     string `json:"strategy" bson:"strategy"` // Values: "tcp", "ip", "oob", "tls", "disorder", "overlap", "extsplit", "firstbyte", "combo", "none"
	ReverseOrder bool   `json:"reverse_order" bson:"reverse_order"`

	TLSRecordPosition int `json:"tlsrec_pos" bson:"tlsrec_pos"` // where to split TLS record

	MiddleSNI   bool `json:"middle_sni" bson:"middle_sni"`
	SNIPosition int  `json:"sni_position" bson:"sni_position"`

	OOBPosition int  `json:"oob_position" bson:"oob_position"` // Position for OOB (0=disabled)
	OOBChar     byte `json:"oob_char" bson:"oob_char"`         // Character for OOB data

	SeqOverlapPattern []string `json:"seq_overlap_pattern" bson:"seq_overlap_pattern"`
	SeqOverlapBytes   []byte   `json:"-" bson:"-"`

	Combo    ComboFragConfig    `json:"combo" bson:"combo"`
	Disorder DisorderFragConfig `json:"disorder" bson:"disorder"`
	Overlap  OverlapFragConfig  `json:"overlap" bson:"overlap"`
}

type FakingConfig struct {
	SNI           bool     `json:"sni" bson:"sni"`
	TTL           uint8    `json:"ttl" bson:"ttl"`
	Strategy      string   `json:"strategy" bson:"strategy"`
	SeqOffset     int32    `json:"seq_offset" bson:"seq_offset"`
	SNISeqLength  int      `json:"sni_seq_length" bson:"sni_seq_length"`
	SNIType       int      `json:"sni_type" bson:"sni_type"`
	CustomPayload string   `json:"custom_payload" bson:"custom_payload"`
	PayloadFile   string   `json:"payload_file" bson:"payload_file"`
	PayloadData   []byte   `json:"-" bson:"-"`
	TLSMod        []string `json:"tls_mod" bson:"tls_mod"` // e.g. ["rnd", "dupsid"]

	SNIMutation SNIMutationConfig `json:"sni_mutation" bson:"sni_mutation"`
}

type SNIMutationConfig struct {
	Mode         string   `json:"mode" bson:"mode"` // "off", "duplicate", "grease", "padding", "reorder", "full"
	GreaseCount  int      `json:"grease_count" bson:"grease_count"`
	PaddingSize  int      `json:"padding_size" bson:"padding_size"`
	FakeExtCount int      `json:"fake_ext_count" bson:"fake_ext_count"`
	FakeSNIs     []string `json:"fake_snis" bson:"fake_snis"` // Additional SNIs to inject
}

type TargetsConfig struct {
	SNIDomains        []string `json:"sni_domains" bson:"sni_domains"`
	IPs               []string `json:"ip" bson:"ip"`
	GeoSiteCategories []string `json:"geosite_categories" bson:"geosite_categories"`
	GeoIpCategories   []string `json:"geoip_categories" bson:"geoip_categories"`
	DomainsToMatch    []string `json:"-" bson:"-"`
	IpsToMatch        []string `json:"-" bson:"-"`
}

type SystemConfig struct {
	Tables    TablesConfig    `json:"tables" bson:"tables"`
	Logging   Logging         `json:"logging" bson:"logging"`
	WebServer WebServerConfig `json:"web_server" bson:"web_server"`
	Checker   DiscoveryConfig `json:"checker" bson:"checker"`
	Geo       GeoDatConfig    `json:"geo" bson:"geo"`
	API       ApiConfig       `json:"api" bson:"api"`
}

type TablesConfig struct {
	MonitorInterval int  `json:"monitor_interval" bson:"monitor_interval"`
	SkipSetup       bool `json:"skip_setup" bson:"skip_setup"`
}

type WebServerConfig struct {
	Port        int    `json:"port" bson:"port"`
	BindAddress string `json:"bind_address" bson:"bind_address"`
	IsEnabled   bool   `json:"-" bson:"-"`
}

type DiscoveryConfig struct {
	// Discovery settings
	DiscoveryTimeoutSec int      `yaml:"discovery_timeout" json:"discovery_timeout"`
	ConfigPropagateMs   int      `yaml:"config_propagate_ms" json:"config_propagate_ms"`
	ReferenceDomain     string   `yaml:"reference_domain" json:"reference_domain"`
	ReferenceDNS        []string `yaml:"reference_dns" json:"reference_dns"`
}

type Logging struct {
	Level      log.Level `json:"level" bson:"level"`
	Instaflush bool      `json:"instaflush" bson:"instaflush"`
	Syslog     bool      `json:"syslog" bson:"syslog"`
	ErrorFile  string    `json:"error_file" bson:"error_file"`
}

type SetConfig struct {
	Id            string              `json:"id" bson:"id"`
	Name          string              `json:"name" bson:"name"`
	TCP           TCPConfig           `json:"tcp" bson:"tcp"`
	UDP           UDPConfig           `json:"udp" bson:"udp"`
	Fragmentation FragmentationConfig `json:"fragmentation" bson:"fragmentation"`
	Faking        FakingConfig        `json:"faking" bson:"faking"`
	Targets       TargetsConfig       `json:"targets" bson:"targets"`
	Enabled       bool                `json:"enabled" bson:"enabled"`
	DNS           DNSConfig           `json:"dns" bson:"dns"`
}

type GeoDatConfig struct {
	GeoSitePath string `json:"sitedat_path" bson:"sitedat_path"`
	GeoIpPath   string `json:"ipdat_path" bson:"ipdat_path"`
	GeoSiteURL  string `json:"sitedat_url" bson:"sitedat_url"`
	GeoIpURL    string `json:"ipdat_url" bson:"ipdat_url"`
}

type ComboFragConfig struct {
	FirstByteSplit bool   `json:"first_byte_split" bson:"first_byte_split"`
	ExtensionSplit bool   `json:"extension_split" bson:"extension_split"`
	ShuffleMode    string `json:"shuffle_mode" bson:"shuffle_mode"` // "middle", "full", "reverse"
	FirstDelayMs   int    `json:"first_delay_ms" bson:"first_delay_ms"`
	JitterMaxUs    int    `json:"jitter_max_us" bson:"jitter_max_us"`
}

type DisorderFragConfig struct {
	ShuffleMode string `json:"shuffle_mode" bson:"shuffle_mode"` // "full", "reverse"
	MinJitterUs int    `json:"min_jitter_us" bson:"min_jitter_us"`
	MaxJitterUs int    `json:"max_jitter_us" bson:"max_jitter_us"`
}

type DNSConfig struct {
	Enabled       bool   `json:"enabled" bson:"enabled"`
	TargetDNS     string `json:"target_dns" bson:"target_dns"`
	FragmentQuery bool   `json:"fragment_query" bson:"fragment_query"`
}

type OverlapFragConfig struct {
	FakeSNIs []string `json:"fake_snis" bson:"fake_snis"`
}
