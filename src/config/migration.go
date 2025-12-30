package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/daniellavrushin/b4/log"
)

type MigrationFunc func(*Config) error

var (
	CurrentConfigVersion = len(migrationRegistry)
	MinSupportedVersion  = 0
)

var migrationRegistry = map[int]MigrationFunc{
	0:  migrateV0to1, // Add enabled field to sets
	1:  migrateV1to2,
	2:  migrateV2to3,
	3:  migrateV3to4,
	4:  migrateV4to5,
	5:  migrateV5to6,
	6:  migrateV6to7, // Add TCP syn TTL and drop SACK settings
	7:  migrateV7to8, // Add DNS redirect settings
	8:  migrateV8to9,
	9:  migrateV9to10,
	10: migrateV10to11,
	11: migrateV11to12,
	12: migrateV12to13,
	13: migrateV13to14,
}

// Migration: v13 -> v14 (add fragmentation strategy field)
func migrateV13to14(c *Config) error {
	log.Tracef("Migration v13->v14: Adding fragmentation strategy field")

	c.System.WebServer.BindAddress = DefaultConfig.System.WebServer.BindAddress
	for _, set := range c.Sets {
		set.TCP.PostDesync = DefaultSetConfig.TCP.PostDesync
		set.Fragmentation.Combo.DecoySNIs = DefaultSetConfig.Fragmentation.Combo.DecoySNIs
		set.Fragmentation.Combo.DecoyEnabled = DefaultSetConfig.Fragmentation.Combo.DecoyEnabled
	}
	return nil
}

// Migration: v12 -> v13 (add payload file/data to faking config)
func migrateV12to13(c *Config) error {
	log.Tracef("Migration v12->v13: Adding TLSMod to faking config")

	for _, set := range c.Sets {
		set.Faking.TLSMod = DefaultSetConfig.Faking.TLSMod
	}
	return nil
}

// Migration: v11 -> v12 (add TCP FilterSYN setting)
func migrateV11to12(c *Config) error {
	log.Tracef("Migration v11->v12: Adding TCP FilterSYN setting")

	for _, set := range c.Sets {
		set.Faking.PayloadFile = DefaultSetConfig.Faking.PayloadFile
		set.Faking.PayloadData = DefaultSetConfig.Faking.PayloadData

		set.Fragmentation.SeqOverlapPattern = DefaultSetConfig.Fragmentation.SeqOverlapPattern
		set.Fragmentation.SeqOverlapBytes = DefaultSetConfig.Fragmentation.SeqOverlapBytes
	}
	return nil
}

// Migration: v10 -> v11 (add devices config to queue)
func migrateV10to11(c *Config) error {
	log.Tracef("Migration v10->v11: Adding devices config to queue")

	c.Queue.Devices = DefaultConfig.Queue.Devices
	return nil
}

// Migration: v9 -> v10 (add error log file setting)
func migrateV9to10(c *Config) error {
	log.Tracef("Migration v9->v10: Adding error log file setting")

	c.Queue.Interfaces = []string{}
	return nil

}

// Migration: v8 -> v9
func migrateV8to9(c *Config) error {
	log.Tracef("Migration v8->v9: No changes, placeholder migration")
	c.System.Logging.ErrorFile = DefaultConfig.System.Logging.ErrorFile
	return nil
}

// Migration: v7 -> v8 (add DNS redirect settings)
func migrateV7to8(c *Config) error {
	log.Tracef("Migration v7->v8: Adding DNS redirect settings")

	for _, set := range c.Sets {
		set.DNS = DefaultSetConfig.DNS
	}
	c.System.Checker.ReferenceDNS = DefaultConfig.System.Checker.ReferenceDNS
	return nil
}

// Migration: v6 -> v7 (add TCP syn TTL and drop SACK settings)
func migrateV6to7(c *Config) error {
	log.Tracef("Migration v6->v7: Adding TCP syn TTL and drop SACK settings")

	for _, set := range c.Sets {
		set.TCP.SynTTL = DefaultSetConfig.TCP.SynTTL
	}
	return nil
}

