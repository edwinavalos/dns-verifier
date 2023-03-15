package datastore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"io"
	"os"
)

type S3Store struct {
	Bucket string
	Client *s3.Client
}

func NewS3Storage(cfg cfg) (FileStore, error) {
	// Check if we have the config we need
	if cfg.CloudProviderBucketName() == "" {
		Log.Fatalf("did not have enough information to get or s3 bucket")
		Log.Fatalf("bucketName: {%s}", cfg.CloudProviderBucketName())
		return nil, fmt.Errorf("missing aws configuration")
	}

	awsConf, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion(cfg.CloudProviderRegion()))
	if err != nil {
		Log.Fatalf("unable to load default aws appConfig")
		return nil, err
	}
	s3Client := s3.NewFromConfig(awsConf)

	// Check if the bucket exists
	_, err = s3Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(cfg.CloudProviderBucketName()),
	})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				return nil, fmt.Errorf("bucket is not taken, but needs to be created")
			default:
				return nil, fmt.Errorf("can't determine if bucket exists")
			}
		}
	}

	// Save the client to the store object because we know its a good client and the bucket existed, or was created
	// with it
	Log.Infof("was able to connect to bucket, and saving s3 client")

	return &S3Store{
		Bucket: cfg.CloudProviderBucketName(),
		Client: s3.NewFromConfig(awsConf),
	}, nil
}

func (store *S3Store) SaveFile(sourcePath string, destinationPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("ran into issue opening file: %w", err)
	}
	defer file.Close()
	_, err = store.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(fmt.Sprintf(store.Bucket)),
		Key:    aws.String(destinationPath),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("ran into issue uploading file: %w", err)
	}
	Log.Infof("uploaded: %s to s3://%s/%s", sourcePath, store.Bucket, destinationPath)
	return nil
}

func (store *S3Store) SaveBuf(buffer bytes.Buffer, destinationPath string) error {
	_, err := store.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(fmt.Sprintf(store.Bucket)),
		Key:    aws.String(destinationPath),
		Body:   bytes.NewReader(buffer.Bytes()),
	})
	if err != nil {
		return fmt.Errorf("ran into issue uploading buffer: %s", err)
	}
	Log.Infof("uploaded the buffer to s3://%s/%s", store.Bucket, destinationPath)
	return nil
}

func (store *S3Store) GetFile(sourcePath string, destinationPath string) error {
	result, err := store.Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(store.Bucket),
		Key:    aws.String(sourcePath),
	})
	if err != nil {
		return fmt.Errorf("unable to get object: s3://%s/%s from bucket: %w", store.Bucket, sourcePath, err)
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
