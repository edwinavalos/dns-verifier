package main

import (
	"context"
	"dnsVerifier/config"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
	"os"
)

func GetOrCreateVerificationFile(ctx context.Context, s3Client *s3.Client, config *config.Config) error {

	if config.Aws.BucketName == "" || config.Aws.VerificationFileName == "" {
		log.Fatal().Msgf("did not have enough information to get or create verification file")
		log.Debug().Msgf("bucketName: {%s}, verificationFileName {%s}", config.Aws.BucketName, config.Aws.VerificationFileName)
		return fmt.Errorf("missing aws configuration")
	}

	getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &config.Aws.BucketName,
		Key:    &config.Aws.VerificationFileName,
	})
	createFile := false
	if err != nil {
		var nske *types.NoSuchKey
		if errors.As(err, &nske) {
			log.Info().Msgf("Did not find key, creating file... s3://%s/%s", config.Aws.BucketName, config.Aws.VerificationFileName)
			createFile = true
		}
		var nsb *types.NoSuchBucket
		if errors.As(err, &nsb) {
			log.Error().Msgf("bucket: %s does not exit... exiting", config.Aws.BucketName)
			return fmt.Errorf("bucket does not exist")
		}
	}
	if createFile {
		emptyFile, err := os.Create(fmt.Sprintf("%s", config.Aws.VerificationFileName))
		if err != nil {
			log.Fatal().Msgf("unable to create empty file: %s", config.Aws.VerificationFileName)
			return err
		}
		file, err := os.Open(emptyFile.Name())
		if err != nil {
			return err
		}
		stat, err := os.Stat(emptyFile.Name())
		if err != nil {
			return err
		}
		s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        &config.Aws.BucketName,
			Key:           &config.Aws.VerificationFileName,
			Body:          file,
			ContentLength: stat.Size(),
		})
	}
	return nil

}
