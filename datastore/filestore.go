package datastore

import "bytes"

type FileStore interface {
	SaveFile(sourcePath string, destinationPath string) error
	SaveBuf(buffer bytes.Buffer, destinationPath string) error
	GetFile(sourcePath string, destinationPath string) error
}
