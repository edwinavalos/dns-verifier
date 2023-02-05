package main

import (
	"context"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/datastore"
	"github.com/edwinavalos/dns-verifier/datastore/dynamo"
	"github.com/edwinavalos/dns-verifier/datastore/s3_filestore"
	"github.com/edwinavalos/dns-verifier/logger"
	v1 "github.com/edwinavalos/dns-verifier/routers/api/v1"
	"github.com/edwinavalos/dns-verifier/server"
	"github.com/edwinavalos/dns-verifier/service/cert_service"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	rootCtx := context.Background()
	var rootLogger = logger.Logger{
		Logger: zerolog.New(os.Stdout),
	}
	setLoggers(rootLogger)

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

	setConfigs(appConfig)

	dbStorage, err := dynamo.NewStorage(appConfig.DB)
	if err != nil {
		panic(err)
	}

	err = dbStorage.Initialize()
	if err != nil {
		panic(err)
	}

	setDBStorage(dbStorage)

	fileStore, err := s3_filestore.NewS3Storage(&appConfig.CloudProvider)
	if err != nil {
		panic(err)
	}

	setFileStorage(fileStore)

	srv := server.NewServer()
	srv.ListenAndServe()
}

func setFileStorage(store datastore.FileStore) {
	cert_service.SetFileStorage(store)
}

func setDBStorage(storage datastore.Datastore) {
	// At some point we can pass in a polymorphic configuration
	domain_service.SetDBStorage(storage)
	cert_service.SetDBStorage(storage)
}

func setConfigs(appConfig *config.Config) {
	domain_service.SetConfig(appConfig)
	v1.SetConfig(appConfig)
	cert_service.SetConfig(appConfig)
	datastore.SetConfig(appConfig)
}

func setLoggers(rootLogger logger.Logger) {
	domain_service.SetLogger(&rootLogger)
	cert_service.SetLogger(&rootLogger)
	v1.SetLogger(&rootLogger)
	datastore.SetLogger(&rootLogger)
}
