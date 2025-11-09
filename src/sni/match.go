package sni

import (
	"net"
	"regexp"
	"strings"
	"sync"

	"github.com/daniellavrushin/b4/config"
)

type ipRange struct {
	ipNet *net.IPNet
	set   *config.SetConfig
}

type SuffixSet struct {
	sets       map[string]*config.SetConfig
	regexes    []*regexWithSet
	regexCache sync.Map
	ipRanges   []ipRange
}

type regexWithSet struct {
	regex *regexp.Regexp
	set   *config.SetConfig
}

type cacheEntry struct {
	matched bool
	set     *config.SetConfig
}

func NewSuffixSet(sets []*config.SetConfig) *SuffixSet {
	s := &SuffixSet{
		sets:    make(map[string]*config.SetConfig),
		regexes: make([]*regexWithSet, 0),
	}

	seenRegexes := make(map[string]bool)

	for _, set := range sets {
		for _, d := range set.Targets.DomainsToMatch {
			d = strings.ToLower(strings.TrimSpace(d))
			if d == "" {
				continue
			}

			// Handle regex patterns
			if strings.HasPrefix(d, "regexp:") {
				pattern := strings.TrimPrefix(d, "regexp:")
				if seenRegexes[pattern] {
					continue
				}
				if re, err := regexp.Compile(pattern); err == nil {
					s.regexes = append(s.regexes, &regexWithSet{regex: re, set: set})
					seenRegexes[pattern] = true
				}
				continue
			}

			// Regular domain
			d = strings.TrimRight(d, ".")
			s.sets[d] = set
		}

		for _, ipStr := range set.Targets.IpsToMatch {
			ipStr = strings.TrimSpace(ipStr)
			if ipStr == "" {
				continue
			}

			// Parse CIDR or single IP
			var ipNet *net.IPNet
			var err error

			if strings.Contains(ipStr, "/") {
				_, ipNet, err = net.ParseCIDR(ipStr)
			} else {
				ip := net.ParseIP(ipStr)
				if ip != nil {
					if ip.To4() != nil {
						_, ipNet, _ = net.ParseCIDR(ipStr + "/32")
					} else {
						_, ipNet, _ = net.ParseCIDR(ipStr + "/128")
					}
				}
			}

			if err == nil && ipNet != nil {
				s.ipRanges = append(s.ipRanges, ipRange{ipNet: ipNet, set: set})
			}
		}
	}

	return s
}

func (s *SuffixSet) MatchSNI(host string) (bool, *config.SetConfig) {
	if s == nil || (len(s.sets) == 0 && len(s.regexes) == 0) || host == "" {
		return false, nil
	}

	lower := strings.ToLower(host)

	// Check exact/suffix match first (fast)
	if matched, set := s.matchDomain(lower); matched {
		return true, set
	}

	if len(s.regexes) > 0 {
		return s.matchRegex(lower)
	}

	return false, nil
}

func (s *SuffixSet) MatchIP(ip net.IP) (bool, *config.SetConfig) {
	if s == nil || len(s.ipRanges) == 0 || ip == nil {
		return false, nil
	}

	for _, r := range s.ipRanges {
		if r.ipNet.Contains(ip) {
			return true, r.set
		}
	}

	return false, nil
}

func (s *SuffixSet) matchDomain(host string) (bool, *config.SetConfig) {
	// Exact match
	if set, ok := s.sets[host]; ok {
		return true, set
	}

	// Check suffixes
	for {
		idx := strings.IndexByte(host, '.')
		if idx == -1 {
			break
		}
		host = host[idx+1:]
		if set, ok := s.sets[host]; ok {
			return true, set
		}
	}

	return false, nil
}

func (s *SuffixSet) matchRegex(host string) (bool, *config.SetConfig) {
	// Check cache
	if cached, ok := s.regexCache.Load(host); ok {
		entry := cached.(cacheEntry)
		return entry.matched, entry.set
	}

	// Test patterns
	for _, rws := range s.regexes {
		if rws.regex.MatchString(host) {
			s.regexCache.Store(host, cacheEntry{matched: true, set: rws.set})
			return true, rws.set
		}
	}

	s.regexCache.Store(host, cacheEntry{matched: false, set: nil})
	return false, nil
}
