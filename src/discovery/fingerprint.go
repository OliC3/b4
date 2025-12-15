package discovery

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
)

// DPI system types we can identify
type DPIType string

const (
	DPITypeUnknown   DPIType = "unknown"
	DPITypeTSPU      DPIType = "tspu"      // Russian TSPU (Technical Means of Countering Threats)
	DPITypeSandvine  DPIType = "sandvine"  // Sandvine PacketLogic
	DPITypeHuawei    DPIType = "huawei"    // Huawei eSight/DPI
	DPITypeAllot     DPIType = "allot"     // Allot NetEnforcer
	DPITypeFortigate DPIType = "fortigate" // Fortinet FortiGate
	DPITypeNone      DPIType = "none"      // No DPI detected
)

// DPI blocking mechanism
type BlockingMethod string

const (
	BlockingRSTInject     BlockingMethod = "rst_inject"     // DPI injects RST packets
	BlockingTimeout       BlockingMethod = "timeout"        // Packets silently dropped
	BlockingRedirect      BlockingMethod = "redirect"       // HTTP redirect to block page
	BlockingContentInject BlockingMethod = "content_inject" // Injects fake response
	BlockingTLSAlert      BlockingMethod = "tls_alert"      // TLS alert injection
	BlockingNone          BlockingMethod = "none"           // No blocking
)

// DPI inspection depth
type InspectionDepth string

const (
	InspectionSNIOnly   InspectionDepth = "sni_only"  // Only inspects SNI
	InspectionTLSFull   InspectionDepth = "tls_full"  // Full TLS inspection
	InspectionHTTPFull  InspectionDepth = "http_full" // HTTP content inspection
	InspectionStateful  InspectionDepth = "stateful"  // Tracks connection state
	InspectionStateless InspectionDepth = "stateless" // Per-packet inspection
)

// DPIFingerprint contains all information about detected DPI system
type DPIFingerprint struct {
	Type            DPIType         `json:"type"`
	BlockingMethod  BlockingMethod  `json:"blocking_method"`
	InspectionDepth InspectionDepth `json:"inspection_depth"`

	// Timing characteristics
	RSTLatencyMs   float64 `json:"rst_latency_ms"`   // Time until RST received
	BlockLatencyMs float64 `json:"block_latency_ms"` // Time until blocking detected

	// Network position
	DPIHopCount int  `json:"dpi_hop_count"` // Estimated hops to DPI (from RST TTL)
	IsInline    bool `json:"is_inline"`     // DPI is inline vs mirror/tap

	// Capabilities detected
	InspectsHTTP bool `json:"inspects_http"`
	InspectsTLS  bool `json:"inspects_tls"`
	InspectsQUIC bool `json:"inspects_quic"`
	TracksState  bool `json:"tracks_state"` // Stateful inspection

	// Evasion hints
	VulnerableToTTL    bool `json:"vulnerable_to_ttl"`
	VulnerableToFrag   bool `json:"vulnerable_to_frag"`
	VulnerableToDesync bool `json:"vulnerable_to_desync"`
	VulnerableToOOB    bool `json:"vulnerable_to_oob"`

	// Optimal parameters discovered
	OptimalTTL      uint8  `json:"optimal_ttl,omitempty"`
	OptimalStrategy string `json:"optimal_strategy,omitempty"`

	// Raw probe results for debugging
	ProbeResults map[string]*ProbeResult `json:"probe_results,omitempty"`

	// Confidence score (0-100)
	Confidence int `json:"confidence"`

	// Recommended strategies (ordered by likelihood of success)
	RecommendedFamilies []StrategyFamily `json:"recommended_families"`
}

// ProbeResult stores result of individual probe
type ProbeResult struct {
	ProbeName    string        `json:"probe_name"`
	Success      bool          `json:"success"`
	Blocked      bool          `json:"blocked"`
	Latency      time.Duration `json:"latency"`
	RSTTTL       int           `json:"rst_ttl,omitempty"`
	ErrorType    string        `json:"error_type,omitempty"`
	HTTPCode     int           `json:"http_code,omitempty"`
	ResponseSize int64         `json:"response_size,omitempty"`
	Notes        string        `json:"notes,omitempty"`
}

