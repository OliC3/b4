package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
)

type FailureMode string

const (
	MIN_BYTES_FOR_SUCCESS = 4 * 1024   // At least 4KB downloaded
	MIN_SPEED_FOR_SUCCESS = 100 * 1024 // At least 100 KB/s
)

const (
	FailureRSTImmediate FailureMode = "rst_immediate"
	FailureTimeout      FailureMode = "timeout"
	FailureTLSError     FailureMode = "tls_error"
	FailureUnknown      FailureMode = "unknown"
)

type PayloadTestResult struct {
	Speed   float64
	Payload int
	Works   bool
}

type DiscoverySuite struct {
	*CheckSuite
	networkBaseline float64

	pool         *nfq.Pool
	cfg          *config.Config
	domainResult *DomainDiscoveryResult

	// Detected working payload(s)
	workingPayloads []PayloadTestResult
	bestPayload     int

	dnsResult *DNSDiscoveryResult
}

func NewDiscoverySuite(input string, pool *nfq.Pool) *DiscoverySuite {

	suite := NewCheckSuite(input)
	return &DiscoverySuite{
		CheckSuite: suite,
		pool:       pool,
		domainResult: &DomainDiscoveryResult{
			Domain:  suite.Domain,
			Results: make(map[string]*DomainPresetResult),
		},
		workingPayloads: []PayloadTestResult{},
		bestPayload:     config.FakePayloadDefault1, // default
	}
}

func parseDiscoveryInput(input string) (domain string, testURL string) {
	input = strings.TrimSpace(input)

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		u, err := url.Parse(input)
		if err == nil && u.Host != "" {
			return u.Host, input
		}
	}

	return input, "https://" + input + "/"
}

func (ds *DiscoverySuite) RunDiscovery() {
	log.SetDiscoveryActive(true)

	log.DiscoveryLogf("═══════════════════════════════════════")
	log.DiscoveryLogf("Starting discovery for domain: %s", ds.Domain)
	log.DiscoveryLogf("═══════════════════════════════════════")

	suitesMu.Lock()
	activeSuites[ds.Id] = ds.CheckSuite
	suitesMu.Unlock()

	defer func() {
		log.SetDiscoveryActive(false)
		ds.EndTime = time.Now()
	}()

	ds.setStatus(CheckStatusRunning)

	phase1Count := len(GetPhase1Presets())
	ds.CheckSuite.mu.Lock()
	ds.TotalChecks = phase1Count
	ds.CheckSuite.mu.Unlock()

	ds.cfg = ds.pool.GetFirstWorkerConfig()

	if ds.cfg == nil {
		log.Errorf("Failed to get original configuration")
		ds.setStatus(CheckStatusFailed)
		return
	}

	ds.networkBaseline = ds.measureNetworkBaseline()

	log.DiscoveryLogf("Starting discovery for domain: %s", ds.Domain)

	ds.setPhase(PhaseDNS)
	dnsResult := ds.runDNSDiscovery()
	ds.domainResult.DNSResult = dnsResult

	if dnsResult != nil && len(dnsResult.ExpectedIPs) > 0 {
		ds.dnsResult = dnsResult
		log.DiscoveryLogf("Stored %d target IPs for preset testing: %v", len(dnsResult.ExpectedIPs), dnsResult.ExpectedIPs)
	}

	if dnsResult != nil && dnsResult.IsPoisoned {
		if dnsResult.hasWorkingConfig() {
			log.DiscoveryLogf("DNS poisoned - applying discovered DNS bypass for TCP testing")
			ds.applyDNSConfig(dnsResult)
		} else if len(dnsResult.ExpectedIPs) > 0 {
			log.DiscoveryLogf("DNS poisoned, no bypass - using direct IPs: %v", dnsResult.ExpectedIPs)
		} else {
			log.DiscoveryLogf("DNS poisoned but no expected IP known - discovery may fail")
		}
	}

	if dnsResult != nil && len(dnsResult.ExpectedIPs) > 0 {
		ds.dnsResult = dnsResult
	}

	phase1Presets := GetPhase1Presets()

	ds.CheckSuite.mu.Lock()
	ds.TotalChecks = len(phase1Presets)
	ds.CheckSuite.mu.Unlock()

	ds.setPhase(PhaseStrategy)
	workingFamilies, baselineSpeed, baselineWorks := ds.runPhase1(phase1Presets)
	ds.determineBest(baselineSpeed)

	if baselineWorks {
		dnsNeeded := dnsResult != nil && dnsResult.IsPoisoned && dnsResult.hasWorkingConfig()

		if !dnsNeeded {
			ds.CheckSuite.mu.Lock()
			ds.TotalChecks = 1
			ds.domainResult.BestPreset = "no-bypass"
			ds.domainResult.BestSpeed = baselineSpeed
			ds.domainResult.BestSuccess = true
			ds.domainResult.BaselineSpeed = baselineSpeed
			ds.domainResult.Improvement = 0
			ds.CheckSuite.mu.Unlock()

			log.DiscoveryLogf("Baseline succeeded for %s - no DPI bypass needed", ds.Domain)
			ds.restoreConfig()
			ds.finalize()
			ds.logDiscoverySummary()
			return
		}

		log.DiscoveryLogf("TCP works for %s but DNS bypass required - testing minimal preset", ds.Domain)
	}

	if len(workingFamilies) == 0 {
		log.Warnf("Phase 1 found no working families, trying extended search")

		ds.setPhase(PhaseOptimize)
		workingFamilies = ds.runExtendedSearch()

		if len(workingFamilies) == 0 {
			log.Warnf("No working bypass strategies found for %s", ds.Domain)
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

func (ds *DiscoverySuite) runPhase1(presets []ConfigPreset) ([]StrategyFamily, float64, bool) {
	var workingFamilies []StrategyFamily
	var baselineSpeed float64

	log.DiscoveryLogf("Phase 1: Testing %d strategy families", len(presets))

	baselineResult := ds.testPreset(presets[0])
	ds.storeResult(presets[0], baselineResult)

	baselineWorks := baselineResult.Status == CheckStatusComplete
	if baselineWorks {
		baselineSpeed = baselineResult.Speed
		return workingFamilies, baselineSpeed, true
	}

	ds.detectWorkingPayloads(presets)

	strategyPresets := ds.filterTestedPresets(presets)

	baselineFailureMode := analyzeFailure(baselineResult)
	suggestedFamilies := suggestFamiliesForFailure(baselineFailureMode)

	if len(suggestedFamilies) > 0 {
		strategyPresets = reorderByFamilies(strategyPresets, suggestedFamilies)
		log.DiscoveryLogf("  Failure mode: %s - prioritizing: %v", baselineFailureMode, suggestedFamilies)
	}

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
			}
		}
	}

	return workingFamilies, baselineSpeed, false
}

