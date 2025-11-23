package config

import "github.com/spf13/cobra"

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
	cmd.Flags().BoolVar(&c.MainSet.TCP.DropSACK, "tcp-drop-sack", c.MainSet.TCP.DropSACK, "Enable dropping SACK option from TCP packets (default false)")

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
	cmd.Flags().StringVar(&c.MainSet.Fragmentation.Strategy, "frag", c.MainSet.Fragmentation.Strategy, "Fragmentation strategy (tcp|ip|tls|oob|none)")
	cmd.Flags().BoolVar(&c.MainSet.Fragmentation.ReverseOrder, "frag-reverse", c.MainSet.Fragmentation.ReverseOrder, "Reverse fragment order")
	cmd.Flags().BoolVar(&c.MainSet.Fragmentation.MiddleSNI, "frag-middle-sni", c.MainSet.Fragmentation.MiddleSNI, "Fragment in middle of SNI")
	cmd.Flags().IntVar(&c.MainSet.Fragmentation.SNIPosition, "frag-sni-pos", c.MainSet.Fragmentation.SNIPosition, "SNI fragment position")
	cmd.Flags().IntVar(&c.MainSet.Fragmentation.OOBPosition, "frag-oob-pos", c.MainSet.Fragmentation.OOBPosition, "OOB data position")
	cmd.Flags().Uint8Var(&c.MainSet.Fragmentation.OOBChar, "frag-oob-char", c.MainSet.Fragmentation.OOBChar, "OOB character (ASCII code)")
	cmd.Flags().IntVar(&c.MainSet.Fragmentation.TLSRecordPosition, "frag-tlsrec-pos", c.MainSet.Fragmentation.TLSRecordPosition, "TLS record split position")

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
