package geodat

import (
	"github.com/urlesistiana/v2dat/v2data"
)

// LoadDomainsFromSites loads domains from geodata file for specified sites
func LoadDomainsFromSites(geodataPath string, sites []string) ([]string, error) {
	if geodataPath == "" || len(sites) == 0 {
		return nil, nil
	}

	allDomains := []string{}

	// Create a callback that collects domains
	save := func(tag string, domainList []*v2data.Domain) error {
		for _, d := range domainList {
			// Extract domain value based on type
			domain := extractDomainValue(d)
			if domain != "" {
				allDomains = append(allDomains, domain)
			}
		}
		return nil
	}

	// Stream and process each site
	if err := streamGeoSite(geodataPath, sites, save); err != nil {
		return nil, err
	}

	return allDomains, nil
}

// extractDomainValue extracts the domain string from a Domain record
func extractDomainValue(d *v2data.Domain) string {
	switch d.Type {
	case v2data.Domain_Plain:
		// Plain/keyword match - just the value
		return d.Value
	case v2data.Domain_Regex:
		// regex patterns - they need special handling
		return "regexp:" + d.Value
	case v2data.Domain_Full:
		// Full domain match
		return d.Value
	case v2data.Domain_Domain:
		// Domain and subdomains
		return d.Value
	default:
		return d.Value
	}
}