func (ds *DiscoverySuite) detectWorkingPayloads(presets []ConfigPreset) {
	log.DiscoveryLogf("  Testing payload variants...")

	var payload1Preset, payload2Preset *ConfigPreset
	for i := range presets {
		if presets[i].Name == "proven-combo" {
			payload1Preset = &presets[i]
		}
		if presets[i].Name == "proven-combo-alt" {
			payload2Preset = &presets[i]
		}
	}

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
				log.DiscoveryLogf("    Payload 1 (google): SUCCESS (%.2f KB/s)", result1.Speed/1024)
			} else {
				log.DiscoveryLogf("    Payload 1 (google): FAILED")
			}
		}
	}

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
				log.DiscoveryLogf("    Payload 2 (duckduckgo): SUCCESS (%.2f KB/s)", result2.Speed/1024)
			} else {
				log.DiscoveryLogf("    Payload 2 (duckduckgo): FAILED")
			}
		}
	}

	ds.selectBestPayload()
}

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
		log.DiscoveryLogf("  Neither payload worked in baseline - will test both during discovery")
	case 1:
		payloadName := "google"
		if ds.bestPayload == config.FakePayloadDefault2 {
			payloadName = "duckduckgo"
		}
		log.DiscoveryLogf("  Selected payload: %s (only one works)", payloadName)
	case 2:
		payloadName := "google"
		if ds.bestPayload == config.FakePayloadDefault2 {
			payloadName = "duckduckgo"
		}
		log.DiscoveryLogf("  Selected payload: %s (faster of both working)", payloadName)
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

	hasWorkingPayload := false
	for _, pr := range ds.workingPayloads {
		if pr.Works {
			hasWorkingPayload = true
			break
		}
	}

	if hasWorkingPayload {
		return ds.testPresetWithPayload(preset, ds.bestPayload)
	}

	result1 := ds.testPresetWithPayload(preset, config.FakePayloadDefault1)
	if result1.Status == CheckStatusComplete {
		ds.updatePayloadKnowledge(config.FakePayloadDefault1, result1.Speed)
		return result1
	}

	result2 := ds.testPresetWithPayload(preset, config.FakePayloadDefault2)
	if result2.Status == CheckStatusComplete {
		ds.updatePayloadKnowledge(config.FakePayloadDefault2, result2.Speed)
		return result2
	}

	return result1
}

