package discovery

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
)

type dohResponse struct {
	Answer []struct {
		Data string `json:"data"`
		Type int    `json:"type"`
	} `json:"Answer"`
}

type DNSProber struct {
	domain  string
	timeout time.Duration
	pool    *nfq.Pool
	cfg     *config.Config
}

func (ds *DiscoverySuite) runDNSDiscovery() *DNSDiscoveryResult {
	log.Infof("Phase DNS: Checking DNS poisoning for %s", ds.Domain)

	prober := NewDNSProber(
		ds.Domain,
		time.Duration(ds.cfg.System.Checker.DiscoveryTimeoutSec)*time.Second,
		ds.pool,
		ds.cfg,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return prober.Probe(ctx)
}

func (ds *DiscoverySuite) applyDNSConfig(dnsResult *DNSDiscoveryResult) {
	if dnsResult == nil || !dnsResult.hasWorkingConfig() {
		return
	}

	ds.cfg.MainSet.DNS = config.DNSConfig{
		Enabled:       true,
		TargetDNS:     dnsResult.BestServer,
		FragmentQuery: dnsResult.NeedsFragment,
	}

	log.Infof("Applied DNS config: server=%s, fragment=%v",
		dnsResult.BestServer, dnsResult.NeedsFragment)
}

func (r *DNSDiscoveryResult) hasWorkingConfig() bool {
	if r == nil {
		return true
	}
	return !r.IsPoisoned || r.BestServer != "" || r.NeedsFragment
}

func NewDNSProber(domain string, timeout time.Duration, pool *nfq.Pool, cfg *config.Config) *DNSProber {
	return &DNSProber{
		domain:  domain,
		timeout: timeout,
		pool:    pool,
		cfg:     cfg,
	}
}

func (p *DNSProber) Probe(ctx context.Context) *DNSDiscoveryResult {
	result := &DNSDiscoveryResult{
		ProbeResults: []DNSProbeResult{},
	}

	expectedIPs := p.getExpectedIPs(ctx)
	if len(expectedIPs) == 0 {
		log.Warnf("DNS Discovery: couldn't get reference IP for %s", p.domain)
		return result
	}
	result.ExpectedIPs = expectedIPs
	expectedIP := expectedIPs[0]
	log.Tracef("DNS Discovery: found %d valid IPs for %s: %v", len(expectedIPs), p.domain, expectedIPs)

	sysResult := p.testDNS(ctx, "", false, expectedIP)
	result.ProbeResults = append(result.ProbeResults, sysResult)

	if !sysResult.Works {
		result.IsPoisoned = true
		log.Infof("DNS Discovery: %s appears poisoned (got %s, expected %s)",
			p.domain, sysResult.ResolvedIP, expectedIP)
	}

	if !result.IsPoisoned {
		return result
	}

	fragResult := p.testDNSWithFragment("", expectedIP)
	result.ProbeResults = append(result.ProbeResults, fragResult)

	if fragResult.Works {
		result.NeedsFragment = true
		log.Infof("DNS Discovery: fragmented query works for %s", p.domain)
		return result
	}

	for _, server := range p.cfg.System.Checker.ReferenceDNS {
		plainResult := p.testDNS(ctx, server, false, expectedIP)
		result.ProbeResults = append(result.ProbeResults, plainResult)

		if plainResult.Works {
			result.BestServer = server
			result.NeedsFragment = false
			log.Infof("DNS Discovery: %s works with DNS %s", p.domain, server)
			return result
		}

		// Fragmented to alternate
		fragAltResult := p.testDNSWithFragment(server, expectedIP)
		result.ProbeResults = append(result.ProbeResults, fragAltResult)

		if fragAltResult.Works {
			result.BestServer = server
			result.NeedsFragment = true
			log.Infof("DNS Discovery: %s works with fragmented DNS to %s", p.domain, server)
			return result
		}
	}

	log.Warnf("DNS Discovery: no working DNS config found for %s", p.domain)
	return result
}

func (p *DNSProber) getExpectedIPs(ctx context.Context) []string {
	recordType := "A"
	if p.cfg.Queue.IPv6Enabled && !p.cfg.Queue.IPv4Enabled {
		recordType = "AAAA"
	}

	dohServers := []string{
		"https://dns.google/resolve?name=%s&type=" + recordType,
		"https://dns.quad9.net:5053/dns-query?name=%s&type=" + recordType,
		"https://cloudflare-dns.com/dns-query?name=%s&type=" + recordType,
	}

	client := &http.Client{Timeout: p.timeout}

	seenIPs := make(map[string]bool)
	var allIPs []string

	for _, endpoint := range dohServers {
		url := fmt.Sprintf(endpoint, p.domain)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Accept", "application/dns-json")

		resp, err := client.Do(req)
		if err != nil {
			log.Tracef("DoH %s failed: %v", endpoint, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var doh dohResponse
		if err := json.Unmarshal(body, &doh); err != nil {
			continue
		}

		wantType := 1
		if recordType == "AAAA" {
			wantType = 28
		}

		for _, ans := range doh.Answer {
			if ans.Type == wantType {
				ip := ans.Data
				if seenIPs[ip] {
					continue
				}
				seenIPs[ip] = true

				if p.testIPServesDomain(ctx, ip) {
					log.Tracef("DoH: verified %s for %s", ip, p.domain)
					allIPs = append(allIPs, ip)
				}
			}
		}

		if len(allIPs) > 0 {
			break
		}
	}

	if len(allIPs) == 0 {
		ip := p.getExpectedIPFallback(ctx)
		if ip != "" {
			return []string{ip}
		}
		return nil
	}

	return allIPs
}

func (p *DNSProber) getExpectedIPFallback(ctx context.Context) string {
	network := "ip4"
	if p.cfg.Queue.IPv6Enabled && !p.cfg.Queue.IPv4Enabled {
		network = "ip6"
	}

	for _, server := range p.cfg.System.Checker.ReferenceDNS {
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: p.timeout / 3}
				return d.DialContext(ctx, "udp", server+":53")
			},
		}

		ips, err := resolver.LookupIP(ctx, network, p.domain)
		if err == nil && len(ips) > 0 {
			ip := ips[0].String()
			if p.testIPServesDomain(ctx, ip) {
				log.Tracef("DNS fallback: verified %s for %s from %s", ip, p.domain, server)
				return ip
			}
		}
	}
	return ""
}