// DPIProber handles all fingerprinting probes
type DPIProber struct {
	domain  string
	timeout time.Duration
	results map[string]*ProbeResult
	mu      sync.Mutex

	// Reference domain that should NOT be blocked (for comparison)
	referenceDomain string
}

// NewDPIProber creates a new prober for the given domain
func NewDPIProber(domain string, refDomain string, timeout time.Duration) *DPIProber {
	return &DPIProber{
		domain:          domain,
		timeout:         timeout,
		results:         make(map[string]*ProbeResult),
		referenceDomain: refDomain,
	}
}

// Fingerprint runs all probes and returns DPI fingerprint
func (p *DPIProber) Fingerprint(ctx context.Context) *DPIFingerprint {
	fp := &DPIFingerprint{
		Type:           DPITypeUnknown,
		BlockingMethod: BlockingNone,
		ProbeResults:   make(map[string]*ProbeResult),
		Confidence:     0,
	}

	log.Infof("DPI Fingerprinting: Starting probes for %s", p.domain)

	// Run probes in order of information value
	// Each probe builds on information from previous probes

	// 1. Baseline: Is the domain actually blocked?
	baselineResult := p.probeBaseline(ctx)
	fp.ProbeResults["baseline"] = baselineResult

	if !baselineResult.Blocked {
		log.Infof("DPI Fingerprinting: Domain %s is NOT blocked, no DPI detected", p.domain)
		fp.Type = DPITypeNone
		fp.BlockingMethod = BlockingNone
		fp.Confidence = 95
		return fp
	}

	log.Infof("DPI Fingerprinting: Domain is blocked, analyzing blocking method...")

	// 2. Determine blocking method
	p.probeBlockingMethod(ctx, fp)

	// 3. Analyze RST characteristics (if RST-based blocking)
	if fp.BlockingMethod == BlockingRSTInject {
		p.probeRSTCharacteristics(ctx, fp)
	}

	// 4. Test inspection depth
	p.probeInspectionDepth(ctx, fp)

	// 5. Test specific evasion vulnerabilities
	p.probeEvasionVulnerabilities(ctx, fp)

	// 6. Try to identify DPI type based on all collected data
	p.identifyDPIType(fp)

	// 7. Generate recommendations
	p.generateRecommendations(fp)

	// Copy results
	p.mu.Lock()
	for k, v := range p.results {
		fp.ProbeResults[k] = v
	}
	p.mu.Unlock()

	p.logFingerprint(fp)

	return fp
}

// probeBaseline checks if domain is actually blocked
func (p *DPIProber) probeBaseline(ctx context.Context) *ProbeResult {
	result := &ProbeResult{
		ProbeName: "baseline",
	}

	// First check reference domain (should work)
	refResult := p.doHTTPSProbe(ctx, p.referenceDomain)
	if !refResult.Success {
		result.Notes = "Reference domain also failed - possible network issue"
		result.Blocked = false
		return result
	}

	// Now check target domain
	targetResult := p.doHTTPSProbe(ctx, p.domain)
	result.Success = targetResult.Success
	result.Blocked = !targetResult.Success
	result.Latency = targetResult.Latency
	result.ErrorType = targetResult.ErrorType
	result.HTTPCode = targetResult.HTTPCode

	return result
}

// probeBlockingMethod determines how the DPI blocks connections
func (p *DPIProber) probeBlockingMethod(ctx context.Context, fp *DPIFingerprint) {
	// Test 1: Check for RST injection (most common)
	rstResult := p.probeForRST(ctx)
	p.storeResult("rst_detection", rstResult)

	if rstResult.RSTTTL > 0 {
		fp.BlockingMethod = BlockingRSTInject
		fp.RSTLatencyMs = float64(rstResult.Latency.Milliseconds())
		return
	}

	// Test 2: Check for HTTP redirect
	redirectResult := p.probeForRedirect(ctx)
	p.storeResult("redirect_detection", redirectResult)

	if redirectResult.HTTPCode >= 300 && redirectResult.HTTPCode < 400 {
		fp.BlockingMethod = BlockingRedirect
		return
	}

	// Test 3: Check for content injection
	injectResult := p.probeForContentInjection(ctx)
	p.storeResult("content_injection", injectResult)

	if injectResult.Notes == "content_injected" {
		fp.BlockingMethod = BlockingContentInject
		return
	}

	// Test 4: Pure timeout = silent drop
	if rstResult.ErrorType == "timeout" {
		fp.BlockingMethod = BlockingTimeout
		return
	}

	// Test 5: Check for TLS alert
	tlsResult := p.probeForTLSAlert(ctx)
	p.storeResult("tls_alert", tlsResult)

	if tlsResult.Notes == "tls_alert_received" {
		fp.BlockingMethod = BlockingTLSAlert
		return
	}
}