func (ds *DiscoverySuite) testPresetWithPayload(preset ConfigPreset, payloadType int) CheckResult {
	modifiedPreset := preset
	modifiedPreset.Config.Faking.SNIType = payloadType
	return ds.testPresetInternal(modifiedPreset)
}

func (ds *DiscoverySuite) updatePayloadKnowledge(payload int, speed float64) {
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

	ds.workingPayloads = append(ds.workingPayloads, PayloadTestResult{
		Payload: payload,
		Works:   true,
		Speed:   speed,
	})
	ds.selectBestPayload()
}

func (ds *DiscoverySuite) runPhase2(families []StrategyFamily) map[StrategyFamily]ConfigPreset {
	bestParams := make(map[StrategyFamily]ConfigPreset)

	log.DiscoveryLogf("Phase 2: Optimizing %d working families", len(families))

	for _, family := range families {
		select {
		case <-ds.cancel:
			return bestParams
		default:
		}

		switch family {
		case FamilyFakeSNI:
			bestParams[family] = ds.optimizeFakeSNI()
		case FamilyTCPFrag:
			bestParams[family] = ds.optimizeTCPFrag()
		case FamilyTLSRec:
			bestParams[family] = ds.optimizeTLSRec()
		default:
			bestParams[family] = ds.optimizeWithPresets(family)
		}
	}

	return bestParams
}

func (ds *DiscoverySuite) optimizeFakeSNI() ConfigPreset {
	log.DiscoveryLogf("  Optimizing FakeSNI with binary search")

	ds.CheckSuite.mu.Lock()
	ds.TotalChecks += 9
	ds.CheckSuite.mu.Unlock()

	base := baseConfig()
	base.Faking.SNI = true
	base.Faking.Strategy = "pastseq"
	base.Faking.SeqOffset = 10000
	base.Faking.SNISeqLength = 1
	base.Faking.SNIType = ds.bestPayload
	base.Fragmentation.Strategy = "tcp"
	base.Fragmentation.SNIPosition = 1
	base.Fragmentation.ReverseOrder = true

	basePreset := ConfigPreset{
		Name:   "fake-optimize",
		Family: FamilyFakeSNI,
		Phase:  PhaseOptimize,
		Config: base,
	}

	optimalTTL, speed := ds.findOptimalTTL(basePreset, 0)
	if optimalTTL == 0 {
		log.DiscoveryLogf("  No working TTL found for FakeSNI")
		return basePreset
	}

	basePreset.Config.Faking.TTL = optimalTTL
	basePreset.Name = fmt.Sprintf("fake-ttl%d-optimized", optimalTTL)

	strategies := []string{"pastseq", "ttl", "randseq"}
	var bestStrategy string = "pastseq"
	var bestSpeed = speed

	for _, strat := range strategies {
		if strat == "pastseq" {
			continue // Already tested
		}

		preset := basePreset
		preset.Name = fmt.Sprintf("fake-%s-ttl%d", strat, optimalTTL)
		preset.Config.Faking.Strategy = strat

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete && result.Speed > bestSpeed {
			bestStrategy = strat
			bestSpeed = result.Speed
		}
	}

	basePreset.Config.Faking.Strategy = bestStrategy
	basePreset.Name = fmt.Sprintf("fake-%s-ttl%d-optimized", bestStrategy, optimalTTL)

	log.DiscoveryLogf("  Best FakeSNI: TTL=%d, strategy=%s (%.2f KB/s)", optimalTTL, bestStrategy, bestSpeed/1024)
	return basePreset
}

func (ds *DiscoverySuite) optimizeTCPFrag() ConfigPreset {
	log.DiscoveryLogf("  Optimizing TCPFrag with binary search")

	ds.CheckSuite.mu.Lock()
	ds.TotalChecks += 6
	ds.CheckSuite.mu.Unlock()

	base := baseConfig()
	base.Fragmentation.Strategy = "tcp"
	base.Fragmentation.ReverseOrder = true
	base.Faking.SNI = true

	base.Faking.TTL = 8

	base.Faking.Strategy = "pastseq"
	base.Faking.SNIType = ds.bestPayload

	basePreset := ConfigPreset{
		Name:   "tcp-optimize",
		Family: FamilyTCPFrag,
		Phase:  PhaseOptimize,
		Config: base,
	}

	optimalPos, speed := ds.findOptimalPosition(basePreset, 16)
	if optimalPos == 0 {
		optimalPos = 1
	}

	basePreset.Config.Fragmentation.SNIPosition = optimalPos
	basePreset.Name = fmt.Sprintf("tcp-pos%d-optimized", optimalPos)

	middlePreset := basePreset
	middlePreset.Name = fmt.Sprintf("tcp-pos%d-middle", optimalPos)
	middlePreset.Config.Fragmentation.MiddleSNI = true

	result := ds.testPresetWithBestPayload(middlePreset)
	ds.storeResult(middlePreset, result)

	if result.Status == CheckStatusComplete && result.Speed > speed {
		basePreset = middlePreset
		speed = result.Speed
		log.DiscoveryLogf("  MiddleSNI improves speed: %.2f KB/s", result.Speed/1024)
	}

	log.DiscoveryLogf("  Best TCPFrag: position=%d (%.2f KB/s)", optimalPos, speed/1024)
	return basePreset
}

