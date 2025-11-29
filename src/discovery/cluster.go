package discovery

import (
	"net"
	"sort"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func ClusterDomains(domains []string) []*DomainCluster {
	if len(domains) == 0 {
		return nil
	}

	// Group by registrable domain (eTLD+1)
	groups := make(map[string][]string)
	ungrouped := []string{}

	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}

		// Get eTLD+1 (e.g., "youtube.com" from "www.youtube.com")
		etld, err := publicsuffix.EffectiveTLDPlusOne(domain)
		if err != nil {
			ungrouped = append(ungrouped, domain)
			continue
		}

		groups[etld] = append(groups[etld], domain)
	}

	clusters := make([]*DomainCluster, 0, len(groups)+len(ungrouped))

	// Create clusters for grouped domains
	for etld, domainList := range groups {
		// Sort to ensure consistent representative selection
		sort.Strings(domainList)

		// Choose representative: prefer the shortest domain, or the eTLD itself
		representative := domainList[0]
		for _, d := range domainList {
			// Prefer exact eTLD match
			if d == etld {
				representative = d
				break
			}
			// Otherwise prefer shorter domains
			if len(d) < len(representative) {
				representative = d
			}
		}

		clusters = append(clusters, &DomainCluster{
			ID:             etld,
			Domains:        domainList,
			Representative: representative,
		})
	}

	// Add ungrouped domains as individual clusters
	for _, d := range ungrouped {
		clusters = append(clusters, &DomainCluster{
			ID:             d,
			Domains:        []string{d},
			Representative: d,
		})
	}

	// Sort clusters by number of domains (larger clusters first - more value to test)
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].Domains) > len(clusters[j].Domains)
	})

	return clusters
}

// MergeClustersByIP groups clusters that resolve to the same IP ranges
// This is useful for CDNs where different domains share infrastructure
func MergeClustersByIP(clusters []*DomainCluster) []*DomainCluster {
	if len(clusters) <= 1 {
		return clusters
	}

	// Resolve representative domains to IPs
	ipToCluster := make(map[string][]*DomainCluster)

	for _, cluster := range clusters {
		ips, err := net.LookupIP(cluster.Representative)
		if err != nil || len(ips) == 0 {
			continue
		}

		// Use first IP's /24 as key (rough grouping)
		ip := ips[0].To4()
		if ip == nil {
			ip = ips[0].To16()
		}
		if ip == nil {
			continue
		}

		// Create /24 key for IPv4, /64 for IPv6
		var key string
		if len(ip) == 4 {
			key = ip[:3].String() + ".0/24"
		} else {
			key = ip[:8].String() + "/64"
		}

		ipToCluster[key] = append(ipToCluster[key], cluster)
	}

	// Merge clusters that share IP ranges
	merged := make([]*DomainCluster, 0)
	seen := make(map[string]bool)

	for _, clusterGroup := range ipToCluster {
		if len(clusterGroup) <= 1 {
			continue
		}

		// Merge all clusters in this IP group
		mergedCluster := &DomainCluster{
			ID:      clusterGroup[0].ID + "+merged",
			Domains: []string{},
		}

		for _, c := range clusterGroup {
			mergedCluster.Domains = append(mergedCluster.Domains, c.Domains...)
			seen[c.ID] = true
		}

		// Choose representative from merged cluster
		sort.Strings(mergedCluster.Domains)
		mergedCluster.Representative = mergedCluster.Domains[0]
		for _, d := range mergedCluster.Domains {
			if len(d) < len(mergedCluster.Representative) {
				mergedCluster.Representative = d
			}
		}

		merged = append(merged, mergedCluster)
	}

	// Add unmerged clusters
	for _, c := range clusters {
		if !seen[c.ID] {
			merged = append(merged, c)
		}
	}

	return merged
}

// GetKnownCDNGroups returns domain patterns that belong to known CDNs
// Domains matching these patterns can be grouped together
var knownCDNPatterns = map[string][]string{
	"google": {
		"google.com", "googleapis.com", "gstatic.com", "googleusercontent.com",
		"googlevideo.com", "youtube.com", "ytimg.com", "ggpht.com",
	},
	"cloudflare": {
		"cloudflare.com", "cloudflare-dns.com", "cloudflareinsights.com",
	},
	"amazon": {
		"amazonaws.com", "cloudfront.net", "amazon.com", "aws.amazon.com",
	},
	"microsoft": {
		"microsoft.com", "msn.com", "live.com", "office.com", "azure.com",
		"windows.net", "microsoftonline.com",
	},
	"meta": {
		"facebook.com", "fbcdn.net", "instagram.com", "whatsapp.com", "whatsapp.net",
	},
	"twitter": {
		"twitter.com", "x.com", "twimg.com", "t.co",
	},
}

// GetCDNGroup returns the CDN group name if domain belongs to a known CDN
func GetCDNGroup(domain string) string {
	domain = strings.ToLower(domain)

	for group, patterns := range knownCDNPatterns {
		for _, pattern := range patterns {
			if domain == pattern || strings.HasSuffix(domain, "."+pattern) {
				return group
			}
		}
	}
	return ""
}

// ClusterByKnownCDN groups domains by known CDN patterns before falling back to eTLD
func ClusterByKnownCDN(domains []string) []*DomainCluster {
	cdnGroups := make(map[string][]string)
	remaining := []string{}

	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}

		cdnGroup := GetCDNGroup(domain)
		if cdnGroup != "" {
			cdnGroups[cdnGroup] = append(cdnGroups[cdnGroup], domain)
		} else {
			remaining = append(remaining, domain)
		}
	}

	clusters := make([]*DomainCluster, 0)

	// Create clusters for CDN groups
	for cdnName, domainList := range cdnGroups {
		sort.Strings(domainList)

		// Choose shortest domain as representative
		representative := domainList[0]
		for _, d := range domainList {
			if len(d) < len(representative) {
				representative = d
			}
		}

		clusters = append(clusters, &DomainCluster{
			ID:             "cdn:" + cdnName,
			Domains:        domainList,
			Representative: representative,
		})
	}

	// Cluster remaining domains by eTLD
	if len(remaining) > 0 {
		remainingClusters := ClusterDomains(remaining)
		clusters = append(clusters, remainingClusters...)
	}

	// Sort by cluster size
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i].Domains) > len(clusters[j].Domains)
	})

	return clusters
}