// probeForRST attempts to detect RST injection and measure its TTL
func (p *DPIProber) probeForRST(ctx context.Context) *ProbeResult {
	result := &ProbeResult{
		ProbeName: "rst_detection",
	}

	// Use raw TCP connection to detect RST
	dialer := &net.Dialer{
		Timeout: p.timeout,
	}

	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:443", p.domain))
	if err != nil {
		result.Latency = time.Since(start)

		// Check error type
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.ErrorType = "timeout"
		} else if strings.Contains(err.Error(), "connection reset") {
			result.ErrorType = "rst"
			result.RSTTTL = p.estimateTTLFromTiming(result.Latency)
		} else if strings.Contains(err.Error(), "connection refused") {
			result.ErrorType = "refused"
		} else {
			result.ErrorType = "other"
		}
		return result
	}
	defer conn.Close()

	// Connected, now send ClientHello with blocked SNI
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         p.domain,
		InsecureSkipVerify: true,
	})

	err = tlsConn.HandshakeContext(ctx)
	result.Latency = time.Since(start)

	if err != nil {
		if strings.Contains(err.Error(), "reset") {
			result.ErrorType = "rst_after_hello"
			result.RSTTTL = p.estimateTTLFromTiming(result.Latency)
		} else {
			result.ErrorType = "tls_error"
		}
	} else {
		result.Success = true
	}

	return result
}

// probeForRedirect checks if blocking uses HTTP redirect
func (p *DPIProber) probeForRedirect(ctx context.Context) *ProbeResult {
	result := &ProbeResult{
		ProbeName: "redirect_detection",
	}

	// Try HTTP (not HTTPS) first - redirects are more common there
	client := &http.Client{
		Timeout: p.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/", p.domain), nil)

	start := time.Now()
	resp, err := client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.ErrorType = "request_failed"
		return result
	}
	defer resp.Body.Close()

	result.HTTPCode = resp.StatusCode

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location != "" && !strings.Contains(location, p.domain) {
			result.Notes = fmt.Sprintf("redirect_to: %s", location)
			result.Blocked = true
		}
	}

	return result
}

// probeForContentInjection checks if DPI injects fake responses
func (p *DPIProber) probeForContentInjection(ctx context.Context) *ProbeResult {
	result := &ProbeResult{
		ProbeName: "content_injection",
	}

	client := &http.Client{
		Timeout: p.timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/", p.domain), nil)

	start := time.Now()
	resp, err := client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		return result
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
	result.ResponseSize = int64(len(body))

	// Check for common block page indicators
	bodyStr := strings.ToLower(string(body))
	blockIndicators := []string{
		"blocked", "запрещен", "access denied", "filtered",
		"blocked by", "this site", "не доступ", "заблокирован",
	}

	for _, indicator := range blockIndicators {
		if strings.Contains(bodyStr, indicator) {
			result.Notes = "content_injected"
			result.Blocked = true
			return result
		}
	}

	// Check if response is suspiciously fast and small (injected)
	if result.Latency < 50*time.Millisecond && result.ResponseSize < 1000 {
		result.Notes = "possibly_injected"
	}

	return result
}

// probeForTLSAlert checks if DPI sends TLS alerts
func (p *DPIProber) probeForTLSAlert(ctx context.Context) *ProbeResult {
	result := &ProbeResult{
		ProbeName: "tls_alert",
	}

	dialer := &net.Dialer{Timeout: p.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:443", p.domain))
	if err != nil {
		result.ErrorType = "connect_failed"
		return result
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         p.domain,
		InsecureSkipVerify: true,
	})

	start := time.Now()
	err = tlsConn.HandshakeContext(ctx)
	result.Latency = time.Since(start)

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "alert") {
			result.Notes = "tls_alert_received"
			result.Blocked = true
		}
	}

	return result
}

