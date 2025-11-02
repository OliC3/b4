package sni

import (
	"regexp"
	"strings"
	"sync"
)

type SuffixSet struct {
	domains    map[string]struct{}
	regexes    []*regexp.Regexp
	regexCache sync.Map
}

func NewSuffixSet(domains []string) *SuffixSet {
	s := &SuffixSet{
		domains: make(map[string]struct{}),
		regexes: make([]*regexp.Regexp, 0),
	}

	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}

		// Handle regex patterns
		if strings.HasPrefix(d, "regexp:") {
			pattern := strings.TrimPrefix(d, "regexp:")
			if re, err := regexp.Compile(pattern); err == nil {
				s.regexes = append(s.regexes, re)
			}
			continue
		}

		// Regular domain
		d = strings.TrimRight(d, ".")
		s.domains[d] = struct{}{}
	}

	return s
}

func (s *SuffixSet) Match(host string) bool {
	if s == nil || len(s.domains) == 0 && len(s.regexes) == 0 || host == "" {
		return false
	}

	lower := strings.ToLower(host)

	// Check exact/suffix match first (fast)
	if s.matchDomain(lower) {
		return true
	}

	if len(s.regexes) > 0 {
		return s.matchRegex(lower)
	}

	return false
}

func (s *SuffixSet) matchDomain(host string) bool {
	// Exact match
	if _, ok := s.domains[host]; ok {
		return true
	}

	// Check suffixes
	for {
		idx := strings.IndexByte(host, '.')
		if idx == -1 {
			break
		}
		host = host[idx+1:]
		if _, ok := s.domains[host]; ok {
			return true
		}
	}

	return false
}

func (s *SuffixSet) matchRegex(host string) bool {
	// Check cache
	if cached, ok := s.regexCache.Load(host); ok {
		return cached.(bool)
	}

	// Test patterns
	for _, re := range s.regexes {
		if re.MatchString(host) {
			s.regexCache.Store(host, true)
			return true
		}
	}

	s.regexCache.Store(host, false)
	return false
}
