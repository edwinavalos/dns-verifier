package config

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-acme/lego/v4/lego"
	"github.com/spf13/viper"
)

type LetsEncryptSettings struct {
	CADirURL           string `json:"ca_dir_url"`
	AdminEmail         string `json:"admin_email"`
	PrivateKeyLocation string `json:"private_key_location"`
	KeyAuth            string `json:"key_auth"`
}

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

type NetworkSettings struct {
	OwnedHosts  []string // Could be a net.url, but do we need to be fancy?
	OwnedCnames []string
}

type Config struct {
	Aws        AWSSettings
	App        AppSettings
	LESettings LetsEncryptSettings
	Env        string
	Network    NetworkSettings
	RootCancel context.CancelFunc
	RootCtx    context.Context
}

func (c *Config) ReadConfig() *Config {
	c.Aws.Region = viper.GetString("aws.region")
	c.Aws.BucketName = viper.GetString("aws.s3BucketName")
	c.Aws.VerificationFileName = viper.GetString("aws.verificationFileName")
	c.App.VerificationTxtRecordName = viper.GetString("app.verificationTxtRecordName")
	c.App.AlwaysRecreate = viper.GetBool("app.alwaysRecreate")
	c.Network.OwnedHosts = viper.GetStringSlice("network.owned_hosts")
	c.Network.OwnedCnames = viper.GetStringSlice("network.owned_cnames")

	c.LESettings.CADirURL = lego.LEDirectoryStaging
	if c.Env == "prod" {
		c.LESettings.CADirURL = lego.LEDirectoryProduction
	}
	c.LESettings.AdminEmail = viper.GetString("le_settings.admin_email")
	c.LESettings.PrivateKeyLocation = viper.GetString("le_settings.private_key_location")
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
		Network: NetworkSettings{
			OwnedHosts:  []string{"0.0.0.0"},
			OwnedCnames: []string{"edwinavalos.com"},
		},
		RootCancel: nil,
		RootCtx:    nil,
	}
}
