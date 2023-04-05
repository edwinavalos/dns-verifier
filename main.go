package main

import (
	"github.com/edwinavalos/common/logger"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/server"
	"github.com/edwinavalos/dns-verifier/service/cert_service"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/edwinavalos/dns-verifier/storage"
	"math/rand"
	"time"
)

func main() {
	logger.New()
	rand.Seed(time.Now().Unix())

	cfg := config.NewConfig()
	datastore, err := storage.NewDataStore(cfg)
	if err != nil {
		panic(err)
	}

	filestore, err := storage.NewFileStore(cfg)
	if err != nil {
		panic(err)
	}

	domainService := domain_service.New(cfg, datastore)
	certService := cert_service.New(cfg, filestore, domainService)
	srv := server.NewServer(cfg, domainService, certService)
	srv.ListenAndServe()
}
