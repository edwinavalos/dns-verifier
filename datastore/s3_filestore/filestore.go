package s3_filestore

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/datastore"
	"io"
	"os"
)

type S3Store struct {
	Bucket string
	Client *s3.Client
}

func (store *S3Store) Initialize(cfg *config.Config) error {

	// Check if we have the config we need
	if cfg.AWS.BucketName == "" {
		datastore.Log.Fatalf("did not have enough information to get or s3 bucket")
		datastore.Log.Fatalf("bucketName: {%s}", cfg.AWS.BucketName)
		return fmt.Errorf("missing aws configuration")
	}

	awsConf, err := awsConfig.LoadDefaultConfig(cfg.RootCtx, awsConfig.WithRegion(cfg.AWS.Region))
	if err != nil {
		datastore.Log.Fatalf("unable to load default aws appConfig")
		return err
	}
	s3Client := s3.NewFromConfig(awsConf)

	// Check if the bucket exists
	_, err = s3Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(cfg.AWS.BucketName),
	})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				return fmt.Errorf("bucket is not taken, but needs to be created")
			default:
				return fmt.Errorf("can't determine if bucket exists")
			}
		}
	}

	// Save the client to the store object because we know its a good client and the bucket existed, or was created
	// with it
	datastore.Log.Infof("was able to connect to bucket, and saving s3 client")
	store.Client = s3.NewFromConfig(awsConf)
	return nil
}

func (store *S3Store) SaveFile(sourcePath string, destinationPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("ran into issue opening file: %w", err)
	}
	defer file.Close()
	_, err = store.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(fmt.Sprintf("s3://%s", store.Bucket)),
		Key:    aws.String(destinationPath),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("ran into issue uploading file: %w", err)
	}
	datastore.Log.Infof("uploaded: %s to s3://%s/%s", sourcePath, store.Bucket, destinationPath)
	return nil
}

func (store *S3Store) GetFile(sourcePath string, destinationPath string) error {
	result, err := store.Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(fmt.Sprintf("s3://%s", store.Bucket)),
		Key:    aws.String(sourcePath),
	})
	if err != nil {
		return fmt.Errorf("unable to get object from bucket: %w", err)
	}
	defer result.Body.Close()
	file, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("unable to create destination file: %w", err)
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("unable to read body of resuly body: %w", err)
	}
	_, err = file.Write(body)
	if err != nil {
		return fmt.Errorf("unable to write file body to destination: %w", err)
	}

	return nil
}
