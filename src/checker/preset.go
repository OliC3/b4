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

	// 3. SYN Fake with Combined Strategies (fixed to match strategies)
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
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: fc.pos, SNIReverse: fc.reverse, MiddleSNI: fc.middle},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 6. Faking Strategies with SeqOffset variations
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

	// 7. UDP/QUIC with Port Filtering
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
		{"fake", 64, 8, "ttl", "parse", "443"},         // HTTPS only
		{"fake", 128, 10, "checksum", "all", "80,443"}, // HTTP+HTTPS
		{"drop", 0, 0, "none", "all", "443"},           // Drop QUIC on 443
		{"fake", 64, 6, "none", "disabled", ""},        // Allow STUN
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

	// 8. IP Fragmentation variations
	ipFragConfigs := []struct {
		pos     int
		reverse bool
	}{
		{1, false},
		{1, true},
		{8, false}, // Fragment at 8-byte boundary
		{20, true}, // Fragment after some data
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
				Fragmentation: config.FragmentationConfig{Strategy: "ip", SNIPosition: ifc.pos, SNIReverse: ifc.reverse},
				Faking:        config.FakingConfig{SNI: true, TTL: 8, Strategy: "pastseq", SeqOffset: 10000, SNISeqLength: 1, SNIType: 2},
			},
		})
	}

	// 9. Delay variations
	delays := []int{5, 10, 20, 50}
	for _, delay := range delays {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("delay-%d", delay),
			Description: fmt.Sprintf("Segment delay %dms", delay),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 19, Seg2Delay: delay},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8, Seg2Delay: delay},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1, SNIReverse: true},
				Faking:        config.FakingConfig{SNI: true, TTL: 5, Strategy: "randseq", SeqOffset: 50000, SNISeqLength: 3, SNIType: 2},
			},
		})
	}

	// 10. Aggressive combinations
	aggressiveConfigs := []struct {
		name   string
		synLen int
		udpSeq int
		ttl    uint8
	}{
		{"max", 256, 15, 3},
		{"ultra", 512, 20, 1},
	}

	for _, ac := range aggressiveConfigs {
		presets = append(presets, ConfigPreset{
			Name:        fmt.Sprintf("aggressive-%s", ac.name),
			Description: fmt.Sprintf("%s bypass: all techniques", ac.name),
			Config: config.SetConfig{
				TCP:           config.TCPConfig{ConnBytesLimit: 1, Seg2Delay: 10, SynFake: true, SynFakeLen: ac.synLen},
				UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: ac.udpSeq, FakeLen: 256, FakingStrategy: "checksum", FilterQUIC: "all", FilterSTUN: true, ConnBytesLimit: 1},
				Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1, SNIReverse: true, MiddleSNI: true},
				Faking:        config.FakingConfig{SNI: true, TTL: ac.ttl, Strategy: "pastseq", SeqOffset: 100000, SNISeqLength: 5, SNIType: 0}, // Random SNI type
			},
		})
	}

	// 11. Special edge cases
	presets = append(presets, ConfigPreset{
		Name:        "no-sni-fake",
		Description: "Fragmentation only, no fake SNI",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 6, FakeLen: 64, FakingStrategy: "none", FilterQUIC: "disabled", FilterSTUN: true, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1, SNIReverse: true, MiddleSNI: true},
			Faking:        config.FakingConfig{SNI: false}, // No fake SNI
		},
	})

	presets = append(presets, ConfigPreset{
		Name:        "fake-only",
		Description: "Fake packets only, no fragmentation",
		Config: config.SetConfig{
			TCP:           config.TCPConfig{ConnBytesLimit: 19},
			UDP:           config.UDPConfig{Mode: "fake", FakeSeqLength: 10, FakeLen: 128, FakingStrategy: "ttl", FilterQUIC: "parse", FilterSTUN: true, ConnBytesLimit: 8},
			Fragmentation: config.FragmentationConfig{Strategy: "none"}, // No fragmentation
			Faking:        config.FakingConfig{SNI: true, TTL: 3, Strategy: "pastseq", SeqOffset: 50000, SNISeqLength: 5, SNIType: 2},
		},
	})

	return presets
}
