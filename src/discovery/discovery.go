package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
)

const (
	// Timeouts
	QUICK_FAIL_TIMEOUT     = 1500 * time.Millisecond // Fast fail for non-responsive
	MAX_PRESETS_PER_DOMAIN = 3                       // Stop after N successful configs

	// Parallelism
	DEFAULT_PARALLEL_TESTS = 3
)

type DiscoverySuite struct {
	*CheckSuite
	pool           *nfq.Pool
	originalConfig *config.Config

	domain          string
	workingFamilies []StrategyFamily
	bestParams      map[StrategyFamily]ConfigPreset
	domainResult    *DomainDiscoveryResult

	mu sync.RWMutex
}

func NewDiscoverySuite(checkConfig CheckConfig, pool *nfq.Pool, domain string) *DiscoverySuite {
	suite := NewCheckSuite(checkConfig)

	return &DiscoverySuite{
		CheckSuite:      suite,
		pool:            pool,
		domain:          domain,
		workingFamilies: []StrategyFamily{},
		bestParams:      make(map[StrategyFamily]ConfigPreset),
		domainResult: &DomainDiscoveryResult{
			Domain:  domain,
			Results: make(map[string]*DomainPresetResult),
		},
	}
}

func (ds *DiscoverySuite) RunDiscovery() {
	suitesMu.Lock()
	activeSuites[ds.Id] = ds.CheckSuite
	suitesMu.Unlock()

	defer func() {
		ds.EndTime = time.Now()
		time.AfterFunc(5*time.Minute, func() {
			suitesMu.Lock()
			delete(activeSuites, ds.Id)
			suitesMu.Unlock()
		})
	}()

	ds.CheckSuite.mu.Lock()
	ds.Status = CheckStatusRunning
	ds.CheckSuite.mu.Unlock()

	ds.originalConfig = ds.pool.GetFirstWorkerConfig()
	if ds.originalConfig == nil {
		log.Errorf("Failed to get original configuration")
		ds.setStatus(CheckStatusFailed)
		return
	}

	log.Infof("Starting discovery for domain: %s", ds.domain)

	phase1Presets := GetPhase1Presets()
	ds.CheckSuite.mu.Lock()
	ds.TotalChecks = len(phase1Presets)
	ds.CheckSuite.mu.Unlock()

	// Phase 1: Strategy Detection
	ds.setPhase(PhaseStrategy)
	ds.workingFamilies = ds.runPhase1(ds.domain)

	if len(ds.workingFamilies) == 0 {
		log.Warnf("No working bypass strategies found for %s", ds.domain)
		ds.restoreConfig()
		ds.CheckSuite.mu.Lock()
		ds.CheckSuite.DomainDiscoveryResults = map[string]*DomainDiscoveryResult{ds.domain: ds.domainResult}
		ds.Status = CheckStatusComplete
		ds.CheckSuite.mu.Unlock()
		return
	}

	log.Infof("Phase 1 complete: %d working families: %v", len(ds.workingFamilies), ds.workingFamilies)

	// Phase 2: Optimization
	ds.setPhase(PhaseOptimize)
	ds.runPhase2(ds.domain, ds.workingFamilies)

	// Phase 3: Combinations
	if len(ds.workingFamilies) >= 2 {
		ds.setPhase(PhaseCombination)
		ds.runPhase3(ds.domain)
	}

	ds.determineBest()
	ds.restoreConfig()

	ds.CheckSuite.mu.Lock()
	ds.CheckSuite.DomainDiscoveryResults = map[string]*DomainDiscoveryResult{ds.domain: ds.domainResult}
	ds.Status = CheckStatusComplete
	ds.CheckSuite.mu.Unlock()

	ds.logDiscoverySummary()
}