func (ds *DiscoverySuite) optimizeTLSRec() ConfigPreset {
	log.DiscoveryLogf("  Optimizing TLSRec with binary search")

	ds.CheckSuite.mu.Lock()
	ds.TotalChecks += 6
	ds.CheckSuite.mu.Unlock()

	base := baseConfig()
	base.Fragmentation.Strategy = "tls"
	base.Faking.SNI = true
	base.Faking.TTL = 8

	base.Faking.Strategy = "pastseq"
	base.Faking.SNIType = ds.bestPayload

	basePreset := ConfigPreset{
		Name:   "tls-optimize",
		Family: FamilyTLSRec,
		Phase:  PhaseOptimize,
		Config: base,
	}

	low, high := 1, 64
	var bestPos int
	var bestSpeed float64

	for low < high {
		mid := (low + high) / 2

		preset := basePreset
		preset.Name = fmt.Sprintf("tls-pos-search-%d", mid)
		preset.Config.Fragmentation.TLSRecordPosition = mid

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete {
			bestPos = mid
			bestSpeed = result.Speed
			high = mid
		} else {
			low = mid + 1
		}
	}

	if bestPos > 0 {
		basePreset.Config.Fragmentation.TLSRecordPosition = bestPos
		basePreset.Name = fmt.Sprintf("tls-pos%d-optimized", bestPos)
	}

	log.DiscoveryLogf("  Best TLSRec: position=%d (%.2f KB/s)", bestPos, bestSpeed/1024)
	return basePreset
}

func (ds *DiscoverySuite) optimizeWithPresets(family StrategyFamily) ConfigPreset {
	presets := GetPhase2Presets(family)
	if len(presets) == 0 {
		return ConfigPreset{Family: family}
	}

	ds.CheckSuite.mu.Lock()
	ds.TotalChecks += len(presets)
	ds.CheckSuite.mu.Unlock()

	log.DiscoveryLogf("  Optimizing %s with %d presets", family, len(presets))

	var bestPreset ConfigPreset
	var bestSpeed float64

	for _, preset := range presets {
		select {
		case <-ds.cancel:
			return bestPreset
		default:
		}

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete && result.Speed > bestSpeed {
			bestSpeed = result.Speed
			bestPreset = preset
			bestPreset.Config.Faking.SNIType = ds.bestPayload
		}
	}

	return bestPreset
}

func (ds *DiscoverySuite) runPhase3(workingFamilies []StrategyFamily, bestParams map[StrategyFamily]ConfigPreset) {
	presets := GetCombinationPresets(workingFamilies, bestParams)
	if len(presets) == 0 {
		return
	}

	ds.CheckSuite.mu.Lock()
	ds.TotalChecks += len(presets)
	ds.CheckSuite.mu.Unlock()

	log.DiscoveryLogf("Phase 3: Testing %d combination presets", len(presets))

	for _, preset := range presets {
		select {
		case <-ds.cancel:
			return
		default:
		}

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete {
			log.DiscoveryLogf("  ✓ %s: %.2f KB/s", preset.Name, result.Speed/1024)
		} else {
			log.DiscoveryLogf("  ✗ %s: failed", preset.Name)
		}
	}
}

func (ds *DiscoverySuite) testPresetInternal(preset ConfigPreset) CheckResult {
	log.DiscoveryLogf("  Testing '%s'...", preset.Name)

	testConfig := ds.buildTestConfig(preset)

	if err := ds.pool.UpdateConfig(testConfig); err != nil {
		log.DiscoveryLogf("    → FAILED (config error: %v)", err)
		return CheckResult{
			Domain: ds.Domain,
			Status: CheckStatusFailed,
			Error:  err.Error(),
		}
	}

	time.Sleep(time.Duration(ds.cfg.System.Checker.ConfigPropagateMs) * time.Millisecond)

	result := ds.fetchWithTimeout(time.Duration(ds.cfg.System.Checker.DiscoveryTimeoutSec) * time.Second)
	result.Set = testConfig.MainSet

	if result.Status == CheckStatusComplete {
		log.DiscoveryLogf("    → OK (%.2f KB/s, %d bytes)", result.Speed/1024, result.BytesRead)
	} else {
		log.DiscoveryLogf("    → FAILED (%s)", result.Error)
	}
	return result
}