// probeRSTCharacteristics analyzes RST packets in detail
func (p *DPIProber) probeRSTCharacteristics(ctx context.Context, fp *DPIFingerprint) {
	// Multiple probes to get consistent TTL reading
	ttlReadings := make([]int, 0, 5)
	latencies := make([]time.Duration, 0, 5)

	for i := 0; i < 5; i++ {
		result := p.probeForRST(ctx)
		if result.RSTTTL > 0 {
			ttlReadings = append(ttlReadings, result.RSTTTL)
			latencies = append(latencies, result.Latency)
		}
		time.Sleep(100 * time.Millisecond)
	}

	if len(ttlReadings) > 0 {
		// Estimate hop count from TTL
		// Common initial TTLs: 64 (Linux), 128 (Windows), 255 (Cisco)
		avgTTL := average(ttlReadings)

		if avgTTL > 200 {
			fp.DPIHopCount = 255 - avgTTL
		} else if avgTTL > 64 {
			fp.DPIHopCount = 128 - avgTTL
		} else {
			fp.DPIHopCount = 64 - avgTTL
		}
		if fp.DPIHopCount < 1 {
			fp.DPIHopCount = 1
		}
		// Very low hop count = inline DPI
		fp.IsInline = fp.DPIHopCount <= 3

		// Calculate average latency
		var totalLatency time.Duration
		for _, l := range latencies {
			totalLatency += l
		}
		fp.BlockLatencyMs = float64(totalLatency.Milliseconds()) / float64(len(latencies))

		p.storeResult("rst_analysis", &ProbeResult{
			ProbeName: "rst_analysis",
			RSTTTL:    avgTTL,
			Latency:   time.Duration(fp.BlockLatencyMs) * time.Millisecond,
			Notes:     fmt.Sprintf("hop_count=%d, inline=%v", fp.DPIHopCount, fp.IsInline),
		})
	}
}

// probeInspectionDepth determines what the DPI actually inspects
func (p *DPIProber) probeInspectionDepth(ctx context.Context, fp *DPIFingerprint) {
	// Test 1: Does it inspect only SNI or full TLS?
	// Try connecting with no SNI
	noSNIResult := p.probeWithoutSNI(ctx)
	p.storeResult("no_sni", noSNIResult)

	if noSNIResult.Success {
		fp.InspectionDepth = InspectionSNIOnly
		fp.InspectsTLS = true
		log.Infof("DPI Fingerprinting: DPI only inspects SNI (no-SNI works)")
	}

	// Test 2: Does it track connection state?
	stateResult := p.probeStateTracking()
	p.storeResult("state_tracking", stateResult)

	fp.TracksState = stateResult.Notes == "stateful"
	if fp.TracksState {
		fp.InspectionDepth = InspectionStateful
	} else {
		fp.InspectionDepth = InspectionStateless
	}

	// Test 3: HTTP inspection
	httpResult := p.probeHTTPBlocking(ctx)
	p.storeResult("http_blocking", httpResult)
	fp.InspectsHTTP = httpResult.Blocked

	// Test 4: QUIC inspection (UDP 443)
	quicResult := p.probeQUICBlocking(ctx)
	p.storeResult("quic_blocking", quicResult)
	fp.InspectsQUIC = quicResult.Blocked
}

// probeWithoutSNI tries TLS connection without SNI extension
func (p *DPIProber) probeWithoutSNI(ctx context.Context) *ProbeResult {
	result := &ProbeResult{
		ProbeName: "no_sni",
	}

	// Resolve IP first
	ips, err := net.LookupIP(p.domain)
	if err != nil || len(ips) == 0 {
		result.ErrorType = "dns_failed"
		return result
	}

	dialer := &net.Dialer{Timeout: p.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:443", ips[0].String()))
	if err != nil {
		result.ErrorType = "connect_failed"
		return result
	}
	defer conn.Close()

	// TLS without ServerName
	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,
		// No ServerName set
	})

	start := time.Now()
	err = tlsConn.HandshakeContext(ctx)
	result.Latency = time.Since(start)

	if err == nil {
		result.Success = true
		result.Notes = "no_sni_works"
	} else {
		result.ErrorType = "tls_failed"
	}

	return result
}