func (ds *DiscoverySuite) runPhase1(domain string) []StrategyFamily {
	presets := GetPhase1Presets()
	workingFamilies := []StrategyFamily{}
	familyResults := make(map[StrategyFamily]*StrategyResult)

	log.Infof("Phase 1: Testing %d strategy families", len(presets))

	// Test baseline first (without any bypass)
	baselinePreset := presets[0]
	baselineResult := ds.testPreset(domain, baselinePreset)
	ds.storeResult(domain, baselinePreset, baselineResult)

	baselineWorks := baselineResult.Status == CheckStatusComplete
	var baselineSpeed float64
	if baselineWorks {
		baselineSpeed = baselineResult.Speed
		log.Infof("  Baseline: SUCCESS (%.2f KB/s) - DPI bypass may not be needed", baselineSpeed/1024)

		// Store baseline speed for improvement calculation
		ds.mu.Lock()
		ds.domainResult.BaselineSpeed = baselineSpeed
		ds.mu.Unlock()
	} else {
		log.Infof("  Baseline: FAILED - DPI bypass needed")
	}

	// Test each strategy family
	for _, preset := range presets[1:] { // Skip baseline
		select {
		case <-ds.cancel:
			return workingFamilies
		default:
		}

		result := ds.testPreset(domain, preset)
		ds.storeResult(domain, preset, result)

		sr := &StrategyResult{
			Family:  preset.Family,
			Works:   result.Status == CheckStatusComplete,
			Speed:   result.Speed,
			Preset:  preset.Name,
			Latency: result.Duration,
		}
		familyResults[preset.Family] = sr

		if sr.Works {
			// Only count as "working" if it's better than baseline or baseline failed
			if !baselineWorks || sr.Speed > baselineSpeed*0.8 {
				workingFamilies = append(workingFamilies, preset.Family)
				log.Infof("  %s: SUCCESS (%.2f KB/s)", preset.Name, sr.Speed/1024)
			} else {
				log.Infof("  %s: SUCCESS but slower than baseline (%.2f vs %.2f KB/s)",
					preset.Name, sr.Speed/1024, baselineSpeed/1024)
			}
		} else {
			log.Tracef("  %s: FAILED (%s)", preset.Name, result.Error)
		}
	}

	return workingFamilies
}

func (ds *DiscoverySuite) runPhase2(domain string, families []StrategyFamily) {
	// Calculate actual phase 2 preset count and update total
	totalPhase2Presets := 0
	for _, family := range families {
		totalPhase2Presets += len(GetPhase2Presets(family))
	}

	ds.CheckSuite.mu.Lock()
	ds.TotalChecks += totalPhase2Presets
	ds.CheckSuite.mu.Unlock()

	log.Infof("Phase 2: Optimizing %d working families (%d presets)", len(families), totalPhase2Presets)

	for _, family := range families {
		select {
		case <-ds.cancel:
			return
		default:
		}

		presets := GetPhase2Presets(family)
		if len(presets) == 0 {
			continue
		}

		log.Infof("  Optimizing %s (%d variants)", family, len(presets))

		var bestPreset ConfigPreset
		var bestSpeed float64
		successCount := 0

		for _, preset := range presets {
			select {
			case <-ds.cancel:
				return
			default:
			}

			// Stop early if we found enough good configs
			if successCount >= 3 {
				log.Tracef("    Found %d good configs for %s, skipping rest", successCount, family)
				break
			}

			result := ds.testPreset(domain, preset)
			ds.storeResult(domain, preset, result)

			if result.Status == CheckStatusComplete {
				successCount++
				if result.Speed > bestSpeed {
					bestSpeed = result.Speed
					bestPreset = preset
				}
				log.Tracef("    %s: %.2f KB/s", preset.Name, result.Speed/1024)
			}
		}

		if bestSpeed > 0 {
			ds.mu.Lock()
			ds.bestParams[family] = bestPreset
			ds.mu.Unlock()
			log.Infof("  Best %s config: %s (%.2f KB/s)", family, bestPreset.Name, bestSpeed/1024)
		}
	}
}

