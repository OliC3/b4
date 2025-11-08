// path: src/config/config.go
package config

import (
	_ "embed"

	"github.com/daniellavrushin/b4/log"
)

type Config struct {
	ConfigPath string `json:"-" bson:"-"`

	Queue   QueueConfig  `json:"queue" bson:"queue"`
	MainSet *SetConfig   `json:"-" bson:"-"`
	System  SystemConfig `json:"system" bson:"system"`
	Sets    []*SetConfig `json:"sets" bson:"sets"`
}

var DefaultSetConfig = SetConfig{
	Id:   "04a6b092-6900-4933-8c76-543a0d18d5e6",
	Name: "default",

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

	TCP: TCPConfig{
		ConnBytesLimit: 19,
		Seg2Delay:      0,
	},

	Fragmentation: FragmentationConfig{
		Strategy:    "tcp",
		SNIReverse:  true,
		MiddleSNI:   true,
		SNIPosition: 1,
	},

	Faking: FakingConfig{
		SNI:           true,
		TTL:           8,
		SNISeqLength:  1,
		SNIType:       FakePayloadDefault,
		CustomPayload: "",
		Strategy:      "pastseq",
		SeqOffset:     10000,
	},

	Domains: DomainsConfig{
		SNIDomains:        []string{},
		GeoSiteCategories: []string{},
		GeoIpCategories:   []string{},
	},
}

var DefaultConfig = Config{
	ConfigPath: "",

	Queue: QueueConfig{
		StartNum:    537,
		Mark:        1 << 15,
		Threads:     4,
		IPv4Enabled: true,
		IPv6Enabled: false,
	},

	Sets: []*SetConfig{},

	MainSet: &DefaultSetConfig,

	System: SystemConfig{
		Geo: GeoDatConfig{
			GeoSitePath: "",
			GeoIpPath:   "",
		},

		Tables: TablesConfig{
			MonitorInterval: 10,
			SkipSetup:       false,
		},

		WebServer: WebServerConfig{
			Port:      7000,
			IsEnabled: true,
		},

		Logging: Logging{
			Level:      log.LevelInfo,
			Instaflush: true,
			Syslog:     false,
		},

		Checker: CheckerConfig{
			TimeoutSeconds: 15,
			MaxConcurrent:  4,
			Domains:        []string{},
		},
	},
}
