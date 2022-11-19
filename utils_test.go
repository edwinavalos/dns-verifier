package main

import (
	"context"
	"net/url"
	"testing"
)

func TestVerifyDomain(t *testing.T) {
	ctx := context.Background()
	type args struct {
		ctx              context.Context
		verificationList []VerificationItem
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		name: "edwinavalos.com ttest"
		args: args{
		ctx: ctx,
		verificationList: []VerificationItem{{
			Domain: url.URL{
				Host: "edwinavalos.com",
			},
			VerificationKey: "11111111",
			Verified:        false,
			},
			},
		},}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifyDomain(tt.args.ctx, tt.args.verificationList); (err != nil) != tt.wantErr {
				t.Errorf("VerifyDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
