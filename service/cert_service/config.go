package cert_service

type Config interface {
	LEPrivateKeyLocation() string
	LECADirURL() string
	LEAdminEmail() string
}
