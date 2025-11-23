package sni

import (
	"container/list"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/daniellavrushin/b4/config"
)

type ipRange struct {
	ipNet *net.IPNet
	set   *config.SetConfig
}

type portRange struct {
	min int
	max int
	set *config.SetConfig
}

type SuffixSet struct {
	sets       map[string]*config.SetConfig
	regexes    []*regexWithSet
	regexCache sync.Map
	ipRanges   []ipRange
	portRanges []portRange

	ipCache      map[string]*cacheEntry
	ipCacheLRU   *list.List
	ipCacheMu    sync.RWMutex
	ipCacheLimit int

	domainCache      map[string]*cacheEntry
	domainCacheLRU   *list.List
	domainCacheMu    sync.RWMutex
	domainCacheLimit int

	regexCacheSize int32
}

type cacheEntry struct {
	matched bool
	set     *config.SetConfig
	element *list.Element
}

type regexWithSet struct {
	regex *regexp.Regexp
	set   *config.SetConfig
}

func NewSuffixSet(sets []*config.SetConfig) *SuffixSet {
	s := &SuffixSet{
		sets:         make(map[string]*config.SetConfig),
		regexes:      make([]*regexWithSet, 0),
		ipCache:      make(map[string]*cacheEntry),
		ipCacheLRU:   list.New(),
		ipCacheLimit: 10000,

		domainCache:      make(map[string]*cacheEntry),
		domainCacheLRU:   list.New(),
		domainCacheLimit: 50000,
	}

	seenRegexes := make(map[string]bool)

	for _, set := range sets {
		if !set.Enabled {
			continue
		}

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
			if _, exists := s.sets[d]; !exists {
				s.sets[d] = set
			}
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

		if set.UDP.DPortFilter != "" {
			ports := strings.Split(set.UDP.DPortFilter, ",")
			for _, part := range ports {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}

				if strings.Contains(part, "-") {
					bounds := strings.SplitN(part, "-", 2)
					if len(bounds) == 2 {
						min, err1 := strconv.Atoi(bounds[0])
						max, err2 := strconv.Atoi(bounds[1])
						if err1 == nil && err2 == nil {
							if min >= 0 && max >= 0 && min <= max {
								s.portRanges = append(s.portRanges, portRange{min: min, max: max, set: set})
							}
						}
					}
				} else {
					port, err := strconv.Atoi(part)
					if err == nil && port >= 0 {
						s.portRanges = append(s.portRanges, portRange{min: port, max: port, set: set})
					}
				}
			}
		}
	}

	if len(s.ipRanges) > 0 {
		sort.Slice(s.ipRanges, func(i, j int) bool {
			onesI, _ := s.ipRanges[i].ipNet.Mask.Size()
			onesJ, _ := s.ipRanges[j].ipNet.Mask.Size()
			return onesI > onesJ // /32 first, /0 last
		})
	}

	return s
}

func (s *SuffixSet) MatchUDPPort(dport uint16) (bool, *config.SetConfig) {
	if s == nil || len(s.portRanges) == 0 {
		return false, nil
	}

	port := int(dport)

	for _, r := range s.portRanges {

		matched := port >= r.min && port <= r.max
		if matched {
			return true, r.set
		}
	}

	return false, nil
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

	ipStr := ip.String()

	// Quick read-only check
	s.ipCacheMu.RLock()
	_, found := s.ipCache[ipStr]
	s.ipCacheMu.RUnlock()

	if found {
		// Need write lock to move element
		s.ipCacheMu.Lock()
		// Recheck (entry might be evicted between unlock/lock)
		if entry, ok := s.ipCache[ipStr]; ok {
			s.ipCacheLRU.MoveToFront(entry.element)
			matched, set := entry.matched, entry.set
			s.ipCacheMu.Unlock()
			return matched, set
		}
		s.ipCacheMu.Unlock()
		// Fall through to cache miss
	}

	// Cache miss - do linear scan
	var matched bool
	var matchedSet *config.SetConfig
	for _, r := range s.ipRanges {
		if r.ipNet.Contains(ip) {
			matched = true
			matchedSet = r.set
			break
		}
	}

	// Update cache
	s.ipCacheMu.Lock()
	defer s.ipCacheMu.Unlock()

	if len(s.ipCache) >= s.ipCacheLimit {
		oldest := s.ipCacheLRU.Back()
		if oldest != nil {
			delete(s.ipCache, oldest.Value.(string))
			s.ipCacheLRU.Remove(oldest)
		}
	}

	element := s.ipCacheLRU.PushFront(ipStr)
	s.ipCache[ipStr] = &cacheEntry{
		matched: matched,
		set:     matchedSet,
		element: element,
	}

	return matched, matchedSet
}

func (s *SuffixSet) matchDomain(host string) (bool, *config.SetConfig) {
	// Quick read-only check
	s.domainCacheMu.RLock()
	_, found := s.domainCache[host]
	s.domainCacheMu.RUnlock()

	if found {
		s.domainCacheMu.Lock()
		if entry, ok := s.domainCache[host]; ok {
			s.domainCacheLRU.MoveToFront(entry.element)
			matched, set := entry.matched, entry.set
			s.domainCacheMu.Unlock()
			return matched, set
		}
		s.domainCacheMu.Unlock()
	}

	var matched bool
	var matchedSet *config.SetConfig

	if set, ok := s.sets[host]; ok {
		matched = true
		matchedSet = set
	} else {
		remaining := host
		for {
			idx := strings.IndexByte(remaining, '.')
			if idx == -1 {
				break
			}
			remaining = remaining[idx+1:]
			if set, ok := s.sets[remaining]; ok {
				matched = true
				matchedSet = set
				break
			}
		}
	}

	// Update cache
	s.domainCacheMu.Lock()
	defer s.domainCacheMu.Unlock()

	if len(s.domainCache) >= s.domainCacheLimit {
		oldest := s.domainCacheLRU.Back()
		if oldest != nil {
			delete(s.domainCache, oldest.Value.(string))
			s.domainCacheLRU.Remove(oldest)
		}
	}

	element := s.domainCacheLRU.PushFront(host)
	s.domainCache[host] = &cacheEntry{
		matched: matched,
		set:     matchedSet,
		element: element,
	}

	return matched, matchedSet
}

func (s *SuffixSet) matchRegex(host string) (bool, *config.SetConfig) {
	if cached, ok := s.regexCache.Load(host); ok {
		entry := cached.(cacheEntry)
		return entry.matched, entry.set
	}

	var matched bool
	var matchedSet *config.SetConfig
	for _, rws := range s.regexes {
		if rws.regex.MatchString(host) {
			matched = true
			matchedSet = rws.set
			break
		}
	}

	if atomic.LoadInt32(&s.regexCacheSize) < 10000 {
		s.regexCache.Store(host, cacheEntry{matched: matched, set: matchedSet})
		atomic.AddInt32(&s.regexCacheSize, 1)
	}

	return matched, matchedSet
}
