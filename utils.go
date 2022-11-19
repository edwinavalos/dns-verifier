package dnsVerifier

import (
	"dnsVerifier/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3 struct {
	s3.Client
}

func (S3) GetOrCreateVerificationFile(config config.Config) {
	config.
		s3.GetObject()
}
