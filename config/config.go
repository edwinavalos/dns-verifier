package config

import "context"

type AWSSettings struct {
	Region               string
	BucketName           string
	VerificationFileName string
	CancelCtx            context.CancelFunc
}

type Config struct {
	Aws        AWSSettings
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
