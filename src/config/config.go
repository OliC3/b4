package config

import (
	"github.com/daniellavrushin/b4/log"
)

var (
	MAIN_SET_ID = "11111111-1111-1111-1111-111111111111"
	NEW_SET_ID  = "00000000-0000-0000-0000-000000000000"

	CurrentConfigVersion = len(migrationRegistry)
	MinSupportedVersion  = 0
)

type Config struct {
	Version    int    `json:"version" bson:"version"`
	ConfigPath string `json:"-" bson:"-"`

	Queue   QueueConfig  `json:"queue" bson:"queue"`
	MainSet *SetConfig   `json:"-" bson:"-"`
	System  SystemConfig `json:"system" bson:"system"`
	Sets    []*SetConfig `json:"sets" bson:"sets"`
}

var DefaultSetConfig = SetConfig{
	Id:      MAIN_SET_ID,
	Name:    "default",
	Enabled: true,

	UDP: UDPConfig{
		Mode:           "fake",
		FakeSeqLength:  6,
		FakeLen:        64,
		FakingStrategy: "none",
		DPortFilter:    "",
		FilterQUIC:     "disabled",
		FilterSTUN:     true,
		ConnBytesLimit: 8,
		Seg2Delay:      0,
	},

	TCP: TCPConfig{
		ConnBytesLimit: 19,
		Seg2Delay:      0,
		SynFake:        false,
		SynFakeLen:     0,
		DropSACK:       false,

		WinMode:   "off",
		WinValues: []int{0, 1460, 8192, 65535},

		DesyncMode:  "off",
		DesyncTTL:   3,
		DesyncCount: 3,
	},

	Fragmentation: FragmentationConfig{
		Strategy:          "tcp", // "tcp", "ip", "tls", "oob", "none"
		ReverseOrder:      true,
		MiddleSNI:         true,
		SNIPosition:       1,
		OOBPosition:       0,
		OOBChar:           'x',
		TLSRecordPosition: 0,
	},

	Faking: FakingConfig{
		SNI:           true,
		TTL:           8,
		SNISeqLength:  1,
		SNIType:       FakePayloadDefault,
		CustomPayload: "",
		Strategy:      "pastseq",
		SeqOffset:     10000,

		SNIMutation: SNIMutationConfig{
			Mode:         "off", // "off", "random", "grease", "padding", "fakeext", "fakesni", "advanced"
			GreaseCount:  3,
			PaddingSize:  2048,
			FakeExtCount: 5,
			FakeSNIs:     []string{"ya.ru", "vk.com", "max.ru"},
		},
	},

	Targets: TargetsConfig{
		SNIDomains:        []string{},
		IPs:               []string{},
		GeoSiteCategories: []string{},
		GeoIpCategories:   []string{},
	},
}

var DefaultConfig = Config{
	Version:    MinSupportedVersion,
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
			GeoSiteURL:  "",
			GeoIpURL:    "",
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
		API: ApiConfig{
			IPInfoToken: "",
		},
	},
}
