package handler

import (
	"net/http"

	"github.com/daniellavrushin/b4/config"
)

type API struct {
	cfg            *config.Config
	mux            *http.ServeMux
	manualDomains  []string            // Track manually added domains
	geositeDomains map[string][]string // Track geosite category -> domains mapping
}

// Response types for API endpoints
type GeositeResponse struct {
	Tags []string `json:"tags"`
}

// ConfigResponse wraps the config with additional metadata
type ConfigResponse struct {
	*config.Config
	DomainStats DomainStatistics `json:"domain_stats"`
}

// DomainStatistics provides overview of domain configuration
type DomainStatistics struct {
	ManualDomains     int            `json:"manual_domains"`
	GeositeDomains    int            `json:"geosite_domains"`
	TotalDomains      int            `json:"total_domains"`
	CategoryBreakdown map[string]int `json:"category_breakdown,omitempty"`
	GeositeAvailable  bool           `json:"geosite_available"`
}

// CategoryPreviewResponse for previewing category contents
type CategoryPreviewResponse struct {
	Category     string   `json:"category"`
	TotalDomains int      `json:"total_domains"`
	PreviewCount int      `json:"preview_count"`
	Preview      []string `json:"preview"`
}

// ConfigUpdateRequest for handling config updates
type ConfigUpdateRequest struct {
	*config.Config
	// Additional fields for UI state if needed
	PreserveManuaDomains bool `json:"preserve_manual_domains,omitempty"`
}

// ConfigUpdateResponse for config update results
type ConfigUpdateResponse struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message"`
	DomainStats DomainStatistics `json:"domain_stats"`
	Warnings    []string         `json:"warnings,omitempty"`
}

// AddDomainRequest represents the request body for adding a domain
type AddDomainRequest struct {
	Domain string `json:"domain"`
}

// AddDomainResponse represents the response for adding a domain
type AddDomainResponse struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	Domain        string   `json:"domain"`
	TotalDomains  int      `json:"total_domains"`
	ManualDomains []string `json:"manual_domains,omitempty"`
}
