package checker

import (
	"fmt"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
)

const MAX_PRESETS_PER_DOMAIN = 3
const DISCOVERY_TIMEOUT = 3 * time.Second

type DiscoverySuite struct {
	*CheckSuite
	pool           *nfq.Pool
	originalConfig *config.Config
	presets        []ConfigPreset
	domainResults  map[string]*DomainDiscoveryResult // domain -> results for all presets
	mu             sync.RWMutex
}

func NewDiscoverySuite(checkConfig CheckConfig, pool *nfq.Pool, presets []ConfigPreset) *DiscoverySuite {
	checkConfig.Timeout = DISCOVERY_TIMEOUT

	suite := NewCheckSuite(checkConfig)
	suite.DomainDiscoveryResults = make(map[string]*DomainDiscoveryResult)

	return &DiscoverySuite{
		CheckSuite:    suite,
		pool:          pool,
		presets:       presets,
		domainResults: make(map[string]*DomainDiscoveryResult),
	}
}

func (ds *DiscoverySuite) RunDiscovery(domains []string) {
	// Register in activeSuites so status endpoint can find it
	suitesMu.Lock()
	activeSuites[ds.Id] = ds.CheckSuite
	suitesMu.Unlock()

	defer func() {
		ds.EndTime = time.Now()

		// Keep in memory for 5 minutes
		time.AfterFunc(5*time.Minute, func() {
			suitesMu.Lock()
			delete(activeSuites, ds.Id)
			suitesMu.Unlock()
		})
	}()

	// Set initial status
	ds.CheckSuite.mu.Lock()
	ds.Status = CheckStatusRunning
	ds.TotalChecks = len(domains) * len(ds.presets)
	ds.CheckSuite.mu.Unlock()

	// Store original configuration
	ds.originalConfig = ds.pool.GetFirstWorkerConfig()
	if ds.originalConfig == nil {
		log.Errorf("Failed to get original configuration")
		ds.CheckSuite.mu.Lock()
		ds.Status = CheckStatusFailed
		ds.CheckSuite.mu.Unlock()
		return
	}

	log.Infof("Starting domain-centric discovery for %d domains across %d presets",
		len(domains), len(ds.presets))
	log.Warnf("Service traffic will be affected during discovery testing")

	// Initialize domain results
	for _, domain := range domains {
		ds.mu.Lock()
		ds.domainResults[domain] = &DomainDiscoveryResult{
			Domain:  domain,
			Results: make(map[string]*DomainPresetResult),
		}
		ds.mu.Unlock()
	}

	for _, domain := range domains {
		select {
		case <-ds.cancel:
			log.Infof("Discovery suite %s canceled", ds.Id)
			ds.CheckSuite.mu.Lock()
			ds.Status = CheckStatusCanceled
			ds.CheckSuite.mu.Unlock()
			return
		default:
		}

		log.Infof("Testing domain: %s (will stop after %d successful configs)", domain, MAX_PRESETS_PER_DOMAIN)

		successfulCount := 0
		testedCount := 0

		for _, preset := range ds.presets {
			select {
			case <-ds.cancel:
				return
			default:
			}

			// Stop testing this domain if we found enough successful configs
			if successfulCount >= MAX_PRESETS_PER_DOMAIN {
				log.Infof("  Domain %s: Found %d successful configs, skipping remaining %d presets",
					domain, successfulCount, len(ds.presets)-testedCount)

				// Update total checks to reflect skipped presets
				ds.CheckSuite.mu.Lock()
				ds.TotalChecks -= (len(ds.presets) - testedCount)
				ds.CheckSuite.mu.Unlock()
				break
			}

			testedCount++
			log.Tracef("  Testing %s with preset %d/%d: %s", domain, testedCount, len(ds.presets), preset.Name)

			// Apply preset configuration by RESTARTING pool
			testConfig := ds.buildTestConfig(preset, domain)

			log.Infof("  Applying preset %s config...", preset.Name)
			if err := ds.pool.UpdateConfig(testConfig); err != nil {
				log.Errorf("Failed to update config for preset %s: %v", preset.Name, err)
				ds.CheckSuite.mu.Lock()
				ds.CompletedChecks++
				ds.CheckSuite.mu.Unlock()
				continue
			}

			// Small delay to let config propagate to all workers
			time.Sleep(500 * time.Millisecond)

			var result CheckResult
			for attempt := 0; attempt < 2; attempt++ {
				result = ds.testDomain(domain)

				// If successful or it's the last attempt, use this result
				if result.Status == CheckStatusComplete || attempt == 1 {
					break
				}

				// First attempt failed, wait a bit longer for config to propagate
				log.Tracef("  First attempt failed, retrying after additional delay...")
				time.Sleep(300 * time.Millisecond)
			}

			// Store result for this domain+preset combination
			ds.mu.Lock()
			ds.domainResults[domain].Results[preset.Name] = &DomainPresetResult{
				PresetName: preset.Name,
				Status:     result.Status,
				Duration:   result.Duration,
				Speed:      result.Speed,
				BytesRead:  result.BytesRead,
				Error:      result.Error,
				StatusCode: result.StatusCode,
			}
			ds.mu.Unlock()

			// Count successful results
			if result.Status == CheckStatusComplete {
				successfulCount++
				log.Infof("  ✓ %s with %s: %.2f KB/s (success %d/%d)",
					domain, preset.Name, result.Speed/1024, successfulCount, MAX_PRESETS_PER_DOMAIN)
			} else {
				log.Tracef("  ✗ %s with %s: %s",
					domain, preset.Name, result.Status)
			}

			// Update progress
			ds.CheckSuite.mu.Lock()
			ds.CompletedChecks++
			ds.CheckSuite.mu.Unlock()
		}

		// Determine best preset for this domain
		ds.determineBestPresetForDomain(domain)

		log.Infof("Domain %s complete: tested %d presets, found %d successful configs",
			domain, testedCount, successfulCount)
	}

	// Copy results to CheckSuite for JSON serialization
	ds.CheckSuite.mu.Lock()
	ds.CheckSuite.DomainDiscoveryResults = ds.domainResults
	ds.CheckSuite.mu.Unlock()

	// Restore original configuration
	log.Infof("Restoring original configuration")
	if err := ds.pool.UpdateConfig(ds.originalConfig); err != nil {
		log.Errorf("Failed to restore original configuration: %v", err)
	}

	ds.CheckSuite.mu.Lock()
	ds.Status = CheckStatusComplete
	ds.CheckSuite.mu.Unlock()

	// Log summary
	ds.logDiscoverySummary()
}