func (ds *DiscoverySuite) testPreset(preset ConfigPreset) CheckResult {
	defer func() {
		ds.CheckSuite.mu.Lock()
		ds.CompletedChecks++
		ds.CheckSuite.mu.Unlock()
	}()

	return ds.testPresetInternal(preset)
}

func (ds *DiscoverySuite) fetchWithTimeout(timeout time.Duration) CheckResult {
	geoip, geosite := GetCDNCategories(ds.Domain)
	if len(geoip) > 0 || len(geosite) > 0 {
		return ds.fetchWithTimeoutUsingIP(timeout, "")
	}

	var allIPs []string
	if ds.dnsResult != nil {
		allIPs = append(allIPs, ds.dnsResult.ExpectedIPs...)
		for _, probe := range ds.dnsResult.ProbeResults {
			if probe.ResolvedIP != "" {
				found := false
				for _, ip := range allIPs {
					if ip == probe.ResolvedIP {
						found = true
						break
					}
				}
				if !found {
					allIPs = append(allIPs, probe.ResolvedIP)
				}
			}
		}
	}

	freshIPs, _ := net.LookupIP(ds.Domain)
	for _, ip := range freshIPs {
		ipStr := ip.String()
		found := false
		for _, existing := range allIPs {
			if existing == ipStr {
				found = true
				break
			}
		}
		if !found {
			allIPs = append([]string{ipStr}, allIPs...)
		}
	}

	for _, ip := range allIPs {
		result := ds.fetchWithTimeoutUsingIP(timeout, ip)
		if result.Status == CheckStatusComplete {
			log.Tracef("Success with IP %s", ip)
			return result
		}
		log.Tracef("IP %s failed, trying next", ip)
	}

	if len(allIPs) > 0 {
		return CheckResult{
			Domain: ds.Domain,
			Status: CheckStatusFailed,
			Error:  fmt.Sprintf("all %d IPs failed", len(allIPs)),
		}
	}

	return ds.fetchWithTimeoutUsingIP(timeout, "")
}

func (ds *DiscoverySuite) fetchWithTimeoutUsingIP(timeout time.Duration, ip string) CheckResult {
	result := CheckResult{
		Domain:    ds.Domain,
		Status:    CheckStatusRunning,
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       timeout,
	}

	if ip != "" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, _ := net.SplitHostPort(addr)
			if port == "" {
				port = "443"
			}
			directAddr := net.JoinHostPort(ip, port)
			log.Tracef("DNS bypass: connecting to %s instead of %s", directAddr, addr)
			return (&net.Dialer{
				Timeout:   timeout / 2,
				KeepAlive: timeout,
			}).DialContext(ctx, network, directAddr)
		}
	} else {
		transport.DialContext = (&net.Dialer{
			Timeout:   timeout / 2,
			KeepAlive: timeout,
		}).DialContext
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ds.CheckURL, nil)
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
	result.ContentSize = resp.ContentLength

	buf := make([]byte, 16*1024)
	var bytesRead int64
	lastProgress := time.Now()

	for bytesRead < 100*1024 {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(start)
			result.BytesRead = bytesRead
			if bytesRead >= MIN_BYTES_FOR_SUCCESS {
				result.Status = CheckStatusComplete
				if result.Duration.Seconds() > 0 {
					result.Speed = float64(bytesRead) / result.Duration.Seconds()
				}
			} else {
				result.Status = CheckStatusFailed
				result.Error = fmt.Sprintf("timeout after %d bytes (need %d)", bytesRead, MIN_BYTES_FOR_SUCCESS)
			}
			return result
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			bytesRead += int64(n)
			lastProgress = time.Now()
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			result.Status = CheckStatusFailed
			result.Error = fmt.Sprintf("read error after %d bytes: %v", bytesRead, err)
			result.Duration = time.Since(start)
			result.BytesRead = bytesRead
			return result
		}

		if time.Since(lastProgress) > 2*time.Second {
			result.Status = CheckStatusFailed
			result.Error = fmt.Sprintf("stalled after %d bytes", bytesRead)
			result.Duration = time.Since(start)
			result.BytesRead = bytesRead
			return result
		}
	}

	duration := time.Since(start)
	result.Duration = duration
	result.BytesRead = bytesRead

	if bytesRead < MIN_BYTES_FOR_SUCCESS {
		if result.ContentSize > 0 {
			if bytesRead >= result.ContentSize {
				result.Status = CheckStatusComplete
				result.Speed = 0
				log.Tracef("Small but complete response (%d/%d bytes, HTTP %d)", bytesRead, result.ContentSize, resp.StatusCode)
				return result
			}
			result.Status = CheckStatusFailed
			result.Error = fmt.Sprintf("truncated: got %d of %d bytes", bytesRead, result.ContentSize)
			return result
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 500 && bytesRead > 0 {
			result.Status = CheckStatusComplete
			result.Speed = 0
			log.Tracef("Small response without Content-Length (%d bytes, HTTP %d)", bytesRead, resp.StatusCode)
			return result
		}

		result.Status = CheckStatusFailed
		result.Error = fmt.Sprintf("insufficient data: %d bytes (need %d)", bytesRead, MIN_BYTES_FOR_SUCCESS)
		return result
	}

	if duration.Seconds() > 0 {
		result.Speed = float64(bytesRead) / duration.Seconds()
	}

	result.Status = CheckStatusComplete
	return result
}

