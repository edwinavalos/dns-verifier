package datastore

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/pem"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestS3Store_Initialize(t *testing.T) {
	SetLogger(&logger.Logger{Logger: zerolog.Logger{}})
	type args struct {
		cfg *config.config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Can initialize a bucket that exists",
			args: args{
				cfg: &config.config{CloudProvider: config.CloudProviderSettings{
					Region:     "us-west-2",
					BucketName: "dns-verifier-test-bucket",
				}},
			},
			wantErr: false,
		},
		{
			name: "Can't initialize a bucket that doesn't exist",
			args: args{
				cfg: &config.config{
					CloudProvider: config.CloudProviderSettings{
						Region:     "us-west-2",
						BucketName: "this-bucket-does-not-exist-freals",
					}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewS3Storage(&tt.args.cfg.CloudProvider)
			if err != nil && tt.wantErr {
				t.Logf("NewS3Storage() error = %+v, wantErr %+v", err, tt.wantErr)
				return
			} else if err != nil {
				t.Errorf("NewS3Storage() error = %+v, wantErr %+v", err, tt.wantErr)
			}
		})
	}
}

func conv(a interface{}) *S3Store {
	return a.(*S3Store)
}

func TestS3Store_GetFile(t *testing.T) {

	type args struct {
		sourcePath      string
		destinationPath string
		hash            string
		bucketName      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Can pull down file in bucket successfully",
			args: args{
				sourcePath:      "20221109_192009smaller.jpg",
				destinationPath: "./test-file.jpg",
				hash:            "f9d8a21bd1f22a0692e1dcf566cc8f29",
				bucketName:      "dns-verifier-test-bucket",
			},
			wantErr: false,
		},
		{
			name: "Can not pull down file because it doesn't exist",
			args: args{
				sourcePath:      "not-a-real-file.jpg",
				destinationPath: "./test-file.jpg",
				hash:            "f9d8a21bd1f22a0692e1dcf566cc8f29",
				bucketName:      "dns-verifier-test-bucket",
			},
			wantErr: true,
		},
		{
			name: "Can not pull down file because the bucket doesn't doesn't exist",
			args: args{
				sourcePath:      "not-a-real-file.jpg",
				destinationPath: "./test-file.jpg",
				hash:            "f9d8a21bd1f22a0692e1dcf566cc8f29",
				bucketName:      "this-bucket-does-not-exist-freals",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Config = &config.config{
				CloudProvider: config.CloudProviderSettings{
					Region:     "us-west-2",
					BucketName: tt.args.bucketName,
				},
			}
			SetLogger(&logger.Logger{Logger: zerolog.Logger{}})

			store, err := NewS3Storage(&Config.CloudProvider)
			if err != nil && tt.wantErr {
				t.Logf("NewS3Storage() error = %+v, wantErr %+v", err, tt.wantErr)
				return
			} else if err != nil {
				t.Errorf("NewS3Storage() error = %+v, wantErr %+v", err, tt.wantErr)
			}
			err = store.GetFile(tt.args.sourcePath, tt.args.destinationPath)
			if err != nil && tt.wantErr {
				t.Logf("GetFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err != nil {
				t.Errorf("GetFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			file, err := os.Open(tt.args.destinationPath)
			if err != nil {
				t.Errorf("os.Open() error = %+v, wantErr %+v", err, tt.wantErr)
			}
			defer file.Close()

			hash := md5.New()
			_, err = io.Copy(hash, file)
			if err != nil {
				t.Errorf("io.Copy() error= %+v, wantErr %+v", err, tt.wantErr)
			}
			hashInBytes := hash.Sum(nil)[:16]
			//Convert the bytes to a string
			actual := hex.EncodeToString(hashInBytes)

			assert.Equal(t, tt.args.hash, actual)
		})
	}
}

func TestS3Store_SaveFile(t *testing.T) {
	type args struct {
		sourcePath      string
		destinationPath string
		bucketName      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "can save file to s3 bucket",
			args: args{
				destinationPath: "test-file.empty",
				bucketName:      "dns-verifier-test-bucket",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			myfile, err := os.CreateTemp("./", "testfile")
			if err != nil {
				t.Errorf("os.CreateTemp() err: %+v tt.wantErr %+v", err, tt.wantErr)
			}
			defer os.Remove(myfile.Name())

			Config = &config.config{
				CloudProvider: config.CloudProviderSettings{
					Region:     "us-west-2",
					BucketName: tt.args.bucketName,
				},
			}
			SetLogger(&logger.Logger{Logger: zerolog.Logger{}})

			store, err := NewS3Storage(&Config.CloudProvider)
			if err != nil {
				t.Errorf("NewS3Storage() error = %+v, wantErr %+v", err, tt.wantErr)
			}

			if err := store.SaveFile(myfile.Name(), tt.args.destinationPath); (err != nil) != tt.wantErr {
				t.Errorf("SaveFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestS3Store_SaveBuf(t *testing.T) {
	type args struct {
		sourcePath      string
		destinationPath string
		bucketName      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "can save buffer to s3 bucket",
			args: args{
				destinationPath: "test-file.empty",
				bucketName:      "dns-verifier-test-bucket",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			Config = &config.config{
				CloudProvider: config.CloudProviderSettings{
					Region:     "us-west-2",
					BucketName: tt.args.bucketName,
				},
			}
			SetLogger(&logger.Logger{Logger: zerolog.Logger{}})
			certificatePEM := &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: []byte("example pem data"),
			}

			// Write the certificate to a file
			// Encode the PEM block to a byte buffer.
			var buf bytes.Buffer
			err := pem.Encode(&buf, certificatePEM)
			if err != nil {
				t.Errorf("failed to encode PEM block: %v\n", err)
			}
			store, err := NewS3Storage(&Config.CloudProvider)
			if err != nil {
				t.Errorf("NewS3Storage() error = %+v, wantErr %+v", err, tt.wantErr)
			}

			err = store.SaveBuf(buf, tt.args.destinationPath)
			if err != nil {
				t.Errorf("store.SaveBuf() err: %+v wantErr: %+v", err, tt.wantErr)
			}

			err = store.GetFile(tt.args.destinationPath, tt.args.destinationPath)
			if err != nil {
				t.Errorf("store.GetFile() err: %+v wantErr: %+v", err, tt.wantErr)
			}

			b, err := os.ReadFile(tt.args.destinationPath)
			if err != nil {
				t.Errorf("os.Open() err: %+v wantErr: %+v", err, tt.wantErr)
			}

			if !bytes.Contains(b, []byte("CERTIFICATE")) {
				t.Errorf("bytes.Contains err: %+v wantErr: %+v", err, tt.wantErr)
			}

		})
	}
}
