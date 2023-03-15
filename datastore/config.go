package datastore

type cfg interface {
	CloudProviderRegion() string
	CloudProviderBucketName() string
	DBAlwaysRecreate() bool
	DBTableName() string
	DBRegion() string
	DBIsLocal() bool
	Env() string
}
