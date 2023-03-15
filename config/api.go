package config

type Config struct {
	appConfig
}

func (c *Config) LECADirURL() string {
	return c.appConfig.LECADirURL
}

func (c *Config) LEAdminEmail() string {
	return c.appConfig.LEAdminEmail
}

func (c *Config) LEPrivateKeyLocation() string {
	return c.appConfig.LEPrivateKeyLocation
}

func (c *Config) LEKeyAuth() string {
	return c.appConfig.LEKeyAuth
}

func (c *Config) CloudProviderRegion() string {
	return c.appConfig.CloudProviderRegion
}

func (c *Config) CloudProviderBucketName() string {
	return c.appConfig.CloudProviderBucketName
}

func (c *Config) VerificationTxtRecordName() string {
	return c.appConfig.VerificationTxtRecordName
}

func (c *Config) DBAlwaysRecreate() bool {
	return c.appConfig.DBAlwaysRecreate
}

func (c *Config) OwnedHosts() []string {
	return c.appConfig.OwnedHosts
}

func (c *Config) OwnedCNames() []string {
	return c.appConfig.OwnedCNames
}

func (c *Config) DBTableName() string {
	return c.appConfig.DBTableName
}

func (c *Config) DBRegion() string {
	return c.appConfig.DBRegion
}

func (c *Config) DBIsLocal() bool {
	return c.appConfig.DBIsLocal
}

func (c *Config) Env() string {
	return c.appConfig.Env
}
