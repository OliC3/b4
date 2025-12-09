package discovery

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
)

type DNSProber struct {
	domain  string
	timeout time.Duration
	pool    *nfq.Pool
	cfg     *config.Config
}

func (ds *DiscoverySuite) runDNSDiscovery() *DNSDiscoveryResult {
	log.Infof("Phase DNS: Checking DNS poisoning for %s", ds.domain)

	prober := NewDNSProber(
		ds.domain,
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

	// Update the stored config that buildTestConfig uses
	ds.cfg.MainSet.DNS = config.DNSConfig{
		Enabled:       true,
		TargetDNS:     dnsResult.BestServer,
		FragmentQuery: dnsResult.NeedsFragment,
	}

	log.Infof("Applied DNS config: server=%s, fragment=%v",
		dnsResult.BestServer, dnsResult.NeedsFragment)
}

// Helper on DNSDiscoveryResult
func (r *DNSDiscoveryResult) hasWorkingConfig() bool {
	if r == nil {
		return true // No result means no poisoning detected
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

// ProbesDNS returns DNS discovery result
func (p *DNSProber) Probe(ctx context.Context) *DNSDiscoveryResult {
	result := &DNSDiscoveryResult{
		ProbeResults: []DNSProbeResult{},
	}

	// Step 1: Get reference IP from known-good resolver (over DoH or trusted path)
	expectedIP := p.getExpectedIP(ctx)
	if expectedIP == "" {
		log.Warnf("DNS Discovery: couldn't get reference IP for %s", p.domain)
		return result
	}

	// Step 2: Test system DNS (detect poisoning)
	sysResult := p.testDNS(ctx, "", false, expectedIP)
	result.ProbeResults = append(result.ProbeResults, sysResult)

	if !sysResult.Works {
		result.IsPoisoned = true
		log.Infof("DNS Discovery: %s appears poisoned (got %s, expected %s)",
			p.domain, sysResult.ResolvedIP, expectedIP)
	}

	if !result.IsPoisoned {
		return result // No DNS bypass needed
	}

	// Step 3: Test fragmented DNS to system resolver
	fragResult := p.testDNSWithFragment("", expectedIP)
	result.ProbeResults = append(result.ProbeResults, fragResult)

	if fragResult.Works {
		result.NeedsFragment = true
		log.Infof("DNS Discovery: fragmented query works for %s", p.domain)
		return result
	}

	// Step 4: Test alternate DNS servers
	for _, server := range p.cfg.System.Checker.ReferenceDNS {
		// Plain
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

func (p *DNSProber) getExpectedIP(ctx context.Context) string {
	for _, server := range p.cfg.System.Checker.ReferenceDNS {
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: p.timeout / 3}
				return d.DialContext(ctx, "udp", server+":53")
			},
		}

		ips, err := resolver.LookupIP(ctx, "ip4", p.domain)
		if err == nil && len(ips) > 0 {
			log.Tracef("DNS reference: got %s for %s from %s", ips[0], p.domain, server)
			return ips[0].String()
		}
	}

	log.Warnf("DNS Discovery: all reference resolvers failed for %s", p.domain)
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

	start := time.Now()
	ips, err := resolver.LookupIP(ctx, "ip4", p.domain)
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
	conn, err := dialer.DialContext(ctx, "tcp", ip+":443")
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
	result.IsPoisoned = result.ResolvedIP != expectedIP
	result.Works = !result.IsPoisoned

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
