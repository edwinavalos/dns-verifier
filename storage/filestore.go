package storage

import (
	"github.com/edwinavalos/common/config"
	"github.com/edwinavalos/common/datastore/s3_filestore"
)

type VerifierFileStore struct {
	*s3_filestore.S3Store
}

func NewFileStore(cfg *config.Config) (*VerifierFileStore, error) {
	filestore, err := s3_filestore.New(cfg)
	if err != nil {
		return nil, err
	}

	return &VerifierFileStore{
		S3Store: filestore,
	}, nil
}
