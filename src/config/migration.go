package config

import (
	"encoding/json"
	"os"

	"github.com/daniellavrushin/b4/log"
)

func (c *Config) LoadWithMigration(path string) error {
	if path == "" {
		log.Tracef("config path is not defined")
		return nil
	}

	shouldMigrate, err := detectV1Format(path)
	if err != nil {
		log.Warnf("Could not detect config format: %v", err)
		// If detection fails, try to load as new format
		return c.LoadFromFile(path)
	}

	if shouldMigrate {
		log.Infof("Old config format detected, will migrate automatically")
		return c.loadAndMigrateV1(path)
	}

	return c.LoadFromFile(path)
}

func (c *Config) migrateFromV1(old *ConfigV1) {
	// Queue settings
	c.Queue.StartNum = old.QueueStartNum
	c.Queue.Mark = old.Mark
	c.Queue.Threads = old.Threads
	c.Queue.IPv4Enabled = old.IPv4Enabled
	c.Queue.IPv6Enabled = old.IPv6Enabled

	// Domains
	c.Sets = []*SetConfig{
		{
			Name: "default",
			Domains: DomainsConfig{
				SNIDomains:        old.Domains.SNIDomains,
				GeoSiteCategories: old.Domains.GeoSiteCategories,
				GeoIpCategories:   old.Domains.GeoIpCategories,
			},
			Fragmentation: old.Fragmentation,
			Faking:        old.Faking,
			UDP:           old.UDP,
			TCP:           TCPConfig{ConnBytesLimit: old.ConnBytesLimit, Seg2Delay: old.Seg2Delay},
		},
	}
	c.MainSet = c.Sets[0]

	// System settings
	c.System.Tables = old.Tables
	c.System.Logging = old.Logging
	c.System.WebServer = old.WebServer
	c.System.Checker = old.Checker
	c.System.Geo = GeoDatConfig{
		GeoSitePath: old.Domains.GeoSitePath,
		GeoIpPath:   old.Domains.GeoIpPath,
	}
	c.ConfigPath = old.ConfigPath
}

func (c *Config) loadAndMigrateV1(path string) error {
	if path == "" {
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

	// Unmarshal as V1 config
	var oldCfg ConfigV1
	err = json.Unmarshal(data, &oldCfg)
	if err != nil {
		return log.Errorf("failed to parse config file as V1 format: %v", err)
	}

	// Migrate to new format
	log.Infof("Migrating config from V1 to V2 format...")
	c.migrateFromV1(&oldCfg)

	// Save the migrated config back to disk
	log.Infof("Saving migrated config...")
	if err := c.SaveToFile(path); err != nil {
		return log.Errorf("failed to save migrated config: %v", err)
	}

	log.Infof("Config migration complete, saved to %s", path)
	return nil
}

// detectV1Format checks if a config file uses the old flat structure
func detectV1Format(path string) (bool, error) {
	if path == "" {
		return false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// Try to unmarshal into a generic map
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false, err
	}

	// Check for old format markers (flat fields that should be nested now)
	_, hasQueueStartNum := raw["queue_start_num"]
	_, hasConnBytesLimit := raw["conn_bytes_limit"]
	_, hasSeg2Delay := raw["seg2delay"]

	// Check for new format markers (nested objects)
	_, hasQueue := raw["queue"]
	_, hasBypass := raw["bypass"]
	_, hasSystem := raw["system"]

	// If we have old markers and no new markers, it's the old format
	isOldFormat := (hasQueueStartNum || hasConnBytesLimit || hasSeg2Delay) &&
		!hasQueue && !hasBypass && !hasSystem

	return isOldFormat, nil
}