func (ds *DiscoverySuite) runPhase3(domain string) {
	ds.mu.RLock()
	workingFamilies := ds.workingFamilies
	bestParams := ds.bestParams
	ds.mu.RUnlock()

	presets := GetCombinationPresets(workingFamilies, bestParams)
	if len(presets) == 0 {
		return
	}

	// Update total count
	ds.CheckSuite.mu.Lock()
	ds.TotalChecks += len(presets)
	ds.CheckSuite.mu.Unlock()

	log.Infof("Phase 3: Testing %d combination presets", len(presets))

	for _, preset := range presets {
		select {
		case <-ds.cancel:
			return
		default:
		}

		result := ds.testPreset(domain, preset)
		ds.storeResult(domain, preset, result)

		if result.Status == CheckStatusComplete {
			log.Infof("  %s: SUCCESS (%.2f KB/s)", preset.Name, result.Speed/1024)
		} else {
			log.Tracef("  %s: FAILED", preset.Name)
		}
	}
}

func (ds *DiscoverySuite) testPreset(domain string, preset ConfigPreset) CheckResult {
	// Build test config
	testConfig := ds.buildTestConfig(preset, domain)

	// Apply config to pool
	if err := ds.pool.UpdateConfig(testConfig); err != nil {
		log.Errorf("Failed to apply preset %s: %v", preset.Name, err)
		return CheckResult{
			Domain: domain,
			Status: CheckStatusFailed,
			Error:  err.Error(),
		}
	}

	// Brief delay for config propagation
	time.Sleep(time.Duration(ds.Config.ConfigPropagateTimeout) * time.Millisecond)

	// Test with quick fail first
	result := ds.quickTest(domain)

	// If quick test failed but not timeout, try full test
	if result.Status == CheckStatusFailed && result.BytesRead == 0 {
		// Give it another shot with full timeout
		result = ds.fullTest(domain)
	}

	result.Set = testConfig.MainSet

	// Update progress
	ds.CheckSuite.mu.Lock()
	ds.CompletedChecks++
	ds.CheckSuite.mu.Unlock()

	return result
}

func (ds *DiscoverySuite) quickTest(domain string) CheckResult {
	return ds.fetchWithTimeout(domain, QUICK_FAIL_TIMEOUT)
}

func (ds *DiscoverySuite) fullTest(domain string) CheckResult {
	return ds.fetchWithTimeout(domain, ds.Config.Timeout)
}

func (ds *DiscoverySuite) fetchWithTimeout(domain string, timeout time.Duration) CheckResult {
	result := CheckResult{
		Domain:    domain,
		Status:    CheckStatusRunning,
		Timestamp: time.Now(),
	}

	testURL := fmt.Sprintf("https://%s/", domain)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			ResponseHeaderTimeout: timeout,
			IdleConnTimeout:       timeout,
			DialContext: (&net.Dialer{
				Timeout:   timeout / 2,
				KeepAlive: timeout,
			}).DialContext,
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		result.Status = CheckStatusFailed
		result.Error = err.Error()
		return result
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		result.Status = CheckStatusFailed
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Read up to 100KB
	bytesRead, _ := io.CopyN(io.Discard, resp.Body, 100*1024)
	duration := time.Since(start)

	result.Duration = duration
	result.BytesRead = bytesRead

	if bytesRead > 0 {
		result.Status = CheckStatusComplete
		if duration.Seconds() > 0 {
			result.Speed = float64(bytesRead) / duration.Seconds()
		}
	} else {
		result.Status = CheckStatusFailed
		result.Error = "no data received"
	}

	return result
}

func (ds *DiscoverySuite) storeResult(domain string, preset ConfigPreset, result CheckResult) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.domainResult.Results[preset.Name] = &DomainPresetResult{
		PresetName: preset.Name,
		Family:     preset.Family,
		Phase:      preset.Phase,
		Status:     result.Status,
		Duration:   result.Duration,
		Speed:      result.Speed,
		BytesRead:  result.BytesRead,
		Error:      result.Error,
		StatusCode: result.StatusCode,
		Set:        result.Set,
	}
}

