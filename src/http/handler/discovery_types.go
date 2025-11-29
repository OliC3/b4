package handler

type DiscoveryRequest struct {
	CheckURL string `json:"check_url,omitempty"`
	Domain   string `json:"domain,omitempty"`
}

type DiscoveryResponse struct {
	Id             string `json:"id"`
	Domain         string `json:"domain"`
	EstimatedTests int    `json:"estimated_tests"`
	Message        string `json:"message"`
}
