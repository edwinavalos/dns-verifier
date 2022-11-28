package utils

import (
	"context"
	"dnsVerifier/config"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
)

func SyncMap2Map(syncMap *sync.Map) map[string]interface{} {
	regMap := make(map[string]interface{})
	syncMap.Range(func(k interface{}, v interface{}) bool {
		regMap[k.(string)] = v
		return true
	})

	return regMap
}

func SaveVerificationFile(ctx context.Context, verifications *sync.Map, config *config.Config) error {
	verificationFileName := config.Aws.VerificationFileName
	log.Debug().Msgf("creating verification_service file at s3://%s/%s", config.Aws.BucketName, config.Aws.VerificationFileName)
	jsonMap := SyncMap2Map(verifications)
	content, _ := json.MarshalIndent(jsonMap, "", " ")
	err := ioutil.WriteFile(verificationFileName, content, 0644)
	if err != nil {
		return err
	}
	file, err := os.Open(verificationFileName)
	if err != nil {
		return err
	}
	stat, err := os.Stat(verificationFileName)
	if err != nil {
		return err
	}
	_, err = config.Aws.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &config.Aws.BucketName,
		Key:           &config.Aws.VerificationFileName,
		Body:          file,
		ContentLength: stat.Size(),
	})
	if err != nil {
		log.Error().Msgf("unable to create file at s3://%s/%s", config.Aws.BucketName, config.Aws.VerificationFileName)
		return err
	}

	return nil
}

func GetOrCreateVerificationFile(ctx context.Context, config *config.Config) (*sync.Map, error) {

	var verifications *sync.Map
	createFile := false
	getObjectOutput, err := config.Aws.S3Client.GetObject(ctx, &s3.GetObjectInput{
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
			return verifications, fmt.Errorf("bucket does not exist")
		}
	}
	if createFile || config.App.AlwaysRecreate {
		err2 := SaveVerificationFile(ctx, verifications, config)
		if err2 != nil {
			return verifications, err2
		}
		return verifications, nil
	}

	if getObjectOutput.ContentLength != 0 {
		err = json.NewDecoder(getObjectOutput.Body).Decode(&verifications)
		if err != nil {
			log.Error().Msgf("unable to decode contents of verification_service file into verification_service list")
			return verifications, err
		}
	}

	return verifications, nil

}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