// Migration: v5 -> v6 (add reference domain to discovery config)
func migrateV5to6(c *Config) error {
	log.Tracef("Migration v5->v6: Initializing missing fields with default values")

	for _, set := range c.Sets {
		set.Fragmentation.Combo = DefaultSetConfig.Fragmentation.Combo
		set.Fragmentation.Disorder = DefaultSetConfig.Fragmentation.Disorder
		//	set.Fragmentation.Overlap = DefaultSetConfig.Fragmentation.Overlap
	}
	return nil
}

// Migration: v4 -> v5 (add reference domain to discovery config)
func migrateV4to5(c *Config) error {
	log.Tracef("Migration v4->v5: Adding reference domain to discovery config")

	c.System.Checker.ReferenceDomain = "max.ru"

	return nil
}

// Migration: v0 -> v1 (add enabled field to sets)
func migrateV0to1(c *Config) error {
	log.Tracef("Migration v0->v1: Adding 'enabled' field to all sets")

	for _, set := range c.Sets {
		set.Enabled = true
	}

	if c.MainSet != nil {
		c.MainSet.Enabled = true
	}

	return nil
}

func migrateV1to2(c *Config) error {
	log.Tracef("Migration v1->v2: Renaming sni_reverse to reverse_order")

	for _, set := range c.Sets {
		set.Fragmentation.ReverseOrder = DefaultSetConfig.Fragmentation.ReverseOrder
		set.Fragmentation.OOBChar = DefaultSetConfig.Fragmentation.OOBChar
		set.Fragmentation.OOBPosition = DefaultSetConfig.Fragmentation.OOBPosition
		set.Fragmentation.TLSRecordPosition = DefaultSetConfig.Fragmentation.TLSRecordPosition
	}

	return nil
}

func migrateV2to3(c *Config) error {
	log.Tracef("Migration v2->v3: Adding TCP desync/window settings and SNI mutation")

	for _, set := range c.Sets {
		// TCP desync settings
		set.TCP.DesyncMode = DefaultSetConfig.TCP.DesyncMode
		set.TCP.DesyncTTL = DefaultSetConfig.TCP.DesyncTTL
		set.TCP.DesyncCount = DefaultSetConfig.TCP.DesyncCount

		// TCP window manipulation
		set.TCP.WinMode = DefaultSetConfig.TCP.WinMode
		set.TCP.WinValues = DefaultSetConfig.TCP.WinValues

		// SNI mutation
		set.Faking.SNIMutation = DefaultSetConfig.Faking.SNIMutation
	}

	return nil
}

func migrateV3to4(c *Config) error {
	log.Tracef("Migration v3->v4: Initializing missing fields with default values")

	c.System.Checker.ConfigPropagateMs = DefaultConfig.System.Checker.ConfigPropagateMs
	c.System.Checker.DiscoveryTimeoutSec = DefaultConfig.System.Checker.DiscoveryTimeoutSec

	return nil
}

func (c *Config) LoadWithMigration(path string) error {
	if path == "" {
		log.Tracef("config path is not defined")
		return nil
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

	// Modern config with version field
	if err := json.Unmarshal(data, c); err != nil {
		return log.Errorf("failed to parse config file: %v", err)
	}

	// Apply migrations if needed
	if c.Version < CurrentConfigVersion {
		log.Infof("Config version %d is older than current version %d, migrating",
			c.Version, CurrentConfigVersion)
		if err := c.applyMigrations(c.Version); err != nil {
			return err
		}
	}

	return nil
}

// applyMigrations applies all migrations from startVersion to CurrentConfigVersion
func (c *Config) applyMigrations(startVersion int) error {
	for v := startVersion; v < CurrentConfigVersion; v++ {
		migrationFunc, exists := migrationRegistry[v]
		if !exists {
			return fmt.Errorf("no migration path from version %d to %d", v, v+1)
		}

		log.Infof("Applying migration: v%d -> v%d", v, v+1)
		if err := migrationFunc(c); err != nil {
			return fmt.Errorf("migration from v%d to v%d failed: %w", v, v+1, err)
		}
		c.Version = v + 1
	}
	return nil
}
