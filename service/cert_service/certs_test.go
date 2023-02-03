package cert_service

import (
	"crypto/x509"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/datastore"
	"github.com/edwinavalos/dns-verifier/datastore/dynamo"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"testing"
	"time"
)

func Test_requestCertificate(t *testing.T) {
	testConfig := config.Config{}
	testConfig.LESettings.PrivateKeyLocation = "C:\\mastodon\\private-key.pem"
	testConfig.LESettings.CADirURL = lego.LEDirectoryStaging
	testConfig.LESettings.KeyAuth = "asufficientlylongenoughstringwithenoughentropy"
	testConfig.DB = config.DatabaseSettings{
		TableName: "dns-verifier-test",
		Region:    "us-east-1",
		IsLocal:   true,
	}
	cfg = &testConfig
	log := logger.Logger{Logger: zerolog.Logger{}}
	datastore.SetConfig(&testConfig)
	datastore.SetLogger(&log)
	SetLogger(&log)

	dbStorage, err := dynamo.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	err = dbStorage.Initialize()
	if err != nil {
		t.Fatal(err)
	}
	storage = dbStorage
	type args struct {
		domain string
		email  string
		userId string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "secondtest.amoslabs.cloud txt record",
			args: args{
				domain: "secondtest.amoslabs.cloud",
				email:  "admin@amoslabs.cloud",
				userId: uuid.New().String(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fqdn, key, err := RequestCertificate(tt.args.userId, tt.args.domain, tt.args.email)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s %s", fqdn, key)
		})
	}
}

func Test_completeCertificateRequest(t *testing.T) {
	testConfig := config.Config{}
	testConfig.LESettings = config.LetsEncryptSettings{
		AdminEmail:         "admin@amoslabs.cloud",
		PrivateKeyLocation: "C:\\mastodon\\private-key.pem",
		KeyAuth:            "asufficientlylongenoughstringwithenoughentropy",
		CADirURL:           lego.LEDirectoryStaging,
	}
	testConfig.DB = config.DatabaseSettings{
		TableName: "dns-verifier-test",
		Region:    "us-east-1",
		IsLocal:   true,
	}
	cfg = &testConfig
	datastore.SetConfig(&testConfig)
	log := logger.Logger{Logger: zerolog.Logger{}}
	SetLogger(&log)
	datastore.SetLogger(&log)

	dbStorage, err := dynamo.NewStorage()
	if err != nil {
		t.Fatal(err)
	}
	err = dbStorage.Initialize()
	if err != nil {
		t.Fatal(err)
	}
	storage = dbStorage

	type args struct {
		domain string
		email  string
		userId string
	}
	tests := []struct {
		name    string
		args    args
		want    *certificate.Resource
		wantErr bool
	}{
		{
			name: "secondtest.amoslabs.cloud",
			args: args{
				domain: "secondtest.amoslabs.cloud",
				email:  "admin@amoslabs.cloud",
				userId: "2c84b63c-9a96-11ed-a8fc-0242ac120002",
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainInformation := models.DomainInformation{
				DomainName: tt.args.domain,
				UserId:     tt.args.userId,
			}
			err := storage.PutDomainInfo(domainInformation)
			if err != nil {
				t.Error(err)
			}
			zone, key, err := RequestCertificate(tt.args.userId, tt.args.domain, tt.args.email)
			if err != nil {
				t.Errorf("RequestCertificate() err %v, wantErr: %v", err, tt.wantErr)
				return
			}
			t.Logf("zone: %s key: %s", zone, key)

			time.Sleep(300 * time.Second)

			ders, err := CompleteCertificateRequest(tt.args.userId, tt.args.domain, tt.args.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompleteCertificateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, der := range ders {
				cert, err := x509.ParseCertificate(der)
				if err != nil {
					return
				}
				t.Logf("Certificate: \n%s", string(cert.Raw))
			}
		})
	}
}
