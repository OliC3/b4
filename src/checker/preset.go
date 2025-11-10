package checker

import "github.com/daniellavrushin/b4/config"

type ConfigPreset struct {
	Name        string
	Description string
	Config      config.SetConfig
}

func GetTestPresets() []ConfigPreset {
	//baseSet := config.DefaultSetConfig

	return []ConfigPreset{
		{
			Name:        "tcp-frag-pos1",
			Description: "TCP fragmentation at position 1",
			Config: config.SetConfig{
				Fragmentation: config.FragmentationConfig{
					Strategy:    "tcp",
					SNIPosition: 1,
					SNIReverse:  false,
				},
				Faking: config.FakingConfig{
					SNI:      true,
					TTL:      8,
					Strategy: "pastseq",
				},
			},
		},
	}
}
