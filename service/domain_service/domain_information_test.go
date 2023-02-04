package domain_service

//
//func TestVerifyDomain(t *testing.T) {
//	edwinavalosDomainName := "edwinavalos.com"
//
//	ctx := context.Background()
//	type args struct {
//		ctx          context.Context
//		verification DomainInformation
//		config       *config.Config
//		shouldVerify bool
//	}
//	tests := []struct {
//		name    string
//		args    args
//		wantErr bool
//	}{{
//		name: "edwinavalos.com should verify on test verification record",
//		args: args{
//			ctx: ctx,
//			verification: DomainInformation{
//				DomainName: edwinavalosDomainName,
//				Verification: Verification{
//					VerificationKey:          "111122223333",
//					Verified:                 false,
//					VerificationWarningStamp: time.Time{},
//					VerificationExpireStamp:  time.Time{},
//				},
//				UserId: uuid.UUID{},
//			},
//			config: &config.Config{
//				CloudProvider: config.AWSSettings{},
//				App: config.AppSettings{
//					VerificationTxtRecordName: "mastodon_ownership_key_test",
//				},
//				RootCancel: nil,
//				RootCtx:    nil,
//			},
//			shouldVerify: true,
//		},
//		wantErr: false,
//	},
//		{
//			name: "edwinavalos.com should fail to verify for wrong verification key",
//			args: args{
//				ctx: nil,
//				verification: DomainInformation{
//					DomainName: edwinavalosDomainName,
//					Verification: Verification{
//						VerificationKey:          "333322221111",
//						Verified:                 false,
//						VerificationWarningStamp: time.Time{},
//						VerificationExpireStamp:  time.Time{},
//					},
//					UserId: uuid.UUID{},
//				},
//				config: &config.Config{
//					CloudProvider: config.AWSSettings{},
//					App: config.AppSettings{
//						VerificationTxtRecordName: "mastodon_ownership_key_test",
//					},
//					RootCancel: nil,
//					RootCtx:    nil,
//				},
//				shouldVerify: false,
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			cfg = tt.args.config
//			verified, err := tt.args.verification.VerifyOwnership(tt.args.ctx)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("VerifyOwnership() error = %v, wantErr %v", err, tt.wantErr)
//			}
//			if verified != tt.args.shouldVerify {
//				t.Errorf("VerifyOwnership() verified = %t, shouldVerify %t", verified, tt.args.shouldVerify)
//			}
//		})
//	}
//}
