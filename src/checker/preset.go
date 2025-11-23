package checker

import (
	"fmt"

	"github.com/daniellavrushin/b4/config"
)

type ConfigPreset struct {
	Name        string
	Description string
	Config      config.SetConfig
}

// GetTestPresets generates focused preset combinations
func GetTestPresets() []ConfigPreset {
	presets := []ConfigPreset{}

	// 1. BASELINE - Always test first to compare
	presets = append(presets, ConfigPreset{
		Name:        "baseline",
		Description: "No bypass techniques",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 0, FakeLen: 0, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: false, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "none"},
			Faking:        config.FakingConfig{SNI: false, TTL: 8, Strategy: "none", SeqOffset: 0, SNISeqLength: 0, SNIType: 2},
		},
	})

	// 2. ConnBytesLimit Variations (CRITICAL - when bypass triggers)
	connBytesConfigs := []struct {
		tcp int
		udp int
	}{
		{1, 1},    // Immediate bypass
		{10, 5},   // Early bypass
		{19, 8},   // Default
		{50, 25},  // Late bypass
		{100, 50}, // Very late
	}

	for _, cb := range connBytesConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("connbytes-tcp%d-udp%d", cb.tcp, cb.udp),
			Description: fmt.Sprintf("Trigger at TCP:%d UDP:%d bytes", cb.tcp, cb.udp),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: cb.tcp},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: cb.udp},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 3. SYN Fake with Combined Strategies
	synConfigs := []struct {
		synLen   int
		strategy string
		ttl      uint8
	}{
		{64, "ttl", 3},
		{64, "pastseq", 8},
		{256, "randseq", 5},
		{256, "tcp_check", 8},
		{512, "md5sum", 3},
	}

	for _, sc := range synConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("syn-%d-%s", sc.synLen, sc.strategy),
			Description: fmt.Sprintf("SYN fake len=%d with %s", sc.synLen, sc.strategy),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19, SynFake: true, SynFakeLen: sc.synLen},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1},
				Faking:        config.FakingConfig{SNI: true, TTL: sc.ttl, Strategy: sc.strategy, SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 4. SNIType Variations (IMPORTANT - different payload types)
	sniTypeConfigs := []struct {
		sniType int
		payload string
		name    string
	}{
		{0, "", "random"},
		{1, "GET / HTTP/1.1\r\nHost: ", "http"},
		{1, "\x00\x00\x00\x00\x00\x00\x00\x00", "nulls"},
		{1, "\xff\xff\xff\xff\xff\xff\xff\xff", "ones"},
		{1, "CONNECT ", "connect"},
		{2, "", "default"},
	}

	for _, st := range sniTypeConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("snitype-%s", st.name),
			Description: fmt.Sprintf("SNI payload type: %s", st.name),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 2, SNIType: st.sniType, CustomPayload: st.payload},
			},
		})
	}

	// 5. TCP Fragmentation with Extreme Positions
	fragConfigs := []struct {
		pos     int
		reverse bool
		middle  bool
	}{
		{1, false, false},  // Very early
		{1, true, false},   // Reverse
		{1, false, true},   // Middle SNI split
		{3, false, false},  // After version
		{11, false, false}, // After TLS header
		{50, false, false}, // Deep in handshake
		{100, true, false}, // Very deep + reverse
	}

	for _, fc := range fragConfigs {
		name := fmt.Sprintf("tcp-pos%d", fc.pos)
		if fc.reverse {
			name += "-rev"
		}
		if fc.middle {
			name += "-mid"
		}

		presets = append(presets, ConfigPreset{
			Name:        name,
			Description: fmt.Sprintf("TCP frag pos=%d reverse=%v middle=%v", fc.pos, fc.reverse, fc.middle),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19, Seg2Delay: 0},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: fc.pos, ReverseOrder: fc.reverse, MiddleSNI: fc.middle},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 6. OOB (Out-of-Band) Variations - NEW
	oobConfigs := []struct {
		pos     int
		reverse bool
		char    byte
		name    string
	}{
		{1, false, 'x', "pos1"},
		{1, true, 'x', "pos1-rev"},
		{2, false, 'x', "pos2"},
		{3, false, 'x', "pos3"},
		{5, false, 'x', "pos5"},
		{1, false, 'a', "pos1-char-a"},
		{1, false, 0x00, "pos1-null"},
		{1, false, 0xFF, "pos1-xff"},
		{2, true, 'y', "pos2-rev-y"},
		{5, true, 'x', "pos5-rev"},
	}

	for _, oc := range oobConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("oob-%s", oc.name),
			Description: fmt.Sprintf("OOB pos=%d reverse=%v char=%#x", oc.pos, oc.reverse, oc.char),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "oob", OOBPosition: oc.pos, ReverseOrder: oc.reverse, OOBChar: oc.char},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 7. OOB + Different Faking Strategies
	oobFakeConfigs := []struct {
		strategy string
		ttl      uint8
		seqLen   int
		pos      int
	}{
		{"ttl", 3, 1, 1},
		{"ttl", 5, 2, 2},
		{"pastseq", 8, 2, 1},
		{"pastseq", 5, 3, 3},
		{"randseq", 8, 1, 1},
		{"tcp_check", 8, 2, 2},
	}

	for _, ofc := range oobFakeConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("oob-fake-%s-pos%d", ofc.strategy, ofc.pos),
			Description: fmt.Sprintf("OOB pos=%d + fake %s seqLen=%d", ofc.pos, ofc.strategy, ofc.seqLen),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "oob", OOBPosition: ofc.pos, ReverseOrder: false, OOBChar: 'x'},
				Faking:        config.FakingConfig{SNI: true, TTL: ofc.ttl, Strategy: ofc.strategy, SeqOffset: 10000, SNISeqLength: ofc.seqLen, SNIType: 2},
			},
		})
	}

	// 7a. SACK Dropping Variations
	sackConfigs := []struct {
		strategy string
		fragPos  int
		reverse  bool
	}{
		{"tcp", 1, false},
		{"tcp", 1, true},
		{"tcp", 3, false},
		{"ip", 1, false},
		{"oob", 1, false},
		{"oob", 2, true},
	}

	for _, sc := range sackConfigs {
		name := fmt.Sprintf("sack-%s", sc.strategy)
		if sc.reverse {
			name += "-rev"
		}
		if sc.fragPos > 1 {
			name += fmt.Sprintf("-pos%d", sc.fragPos)
		}

		fragConfig := config.FragmentationConfig{Strategy: sc.strategy, SNIPosition: sc.fragPos, ReverseOrder: sc.reverse}
		if sc.strategy == "oob" {
			fragConfig.OOBPosition = sc.fragPos
			fragConfig.OOBChar = 'x'
		}

		presets = append(presets, ConfigPreset{
			Name:        name,
			Description: fmt.Sprintf("SACK drop + %s frag pos=%d", sc.strategy, sc.fragPos),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19, DropSACK: true},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: fragConfig,
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 7b. SACK + SYN Fake combinations
	presets = append(presets, ConfigPreset{
		Name:        "sack-syn-aggressive",
		Description: "SACK drop + SYN fake + TCP frag",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19, DropSACK: true, SynFake: true, SynFakeLen: 256},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1, ReverseOrder: true},
			Faking:        config.FakingConfig{SNI: true, TTL: 5, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 2, SNIType: 2},
		},
	})

	presets = append(presets, ConfigPreset{
		Name:        "sack-oob-ultra",
		Description: "SACK drop + OOB + immediate trigger",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 1, DropSACK: true},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 1},
			Fragmentation: config.FragmentationConfig{Strategy: "oob", OOBPosition: 1, ReverseOrder: true, OOBChar: 'x'},
			Faking:        config.FakingConfig{SNI: true, TTL: 3, Strategy: "pastseq", SeqOffset: 50000, SNISeqLength: 3, SNIType: 2},
		},
	})

	// 8. OOB + Early Triggering
	presets = append(presets, ConfigPreset{
		Name:        "oob-immediate",
		Description: "OOB with immediate trigger (connbytes=1)",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 1},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 1},
			Fragmentation: config.FragmentationConfig{Strategy: "oob", OOBPosition: 1, ReverseOrder: false, OOBChar: 'x'},
			Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 2, SNIType: 2},
		},
	})

	// 9. Faking Strategies with SeqOffset variations
	fakeConfigs := []struct {
		strategy string
		ttl      uint8
		seqLen   int
		offset   int32
	}{
		{"ttl", 3, 1, 0},
		{"ttl", 5, 3, 0},
		{"ttl", 8, 5, 0},
		{"pastseq", 8, 1, 10000},
		{"pastseq", 5, 2, 50000},
		{"pastseq", 3, 3, 100000},
		{"randseq", 8, 1, 10000},
		{"randseq", 5, 2, 100000},
		{"md5sum", 8, 1, 0},
		{"tcp_check", 8, 2, 0},
	}

	for _, fc := range fakeConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("fake-%s-ttl%d-len%d", fc.strategy, fc.ttl, fc.seqLen),
			Description: fmt.Sprintf("Fake %s TTL=%d seqLen=%d", fc.strategy, fc.ttl, fc.seqLen),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1},
				Faking:        config.FakingConfig{SNI: true, TTL: fc.ttl, Strategy: fc.strategy, SeqOffset: fc.offset, SNISeqLength: fc.seqLen, SNIType: 2},
			},
		})
	}

	// 10. UDP/QUIC with Port Filtering
	udpConfigs := []struct {
		mode       string
		fakeLen    int
		fakeSeq    int
		strategy   string
		quicFilter string
		dPort      string
	}{
		{"fake", 64, 6, "ttl", "disabled", ""},
		{"fake", 128, 10, "checksum", "parse", ""},
		{"fake", 256, 12, "none", "all", ""},
		{"fake", 64, 8, "ttl", "parse", "443"},
		{"fake", 128, 10, "checksum", "all", "80,443"},
		{"drop", 0, 0, "none", "all", "443"},
		{"fake", 64, 6, "none", "disabled", ""},
	}

	for _, uc := range udpConfigs {
		name := fmt.Sprintf("udp-%s", uc.mode)
		if uc.quicFilter != "" {
			name += fmt.Sprintf("-q%s", uc.quicFilter)
		}
		if uc.dPort != "" {
			name += fmt.Sprintf("-p%s", uc.dPort)
		}

		presets = append(presets, ConfigPreset{
			Name:        name,
			Description: fmt.Sprintf("UDP %s QUIC=%s ports=%s", uc.mode, uc.quicFilter, uc.dPort),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19},
				UDP:           config.UDPConfig{Mode: uc.mode, FakeSeqLength: uc.fakeSeq, FakeLen: uc.fakeLen, FakingStrategy: uc.strategy, FilterQUIC: uc.quicFilter, FilterSTUN: uc.dPort == "", ConnBytesLimit: 8, DPortFilter: uc.dPort},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 11. IP Fragmentation variations
	ipFragConfigs := []struct {
		pos     int
		reverse bool
	}{
		{1, false},
		{1, true},
		{8, false},
		{20, true},
	}

	for _, ifc := range ipFragConfigs {
		name := fmt.Sprintf("ip-frag-pos%d", ifc.pos)
		if ifc.reverse {
			name += "-rev"
		}

		presets = append(presets, ConfigPreset{
			Name:        name,
			Description: fmt.Sprintf("IP frag at %d reverse=%v", ifc.pos, ifc.reverse),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "ip", SNIPosition: ifc.pos, ReverseOrder: ifc.reverse},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 11a. TLS Record Splitting Variations
	tlsConfigs := []struct {
		pos     int
		reverse bool
		name    string
	}{
		{1, false, "early"},
		{1, true, "early-rev"},
		{5, false, "mid"},
		{5, true, "mid-rev"},
		{10, false, "deep"},
		{20, false, "late"},
		{50, true, "late-rev"},
		{100, false, "extreme"},
	}

	for _, tc := range tlsConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("tls-%s", tc.name),
			Description: fmt.Sprintf("TLS record split at %d bytes reverse=%v", tc.pos, tc.reverse),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: tc.pos, ReverseOrder: tc.reverse},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 11b. TLS + Faking Strategy combinations
	tlsFakeConfigs := []struct {
		tlsPos   int
		strategy string
		ttl      uint8
		seqLen   int
	}{
		{1, "ttl", 3, 1},
		{5, "ttl", 5, 2},
		{1, "pastseq", 8, 2},
		{10, "pastseq", 5, 3},
		{1, "randseq", 8, 1},
		{5, "tcp_check", 8, 2},
		{20, "md5sum", 8, 1},
	}

	for _, tfc := range tlsFakeConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("tls-pos%d-%s", tfc.tlsPos, tfc.strategy),
			Description: fmt.Sprintf("TLS pos=%d + fake %s TTL=%d", tfc.tlsPos, tfc.strategy, tfc.ttl),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: tfc.tlsPos, ReverseOrder: false},
				Faking:        config.FakingConfig{SNI: true, TTL: tfc.ttl, Strategy: tfc.strategy, SeqOffset: 10000, SNISeqLength: tfc.seqLen, SNIType: 2},
			},
		})
	}

	// 11c. TLS + SYN Fake combinations
	tlsSynConfigs := []struct {
		tlsPos int
		synLen int
		ttl    uint8
	}{
		{1, 64, 3},
		{5, 256, 5},
		{10, 512, 8},
	}

	for _, tsc := range tlsSynConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("tls-syn-pos%d-len%d", tsc.tlsPos, tsc.synLen),
			Description: fmt.Sprintf("TLS pos=%d + SYN fake len=%d", tsc.tlsPos, tsc.synLen),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19, SynFake: true, SynFakeLen: tsc.synLen},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
				Fragmentation: config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: tsc.tlsPos, ReverseOrder: true},
				Faking:        config.FakingConfig{SNI: true, TTL: tsc.ttl, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 2, SNIType: 2},
			},
		})
	}

	// 11d. TLS + SACK combinations
	presets = append(presets, ConfigPreset{
		Name:        "tls-sack-basic",
		Description: "TLS record split + SACK drop",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19, DropSACK: true},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: 1, ReverseOrder: false},
			Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
		},
	})

	presets = append(presets, ConfigPreset{
		Name:        "tls-sack-aggressive",
		Description: "TLS + SACK + SYN fake + immediate trigger",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 1, DropSACK: true, SynFake: true, SynFakeLen: 256},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 10, FakeLen: 128, FakingStrategy: "checksum", FilterQUIC: "all", FilterSTUN: true, ConnBytesLimit: 1},
			Fragmentation: config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: 5, ReverseOrder: true},
			Faking:        config.FakingConfig{SNI: true, TTL: 3, Strategy: "randseq", SeqOffset: 100000, SNISeqLength: 3, SNIType: 0},
		},
	})

	// 12. Delay variations
	delays := []int{5, 10, 20, 50}
	for _, delay := range delays {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("delay-%d", delay),
			Description: fmt.Sprintf("Segment delay %dms", delay),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19, Seg2Delay: delay},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8, Seg2Delay: delay},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1, ReverseOrder: true},
				Faking:        config.FakingConfig{SNI: true, TTL: 5, Strategy: "randseq", SeqOffset: 50000, SNISeqLength: 3, SNIType: 2},
			},
		})
	}

	// 13. Aggressive combinations
	aggressiveConfigs := []struct {
		name   string
		synLen int
		udpSeq int
		ttl    uint8
		strat  string
	}{
		{"max-tcp", 256, 15, 3, "tcp"},
		{"max-oob", 512, 20, 1, "oob"},
		{"ultra-tcp", 512, 20, 1, "tcp"},
		{"ultra-oob", 512, 25, 1, "oob"},
		{"max-tls", 256, 15, 3, "tls"},
		{"max-tls-sack", 512, 20, 1, "tls"},
		{"ultra-tls", 512, 20, 1, "tls"},
	}

	for _, ac := range aggressiveConfigs {
		var fragConfig config.FragmentationConfig
		var tcpConfig config.TCPConfig

		switch ac.strat {
		case "oob":
			fragConfig = config.FragmentationConfig{Strategy: "oob", OOBPosition: 1, ReverseOrder: true, OOBChar: 'x'}
			tcpConfig = config.TCPConfig{ConnBytesLimit: 1, Seg2Delay: 10, SynFake: true, SynFakeLen: ac.synLen}
		case "tls":
			fragConfig = config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: 5, ReverseOrder: true}
			tcpConfig = config.TCPConfig{ConnBytesLimit: 1, Seg2Delay: 10, SynFake: true, SynFakeLen: ac.synLen}
			// Add SACK for "sack" variants
			if len(ac.name) > 4 && ac.name[len(ac.name)-4:] == "sack" {
				tcpConfig.DropSACK = true
			}
		default:
			fragConfig = config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1, ReverseOrder: true, MiddleSNI: true}
			tcpConfig = config.TCPConfig{ConnBytesLimit: 1, Seg2Delay: 10, SynFake: true, SynFakeLen: ac.synLen}
		}

		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("aggressive-%s", ac.name),
			Description: fmt.Sprintf("%s bypass: all techniques", ac.name),
			Config: config.SetConfig{
				TCP:           tcpConfig,
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: ac.udpSeq, FakeLen: 256, FakingStrategy: "checksum", FilterQUIC: "all", FilterSTUN: true, ConnBytesLimit: 1},
				Fragmentation: fragConfig,
				Faking:        config.FakingConfig{SNI: true, TTL: ac.ttl, Strategy: "pastseq", SeqOffset: 100000, SNISeqLength: 5, SNIType: 0},
			},
		})
	}

	// 14. Special edge cases
	presets = append(presets, ConfigPreset{
		Name:        "tls-only",
		Description: "TLS record split only, no fake SNI",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: 1, ReverseOrder: false},
			Faking:        config.FakingConfig{SNI: false},
		},
	})

	presets = append(presets, ConfigPreset{
		Name:        "sack-only",
		Description: "SACK drop only, no fragmentation",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19, DropSACK: true},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 0, FakeLen: 0, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "none"},
			Faking:        config.FakingConfig{SNI: false},
		},
	})

	presets = append(presets, ConfigPreset{
		Name:        "tls-sack-only",
		Description: "TLS + SACK only, no fake SNI",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19, DropSACK: true},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 0, FakeLen: 0, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: 5, ReverseOrder: false},
			Faking:        config.FakingConfig{SNI: false},
		},
	})

	presets = append(presets, ConfigPreset{
		Name:        "oob-only",
		Description: "OOB only, no fake SNI",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "oob", OOBPosition: 1, ReverseOrder: false, OOBChar: 'x'},
			Faking:        config.FakingConfig{SNI: false},
		},
	})

	return presets
}
