package config

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type AWSSettings struct {
	Region               string
	BucketName           string
	VerificationFileName string
	CancelCtx            context.CancelFunc
	S3Client             *s3.Client
}

type AppSettings struct {
	VerificationTxtRecordName string
}

type Config struct {
	Aws        AWSSettings
	App        AppSettings
	RootCancel context.CancelFunc
	RootCtx    context.Context
}

func NewConfig() *Config {
	return &Config{
		Aws: AWSSettings{
			Region:               "us-west-2",
			BucketName:           "test-bucket",
			VerificationFileName: "example-file.json",
		},
		RootCtx: nil,
	}
}
