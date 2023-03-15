package config

import (
	"github.com/go-acme/lego/v4/lego"
	"github.com/spf13/viper"
)

type appConfig struct {
	LECADirURL                string   `json:"ca_dir_url"`
	LEAdminEmail              string   `json:"admin_email"`
	LEPrivateKeyLocation      string   `json:"private_key_location"`
	LEKeyAuth                 string   `json:"key_auth"`
	CloudProviderRegion       string   `json:"cloud_provider_region"`
	CloudProviderBucketName   string   `json:"cloud_provider_bucket_name"`
	VerificationTxtRecordName string   `json:"verification_txt_record_name"`
	DBAlwaysRecreate          bool     `json:"db_always_recreate"`
	OwnedHosts                []string `json:"owned_hosts"`
	OwnedCNames               []string `json:"owned_c_names"`
	DBTableName               string   `json:"db_table_name"`
	DBRegion                  string   `json:"db_region"`
	DBIsLocal                 bool     `json:"db_is_local"`
	Env                       string   `json:"env"`
	StripeKey                 string   `json:"stripe_key"`
	EncKey                    string   `json:"enc_key"`
	SessionName               string   `json:"session_name"`
	CookieSecret              string   `json:"cookie_secret"`
	OauthClientID             string   `json:"oauth_client_id"`
	OauthClientSecret         string   `json:"oauth_client_secret"`
	OauthRedirectURL          string   `json:"oauth_redirect_url"`
	OauthIssuerURL            string   `json:"oauth_issuer_url"`
	OauthScopes               []string `json:"oauth_scopes"`
}

func (c *appConfig) ReadConfig() {
	c.CloudProviderRegion = viper.GetString("cloud_provider.region")
	c.CloudProviderBucketName = viper.GetString("cloud_provider.bucket_name")
	c.VerificationTxtRecordName = viper.GetString("app.verificationTxtRecordName")
	c.DBAlwaysRecreate = viper.GetBool("app.alwaysRecreate")
	c.OwnedHosts = viper.GetStringSlice("network.owned_hosts")
	c.OwnedCNames = viper.GetStringSlice("network.owned_cnames")

	c.LECADirURL = lego.LEDirectoryStaging
	if c.Env == "prod" {
		c.LECADirURL = lego.LEDirectoryProduction
	}
	c.LEAdminEmail = viper.GetString("le_settings.admin_email")
	c.LEPrivateKeyLocation = viper.GetString("le_settings.private_key_location")

	c.DBTableName = viper.GetString("db.table_name")
	c.DBRegion = viper.GetString("db.region")
	c.DBIsLocal = viper.GetBool("db.is_local")

	c.StripeKey = viper.GetString("stripe.key")

	c.EncKey = viper.GetString("app.enc_key")
	c.SessionName = viper.GetString("app.session_name")

	c.CookieSecret = viper.GetString("app.cookie_secret")
	c.OauthClientID = viper.GetString("oauth_settings.client_id")
	c.OauthClientSecret = viper.GetString("oauth_settings.client_secret")
	c.OauthRedirectURL = viper.GetString("oauth_settings.redirect_url")
	c.OauthIssuerURL = viper.GetString("oauth_settings.issuer_url")
	c.OauthScopes = viper.GetStringSlice("oauth_settings.scopes")
	return
}

func NewConfig() *Config {
	return &Config{
		appConfig: appConfig{},
	}
}