func (ds *DiscoverySuite) storeResult(preset ConfigPreset, result CheckResult) {
	ds.CheckSuite.mu.Lock()
	defer ds.CheckSuite.mu.Unlock()

	switch result.Status {
	case CheckStatusComplete:
		ds.SuccessfulChecks++
	case CheckStatusFailed:
		ds.FailedChecks++
	}

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
			oldBest := ds.domainResult.BestSpeed
			ds.domainResult.BestPreset = preset.Name
			ds.domainResult.BestSpeed = result.Speed
			ds.domainResult.BestSuccess = true
			if oldBest > 0 {
				improvement := ((result.Speed - oldBest) / oldBest) * 100
				log.DiscoveryLogf("★ New best: %s at %.2f KB/s (+%.0f%%)", preset.Name, result.Speed/1024, improvement)
			} else {
				log.DiscoveryLogf("★ First success: %s at %.2f KB/s", preset.Name, result.Speed/1024)
			}
		}
	}

	ds.DomainDiscoveryResults = map[string]*DomainDiscoveryResult{ds.Domain: ds.domainResult}
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
	mainSet.Id = ds.cfg.MainSet.Id
	mainSet.Name = preset.Name
	mainSet.TCP = preset.Config.TCP
	mainSet.UDP = preset.Config.UDP
	mainSet.Fragmentation = preset.Config.Fragmentation
	mainSet.Faking = preset.Config.Faking
	mainSet.DNS = ds.cfg.MainSet.DNS

	if mainSet.TCP.WinMode == "" {
		mainSet.TCP.WinMode = config.ConfigOff
	}
	if mainSet.TCP.DesyncMode == "" {
		mainSet.TCP.DesyncMode = config.ConfigOff
	}

	if mainSet.Faking.SNIMutation.Mode == "" {
		mainSet.Faking.SNIMutation.Mode = config.ConfigOff
	}
	if mainSet.Faking.SNIMutation.FakeSNIs == nil {
		mainSet.Faking.SNIMutation.FakeSNIs = []string{}
	}

	if preset.Name == "no-bypass" {
		mainSet.Enabled = false
	} else {
		mainSet.Enabled = true
		mainSet.Targets.SNIDomains = []string{ds.Domain}
		mainSet.Targets.DomainsToMatch = []string{ds.Domain}

		geoip, geosite := GetCDNCategories(ds.Domain)
		if len(geoip) > 0 || len(geosite) > 0 {
			if len(geoip) > 0 {
				mainSet.Targets.GeoIpCategories = geoip
			}
			if len(geosite) > 0 {
				mainSet.Targets.GeoSiteCategories = geosite
			}

			if len(ds.cfg.System.Checker.ReferenceDNS) > 0 {
				mainSet.DNS = config.DNSConfig{
					Enabled:       true,
					TargetDNS:     ds.cfg.System.Checker.ReferenceDNS[0],
					FragmentQuery: true,
				}
			} else {
				mainSet.DNS = config.DNSConfig{
					Enabled:       true,
					TargetDNS:     "9.9.9.9",
					FragmentQuery: true,
				}
			}

			tempCfg := &config.Config{System: ds.cfg.System}
			domains, ips, err := tempCfg.GetTargetsForSet(&mainSet)
			if err != nil {
				log.DiscoveryLogf("Discovery: failed to load CDN categories: %v", err)
			} else {
				log.Tracef("Discovery: CDN %s - loaded %d domains, %d IPs", ds.Domain, len(domains), len(ips))
			}
		} else {
			var ipsToAdd []string
			if ds.dnsResult != nil {
				ipsToAdd = append(ipsToAdd, ds.dnsResult.ExpectedIPs...)
				for _, probe := range ds.dnsResult.ProbeResults {
					if probe.ResolvedIP != "" {
						found := false
						for _, ip := range ipsToAdd {
							if ip == probe.ResolvedIP {
								found = true
								break
							}
						}
						if !found {
							ipsToAdd = append(ipsToAdd, probe.ResolvedIP)
						}
					}
				}
			}

			if len(ipsToAdd) > 0 {
				cidrIPs := make([]string, len(ipsToAdd))
				for i, ip := range ipsToAdd {
					if strings.Contains(ip, "/") {
						cidrIPs[i] = ip
					} else if strings.Contains(ip, ":") {
						cidrIPs[i] = ip + "/128"
					} else {
						cidrIPs[i] = ip + "/32"
					}
				}
				mainSet.Targets.IPs = cidrIPs
				mainSet.Targets.IpsToMatch = cidrIPs
				log.Tracef("Discovery: added %d IPs to test config: %v", len(cidrIPs), cidrIPs)
			}
		}
	}

	return &config.Config{
		ConfigPath: ds.cfg.ConfigPath,
		Queue:      ds.cfg.Queue,
		System:     ds.cfg.System,
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
	ds.DomainDiscoveryResults = map[string]*DomainDiscoveryResult{ds.Domain: ds.domainResult}
	ds.Status = CheckStatusComplete
	ds.CheckSuite.mu.Unlock()

	go func() {
		time.Sleep(30 * time.Second)
		suitesMu.Lock()
		delete(activeSuites, ds.Id)
		suitesMu.Unlock()
	}()
}

