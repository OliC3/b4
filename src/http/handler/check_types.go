package handler

import "time"

type StartCheckRequest struct {
	CheckURL      string        `json:"check_url,omitempty"`
	Timeout       time.Duration `json:"timeout"`
	MaxConcurrent int           `json:"max_concurrent"`
}

type StartCheckResponse struct {
	Id          string `json:"id"`
	TotalChecks int    `json:"total_checks"`
	Message     string `json:"message"`
}

type AddDomainRequest struct {
	Domain string `json:"domain"`
	SetId  string `json:"set_id"`
}

type AddDomainResponse struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	Domain        string   `json:"domain"`
	TotalDomains  int      `json:"total_domains"`
	ManualDomains []string `json:"manual_domains,omitempty"`
}
