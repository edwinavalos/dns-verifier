package config

import (
	"context"
	"github.com/go-acme/lego/v4/lego"
	"github.com/spf13/viper"
)

type LetsEncryptSettings struct {
	CADirURL           string `json:"ca_dir_url"`
	AdminEmail         string `json:"admin_email"`
	PrivateKeyLocation string `json:"private_key_location"`
	KeyAuth            string `json:"key_auth"`
}

type CloudProviderSettings struct {
	Region     string
	BucketName string
}

type AppSettings struct {
	VerificationTxtRecordName string
	AlwaysRecreate            bool
}

type NetworkSettings struct {
	OwnedHosts  []string // Could be a net.url, but do we need to be fancy?
	OwnedCNames []string
}

type DatabaseSettings struct {
	TableName string
	Region    string
	IsLocal   bool
}

type Config struct {
	CloudProvider CloudProviderSettings
	App           AppSettings
	LESettings    LetsEncryptSettings
	DB            DatabaseSettings
	Env           string
	Network       NetworkSettings
	RootCtx       context.Context
}

func (c *Config) ReadConfig() *Config {
	c.CloudProvider.Region = viper.GetString("cloud_provider.region")
	c.CloudProvider.BucketName = viper.GetString("cloud_provider.bucket_name")
	c.App.VerificationTxtRecordName = viper.GetString("app.verificationTxtRecordName")
	c.App.AlwaysRecreate = viper.GetBool("app.alwaysRecreate")
	c.Network.OwnedHosts = viper.GetStringSlice("network.owned_hosts")
	c.Network.OwnedCNames = viper.GetStringSlice("network.owned_cnames")

	c.LESettings.CADirURL = lego.LEDirectoryStaging
	if c.Env == "prod" {
		c.LESettings.CADirURL = lego.LEDirectoryProduction
	}
	c.LESettings.AdminEmail = viper.GetString("le_settings.admin_email")
	c.LESettings.PrivateKeyLocation = viper.GetString("le_settings.private_key_location")

	c.DB.TableName = viper.GetString("db.table_name")
	c.DB.Region = viper.GetString("db.region")
	c.DB.IsLocal = viper.GetBool("db.is_local")

	return c
}

func NewConfig() *Config {
	return &Config{
		CloudProvider: CloudProviderSettings{
			Region:     "us-west-2",
			BucketName: "test-bucket",
		},
		App: AppSettings{
			VerificationTxtRecordName: "mastodon_ownership_key",
		},
		Network: NetworkSettings{
			OwnedHosts:  []string{"0.0.0.0"},
			OwnedCNames: []string{"edwinavalos.com"},
		},
		RootCtx: nil,
	}
}