func (ds *DiscoverySuite) restoreConfig() {
	log.DiscoveryLogf("Restoring original configuration")
	if err := ds.pool.UpdateConfig(ds.cfg); err != nil {
		log.DiscoveryLogf("Failed to restore original configuration: %v", err)
	}
}

func (ds *DiscoverySuite) logDiscoverySummary() {
	ds.CheckSuite.mu.RLock()
	defer ds.CheckSuite.mu.RUnlock()

	duration := time.Since(ds.StartTime)
	totalConfigs := len(ds.domainResult.Results)

	log.DiscoveryLogf("═══════════════════════════════════════")
	if ds.domainResult.BestSuccess {
		improvement := ""
		if ds.domainResult.Improvement > 0 {
			improvement = fmt.Sprintf(" (+%.0f%% vs baseline)", ds.domainResult.Improvement)
		}
		log.DiscoveryLogf("✓ Discovery complete: %s", ds.Domain)
		log.DiscoveryLogf("  Best config: %s", ds.domainResult.BestPreset)
		log.DiscoveryLogf("  Speed: %.2f KB/s%s", ds.domainResult.BestSpeed/1024, improvement)
	} else {
		log.DiscoveryLogf("✗ Discovery complete: no working config found")
	}
	log.DiscoveryLogf("  Tested %d configurations in %v", totalConfigs, duration.Round(time.Second))
	log.DiscoveryLogf("═══════════════════════════════════════")
}

func (ds *DiscoverySuite) runExtendedSearch() []StrategyFamily {
	families := []StrategyFamily{
		FamilyCombo,
		FamilyDisorder,
		FamilyOverlap,
		FamilyExtSplit,
		FamilyFirstByte,
		FamilyTCPFrag,
		FamilyTLSRec,
		FamilyOOB,
		FamilyFakeSNI,
		FamilyIPFrag,
		FamilySACK,
		FamilyDesync,
		FamilySynFake,
		FamilyDelay,
		FamilyHybrid,
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

		log.DiscoveryLogf("  Extended search: %s (%d variants)", family, len(presets))

		for _, preset := range presets {
			select {
			case <-ds.cancel:
				return workingFamilies
			default:
			}

			result := ds.testPresetWithBestPayload(preset)
			ds.storeResult(preset, result)

			if result.Status == CheckStatusComplete {
				log.DiscoveryLogf("    %s: SUCCESS (%.2f KB/s)", preset.Name, result.Speed/1024)
				if !containsFamily(workingFamilies, family) {
					workingFamilies = append(workingFamilies, family)
				}
			}
		}
	}

	return workingFamilies
}

