package verification_service

import (
	"context"
	models2 "dnsVerifier"
	"dnsVerifier/config"
	"dnsVerifier/models"
	"testing"
	"time"
)

func TestVerifyDomain(t *testing.T) {
	ctx := context.Background()
	type args struct {
		ctx              context.Context
		verificationList []models.VerificationFile
		config           *config.Config
		shouldVerify     bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name: "edwinavalos.com test",
		args: args{
			ctx: ctx,
			verificationList: []models.VerificationFile{{
				Domain:          models2.URL{Host: "edwinavalos.com"},
				VerificationKey: "111122223333",
				Verified:        false,
				WarningStamp:    time.Time{},
				ExpireStamp:     time.Time{},
			}},
			config: &config.Config{
				Aws: config.AWSSettings{},
				App: config.AppSettings{
					VerificationTxtRecordName: "mastodon_ownership_key",
				},
				RootCancel: nil,
				RootCtx:    nil,
			},
			shouldVerify: true,
		},
		wantErr: false,
	},
		{
			name: "edwinavalos.com test",
			args: args{
				ctx: ctx,
				verificationList: []models.VerificationFile{{
					Domain:          models2.URL{Host: "edwinavalos.com"},
					VerificationKey: "333322221111",
					Verified:        false,
					WarningStamp:    time.Time{},
					ExpireStamp:     time.Time{},
				}},
				config: &config.Config{
					Aws: config.AWSSettings{},
					App: config.AppSettings{
						VerificationTxtRecordName: "mastodon_ownership_key",
					},
					RootCancel: nil,
					RootCtx:    nil,
				},
				shouldVerify: false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verified, err := VerifyDomain(tt.args.ctx, tt.args.verificationList, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
			if verified != tt.args.shouldVerify {
				t.Errorf("VerifyDomain() verified = %t, shouldVerify %t", verified, tt.args.shouldVerify)
			}
		})
	}
}