func (p *DNSProber) testDNS(ctx context.Context, server string, fragmented bool, expectedIP string) DNSProbeResult {
	result := DNSProbeResult{
		Server:     server,
		Fragmented: fragmented,
		ExpectedIP: expectedIP,
	}

	resolver := net.DefaultResolver
	if server != "" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: p.timeout}
				return d.DialContext(ctx, network, server+":53")
			},
		}
	}

	network := "ip4"
	if p.cfg.Queue.IPv6Enabled && !p.cfg.Queue.IPv4Enabled {
		network = "ip6"
	}

	start := time.Now()
	ips, err := resolver.LookupIP(ctx, network, p.domain)
	result.Latency = time.Since(start)

	if err != nil || len(ips) == 0 {
		result.IsPoisoned = true
		return result
	}

	result.ResolvedIP = ips[0].String()

	result.Works = p.testIPServesDomain(ctx, result.ResolvedIP)
	result.IsPoisoned = !result.Works

	return result
}

func (p *DNSProber) testIPServesDomain(ctx context.Context, ip string) bool {
	dialer := &net.Dialer{Timeout: p.timeout / 2}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, "443"))
	if err != nil {
		return false
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         p.domain,
		InsecureSkipVerify: false,
	})

	err = tlsConn.HandshakeContext(ctx)
	if err != nil {
		return false
	}
	tlsConn.Close()
	return true
}

func (p *DNSProber) testDNSWithFragment(server string, expectedIP string) DNSProbeResult {
	result := DNSProbeResult{
		Server:     server,
		Fragmented: true,
		ExpectedIP: expectedIP,
	}

	// Apply DNS config to pool temporarily
	testCfg := p.buildDNSTestConfig(server, true)
	if err := p.pool.UpdateConfig(testCfg); err != nil {
		return result
	}
	defer p.pool.UpdateConfig(p.cfg) // Restore

	time.Sleep(time.Duration(p.cfg.System.Checker.ConfigPropagateMs) * time.Millisecond)

	// Now DNS queries should be fragmented via NFQ
	start := time.Now()
	ips, err := net.LookupIP(p.domain)
	result.Latency = time.Since(start)

	if err != nil || len(ips) == 0 {
		return result
	}

	result.ResolvedIP = ips[0].String()
	result.Works = p.testIPServesDomain(context.Background(), result.ResolvedIP)
	result.IsPoisoned = !result.Works

	return result
}

func (p *DNSProber) buildDNSTestConfig(targetDNS string, fragment bool) *config.Config {
	mainSet := config.NewSetConfig()
	mainSet.Id = p.cfg.MainSet.Id
	mainSet.Name = "dns-test"
	mainSet.Enabled = true
	mainSet.Targets.SNIDomains = []string{p.domain}
	mainSet.Targets.DomainsToMatch = []string{p.domain}

	mainSet.DNS = config.DNSConfig{
		Enabled:       true,
		TargetDNS:     targetDNS,
		FragmentQuery: fragment,
	}

	return &config.Config{
		ConfigPath: p.cfg.ConfigPath,
		Queue:      p.cfg.Queue,
		System:     p.cfg.System,
		MainSet:    &mainSet,
		Sets:       []*config.SetConfig{&mainSet},
	}
}
