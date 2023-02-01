package main

import (
	"context"
	"fmt"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	cCtx, cancel := context.WithCancel(rootCtx)

	appConfig := config.NewConfig()
	appConfig.RootCtx = rootCtx
	appConfig.Aws.CancelCtx = cancel
	appConfig.ReadConfig()

	if appConfig.Aws.BucketName == "" || appConfig.Aws.VerificationFileName == "" {
		log.Fatal().Msgf("did not have enough information to get or create domain_service file")
		log.Debug().Msgf("bucketName: {%s}, verificationFileName {%s}", appConfig.Aws.BucketName, appConfig.Aws.VerificationFileName)
		panic(fmt.Errorf("missing aws configuration"))
	}

	cfg, err := awsConfig.LoadDefaultConfig(cCtx, awsConfig.WithRegion(appConfig.Aws.Region))
	if err != nil {
		log.Panic().Msg("unable to load default aws appConfig")
		panic(err)
	}

	awsS3Client := s3.NewFromConfig(cfg)
	appConfig.Aws.S3Client = awsS3Client
	SetConfigs(appConfig)

	storage, err := dynamo.NewStorage()
	if err != nil {
		panic(err)
	}

	err = storage.Initialize()
	if err != nil {
		panic(err)
	}

	// At some point we can pass in a polymorphic configuration
	domain_service.SetStorage(storage)
	cert_service.SetStorage(storage)
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
