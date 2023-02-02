package cert_service

import (
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/datastore"
	"github.com/edwinavalos/dns-verifier/datastore/dynamo"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"testing"
)

func Test_requestCertificate(t *testing.T) {
	testConfig := config.Config{}
	testConfig.LESettings.PrivateKeyLocation = "C:\\mastodon\\private-key.pem"
	testConfig.LESettings.CADirURL = lego.LEDirectoryStaging
	testConfig.DB = config.DatabaseSettings{
		TableName: "dns-verifier-test",
		Region:    "us-east-1",
		IsLocal:   true,
	}
	cfg = &testConfig
	datastore.SetConfig(&testConfig)

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
			if _, _, err := RequestCertificate(tt.args.userId, tt.args.domain, tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("RequestCertificate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_completeCertificateRequest(t *testing.T) {
	testConfig := config.Config{}
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

	privateKey, err := getRequestUserCert()
	if err != nil {
		t.Fatal(err)
	}

	myUser := certRequestUser{
		Email: "admin@amoslabs.cloud",
		key:   privateKey,
	}

	leConfig := lego.NewConfig(&myUser)

	// This CA URL is configured for a local dev instance of Boulder running in Docker in a VM.

	leConfig.CADirURL = cfg.LESettings.CADirURL
	leConfig.Certificate.KeyType = certcrypto.RSA2048

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(leConfig)
	if err != nil {
		t.Fatal(err)
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		t.Fatal(err)
	}
	myUser.Registration = reg

	type args struct {
		domain         string
		client         *lego.Client
		manualProvider *challenge.Provider
		email          string
		userId         string
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
				client: client,
				email:  "admin@amoslabs.cloud",
				userId: uuid.New().String(),
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
			_, _, err = RequestCertificate(tt.args.userId, tt.args.domain, tt.args.email)
			if err != nil {
				t.Errorf("RequestCertificate() err %v, wantErr: %v", err, tt.wantErr)
				return
			}
			_, err = CompleteCertificateRequest(tt.args.userId, tt.args.domain, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompleteCertificateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