func (p *DPIProber) probeStateTracking() *ProbeResult {
	result := &ProbeResult{
		ProbeName: "state_tracking",
	}

	p.mu.Lock()
	rstResult := p.results["rst_detection"]
	p.mu.Unlock()

	if rstResult != nil {
		if rstResult.Latency < 20*time.Millisecond && rstResult.ErrorType == "rst_after_hello" {
			result.Notes = "stateful"
		} else if rstResult.ErrorType == "timeout" {
			result.Notes = "stateless"
		} else {
			result.Notes = "stateful"
		}
	} else {
		result.Notes = "unknown"
	}

	return result
}

// probeHTTPBlocking checks if plain HTTP is blocked
func (p *DPIProber) probeHTTPBlocking(ctx context.Context) *ProbeResult {
	result := &ProbeResult{
		ProbeName: "http_blocking",
	}

	client := &http.Client{
		Timeout: p.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/", p.domain), nil)

	start := time.Now()
	resp, err := client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.Blocked = true
		result.ErrorType = "request_failed"
		return result
	}
	defer resp.Body.Close()

	result.HTTPCode = resp.StatusCode

	// Check if response is a block page
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		result.Blocked = true
	}

	return result
}

// probeQUICBlocking checks if QUIC/UDP 443 is blocked
func (p *DPIProber) probeQUICBlocking(ctx context.Context) *ProbeResult {
	result := &ProbeResult{
		ProbeName: "quic_blocking",
	}

	// Simple UDP probe to port 443
	// A full QUIC handshake would be more accurate but complex

	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:443", p.domain), p.timeout)
	if err != nil {
		result.Blocked = true
		result.ErrorType = "connect_failed"
		return result
	}
	defer conn.Close()

	// Send a minimal QUIC-like packet (not a real QUIC Initial)
	// Real implementation would use proper QUIC Initial packet
	fakeQUIC := make([]byte, 100)
	fakeQUIC[0] = 0xC0 // Long header, QUIC initial

	conn.SetWriteDeadline(time.Now().Add(p.timeout))
	_, err = conn.Write(fakeQUIC)

	if err != nil {
		result.Blocked = true
		result.ErrorType = "write_failed"
		return result
	}

	// Try to read response
	conn.SetReadDeadline(time.Now().Add(p.timeout / 2))
	buf := make([]byte, 1500)
	_, err = conn.Read(buf)

	if err != nil {
		// Timeout is expected if no QUIC server, but DPI might also drop
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.Notes = "timeout_no_response"
		}
	} else {
		result.Success = true
	}

	return result
}

// probeEvasionVulnerabilities tests specific bypass techniques
func (p *DPIProber) probeEvasionVulnerabilities(ctx context.Context, fp *DPIFingerprint) {
	log.Infof("DPI Fingerprinting: Testing evasion vulnerabilities...")

	// These probes require the B4 engine to be active
	// For now, we estimate based on DPI characteristics

	// TTL-based evasion works if DPI is not inline
	fp.VulnerableToTTL = fp.DPIHopCount > 2 && fp.DPIHopCount < 20

	// Fragmentation works against stateless DPI
	fp.VulnerableToFrag = !fp.TracksState || fp.InspectionDepth == InspectionSNIOnly

	// Desync works against stateful DPI that trusts sequence numbers
	fp.VulnerableToDesync = fp.TracksState && fp.BlockingMethod == BlockingRSTInject

	// OOB works against DPI that doesn't handle urgent data
	fp.VulnerableToOOB = fp.BlockingMethod == BlockingTimeout && !fp.TracksState

	// Calculate optimal TTL if vulnerable
	if fp.VulnerableToTTL && fp.DPIHopCount > 0 {
		fp.OptimalTTL = uint8(fp.DPIHopCount - 1)
		if fp.OptimalTTL < 1 {
			fp.OptimalTTL = 1
		}
	}

	p.storeResult("vuln_analysis", &ProbeResult{
		ProbeName: "vuln_analysis",
		Notes: fmt.Sprintf("ttl=%v frag=%v desync=%v oob=%v optimal_ttl=%d",
			fp.VulnerableToTTL, fp.VulnerableToFrag,
			fp.VulnerableToDesync, fp.VulnerableToOOB, fp.OptimalTTL),
	})
}

