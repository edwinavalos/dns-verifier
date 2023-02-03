package main

import (
	"context"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/datastore"
	"github.com/edwinavalos/dns-verifier/datastore/dynamo"
	"github.com/edwinavalos/dns-verifier/logger"
	v1 "github.com/edwinavalos/dns-verifier/routers/api/v1"
	"github.com/edwinavalos/dns-verifier/server"
	"github.com/edwinavalos/dns-verifier/service/cert_service"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	rootCtx := context.Background()
	rootLogger := logger.Logger{
		Logger: zerolog.Logger{},
	}

	SetLoggers(rootLogger)

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./resources")
	err := viper.ReadInConfig()
	if err != nil {
		log.Panic().Msgf("unable to read configuration file, exiting.")
		panic(err)
	}

	appConfig := config.NewConfig()
	appConfig.RootCtx = rootCtx
	appConfig.ReadConfig()

	SetConfigs(appConfig)

	storage, err := dynamo.NewStorage(appConfig)
	if err != nil {
		panic(err)
	}

	err = storage.Initialize()
	if err != nil {
		panic(err)
	}

	// At some point we can pass in a polymorphic configuration
	domain_service.SetDBStorage(storage)
	cert_service.SetDBStorage(storage)
	srv := server.NewServer()
	srv.ListenAndServe()
}

func SetConfigs(appConfig *config.Config) {
	domain_service.SetConfig(appConfig)
	v1.SetConfig(appConfig)
	cert_service.SetConfig(appConfig)
	datastore.SetConfig(appConfig)
}

func SetLoggers(rootLogger logger.Logger) {
	domain_service.SetLogger(&rootLogger)
	cert_service.SetLogger(&rootLogger)
	v1.SetLogger(&rootLogger)
	datastore.SetLogger(&rootLogger)
}
