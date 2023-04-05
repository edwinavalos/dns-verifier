package server

import (
	"github.com/edwinavalos/common/config"
	"github.com/edwinavalos/dns-verifier/routers"
	"github.com/edwinavalos/dns-verifier/service/cert_service"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"net/http"
	"time"
)

func NewServer(conf *config.Config, domainService *domain_service.Service, certService *cert_service.Service) *http.Server {
	routes := routers.InitRouter(conf, domainService, certService)
	return &http.Server{
		Addr:         ":8081",
		Handler:      routes,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}