// identifyDPIType attempts to identify the specific DPI system
func (p *DPIProber) identifyDPIType(fp *DPIFingerprint) {
	// Scoring system for each DPI type
	scores := map[DPIType]int{
		DPITypeTSPU:      0,
		DPITypeSandvine:  0,
		DPITypeHuawei:    0,
		DPITypeAllot:     0,
		DPITypeFortigate: 0,
	}

	// TSPU characteristics (Russian DPI)
	// - Very fast RST injection (<10ms)
	// - Inline deployment
	// - Blocks SNI specifically
	// - Often hop count 1-3
	if fp.RSTLatencyMs < 15 && fp.IsInline {
		scores[DPITypeTSPU] += 30
	}
	if fp.DPIHopCount <= 3 && fp.DPIHopCount > 0 {
		scores[DPITypeTSPU] += 20
	}
	if fp.InspectionDepth == InspectionSNIOnly {
		scores[DPITypeTSPU] += 15
	}
	if fp.BlockingMethod == BlockingRSTInject {
		scores[DPITypeTSPU] += 10
	}

	// Sandvine characteristics
	// - Moderate latency (10-50ms)
	// - Often uses content injection
	// - Full TLS inspection capable
	if fp.RSTLatencyMs >= 10 && fp.RSTLatencyMs < 50 {
		scores[DPITypeSandvine] += 20
	}
	if fp.BlockingMethod == BlockingContentInject {
		scores[DPITypeSandvine] += 30
	}
	if fp.InspectionDepth == InspectionStateful {
		scores[DPITypeSandvine] += 15
	}

	// Huawei characteristics
	// - HTTP redirect common
	// - Moderate hop count (ISP level)
	if fp.BlockingMethod == BlockingRedirect {
		scores[DPITypeHuawei] += 25
	}
	if fp.DPIHopCount >= 3 && fp.DPIHopCount <= 8 {
		scores[DPITypeHuawei] += 15
	}

	// Fortigate characteristics
	// - TLS alerts
	// - Enterprise deployment (low hop count)
	if fp.BlockingMethod == BlockingTLSAlert {
		scores[DPITypeFortigate] += 35
	}
	if fp.DPIHopCount <= 2 {
		scores[DPITypeFortigate] += 15
	}

	// Find highest score
	maxScore := 0
	bestType := DPITypeUnknown
	for dpiType, score := range scores {
		if score > maxScore {
			maxScore = score
			bestType = dpiType
		}
	}

	// Only assign type if confidence is reasonable
	if maxScore >= 40 {
		fp.Type = bestType
		fp.Confidence = min(maxScore, 95)
	} else {
		fp.Type = DPITypeUnknown
		fp.Confidence = maxScore
	}
}

// generateRecommendations suggests strategies based on fingerprint
func (p *DPIProber) generateRecommendations(fp *DPIFingerprint) {
	recommendations := make([]StrategyFamily, 0)

	// Combo is generally most effective - test first
	recommendations = append(recommendations, FamilyTCPFrag)
	recommendations = append(recommendations, FamilyCombo)
	recommendations = append(recommendations, FamilyHybrid)

	// Priority based on vulnerabilities
	if fp.VulnerableToDesync {
		recommendations = append(recommendations, FamilyDesync)
		recommendations = append(recommendations, FamilyFirstByte)
	}

	if fp.VulnerableToFrag {
		recommendations = append(recommendations, FamilyDisorder)
		recommendations = append(recommendations, FamilyOverlap)
		recommendations = append(recommendations, FamilyTCPFrag)
		if fp.InspectionDepth == InspectionSNIOnly {
			recommendations = append(recommendations, FamilyExtSplit)
			recommendations = append(recommendations, FamilyTLSRec)
		}
	}

	if fp.VulnerableToTTL {
		recommendations = append(recommendations, FamilyFakeSNI)
	}

	if fp.VulnerableToOOB {
		recommendations = append(recommendations, FamilyOOB)
	}

	// Type-specific recommendations
	switch fp.Type {
	case DPITypeTSPU:
		// TSPU responds well to combo and disorder
		if !containsFamily(recommendations, FamilyDisorder) {
			recommendations = append(recommendations, FamilyDisorder)
		}
		recommendations = append(recommendations, FamilySACK)

	case DPITypeSandvine:
		// Sandvine needs timing-based attacks
		recommendations = append(recommendations, FamilyFirstByte)
		recommendations = append(recommendations, FamilySynFake)
		recommendations = append(recommendations, FamilyDelay)

	case DPITypeHuawei, DPITypeFortigate:
		// Enterprise DPI - overlap and extsplit often work
		recommendations = append(recommendations, FamilyOverlap)
		recommendations = append(recommendations, FamilyExtSplit)
		recommendations = append(recommendations, FamilyIPFrag)
	}
	// Ensure we have at least some recommendations
	if len(recommendations) == 0 {
		recommendations = []StrategyFamily{
			FamilyTCPFrag,
			FamilyFakeSNI,
			FamilyOOB,
			FamilyDesync,
		}
	}

	// Remove duplicates while preserving order
	seen := make(map[StrategyFamily]bool)
	unique := make([]StrategyFamily, 0, len(recommendations))
	for _, f := range recommendations {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}

	fp.RecommendedFamilies = unique

	// Determine optimal strategy name
	if len(unique) > 0 {
		fp.OptimalStrategy = string(unique[0])
	}
}