func (ds *DiscoverySuite) determineBest() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	var bestPreset string
	var bestSpeed float64
	var bestSuccess bool

	for presetName, result := range ds.domainResult.Results {
		if result.Status == CheckStatusComplete {
			if !bestSuccess || result.Speed > bestSpeed {
				bestSuccess = true
				bestPreset = presetName
				bestSpeed = result.Speed
			}
		}
	}

	ds.domainResult.BestPreset = bestPreset
	ds.domainResult.BestSpeed = bestSpeed
	ds.domainResult.BestSuccess = bestSuccess

	if ds.domainResult.BaselineSpeed > 0 && bestSpeed > 0 {
		ds.domainResult.Improvement = ((bestSpeed - ds.domainResult.BaselineSpeed) / ds.domainResult.BaselineSpeed) * 100
	}
}

func (ds *DiscoverySuite) buildTestConfig(preset ConfigPreset, testDomain string) *config.Config {
	mainSet := config.NewSetConfig()

	mainSet.Id = ds.originalConfig.MainSet.Id
	mainSet.Name = preset.Name
	mainSet.TCP = preset.Config.TCP
	mainSet.UDP = preset.Config.UDP
	mainSet.Fragmentation = preset.Config.Fragmentation
	mainSet.Faking = preset.Config.Faking
	mainSet.Targets.SNIDomains = []string{testDomain}
	mainSet.Targets.DomainsToMatch = []string{testDomain}

	return &config.Config{
		ConfigPath: ds.originalConfig.ConfigPath,
		Queue:      ds.originalConfig.Queue,
		System:     ds.originalConfig.System,
		MainSet:    &mainSet,
		Sets:       []*config.SetConfig{&mainSet},
	}
}

func (ds *DiscoverySuite) setStatus(status CheckStatus) {
	ds.CheckSuite.mu.Lock()
	ds.Status = status
	ds.CheckSuite.mu.Unlock()
}

func (ds *DiscoverySuite) setPhase(phase DiscoveryPhase) {
	ds.CheckSuite.mu.Lock()
	ds.CurrentPhase = phase
	ds.CheckSuite.mu.Unlock()
}

func (ds *DiscoverySuite) restoreConfig() {
	log.Infof("Restoring original configuration")
	if err := ds.pool.UpdateConfig(ds.originalConfig); err != nil {
		log.Errorf("Failed to restore original configuration: %v", err)
	}
}

func (ds *DiscoverySuite) logDiscoverySummary() {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	log.Infof("\n=== Discovery Results for %s ===", ds.domain)

	if ds.domainResult.BestSuccess {
		improvement := ""
		if ds.domainResult.Improvement > 0 {
			improvement = fmt.Sprintf(" (+%.0f%%)", ds.domainResult.Improvement)
		}
		log.Infof("✓ Best config: %s (%.2f KB/s%s)",
			ds.domainResult.BestPreset, ds.domainResult.BestSpeed/1024, improvement)
	} else {
		log.Warnf("✗ No successful configuration found")
	}
}

func (ds *DiscoverySuite) GetDiscoveryReport() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	report := fmt.Sprintf("Discovery Results for %s:\n", ds.domain)
	report += "=========================================\n\n"

	if ds.domainResult.BestSuccess {
		report += fmt.Sprintf("  Best Config: %s\n", ds.domainResult.BestPreset)
		report += fmt.Sprintf("  Speed: %.2f KB/s\n", ds.domainResult.BestSpeed/1024)
		if ds.domainResult.Improvement > 0 {
			report += fmt.Sprintf("  Improvement: +%.0f%%\n", ds.domainResult.Improvement)
		}
	} else {
		report += "  Status: No successful configuration\n"
	}

	return report
}
