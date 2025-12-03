package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
)

type FailureMode string

const (
	QUICK_FAIL_TIMEOUT = 1500 * time.Millisecond
)

const (
	FailureRSTImmediate FailureMode = "rst_immediate"
	FailureTimeout      FailureMode = "timeout"
	FailureTLSError     FailureMode = "tls_error"
	FailureUnknown      FailureMode = "unknown"
)

type PayloadTestResult struct {
	Payload int
	Works   bool
	Speed   float64
}

type DiscoverySuite struct {
	*CheckSuite
	pool           *nfq.Pool
	originalConfig *config.Config
	domain         string
	domainResult   *DomainDiscoveryResult

	// Detected working payload(s)
	workingPayloads []PayloadTestResult
	bestPayload     int
}

func NewDiscoverySuite(checkConfig CheckConfig, pool *nfq.Pool, domain string) *DiscoverySuite {
	return &DiscoverySuite{
		CheckSuite: NewCheckSuite(checkConfig),
		pool:       pool,
		domain:     domain,
		domainResult: &DomainDiscoveryResult{
			Domain:  domain,
			Results: make(map[string]*DomainPresetResult),
		},
		workingPayloads: []PayloadTestResult{},
		bestPayload:     config.FakePayloadDefault1, // default
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

	ds.setStatus(CheckStatusRunning)

	ds.originalConfig = ds.pool.GetFirstWorkerConfig()
	if ds.originalConfig == nil {
		log.Errorf("Failed to get original configuration")
		ds.setStatus(CheckStatusFailed)
		return
	}

	log.Infof("Starting discovery for domain: %s", ds.domain)

	ds.setPhase(PhaseFingerprint)
	fingerprint := ds.runFingerprinting()
	ds.domainResult.Fingerprint = fingerprint

	if fingerprint != nil && fingerprint.Type == DPITypeNone {
		log.Infof("No DPI detected for %s - skipping bypass discovery", ds.domain)
		ds.domainResult.BestPreset = "no-bypass"
		ds.domainResult.BestSuccess = true
		ds.restoreConfig()
		ds.finalize()
		return
	}

	phase1Presets := GetPhase1Presets()
	if fingerprint != nil && len(fingerprint.RecommendedFamilies) > 0 {
		phase1Presets = FilterPresetsByFingerprint(phase1Presets, fingerprint)
		// Apply optimal TTL to all presets
		for i := range phase1Presets {
			ApplyFingerprintToPreset(&phase1Presets[i], fingerprint)
		}
	}

	ds.CheckSuite.mu.Lock()
	ds.TotalChecks = len(phase1Presets)
	ds.CheckSuite.mu.Unlock()

	// Phase 1: Strategy Detection
	ds.setPhase(PhaseStrategy)
	workingFamilies, baselineSpeed, baselineWorks := ds.runPhase1(phase1Presets)
	ds.determineBest(baselineSpeed)

	if baselineWorks {
		log.Infof("Baseline succeeded for %s - no DPI bypass needed, skipping optimization", ds.domain)

		ds.CheckSuite.mu.Lock()
		ds.TotalChecks = 1
		ds.domainResult.BestPreset = "no-bypass"
		ds.domainResult.BestSpeed = baselineSpeed
		ds.domainResult.BestSuccess = true
		ds.domainResult.BaselineSpeed = baselineSpeed
		ds.domainResult.Improvement = 0
		ds.CheckSuite.mu.Unlock()

		ds.restoreConfig()
		ds.finalize()
		ds.logDiscoverySummary()
		return
	}

	if len(workingFamilies) == 0 {
		log.Warnf("Phase 1 found no working families, trying extended search")

		// Try all Phase 2 presets for each family anyway
		ds.setPhase(PhaseOptimize)
		workingFamilies = ds.runExtendedSearch()

		if len(workingFamilies) == 0 {
			log.Warnf("No working bypass strategies found for %s", ds.domain)
			ds.restoreConfig()
			ds.finalize()
			ds.logDiscoverySummary()
			return
		}
	}

	log.Infof("Phase 1 complete: %d working families: %v", len(workingFamilies), workingFamilies)

	// Phase 2: Optimization
	ds.setPhase(PhaseOptimize)
	bestParams := ds.runPhase2(workingFamilies)
	ds.determineBest(baselineSpeed)

	// Phase 3: Combinations
	if len(workingFamilies) >= 2 {
		ds.setPhase(PhaseCombination)
		ds.runPhase3(workingFamilies, bestParams)
	}

	ds.determineBest(baselineSpeed)
	ds.restoreConfig()
	ds.finalize()
	ds.logDiscoverySummary()
}

func (ds *DiscoverySuite) runFingerprinting() *DPIFingerprint {
	log.Infof("Phase 0: DPI Fingerprinting for %s", ds.domain)

	prober := NewDPIProber(ds.domain, ds.Config.Timeout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fingerprint := prober.Fingerprint(ctx)

	ds.CheckSuite.mu.Lock()
	ds.Fingerprint = fingerprint
	ds.CheckSuite.mu.Unlock()

	return fingerprint
}

func (ds *DiscoverySuite) runPhase1(presets []ConfigPreset) ([]StrategyFamily, float64, bool) {
	var workingFamilies []StrategyFamily
	var baselineSpeed float64

	log.Infof("Phase 1: Testing %d strategy families", len(presets))

	// Test baseline first (index 0)
	baselineResult := ds.testPreset(presets[0])
	ds.storeResult(presets[0], baselineResult)

	baselineWorks := baselineResult.Status == CheckStatusComplete
	if baselineWorks {
		baselineSpeed = baselineResult.Speed
		log.Infof("  Baseline: SUCCESS (%.2f KB/s) - no DPI detected", baselineSpeed/1024)
		return workingFamilies, baselineSpeed, true
	}

	log.Infof("  Baseline: FAILED - DPI bypass needed, testing strategies")

	// Test payload variants early (proven-combo and proven-combo-alt)
	ds.detectWorkingPayloads(presets)

	// Get non-baseline presets (skip baseline and the two proven-combo variants we already tested)
	strategyPresets := ds.filterTestedPresets(presets)

	baselineFailureMode := analyzeFailure(baselineResult)
	suggestedFamilies := suggestFamiliesForFailure(baselineFailureMode)

	if len(suggestedFamilies) > 0 {
		strategyPresets = reorderByFamilies(strategyPresets, suggestedFamilies)
		log.Infof("  Failure mode: %s - prioritizing: %v", baselineFailureMode, suggestedFamilies)
	}

	// Test each strategy with the best detected payload
	for _, preset := range strategyPresets {
		select {
		case <-ds.cancel:
			return workingFamilies, baselineSpeed, false
		default:
		}

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete {
			if result.Speed > baselineSpeed*0.8 {
				workingFamilies = append(workingFamilies, preset.Family)
				log.Infof("  %s: SUCCESS (%.2f KB/s)", preset.Name, result.Speed/1024)
			} else {
				log.Infof("  %s: SUCCESS but slower than baseline (%.2f vs %.2f KB/s)",
					preset.Name, result.Speed/1024, baselineSpeed/1024)
			}
		} else {
			log.Tracef("  %s: FAILED (%s)", preset.Name, result.Error)
		}
	}

	return workingFamilies, baselineSpeed, false
}

// detectWorkingPayloads tests both payload types and determines which work
func (ds *DiscoverySuite) detectWorkingPayloads(presets []ConfigPreset) {
	log.Infof("  Testing payload variants...")

	// Find proven-combo and proven-combo-alt presets
	var payload1Preset, payload2Preset *ConfigPreset
	for i := range presets {
		if presets[i].Name == "proven-combo" {
			payload1Preset = &presets[i]
		}
		if presets[i].Name == "proven-combo-alt" {
			payload2Preset = &presets[i]
		}
	}

	// Test payload 1 (if not already tested)
	if payload1Preset != nil {
		if _, exists := ds.domainResult.Results["proven-combo"]; !exists {
			result1 := ds.testPreset(*payload1Preset)
			ds.storeResult(*payload1Preset, result1)

			ds.workingPayloads = append(ds.workingPayloads, PayloadTestResult{
				Payload: config.FakePayloadDefault1,
				Works:   result1.Status == CheckStatusComplete,
				Speed:   result1.Speed,
			})

			if result1.Status == CheckStatusComplete {
				log.Infof("    Payload 1 (google): SUCCESS (%.2f KB/s)", result1.Speed/1024)
			} else {
				log.Infof("    Payload 1 (google): FAILED")
			}
		}
	}

	// Test payload 2
	if payload2Preset != nil {
		if _, exists := ds.domainResult.Results["proven-combo-alt"]; !exists {
			result2 := ds.testPreset(*payload2Preset)
			ds.storeResult(*payload2Preset, result2)

			ds.workingPayloads = append(ds.workingPayloads, PayloadTestResult{
				Payload: config.FakePayloadDefault2,
				Works:   result2.Status == CheckStatusComplete,
				Speed:   result2.Speed,
			})

			if result2.Status == CheckStatusComplete {
				log.Infof("    Payload 2 (duckduckgo): SUCCESS (%.2f KB/s)", result2.Speed/1024)
			} else {
				log.Infof("    Payload 2 (duckduckgo): FAILED")
			}
		}
	}

	// Determine best payload
	ds.selectBestPayload()
}

// selectBestPayload chooses the best payload based on test results
func (ds *DiscoverySuite) selectBestPayload() {
	var bestSpeed float64
	ds.bestPayload = config.FakePayloadDefault1 // default fallback

	workingCount := 0
	for _, pr := range ds.workingPayloads {
		if pr.Works {
			workingCount++
			if pr.Speed > bestSpeed {
				bestSpeed = pr.Speed
				ds.bestPayload = pr.Payload
			}
		}
	}

	switch workingCount {
	case 0:
		log.Infof("  Neither payload worked in baseline - will test both during discovery")
	case 1:
		payloadName := "google"
		if ds.bestPayload == config.FakePayloadDefault2 {
			payloadName = "duckduckgo"
		}
		log.Infof("  Selected payload: %s (only one works)", payloadName)
	case 2:
		payloadName := "google"
		if ds.bestPayload == config.FakePayloadDefault2 {
			payloadName = "duckduckgo"
		}
		log.Infof("  Selected payload: %s (faster of both working)", payloadName)
	}
}

// filterTestedPresets removes presets we've already tested
func (ds *DiscoverySuite) filterTestedPresets(presets []ConfigPreset) []ConfigPreset {
	filtered := []ConfigPreset{}
	for _, p := range presets {
		if p.Name == "no-bypass" || p.Name == "proven-combo" || p.Name == "proven-combo-alt" {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

// testPresetWithBestPayload tests a preset using the detected best payload
func (ds *DiscoverySuite) testPresetWithBestPayload(preset ConfigPreset) CheckResult {

	defer func() {
		ds.CheckSuite.mu.Lock()
		ds.CompletedChecks++
		ds.CheckSuite.mu.Unlock()
	}()

	// If we have a clearly working payload, use it
	hasWorkingPayload := false
	for _, pr := range ds.workingPayloads {
		if pr.Works {
			hasWorkingPayload = true
			break
		}
	}

	if hasWorkingPayload {
		// Use the best payload
		return ds.testPresetWithPayload(preset, ds.bestPayload)
	}

	// Neither payload worked in baseline - test both and return best
	result1 := ds.testPresetWithPayload(preset, config.FakePayloadDefault1)
	if result1.Status == CheckStatusComplete {
		// Payload 1 works for this strategy - update our knowledge
		ds.updatePayloadKnowledge(config.FakePayloadDefault1, result1.Speed)
		return result1
	}

	result2 := ds.testPresetWithPayload(preset, config.FakePayloadDefault2)
	if result2.Status == CheckStatusComplete {
		// Payload 2 works for this strategy - update our knowledge
		ds.updatePayloadKnowledge(config.FakePayloadDefault2, result2.Speed)
		return result2
	}

	// Neither worked
	return result1
}

// testPresetWithPayload tests a specific preset with a specific payload type
func (ds *DiscoverySuite) testPresetWithPayload(preset ConfigPreset, payloadType int) CheckResult {
	// Override the payload type
	modifiedPreset := preset
	modifiedPreset.Config.Faking.SNIType = payloadType

	return ds.testPresetInternal(modifiedPreset)
}

// updatePayloadKnowledge updates our knowledge about working payloads
func (ds *DiscoverySuite) updatePayloadKnowledge(payload int, speed float64) {
	// Check if we already have this payload recorded
	for i, pr := range ds.workingPayloads {
		if pr.Payload == payload {
			if !pr.Works || speed > pr.Speed {
				ds.workingPayloads[i].Works = true
				ds.workingPayloads[i].Speed = speed
			}
			ds.selectBestPayload()
			return
		}
	}

	// Add new payload knowledge
	ds.workingPayloads = append(ds.workingPayloads, PayloadTestResult{
		Payload: payload,
		Works:   true,
		Speed:   speed,
	})
	ds.selectBestPayload()
}

func (ds *DiscoverySuite) runPhase2(families []StrategyFamily) map[StrategyFamily]ConfigPreset {
	bestParams := make(map[StrategyFamily]ConfigPreset)

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
			return bestParams
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
				return bestParams
			default:
			}

			if successCount >= 3 {
				log.Tracef("    Found %d good configs for %s, skipping rest", successCount, family)
				break
			}

			result := ds.testPresetWithBestPayload(preset)
			ds.storeResult(preset, result)

			if result.Status == CheckStatusComplete {
				successCount++
				if result.Speed > bestSpeed {
					bestSpeed = result.Speed
					bestPreset = preset
					// Store with best payload
					bestPreset.Config.Faking.SNIType = ds.bestPayload
				}
				log.Tracef("    %s: %.2f KB/s", preset.Name, result.Speed/1024)
			}
		}

		if bestSpeed > 0 {
			bestParams[family] = bestPreset
			log.Infof("  Best %s config: %s (%.2f KB/s)", family, bestPreset.Name, bestSpeed/1024)
		}
	}

	return bestParams
}

func (ds *DiscoverySuite) runPhase3(workingFamilies []StrategyFamily, bestParams map[StrategyFamily]ConfigPreset) {
	presets := GetCombinationPresets(workingFamilies, bestParams)
	if len(presets) == 0 {
		return
	}

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

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete {
			log.Infof("  %s: SUCCESS (%.2f KB/s)", preset.Name, result.Speed/1024)
		} else {
			log.Tracef("  %s: FAILED", preset.Name)
		}
	}
}

func (ds *DiscoverySuite) testPresetInternal(preset ConfigPreset) CheckResult {
	testConfig := ds.buildTestConfig(preset)

	if err := ds.pool.UpdateConfig(testConfig); err != nil {
		log.Errorf("Failed to apply preset %s: %v", preset.Name, err)
		return CheckResult{
			Domain: ds.domain,
			Status: CheckStatusFailed,
			Error:  err.Error(),
		}
	}

	time.Sleep(time.Duration(ds.Config.ConfigPropagateTimeout) * time.Millisecond)

	result := ds.fetchWithTimeout(QUICK_FAIL_TIMEOUT)
	if result.Status == CheckStatusFailed && result.BytesRead == 0 {
		result = ds.fetchWithTimeout(ds.Config.Timeout)
	}

	result.Set = testConfig.MainSet

	return result
}

func (ds *DiscoverySuite) testPreset(preset ConfigPreset) CheckResult {
	result := ds.testPresetInternal(preset)

	ds.CheckSuite.mu.Lock()
	ds.CompletedChecks++
	ds.CheckSuite.mu.Unlock()

	return result
}

func (ds *DiscoverySuite) fetchWithTimeout(timeout time.Duration) CheckResult {
	result := CheckResult{
		Domain:    ds.domain,
		Status:    CheckStatusRunning,
		Timestamp: time.Now(),
	}

	testURL := fmt.Sprintf("https://%s/", ds.domain)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
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

func (ds *DiscoverySuite) storeResult(preset ConfigPreset, result CheckResult) {
	ds.CheckSuite.mu.Lock()
	defer ds.CheckSuite.mu.Unlock()

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

	if result.Status == CheckStatusComplete && preset.Name != "no-bypass" {
		if result.Speed > ds.domainResult.BestSpeed {
			ds.domainResult.BestPreset = preset.Name
			ds.domainResult.BestSpeed = result.Speed
			ds.domainResult.BestSuccess = true
		}
	}

	ds.DomainDiscoveryResults = map[string]*DomainDiscoveryResult{ds.domain: ds.domainResult}
}

func (ds *DiscoverySuite) determineBest(baselineSpeed float64) {
	ds.CheckSuite.mu.Lock()
	defer ds.CheckSuite.mu.Unlock()

	var bestPreset string
	var bestSpeed float64

	for presetName, result := range ds.domainResult.Results {
		if result.Status == CheckStatusComplete && result.Speed > bestSpeed {
			if presetName == "no-bypass" {
				continue
			}
			bestPreset = presetName
			bestSpeed = result.Speed
		}
	}

	ds.domainResult.BestPreset = bestPreset
	ds.domainResult.BestSpeed = bestSpeed
	ds.domainResult.BestSuccess = bestSpeed > 0
	ds.domainResult.BaselineSpeed = baselineSpeed

	if baselineSpeed > 0 && bestSpeed > 0 {
		ds.domainResult.Improvement = ((bestSpeed - baselineSpeed) / baselineSpeed) * 100
	}
}

func (ds *DiscoverySuite) buildTestConfig(preset ConfigPreset) *config.Config {
	mainSet := config.NewSetConfig()
	mainSet.Id = ds.originalConfig.MainSet.Id
	mainSet.Name = preset.Name
	mainSet.TCP = preset.Config.TCP
	mainSet.UDP = preset.Config.UDP
	mainSet.Fragmentation = preset.Config.Fragmentation
	mainSet.Faking = preset.Config.Faking
	mainSet.Enabled = preset.Config.Enabled

	if preset.Config.Enabled {
		mainSet.Targets.SNIDomains = []string{ds.domain}
		mainSet.Targets.DomainsToMatch = []string{ds.domain}
	}

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

func (ds *DiscoverySuite) finalize() {
	ds.CheckSuite.mu.Lock()
	ds.DomainDiscoveryResults = map[string]*DomainDiscoveryResult{ds.domain: ds.domainResult}
	ds.Status = CheckStatusComplete
	ds.CheckSuite.mu.Unlock()
}

func (ds *DiscoverySuite) restoreConfig() {
	log.Infof("Restoring original configuration")
	if err := ds.pool.UpdateConfig(ds.originalConfig); err != nil {
		log.Errorf("Failed to restore original configuration: %v", err)
	}
}

func (ds *DiscoverySuite) logDiscoverySummary() {
	ds.CheckSuite.mu.RLock()
	defer ds.CheckSuite.mu.RUnlock()

	log.Infof("\n=== Discovery Results for %s ===", ds.domain)

	// Log payload detection results
	for _, pr := range ds.workingPayloads {
		payloadName := "google"
		if pr.Payload == config.FakePayloadDefault2 {
			payloadName = "duckduckgo"
		}
		status := "FAILED"
		if pr.Works {
			status = fmt.Sprintf("SUCCESS (%.2f KB/s)", pr.Speed/1024)
		}
		log.Infof("  Payload %s: %s", payloadName, status)
	}

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

func (ds *DiscoverySuite) runExtendedSearch() []StrategyFamily {
	families := []StrategyFamily{
		FamilyTCPFrag,
		FamilyTLSRec,
		FamilyOOB,
		FamilyFakeSNI,
		FamilyIPFrag,
		FamilySACK,
		FamilyDesync,
		FamilySynFake,
		FamilyDelay,
	}

	var workingFamilies []StrategyFamily

	for _, family := range families {
		select {
		case <-ds.cancel:
			return workingFamilies
		default:
		}

		presets := GetPhase2Presets(family)

		ds.CheckSuite.mu.Lock()
		ds.TotalChecks += len(presets)
		ds.CheckSuite.mu.Unlock()

		log.Infof("  Extended search: %s (%d variants)", family, len(presets))

		for _, preset := range presets {
			select {
			case <-ds.cancel:
				return workingFamilies
			default:
			}

			result := ds.testPresetWithBestPayload(preset)
			ds.storeResult(preset, result)

			if result.Status == CheckStatusComplete {
				log.Infof("    %s: SUCCESS (%.2f KB/s)", preset.Name, result.Speed/1024)
				if !containsFamily(workingFamilies, family) {
					workingFamilies = append(workingFamilies, family)
				}
			}
		}
	}

	return workingFamilies
}

func analyzeFailure(result CheckResult) FailureMode {
	if result.Error == "" {
		return FailureUnknown
	}
	err := strings.ToLower(result.Error)

	if strings.Contains(err, "reset") || strings.Contains(err, "rst") {
		if result.Duration < 100*time.Millisecond {
			return FailureRSTImmediate
		}
	}
	if strings.Contains(err, "timeout") || strings.Contains(err, "deadline") {
		return FailureTimeout
	}
	if strings.Contains(err, "tls") || strings.Contains(err, "certificate") {
		return FailureTLSError
	}
	return FailureUnknown
}

func suggestFamiliesForFailure(mode FailureMode) []StrategyFamily {
	switch mode {
	case FailureRSTImmediate:
		// DPI inline, stateful - need desync/fake
		return []StrategyFamily{FamilyDesync, FamilyFakeSNI, FamilySynFake}
	case FailureTimeout:
		// Packets dropped - fragmentation helps
		return []StrategyFamily{FamilyTCPFrag, FamilyTLSRec, FamilyOOB}
	default:
		return nil
	}
}

func reorderByFamilies(presets []ConfigPreset, priority []StrategyFamily) []ConfigPreset {
	priorityMap := make(map[StrategyFamily]int)
	for i, f := range priority {
		priorityMap[f] = i
	}

	sort.SliceStable(presets, func(i, j int) bool {
		pi, oki := priorityMap[presets[i].Family]
		pj, okj := priorityMap[presets[j].Family]
		if oki && !okj {
			return true
		}
		if !oki && okj {
			return false
		}
		if oki && okj {
			return pi < pj
		}
		return false
	})
	return presets
}