// Helper methods

func (p *DPIProber) doHTTPSProbe(ctx context.Context, domain string) *ProbeResult {
	result := &ProbeResult{
		ProbeName: fmt.Sprintf("https_%s", domain),
	}

	client := &http.Client{
		Timeout: p.timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout: p.timeout / 2,
			}).DialContext,
		},
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/", domain), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	start := time.Now()
	resp, err := client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.ErrorType = categorizeError(err)
		return result
	}
	defer resp.Body.Close()

	result.Success = true
	result.HTTPCode = resp.StatusCode

	// Read some body to ensure connection is complete
	io.CopyN(io.Discard, resp.Body, 1024)

	return result
}

func (p *DPIProber) estimateTTLFromTiming(latency time.Duration) int {
	// Rough heuristic: faster response = closer DPI
	// This is very approximate
	ms := latency.Milliseconds()

	if ms < 5 {
		return 62 // Very close, likely TTL ~64-2
	} else if ms < 20 {
		return 58 // Close, TTL ~64-6
	} else if ms < 50 {
		return 50 // Medium distance
	} else {
		return 40 // Far away
	}
}

func (p *DPIProber) storeResult(name string, result *ProbeResult) {
	p.mu.Lock()
	p.results[name] = result
	p.mu.Unlock()
}

func (p *DPIProber) logFingerprint(fp *DPIFingerprint) {
	log.Infof("=== DPI Fingerprint Results ===")
	log.Infof("  Type: %s (confidence: %d%%)", fp.Type, fp.Confidence)
	log.Infof("  Blocking Method: %s", fp.BlockingMethod)
	log.Infof("  Inspection Depth: %s", fp.InspectionDepth)
	log.Infof("  DPI Hop Count: %d (inline: %v)", fp.DPIHopCount, fp.IsInline)
	log.Infof("  RST Latency: %.2fms", fp.RSTLatencyMs)
	log.Infof("  Vulnerabilities: TTL=%v Frag=%v Desync=%v OOB=%v",
		fp.VulnerableToTTL, fp.VulnerableToFrag, fp.VulnerableToDesync, fp.VulnerableToOOB)
	if fp.OptimalTTL > 0 {
		log.Infof("  Optimal TTL: %d", fp.OptimalTTL)
	}
	log.Infof("  Recommended Families: %v", fp.RecommendedFamilies)
	log.Infof("===============================")
}

// Utility functions

func categorizeError(err error) string {
	errStr := err.Error()

	if strings.Contains(errStr, "timeout") {
		return "timeout"
	}
	if strings.Contains(errStr, "reset") {
		return "rst"
	}
	if strings.Contains(errStr, "refused") {
		return "refused"
	}
	if strings.Contains(errStr, "no route") {
		return "no_route"
	}
	if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls") {
		return "tls_error"
	}

	return "other"
}