// FindOptimalTTL uses binary search to find minimum working TTL, then verifies speed
func (ds *DiscoverySuite) findOptimalTTL(basePreset ConfigPreset, hint uint8) (uint8, float64) {
	var bestTTL uint8
	var bestSpeed float64
	low, high := uint8(1), uint8(32)

	// If hint provided, test it first and narrow search range
	if hint > 0 {
		preset := basePreset
		preset.Name = fmt.Sprintf("ttl-hint-%d", hint)
		preset.Config.Faking.TTL = hint

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete {
			bestTTL = hint
			bestSpeed = result.Speed
			log.DiscoveryLogf("  TTL hint %d: SUCCESS (%.2f KB/s) - narrowing search", hint, result.Speed/1024)

			// Narrow search: find minimum between 1 and hint
			high = hint
			if hint > 8 {
				low = hint - 8 // Don't search too far below
			}
		} else {
			log.DiscoveryLogf("  TTL hint %d: FAILED - falling back to full search", hint)
		}
	}

	log.DiscoveryLogf("Binary search for optimal TTL (range %d-%d)", low, high)

	for low < high {
		mid := (low + high) / 2

		preset := basePreset
		preset.Name = fmt.Sprintf("ttl-search-%d", mid)
		preset.Config.Faking.TTL = mid

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete {
			bestTTL = mid
			bestSpeed = result.Speed
			high = mid
			log.DiscoveryLogf("  TTL %d: SUCCESS (%.2f KB/s)", mid, result.Speed/1024)
		} else {
			low = mid + 1
			log.Tracef("  TTL %d: FAILED", mid)
		}
	}

	if bestTTL == 0 {
		return 0, 0
	}

	// Test slightly higher TTLs - sometimes better speed
	for _, offset := range []uint8{2, 4} {
		testTTL := bestTTL + offset
		if testTTL > 32 {
			continue
		}

		preset := basePreset
		preset.Name = fmt.Sprintf("ttl-verify-%d", testTTL)
		preset.Config.Faking.TTL = testTTL

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete && result.Speed > bestSpeed*1.1 {
			bestTTL = testTTL
			bestSpeed = result.Speed
			log.DiscoveryLogf("  TTL %d: Better (%.2f KB/s)", testTTL, result.Speed/1024)
		}
	}

	log.DiscoveryLogf("Optimal TTL found: %d (%.2f KB/s)", bestTTL, bestSpeed/1024)
	return bestTTL, bestSpeed
}

// FindOptimalPosition binary searches for minimum working fragmentation position
func (ds *DiscoverySuite) findOptimalPosition(basePreset ConfigPreset, maxPos int) (int, float64) {
	low, high := 1, maxPos
	var bestPos int
	var bestSpeed float64

	log.DiscoveryLogf("Binary search for optimal position (range %d-%d)", low, high)

	for low < high {
		mid := (low + high) / 2

		preset := basePreset
		preset.Name = fmt.Sprintf("pos-search-%d", mid)
		preset.Config.Fragmentation.SNIPosition = mid

		result := ds.testPresetWithBestPayload(preset)
		ds.storeResult(preset, result)

		if result.Status == CheckStatusComplete {
			bestPos = mid
			bestSpeed = result.Speed
			high = mid
			log.DiscoveryLogf("  Position %d: SUCCESS (%.2f KB/s)", mid, result.Speed/1024)
		} else {
			low = mid + 1
			log.Tracef("  Position %d: FAILED", mid)
		}
	}

	return bestPos, bestSpeed
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

func (ds *DiscoverySuite) measureNetworkBaseline() float64 {
	// Test a known-good domain to establish actual network speed
	timeout := time.Duration(ds.cfg.System.Checker.DiscoveryTimeoutSec) * time.Second
	referenceDomain := ds.cfg.System.Checker.ReferenceDomain
	if referenceDomain == "" {
		referenceDomain = config.DefaultConfig.System.Checker.ReferenceDomain
	}

	log.DiscoveryLogf("Measuring network baseline using %s", referenceDomain)

	testURL := fmt.Sprintf("https://%s/", referenceDomain)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout: timeout / 2,
			}).DialContext,
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		log.DiscoveryLogf("Failed to create baseline request: %v", err)
		return 0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.DiscoveryLogf("Baseline measurement failed: %v", err)
		return 0
	}
	defer resp.Body.Close()

	bytesRead, _ := io.CopyN(io.Discard, resp.Body, 100*1024)
	duration := time.Since(start)

	if bytesRead == 0 || duration.Seconds() == 0 {
		return 0
	}

	speed := float64(bytesRead) / duration.Seconds()
	log.DiscoveryLogf("Network baseline: %.2f KB/s (%d bytes in %v)", speed/1024, bytesRead, duration)

	return speed
}
