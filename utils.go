package main

import (
	"context"
	"dnsVerifier/config"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
	"net"
	"os"
)

func GetOrCreateVerificationFile(ctx context.Context, s3Client *s3.Client, config *config.Config) ([]VerificationItem, error) {

	if config.Aws.BucketName == "" || config.Aws.VerificationFileName == "" {
		log.Fatal().Msgf("did not have enough information to get or create verification file")
		log.Debug().Msgf("bucketName: {%s}, verificationFileName {%s}", config.Aws.BucketName, config.Aws.VerificationFileName)
		return nil, fmt.Errorf("missing aws configuration")
	}

	createFile := false
	getObjectOutput, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &config.Aws.BucketName,
		Key:    &config.Aws.VerificationFileName,
	})
	if err != nil {
		var nske *types.NoSuchKey
		if errors.As(err, &nske) {
			log.Info().Msgf("Did not find key, creating file... s3://%s/%s", config.Aws.BucketName, config.Aws.VerificationFileName)
			createFile = true
		}
		var nsb *types.NoSuchBucket
		if errors.As(err, &nsb) {
			log.Error().Msgf("bucket: %s does not exit... exiting", config.Aws.BucketName)
			return nil, fmt.Errorf("bucket does not exist")
		}
	}

	if createFile {
		emptyFile, err := os.Create(fmt.Sprintf("%s", config.Aws.VerificationFileName))
		if err != nil {
			log.Fatal().Msgf("unable to create empty file: %s", config.Aws.VerificationFileName)
			return nil, err
		}
		file, err := os.Open(emptyFile.Name())
		if err != nil {
			return nil, err
		}
		stat, err := os.Stat(emptyFile.Name())
		if err != nil {
			return nil, err
		}
		_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        &config.Aws.BucketName,
			Key:           &config.Aws.VerificationFileName,
			Body:          file,
			ContentLength: stat.Size(),
		})
		if err != nil {
			log.Error().Msgf("unable to create file at s3://%s/%s", config.Aws.BucketName, config.Aws.VerificationFileName)
			return nil, err
		}
		log.Info().Msgf("created file at s3://%s/%s", config.Aws.BucketName, config.Aws.VerificationFileName)
		return []VerificationItem{}, nil
	}

	var verificationList []VerificationItem
	err = json.NewDecoder(getObjectOutput.Body).Decode(&verificationList)
	if err != nil {
		log.Error().Msgf("unable to decode contents of verification file into verification list")
		return nil, err
	}

	return verificationList, nil

}

func VerifyDomain(ctx context.Context, verificationList []VerificationItem) error {

	for _, item := range verificationList {
		txtrecords, err := net.LookupTXT(item.Domain)
		if err != nil {
			return err
		}

		for _, txt := range txtrecords {
			if txt
		}

	}
}