func average(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum / len(values)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FilterPresetsByFingerprint returns only presets that match the fingerprint recommendations
func FilterPresetsByFingerprint(presets []ConfigPreset, fp *DPIFingerprint) []ConfigPreset {
	if fp == nil || fp.Type == DPITypeNone || len(fp.RecommendedFamilies) == 0 {
		return presets
	}

	// Create map of recommended families for fast lookup
	recommended := make(map[StrategyFamily]bool)
	for _, f := range fp.RecommendedFamilies {
		recommended[f] = true
	}

	// Always include baseline/proven presets
	filtered := make([]ConfigPreset, 0, len(presets))
	for _, preset := range presets {
		// Keep baseline/none family presets
		if preset.Family == FamilyNone {
			filtered = append(filtered, preset)
			continue
		}

		// Keep if family is recommended
		if recommended[preset.Family] {
			filtered = append(filtered, preset)
		}
	}

	log.Infof("Fingerprint filtering: %d -> %d presets", len(presets), len(filtered))
	return filtered
}

// ApplyFingerprintToPreset modifies a preset based on fingerprint data
func ApplyFingerprintToPreset(preset *ConfigPreset, fp *DPIFingerprint) {
	if fp == nil {
		return
	}

	// Apply optimal TTL if discovered
	if fp.OptimalTTL > 0 && preset.Config.Faking.SNI {
		preset.Config.Faking.TTL = fp.OptimalTTL
	}

	// If DPI is stateful, enable desync
	if fp.TracksState && preset.Config.TCP.DesyncMode == "off" {
		preset.Config.TCP.DesyncMode = "rst"
		preset.Config.TCP.DesyncTTL = fp.OptimalTTL
		preset.Config.TCP.DesyncCount = 2
	}
}

// GenerateOptimizedPresets creates presets specifically tuned for the fingerprint
func GenerateOptimizedPresets(fp *DPIFingerprint, baseConfig config.SetConfig) []ConfigPreset {
	if fp == nil || fp.Type == DPITypeNone {
		return nil
	}

	presets := make([]ConfigPreset, 0)

	// Generate preset optimized for this specific DPI
	optimized := ConfigPreset{
		Name:        fmt.Sprintf("fingerprint-optimized-%s", fp.Type),
		Description: fmt.Sprintf("Auto-generated for %s DPI", fp.Type),
		Family:      FamilyNone,
		Phase:       PhaseStrategy,
		Priority:    0, // Highest priority
		Config:      baseConfig,
	}

	// Apply fingerprint-specific settings
	if fp.OptimalTTL > 0 {
		optimized.Config.Faking.TTL = fp.OptimalTTL
	}

	// Set strategy based on vulnerabilities
	if fp.VulnerableToDesync {
		optimized.Config.TCP.DesyncMode = "combo"
		optimized.Config.TCP.DesyncTTL = fp.OptimalTTL
		optimized.Config.TCP.DesyncCount = 3
	}

	if fp.VulnerableToFrag {
		optimized.Config.Fragmentation.Strategy = "tcp"
		optimized.Config.Fragmentation.ReverseOrder = true
		optimized.Config.Fragmentation.MiddleSNI = true
	}

	if fp.VulnerableToOOB {
		// Create separate OOB variant
		oobPreset := optimized
		oobPreset.Name = fmt.Sprintf("fingerprint-oob-%s", fp.Type)
		oobPreset.Config.Fragmentation.Strategy = "oob"
		oobPreset.Config.Fragmentation.OOBPosition = 1
		oobPreset.Config.Fragmentation.OOBChar = 'x'
		presets = append(presets, oobPreset)
	}

	presets = append([]ConfigPreset{optimized}, presets...)

	return presets
}

// FingerprintToJSON returns fingerprint as JSON for API response
func (fp *DPIFingerprint) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"type":                 string(fp.Type),
		"blocking_method":      string(fp.BlockingMethod),
		"inspection_depth":     string(fp.InspectionDepth),
		"rst_latency_ms":       fp.RSTLatencyMs,
		"dpi_hop_count":        fp.DPIHopCount,
		"is_inline":            fp.IsInline,
		"confidence":           fp.Confidence,
		"optimal_ttl":          fp.OptimalTTL,
		"vulnerable_to_ttl":    fp.VulnerableToTTL,
		"vulnerable_to_frag":   fp.VulnerableToFrag,
		"vulnerable_to_desync": fp.VulnerableToDesync,
		"vulnerable_to_oob":    fp.VulnerableToOOB,
		"recommended_families": fp.RecommendedFamilies,
	}
}
