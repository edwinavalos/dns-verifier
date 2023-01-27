package certs

import (
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"reflect"
	"testing"
)

func Test_requestCertificate(t *testing.T) {
	testConfig := config.Config{}
	testConfig.LESettings.PrivateKeyLocation = "C:\\mastodon\\private-key.pem"
	testConfig.LESettings.CADirURL = lego.LEDirectoryStaging
	type args struct {
		domain string
		email  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test.amoslabs.cloud txt record",
			args: args{
				domain: "test.amoslabs.cloud",
				email:  "admin@amoslabs.cloud",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg = &testConfig
			if err := requestCertificate(tt.args.domain, tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("requestCertificate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_completeCertificateRequest(t *testing.T) {

	testConfig := config.Config{}
	testConfig.LESettings.PrivateKeyLocation = "C:\\mastodon\\private-key.pem"
	testConfig.LESettings.CADirURL = lego.LEDirectoryStaging
	cfg = &testConfig
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
	}
	tests := []struct {
		name    string
		args    args
		want    *certificate.Resource
		wantErr bool
	}{
		{
			name: "test.amoslabs.cloud",
			args: args{
				domain: "test.amoslabs.cloud",
				client: client,
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := completeCertificateRequest(tt.args.domain, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("completeCertificateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("completeCertificateRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