func (ds *DiscoverySuite) determineBestPresetForDomain(domain string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	domainResult := ds.domainResults[domain]
	if domainResult == nil {
		return
	}

	var bestPreset string
	var bestSpeed float64
	var bestSuccess bool

	for presetName, result := range domainResult.Results {
		// Prioritize success status first, then speed
		isSuccess := result.Status == CheckStatusComplete

		if !bestSuccess && isSuccess {
			// First successful result
			bestSuccess = true
			bestPreset = presetName
			bestSpeed = result.Speed
		} else if bestSuccess == isSuccess {
			// Both succeeded or both failed - compare speed
			if result.Speed > bestSpeed {
				bestPreset = presetName
				bestSpeed = result.Speed
			}
		}
		// If current best is successful but this one failed, skip
	}

	domainResult.BestPreset = bestPreset
	domainResult.BestSpeed = bestSpeed
	domainResult.BestSuccess = bestSuccess
}

func (ds *DiscoverySuite) buildTestConfig(preset ConfigPreset, testDomain string) *config.Config {
	cfg := &config.Config{
		ConfigPath: ds.originalConfig.ConfigPath,
		Queue:      ds.originalConfig.Queue,
		System:     ds.originalConfig.System,
	}

	mainSet := &config.SetConfig{
		Id:            ds.originalConfig.MainSet.Id,
		Name:          ds.originalConfig.MainSet.Name,
		TCP:           preset.Config.TCP,
		UDP:           preset.Config.UDP,
		Fragmentation: preset.Config.Fragmentation,
		Faking:        preset.Config.Faking,
		Targets: config.TargetsConfig{
			SNIDomains:        []string{testDomain},
			DomainsToMatch:    []string{testDomain},
			IPs:               []string{},
			IpsToMatch:        []string{},
			GeoSiteCategories: []string{},
			GeoIpCategories:   []string{},
		},
	}

	cfg.MainSet = mainSet
	cfg.Sets = []*config.SetConfig{mainSet}

	return cfg
}

func (ds *DiscoverySuite) logDiscoverySummary() {
	log.Infof("\n=== Discovery Results Summary ===")

	ds.mu.RLock()
	defer ds.mu.RUnlock()

	for _, domain := range ds.sortedDomains() {
		result := ds.domainResults[domain]
		if result.BestSuccess {
			log.Infof("✓ %s: %s (%.2f KB/s)",
				domain, result.BestPreset, result.BestSpeed/1024)
		} else {
			log.Warnf("✗ %s: No successful configuration found", domain)
		}
	}
}

func (ds *DiscoverySuite) sortedDomains() []string {
	domains := make([]string, 0, len(ds.domainResults))
	for domain := range ds.domainResults {
		domains = append(domains, domain)
	}
	return domains
}

// GetDiscoveryReport returns formatted report
func (ds *DiscoverySuite) GetDiscoveryReport() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	report := "Domain-Specific Configuration Discovery:\n"
	report += "=========================================\n\n"

	for _, domain := range ds.sortedDomains() {
		result := ds.domainResults[domain]
		report += fmt.Sprintf("Domain: %s\n", domain)
		if result.BestSuccess {
			report += fmt.Sprintf("  Best Config: %s\n", result.BestPreset)
			report += fmt.Sprintf("  Speed: %.2f KB/s\n", result.BestSpeed/1024)
		} else {
			report += "  Status: No successful configuration\n"
		}
		report += "\n"
	}

	return report
}
