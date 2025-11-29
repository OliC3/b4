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
		// 0. Proven working config - this is the baseline that works for most Russian DPI
		{
			Name:        "proven-combo",
			Description: "Proven combination: TCP frag + reverse + middle SNI + fake pastseq",
			Family:      FamilyNone,
			Phase:       PhaseBaseline,
			Priority:    0,
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
					SNIType:      config.FakePayloadDefault,
				},
			},
		},

		// 1. Raw baseline - no bypass at all (to detect if DPI even blocks)
		{
			Name:        "no-bypass",
			Description: "No bypass techniques - test raw connectivity",
			Family:      FamilyNone,
			Phase:       PhaseBaseline,
			Priority:    1,
			Config:      baselineConfig(),
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
					SNIType:      config.FakePayloadDefault,
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
					SNIType:      config.FakePayloadDefault,
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
					SNIType:      config.FakePayloadDefault,
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
					SNIType:      config.FakePayloadDefault,
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
					SNIType:      config.FakePayloadDefault,
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
					SNIType:      config.FakePayloadDefault,
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
					SNIType:      config.FakePayloadDefault,
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
					SNIType:      config.FakePayloadDefault,
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
					SNIType:      config.FakePayloadDefault,
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
				SNIType:      config.FakePayloadDefault,
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
