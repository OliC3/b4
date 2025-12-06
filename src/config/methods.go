package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/daniellavrushin/b4/geodat"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/utils"
)

func (c *Config) SaveToFile(path string) error {
	if path == "" {
		log.Tracef("config path is not defined")
		return nil
	}

	c.Version = CurrentConfigVersion
	if len(c.Sets) == 0 {
		defaultCopy := NewSetConfig()
		c.Sets = []*SetConfig{&defaultCopy}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return log.Errorf("failed to marshal config: %v", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return log.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return log.Errorf("failed to write config file: %v", err)
	}
	return nil
}

func (c *Config) LoadFromFile(path string) error {
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
	err = json.Unmarshal(data, c)

	if err != nil {
		return log.Errorf("failed to parse config file: %v", err)
	}

	if len(c.Sets) == 0 {
		defaultCopy := NewSetConfig()
		c.Sets = []*SetConfig{&defaultCopy}
	}
	return nil
}

func (cfg *Config) ApplyLogLevel(level string) {
	switch level {
	case "debug":
		cfg.System.Logging.Level = log.LevelDebug
	case "trace":
		cfg.System.Logging.Level = log.LevelTrace
	case "info":
		cfg.System.Logging.Level = log.LevelInfo
	case "error":
		cfg.System.Logging.Level = log.LevelError
	case "silent":
		cfg.System.Logging.Level = -1
	default:
		cfg.System.Logging.Level = log.LevelInfo
	}
}

func (c *Config) Validate() error {
	c.System.WebServer.IsEnabled = c.System.WebServer.Port > 0 && c.System.WebServer.Port <= 65535

	if c.MainSet == nil {
		for _, set := range c.Sets {
			if set.Id == MAIN_SET_ID {
				c.MainSet = set
				break
			}
		}
		if c.MainSet == nil {
			defaultCopy := NewSetConfig()
			c.MainSet = &defaultCopy
		}
	}

	if c.MainSet == nil {
		return fmt.Errorf("main set configuration is missing")
	}

	if len(c.MainSet.Targets.GeoSiteCategories) > 0 && c.System.Geo.GeoSitePath == "" {
		return fmt.Errorf("--geosite must be specified when using --geo-categories")
	}

	if len(c.MainSet.Targets.GeoIpCategories) > 0 && c.System.Geo.GeoIpPath == "" {
		return fmt.Errorf("--geoip must be specified when using --geoip-categories")
	}

	if c.Queue.Threads < 1 {
		return fmt.Errorf("threads must be at least 1")
	}

	if c.Queue.StartNum < 0 || c.Queue.StartNum > 65535 {
		return fmt.Errorf("queue-num must be between 0 and 65535")
	}

	if len(c.Sets) >= 1 {
		for _, set := range c.Sets {
			if set.Id == "" {
				return fmt.Errorf("each set must have a unique non-empty ID")
			}

			if set.Id == MAIN_SET_ID {
				set.UDP.DPortFilter = utils.ValidatePorts(set.UDP.DPortFilter)
				continue
			}

			if set.TCP.ConnBytesLimit > c.MainSet.TCP.ConnBytesLimit {
				return fmt.Errorf("set '%s' has TCP ConnBytesLimit greater than main set", set.Name)
			}

			if set.UDP.ConnBytesLimit > c.MainSet.UDP.ConnBytesLimit {
				return fmt.Errorf("set '%s' has UDP ConnBytesLimit greater than main set", set.Name)
			}

			set.UDP.DPortFilter = utils.ValidatePorts(set.UDP.DPortFilter)
		}
	}

	return nil
}

func (c *Config) LogString() string {
	return ""
}

// LoadTargets returns all targets (domains and IPs) from all sets grouped by set name
func (c *Config) LoadTargets() ([]*SetConfig, int, int, error) {
	result := make([]*SetConfig, 0, len(c.Sets))
	totalDomains := 0
	totalIps := 0

	// Process all sets
	for _, set := range c.Sets {

		if !set.Enabled {
			continue
		}

		domains, ips, err := c.GetTargetsForSet(set)
		if err != nil {
			return nil, -1, -1, fmt.Errorf("failed to load domains for set '%s': %w", set.Name, err)
		}
		if len(domains) > 0 {
			totalDomains += len(domains)
		}
		if len(ips) > 0 {
			totalIps += len(ips)
		}
		result = append(result, set)
	}

	return result, totalDomains, totalIps, nil
}

func (c *Config) GetTargetsForSet(set *SetConfig) ([]string, []string, error) {
	return c.GetTargetsForSetWithCache(set, nil, nil)
}

func (c *Config) GetTargetsForSetWithCache(set *SetConfig, geositeDomains, geoipIPs map[string][]string) ([]string, []string, error) {
	domains := []string{}
	ips := []string{}

	if len(set.Targets.GeoSiteCategories) > 0 && c.System.Geo.GeoSitePath != "" {
		if geositeDomains != nil {
			// Use cached data
			for _, cat := range set.Targets.GeoSiteCategories {
				if cached, ok := geositeDomains[cat]; ok {
					domains = append(domains, cached...)
				}
			}
		} else {
			// Fallback to disk (slow path)
			geoDomains, err := geodat.LoadDomainsFromCategories(
				c.System.Geo.GeoSitePath,
				set.Targets.GeoSiteCategories,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load geosite domains for set '%s': %w", set.Name, err)
			}
			domains = append(domains, geoDomains...)
		}
	}

	if len(set.Targets.SNIDomains) > 0 {
		domains = append(domains, set.Targets.SNIDomains...)
	}
	set.Targets.DomainsToMatch = domains

	if len(set.Targets.GeoIpCategories) > 0 && c.System.Geo.GeoIpPath != "" {
		if geoipIPs != nil {
			for _, cat := range set.Targets.GeoIpCategories {
				if cached, ok := geoipIPs[cat]; ok {
					ips = append(ips, cached...)
				}
			}
		} else {
			geoIps, err := geodat.LoadIpsFromCategories(
				c.System.Geo.GeoIpPath,
				set.Targets.GeoIpCategories,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load geoip for set '%s': %w", set.Name, err)
			}
			ips = append(ips, geoIps...)
		}
	}

	if len(set.Targets.IPs) > 0 {
		ips = append(ips, set.Targets.IPs...)
	}

	set.Targets.IpsToMatch = ips
	return domains, ips, nil
}

func (c *Config) GetSetById(id string) *SetConfig {
	for _, set := range c.Sets {
		if set.Id == id {
			return set
		}
	}
	return nil
}

func (set *SetConfig) ResetToDefaults() {
	defaultSet := DefaultSetConfig

	id := set.Id
	name := set.Name
	targets := set.Targets

	*set = defaultSet

	set.Id = id
	set.Name = name
	set.Targets = targets

	set.TCP.WinValues = make([]int, len(defaultSet.TCP.WinValues))
	copy(set.TCP.WinValues, defaultSet.TCP.WinValues)

	set.Faking.SNIMutation.FakeSNIs = make([]string, len(defaultSet.Faking.SNIMutation.FakeSNIs))
	copy(set.Faking.SNIMutation.FakeSNIs, defaultSet.Faking.SNIMutation.FakeSNIs)

	set.Fragmentation.Overlap.FakeSNIs = make([]string, len(defaultSet.Faking.SNIMutation.FakeSNIs))
	copy(set.Faking.SNIMutation.FakeSNIs, defaultSet.Faking.SNIMutation.FakeSNIs)

}

func (t *TargetsConfig) AppendIP(ip []string) error {
	for _, newIP := range ip {
		exists := false
		for _, existingIP := range t.IPs {
			if existingIP == newIP {
				exists = true
				break
			}
		}
		if !exists {
			t.IPs = append(t.IPs, newIP)
		}
	}

	for _, newIP := range ip {
		exists := false
		for _, existingIP := range t.IpsToMatch {
			if existingIP == newIP {
				exists = true
				break
			}
		}
		if !exists {
			t.IpsToMatch = append(t.IpsToMatch, newIP)
		}
	}

	return nil
}
func (t *TargetsConfig) AppendSNI(sni string) error {

	for _, existingDomain := range t.SNIDomains {
		if existingDomain == sni {
			return log.Errorf("SNI '%s' already exists in the set", sni)
		}
	}
	t.SNIDomains = append(t.SNIDomains, sni)

	for _, existingDomain := range t.DomainsToMatch {
		if existingDomain == sni {
			return log.Errorf("SNI '%s' already exists in the set", sni)
		}
	}
	t.DomainsToMatch = append(t.DomainsToMatch, sni)
	return nil
}
