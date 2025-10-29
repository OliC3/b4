package sni

import (
	"strings"
)

type SuffixSet struct {
	m map[string]struct{}
}

func NewSuffixSet(domains []string) *SuffixSet {
	m := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if strings.HasSuffix(d, ".") {
			d = strings.TrimRight(d, ".")
		}
		m[d] = struct{}{}
	}
	return &SuffixSet{m: m}
}

func (s *SuffixSet) Match(host string) bool {
	if s == nil || len(s.m) == 0 || host == "" {
		return false
	}

	lower := strings.ToLower(host)
	if _, ok := s.m[lower]; ok {
		return true
	}

	for {
		idx := strings.IndexByte(lower, '.')
		if idx == -1 {
			break
		}
		lower = lower[idx+1:]
		if _, ok := s.m[lower]; ok {
			return true
		}
	}

	return false
}
