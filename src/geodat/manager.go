// path: src/geodat/manager.go
package geodat

import (
	"sync"

	"github.com/daniellavrushin/b4/log"
)

// GeodataManager handles geodata file operations with caching and statistics
type GeodataManager struct {
	mu              sync.RWMutex
	geositePath     string
	geoipPath       string
	categoryDomains map[string][]string // category -> domains (cached)
	categoryCounts  map[string]int      // category -> domain count (fast lookup)
}

// NewGeodataManager creates a new geodata manager instance
func NewGeodataManager(geositePath, geoipPath string) *GeodataManager {
	return &GeodataManager{
		geositePath:     geositePath,
		geoipPath:       geoipPath,
		categoryDomains: make(map[string][]string),
		categoryCounts:  make(map[string]int),
	}
}

// UpdatePaths updates the geodata file paths and clears cache if paths changed
func (gm *GeodataManager) UpdatePaths(geositePath, geoipPath string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	pathsChanged := gm.geositePath != geositePath || gm.geoipPath != geoipPath

	gm.geositePath = geositePath
	gm.geoipPath = geoipPath

	if pathsChanged {
		gm.categoryDomains = make(map[string][]string)
		gm.categoryCounts = make(map[string]int)
		log.Infof("Geodata paths updated, cache cleared")
	}
}

// LoadCategory loads domains for a single category (uses cache if available)
func (gm *GeodataManager) LoadCategory(category string) ([]string, error) {
	gm.mu.RLock()
	if domains, exists := gm.categoryDomains[category]; exists {
		gm.mu.RUnlock()
		log.Tracef("Using cached domains for category: %s (%d domains)", category, len(domains))
		return domains, nil
	}
	gm.mu.RUnlock()

	// Load from file
	if gm.geositePath == "" {
		return nil, log.Errorf("geosite path not configured")
	}

	domains, err := LoadDomainsFromSites(gm.geositePath, []string{category})
	if err != nil {
		return nil, err
	}

	// Cache the result
	gm.mu.Lock()
	gm.categoryDomains[category] = domains
	gm.categoryCounts[category] = len(domains)
	gm.mu.Unlock()

	log.Tracef("Loaded and cached %d domains for category: %s", len(domains), category)
	return domains, nil
}

// LoadCategories loads domains for multiple categories and returns combined domains + counts
// LoadCategories loads domains for multiple categories and returns combined domains + counts
func (gm *GeodataManager) LoadCategories(categories []string) ([]string, map[string]int, error) {
	if len(categories) == 0 {
		return []string{}, make(map[string]int), nil
	}

	if gm.geositePath == "" {
		return nil, nil, log.Errorf("geosite path not configured")
	}

	// Create a set of requested categories for easy lookup
	requestedCategories := make(map[string]bool)
	for _, cat := range categories {
		requestedCategories[cat] = true
	}

	// Remove categories from cache that are no longer requested
	gm.mu.Lock()
	for cachedCategory := range gm.categoryDomains {
		if !requestedCategories[cachedCategory] {
			delete(gm.categoryDomains, cachedCategory)
			delete(gm.categoryCounts, cachedCategory)
			log.Tracef("Removed category %s from cache (no longer selected)", cachedCategory)
		}
	}
	gm.mu.Unlock()

	uniqueDomains := make(map[string]bool)
	categoryStats := make(map[string]int)

	for _, category := range categories {
		domains, err := gm.LoadCategory(category)
		if err != nil {
			log.Errorf("Failed to load category %s: %v", category, err)
			continue
		}

		for _, domain := range domains {
			uniqueDomains[domain] = true
		}
		categoryStats[category] = len(domains)
	}

	allDomains := make([]string, 0, len(uniqueDomains))
	for domain := range uniqueDomains {
		allDomains = append(allDomains, domain)
	}

	log.Tracef("Loaded %d total domains from %d categories", len(allDomains), len(categories))
	return allDomains, categoryStats, nil
}

// GetCategoryCounts returns domain counts for specified categories (loads if not cached)
func (gm *GeodataManager) GetCategoryCounts(categories []string) (map[string]int, error) {
	if len(categories) == 0 {
		return make(map[string]int), nil
	}

	counts := make(map[string]int)

	for _, category := range categories {
		// Check cache first
		gm.mu.RLock()
		if count, exists := gm.categoryCounts[category]; exists {
			counts[category] = count
			gm.mu.RUnlock()
			continue
		}
		gm.mu.RUnlock()

		// Not in cache, load it
		domains, err := gm.LoadCategory(category)
		if err != nil {
			log.Errorf("Failed to get count for category %s: %v", category, err)
			counts[category] = 0
			continue
		}
		counts[category] = len(domains)
	}

	return counts, nil
}

// GetCachedCategoryBreakdown returns counts for all cached categories
func (gm *GeodataManager) GetCachedCategoryBreakdown() map[string]int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	breakdown := make(map[string]int, len(gm.categoryCounts))
	for category, count := range gm.categoryCounts {
		breakdown[category] = count
	}
	return breakdown
}

// PreloadCategories loads and caches categories at startup
func (gm *GeodataManager) PreloadCategories(categories []string) (map[string]int, error) {
	log.Infof("Preloading %d geosite categories...", len(categories))

	_, counts, err := gm.LoadCategories(categories)
	if err != nil {
		return nil, err
	}

	totalDomains := 0
	for _, count := range counts {
		totalDomains += count
	}

	log.Infof("Preloaded %d domains across %d categories", totalDomains, len(counts))
	return counts, nil
}

// ClearCache clears all cached data
func (gm *GeodataManager) ClearCache() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.categoryDomains = make(map[string][]string)
	gm.categoryCounts = make(map[string]int)
	log.Infof("Geodata cache cleared")
}

// GetTotalCachedDomains returns the total number of domains across all cached categories
func (gm *GeodataManager) GetTotalCachedDomains() int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	total := 0
	for _, count := range gm.categoryCounts {
		total += count
	}
	return total
}

// IsConfigured returns true if geodata paths are configured
func (gm *GeodataManager) IsConfigured() bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.geositePath != ""
}

// GetGeositePath returns the current geosite path
func (gm *GeodataManager) GetGeositePath() string {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	return gm.geositePath
}
