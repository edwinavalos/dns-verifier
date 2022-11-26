package config

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
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
	AlwaysRecreate            bool
}

type Config struct {
	Aws        AWSSettings
	App        AppSettings
	RootCancel context.CancelFunc
	RootCtx    context.Context
}

func (c *Config) ReadConfig() *Config {
	c.Aws.Region = viper.GetString("aws.region")
	c.Aws.BucketName = viper.GetString("aws.s3BucketName")
	c.Aws.VerificationFileName = viper.GetString("aws.verificationFileName")
	c.App.VerificationTxtRecordName = viper.GetString("app.verificationTxtRecordName")
	c.App.AlwaysRecreate = viper.GetBool("app.alwaysRecreate")
	return c
}

func NewConfig() *Config {
	return &Config{
		Aws: AWSSettings{
			Region:               "us-west-2",
			BucketName:           "test-bucket",
			VerificationFileName: "example-file.json",
		},
		App: AppSettings{
			VerificationTxtRecordName: "mastodon_ownership_key",
		},
		RootCtx: nil,
	}
}
