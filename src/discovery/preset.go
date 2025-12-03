package discovery

import (
	"fmt"

	"github.com/daniellavrushin/b4/config"
)

// GetPhase1Presets returns minimal presets for strategy family detection
// These are the "does this approach work at all?" tests
// IMPORTANT: Most DPI requires COMBINATIONS of techniques, not single techniques
func GetPhase1Presets() []ConfigPreset {
	return []ConfigPreset{

		// 0. Raw baseline - no bypass at all (to detect if DPI even blocks)
		{
			Name:        "no-bypass",
			Description: "No bypass techniques - test raw connectivity",
			Family:      FamilyNone,
			Phase:       PhaseBaseline,
			Priority:    0,
			Config:      baselineConfig(),
		},

		// 1. Proven working config - this is the baseline that works for most Russian DPI
		{
			Name:        "proven-combo",
			Description: "Proven combination: TCP frag + reverse + middle SNI + fake pastseq",
			Family:      FamilyNone,
			Phase:       PhaseBaseline,
			Priority:    1,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
				},
				UDP: config.UDPConfig{
					Mode:           "fake",
					FakeSeqLength:  6,
					FakeLen:        64,
					FakingStrategy: "none",
					FilterQUIC:     "disabled",
					FilterSTUN:     true,
					ConnBytesLimit: 8,
				},
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					ReverseOrder: true,
					MiddleSNI:    true,
					SNIPosition:  1,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 1b. Proven config with alternate payload
		{
			Name:        "proven-combo-alt",
			Description: "Proven combination with alternate fake payload",
			Family:      FamilyNone,
			Phase:       PhaseBaseline,
			Priority:    1,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
				},
				UDP: config.UDPConfig{
					Mode:           "fake",
					FakeSeqLength:  6,
					FakeLen:        64,
					FakingStrategy: "none",
					FilterQUIC:     "disabled",
					FilterSTUN:     true,
					ConnBytesLimit: 8,
				},
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					ReverseOrder: true,
					MiddleSNI:    true,
					SNIPosition:  1,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault2,
				},
			},
		},

		// 2. TCP Frag + Fake (common combo)
		{
			Name:        "tcp-frag-fake",
			Description: "TCP fragmentation with fake SNI",
			Family:      FamilyTCPFrag,
			Phase:       PhaseStrategy,
			Priority:    2,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:    "tcp",
					SNIPosition: 1,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 3. TCP Frag + Reverse + Fake
		{
			Name:        "tcp-frag-rev-fake",
			Description: "TCP frag reverse order with fake SNI",
			Family:      FamilyTCPFrag,
			Phase:       PhaseStrategy,
			Priority:    3,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 4. TLS Record + Fake
		{
			Name:        "tls-rec-fake",
			Description: "TLS record splitting with fake SNI",
			Family:      FamilyTLSRec,
			Phase:       PhaseStrategy,
			Priority:    4,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:          "tls",
					TLSRecordPosition: 1,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 5. OOB + Fake
		{
			Name:        "oob-fake",
			Description: "Out-of-band with fake SNI",
			Family:      FamilyOOB,
			Phase:       PhaseStrategy,
			Priority:    5,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:    "oob",
					OOBPosition: 1,
					OOBChar:     'x',
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 6. Fake only (low TTL)
		{
			Name:        "fake-ttl-low",
			Description: "Fake SNI with low TTL (no fragmentation)",
			Family:      FamilyFakeSNI,
			Phase:       PhaseStrategy,
			Priority:    6,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy: "none",
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          3,
					Strategy:     "ttl",
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 7. SACK Drop + TCP Frag + Fake
		{
			Name:        "sack-frag-fake",
			Description: "SACK drop with TCP frag and fake",
			Family:      FamilySACK,
			Phase:       PhaseStrategy,
			Priority:    7,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
					DropSACK:       true,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 8. Desync RST + Frag
		{
			Name:        "desync-rst-frag",
			Description: "TCP desync RST attack with fragmentation",
			Family:      FamilyDesync,
			Phase:       PhaseStrategy,
			Priority:    8,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
					DesyncMode:     "rst",
					DesyncTTL:      3,
					DesyncCount:    3,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 9. Desync Combo (RST+FIN+ACK flood)
		{
			Name:        "desync-combo",
			Description: "TCP desync combo attack",
			Family:      FamilyDesync,
			Phase:       PhaseStrategy,
			Priority:    9,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
					DesyncMode:     "combo",
					DesyncTTL:      2,
					DesyncCount:    5,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
					MiddleSNI:    true,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          3,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 2,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 10. SYN Fake with TCP frag
		{
			Name:        "synfake-frag",
			Description: "SYN fake packets with fragmentation",
			Family:      FamilySynFake,
			Phase:       PhaseStrategy,
			Priority:    10,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
					SynFake:        true,
					SynFakeLen:     256,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 11. Seg2Delay with fragmentation (timing-based)
		{
			Name:        "delay-frag",
			Description: "Delayed segments with fragmentation",
			Family:      FamilyDelay,
			Phase:       PhaseStrategy,
			Priority:    11,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
					Seg2Delay:      10,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
					MiddleSNI:    true,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 12. Very low TTL fake (TTL=1-2)
		{
			Name:        "fake-ttl-ultra-low",
			Description: "Fake SNI with ultra-low TTL",
			Family:      FamilyFakeSNI,
			Phase:       PhaseStrategy,
			Priority:    12,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          1,
					Strategy:     "ttl",
					SNISeqLength: 3,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},

		// 13. Full desync attack
		{
			Name:        "desync-full",
			Description: "Full desync attack sequence",
			Family:      FamilyDesync,
			Phase:       PhaseStrategy,
			Priority:    13,
			Config: config.SetConfig{
				TCP: config.TCPConfig{
					ConnBytesLimit: 19,
					DesyncMode:     "full",
					DesyncTTL:      3,
					DesyncCount:    5,
				},
				UDP: defaultUDP(),
				Fragmentation: config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
					MiddleSNI:    true,
				},
				Faking: config.FakingConfig{
					SNI:          true,
					TTL:          3,
					Strategy:     "pastseq",
					SeqOffset:    50000,
					SNISeqLength: 3,
					SNIType:      config.FakePayloadDefault1,
				},
			},
		},
	}
}

func defaultUDP() config.UDPConfig {
	return config.UDPConfig{
		Mode:           "fake",
		FakeSeqLength:  6,
		FakeLen:        64,
		FakingStrategy: "none",
		FilterQUIC:     "disabled",
		FilterSTUN:     true,
		ConnBytesLimit: 8,
	}
}

// GetPhase2Presets generates optimization presets for a specific working family
func GetPhase2Presets(family StrategyFamily) []ConfigPreset {
	base := baseConfig()
	presets := []ConfigPreset{}

	switch family {
	case FamilyTCPFrag:
		positions := []int{1, 2, 3, 5, 10}
		for _, pos := range positions {
			for _, reverse := range []bool{false, true} {
				name := formatName("tcp-pos%d", pos)
				if reverse {
					name += "-rev"
				}
				presets = append(presets, ConfigPreset{
					Name:     name,
					Family:   FamilyTCPFrag,
					Phase:    PhaseOptimize,
					Priority: pos,
					Config: withFragmentation(base, config.FragmentationConfig{
						Strategy:     "tcp",
						SNIPosition:  pos,
						ReverseOrder: reverse,
					}),
				})
			}
		}
		// Add middle SNI variant
		presets = append(presets, ConfigPreset{
			Name:     "tcp-middle-sni",
			Family:   FamilyTCPFrag,
			Phase:    PhaseOptimize,
			Priority: 10,
			Config: withFragmentation(base, config.FragmentationConfig{
				Strategy:    "tcp",
				SNIPosition: 1,
				MiddleSNI:   true,
			}),
		})

	case FamilyTLSRec:
		positions := []int{1, 5, 10, 20, 50}
		for _, pos := range positions {
			for _, reverse := range []bool{false, true} {
				name := formatName("tls-pos%d", pos)
				if reverse {
					name += "-rev"
				}
				presets = append(presets, ConfigPreset{
					Name:     name,
					Family:   FamilyTLSRec,
					Phase:    PhaseOptimize,
					Priority: pos,
					Config: withFragmentation(base, config.FragmentationConfig{
						Strategy:          "tls",
						TLSRecordPosition: pos,
						ReverseOrder:      reverse,
					}),
				})
			}
		}

	case FamilyOOB:
		positions := []int{1, 2, 3, 5}
		chars := []byte{'x', 'a', 0x00, 0xFF}
		for _, pos := range positions {
			for _, ch := range chars {
				name := formatName("oob-pos%d-0x%02x", pos, ch)
				presets = append(presets, ConfigPreset{
					Name:     name,
					Family:   FamilyOOB,
					Phase:    PhaseOptimize,
					Priority: pos,
					Config: withFragmentation(base, config.FragmentationConfig{
						Strategy:    "oob",
						OOBPosition: pos,
						OOBChar:     ch,
					}),
				})
			}
		}

	case FamilyFakeSNI:
		// TTL variations
		ttls := []uint8{1, 2, 3, 5, 8}
		for _, ttl := range ttls {
			presets = append(presets, ConfigPreset{
				Name:     formatName("fake-ttl%d", ttl),
				Family:   FamilyFakeSNI,
				Phase:    PhaseOptimize,
				Priority: int(ttl),
				Config: withFaking(base, config.FakingConfig{
					SNI:          true,
					TTL:          ttl,
					Strategy:     "ttl",
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				}),
			})
		}

		// Sequence length variations
		seqLens := []int{1, 2, 3, 5}
		for _, sl := range seqLens {
			presets = append(presets, ConfigPreset{
				Name:     formatName("fake-seq%d", sl),
				Family:   FamilyFakeSNI,
				Phase:    PhaseOptimize,
				Priority: sl + 10,
				Config: withFaking(base, config.FakingConfig{
					SNI:          true,
					TTL:          3,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: sl,
					SNIType:      config.FakePayloadDefault1,
				}),
			})
		}

		// Strategy variations
		strategies := []string{"ttl", "pastseq", "randseq", "tcp_check", "md5sum"}
		for i, strat := range strategies {
			presets = append(presets, ConfigPreset{
				Name:     formatName("fake-%s", strat),
				Family:   FamilyFakeSNI,
				Phase:    PhaseOptimize,
				Priority: i + 20,
				Config: withFaking(base, config.FakingConfig{
					SNI:          true,
					TTL:          3,
					Strategy:     strat,
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      config.FakePayloadDefault1,
				}),
			})
		}

		// Payload type variations
		payloadTypes := []struct {
			name    string
			sniType int
		}{
			{"payload1", config.FakePayloadDefault1},
			{"payload2", config.FakePayloadDefault2},
		}
		for _, pt := range payloadTypes {
			presets = append(presets, ConfigPreset{
				Name:     formatName("fake-%s", pt.name),
				Family:   FamilyFakeSNI,
				Phase:    PhaseOptimize,
				Priority: 30 + pt.sniType,
				Config: withFaking(base, config.FakingConfig{
					SNI:          true,
					TTL:          8,
					Strategy:     "pastseq",
					SeqOffset:    10000,
					SNISeqLength: 1,
					SNIType:      pt.sniType,
				}),
			})
		}

	case FamilyIPFrag:
		positions := []int{1, 8, 16, 24}
		for _, pos := range positions {
			for _, reverse := range []bool{false, true} {
				name := formatName("ip-pos%d", pos)
				if reverse {
					name += "-rev"
				}
				presets = append(presets, ConfigPreset{
					Name:     name,
					Family:   FamilyIPFrag,
					Phase:    PhaseOptimize,
					Priority: pos,
					Config: withFragmentation(base, config.FragmentationConfig{
						Strategy:     "ip",
						SNIPosition:  pos,
						ReverseOrder: reverse,
					}),
				})
			}
		}

	case FamilySACK:
		// SACK + different fragmentation strategies
		fragStrategies := []string{"tcp", "tls", "oob"}
		for i, fs := range fragStrategies {
			cfg := withTCP(base, config.TCPConfig{
				ConnBytesLimit: 19,
				DropSACK:       true,
			})
			switch fs {
			case "tcp":
				cfg = withFragmentation(cfg, config.FragmentationConfig{Strategy: "tcp", SNIPosition: 1})
			case "tls":
				cfg = withFragmentation(cfg, config.FragmentationConfig{Strategy: "tls", TLSRecordPosition: 1})
			case "oob":
				cfg = withFragmentation(cfg, config.FragmentationConfig{Strategy: "oob", OOBPosition: 1, OOBChar: 'x'})
			}
			presets = append(presets, ConfigPreset{
				Name:     formatName("sack-%s", fs),
				Family:   FamilySACK,
				Phase:    PhaseOptimize,
				Priority: i,
				Config:   cfg,
			})
		}

	case FamilyDesync:
		modes := []string{"rst", "fin", "ack", "combo", "full"}
		ttls := []uint8{1, 2, 3, 5}
		counts := []int{2, 3, 5}

		for _, mode := range modes {
			for _, ttl := range ttls {
				for _, count := range counts {
					presets = append(presets, ConfigPreset{
						Name:     formatName("desync-%s-ttl%d-c%d", mode, ttl, count),
						Family:   FamilyDesync,
						Phase:    PhaseOptimize,
						Priority: int(ttl),
						Config: withTCP(withFragmentation(base, config.FragmentationConfig{
							Strategy:     "tcp",
							SNIPosition:  1,
							ReverseOrder: true,
						}), config.TCPConfig{
							ConnBytesLimit: 19,
							DesyncMode:     mode,
							DesyncTTL:      ttl,
							DesyncCount:    count,
						}),
					})
				}
			}
		}

	case FamilySynFake:
		lengths := []int{0, 64, 128, 256, 512}
		for _, l := range lengths {
			presets = append(presets, ConfigPreset{
				Name:     formatName("synfake-len%d", l),
				Family:   FamilySynFake,
				Phase:    PhaseOptimize,
				Priority: l,
				Config: withTCP(withFragmentation(base, config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
				}), config.TCPConfig{
					ConnBytesLimit: 19,
					SynFake:        true,
					SynFakeLen:     l,
				}),
			})
		}

	case FamilyDelay:
		delays := []int{1, 5, 10, 20, 50, 100}
		for _, d := range delays {
			presets = append(presets, ConfigPreset{
				Name:     formatName("delay-%dms", d),
				Family:   FamilyDelay,
				Phase:    PhaseOptimize,
				Priority: d,
				Config: withTCP(withFragmentation(base, config.FragmentationConfig{
					Strategy:     "tcp",
					SNIPosition:  1,
					ReverseOrder: true,
					MiddleSNI:    true,
				}), config.TCPConfig{
					ConnBytesLimit: 19,
					Seg2Delay:      d,
				}),
			})
		}
	}

	return presets
}

// GetCombinationPresets generates presets combining multiple working families
func GetCombinationPresets(workingFamilies []StrategyFamily, bestParams map[StrategyFamily]ConfigPreset) []ConfigPreset {
	presets := []ConfigPreset{}

	// If we have both fragmentation and faking working, combine them
	hasFrag := containsFamily(workingFamilies, FamilyTCPFrag) || containsFamily(workingFamilies, FamilyTLSRec) || containsFamily(workingFamilies, FamilyOOB)
	hasFake := containsFamily(workingFamilies, FamilyFakeSNI)
	hasSACK := containsFamily(workingFamilies, FamilySACK)

	base := baseConfig()

	if hasFrag && hasFake {
		// Combine best frag with best fake
		var fragConfig config.FragmentationConfig
		var fakingConfig config.FakingConfig

		// Get best fragmentation params
		for _, fam := range []StrategyFamily{FamilyTCPFrag, FamilyTLSRec, FamilyOOB} {
			if bp, ok := bestParams[fam]; ok {
				fragConfig = bp.Config.Fragmentation
				break
			}
		}

		// Get best faking params
		if bp, ok := bestParams[FamilyFakeSNI]; ok {
			fakingConfig = bp.Config.Faking
		}

		combined := withFragmentation(base, fragConfig)
		combined = withFaking(combined, fakingConfig)

		presets = append(presets, ConfigPreset{
			Name:        "combo-frag-fake",
			Description: "Combined fragmentation + fake SNI",
			Family:      FamilyNone,
			Phase:       PhaseCombination,
			Priority:    1,
			Config:      combined,
		})
	}

	if hasSACK && hasFrag {
		// SACK + fragmentation
		var fragConfig config.FragmentationConfig
		for _, fam := range []StrategyFamily{FamilyTCPFrag, FamilyTLSRec, FamilyOOB} {
			if bp, ok := bestParams[fam]; ok {
				fragConfig = bp.Config.Fragmentation
				break
			}
		}

		combined := withTCP(base, config.TCPConfig{ConnBytesLimit: 19, DropSACK: true})
		combined = withFragmentation(combined, fragConfig)

		presets = append(presets, ConfigPreset{
			Name:        "combo-sack-frag",
			Description: "SACK drop + fragmentation",
			Family:      FamilyNone,
			Phase:       PhaseCombination,
			Priority:    2,
			Config:      combined,
		})
	}

	// Aggressive combo - everything together
	if len(workingFamilies) >= 2 {
		aggressive := config.SetConfig{
			TCP: config.TCPConfig{
				ConnBytesLimit: 1,
				Seg2Delay:      5,
				DropSACK:       hasSACK,
				SynFake:        true,
				SynFakeLen:     256,
			},
			UDP: config.UDPConfig{
				Mode:           "fake",
				FakeSeqLength:  10,
				FakeLen:        128,
				FakingStrategy: "checksum",
				FilterQUIC:     "all",
				FilterSTUN:     true,
				ConnBytesLimit: 1,
			},
			Fragmentation: config.FragmentationConfig{
				Strategy:     "tcp",
				SNIPosition:  1,
				ReverseOrder: true,
				MiddleSNI:    true,
			},
			Faking: config.FakingConfig{
				SNI:          true,
				TTL:          3,
				Strategy:     "pastseq",
				SeqOffset:    50000,
				SNISeqLength: 3,
				SNIType:      config.FakePayloadDefault1,
			},
		}

		presets = append(presets, ConfigPreset{
			Name:        "aggressive",
			Description: "All bypass techniques combined",
			Family:      FamilyNone,
			Phase:       PhaseCombination,
			Priority:    10,
			Config:      aggressive,
		})
	}

	return presets
}

// Helper functions

func baseConfig() config.SetConfig {
	return config.NewSetConfig()
}

func baselineConfig() config.SetConfig {
	return config.SetConfig{
		Enabled: false,
		TCP: config.TCPConfig{
			ConnBytesLimit: 19,
		},
		UDP: config.UDPConfig{
			Mode:           "fake",
			FakeSeqLength:  0,
			FakeLen:        0,
			FakingStrategy: "none",
			FilterQUIC:     "disabled",
			FilterSTUN:     false,
			ConnBytesLimit: 8,
		},
		Fragmentation: config.FragmentationConfig{
			Strategy: "none",
		},
		Faking: config.FakingConfig{
			SNI: false,
		},
	}
}

func withFragmentation(base config.SetConfig, frag config.FragmentationConfig) config.SetConfig {
	base.Fragmentation = frag
	return base
}

func withFaking(base config.SetConfig, faking config.FakingConfig) config.SetConfig {
	base.Faking = faking
	return base
}

func withTCP(base config.SetConfig, tcp config.TCPConfig) config.SetConfig {
	base.TCP = tcp
	return base
}

func formatName(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func containsFamily(families []StrategyFamily, target StrategyFamily) bool {
	for _, f := range families {
		if f == target {
			return true
		}
	}
	return false
}
