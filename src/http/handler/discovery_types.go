package handler

type StartCheckRequest struct {
	CheckURL string   `json:"check_url,omitempty"`
	Domains  []string `json:"domains,omitempty"`
}

type StartCheckResponse struct {
	Id          string `json:"id"`
	TotalChecks int    `json:"total_checks"`
	Message     string `json:"message"`
}

type AddDomainRequest struct {
	Domain  string `json:"domain"`
	SetId   string `json:"set_id"`
	SetName string `json:"set_name,omitempty"`
}

type AddDomainResponse struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	Domain        string   `json:"domain"`
	TotalDomains  int      `json:"total_domains"`
	ManualDomains []string `json:"manual_domains,omitempty"`
}
