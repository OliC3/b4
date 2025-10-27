package handler

import (
	"net/http"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/nfq"
)

var (
	globalPool *nfq.Pool
)

func SetNFQPool(pool *nfq.Pool) {
	globalPool = pool
}

func NewAPIHandler(cfg *config.Config) *API {
	return &API{
		cfg:            cfg,
		manualDomains:  []string{},
		geositeDomains: make(map[string][]string),
	}
}

func (api *API) RegisterEndpoints(mux *http.ServeMux, cfg *config.Config) {

	api.cfg = cfg
	api.mux = mux
	api.RegisterConfigApi()
	api.RegisterMetricsApi()
	api.RegisterGeositeApi()
}
