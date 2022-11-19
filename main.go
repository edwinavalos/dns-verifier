package main

import (
	"context"
	"dnsVerifier/config"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	rootCtx := context.Background()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./resources")
	err := viper.ReadInConfig()
	if err != nil {
		log.Panic().Msgf("unable to read configuration file, exiting.")
		panic(err)
	}

	cCtx, cancel := context.WithCancel(rootCtx)

	config := config.NewConfig()
	config.RootCtx = rootCtx
	config.Aws.Region = viper.GetString("aws.region")
	config.Aws.BucketName = viper.GetString("aws.s3BucketName")
	config.Aws.VerificationFileName = viper.GetString("aws.verificationFileName")
	config.Aws.CancelCtx = cancel
	config.App.VerificationTxtRecordName = viper.GetString("app.verificationTxtRecordName")

	cfg, err := awsConfig.LoadDefaultConfig(cCtx, awsConfig.WithRegion(config.Aws.Region))
	if err != nil {
		log.Panic().Msg("unable to load default aws config")
		panic(err)
	}

	awsS3Client := s3.NewFromConfig(cfg)
	config.Aws.S3Client = awsS3Client

	verificationList, err := GetOrCreateVerificationFile(cCtx, awsS3Client, config)
	if err != nil {
		log.Panic().Msgf("unable to get verification file from s3")
		panic(err)
	}
	fmt.Printf("verificationList: %s", verificationList)

}
