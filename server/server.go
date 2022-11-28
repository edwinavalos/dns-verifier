package server

import (
	"dnsVerifier/config"
	"dnsVerifier/routers"
	"dnsVerifier/service/verification_service"
	"net/http"
	"sync"
	"time"
)

func NewServer(appConfig *config.Config, verifications sync.Map) *http.Server {
	verification_service.SvConfig = appConfig
	verification_service.VerificationMap = verifications
	routes := routers.InitRouter()
	return &http.Server{
		Addr:         ":8080",
		Handler:      routes,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}
