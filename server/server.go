package server

import (
	"github.com/edwinavalos/dns-verifier/routers"
	"net/http"
	"time"
)

func NewServer() *http.Server {
	routes := routers.InitRouter()
	return &http.Server{
		Addr:         ":8080",
		Handler:      routes,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}
