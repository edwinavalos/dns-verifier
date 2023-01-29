package main

import (
	"context"
	"fmt"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/edwinavalos/dns-verifier/config"
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
	domain_service.SetLogger(&rootLogger)

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
	domain_service.SetConfig(appConfig)
	v1.SetConfig(appConfig)
	cert_service.SetConfig(appConfig)

	verifications, err := domain_service.GetOrCreateDomainInformationFile(cCtx)
	if err != nil {
		log.Panic().Msgf("unable to get verification file from s3")
		panic(err)
	}
	log.Debug().Msgf("verifications: %+v\n", domain_service.SyncMap2Map(verifications))

	srv := server.NewServer(verifications)
	srv.ListenAndServe()
}
