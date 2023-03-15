package datastore

import (
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"testing"
	"time"
)

func resetStorage(storage Datastore) error {
	err := storage.DropTable()
	if err != nil {
		return err
	}

	return nil
}

func TestLocalDynamoStorage(t *testing.T) {
	cfg := config.config{
		DB: config.DatabaseSettings{
			TableName: "dns-verifier",
			Region:    "us-east-1",
			IsLocal:   true,
		},
	}
	Log = &logger.Logger{
		Logger: zerolog.Logger{},
	}

	Config = &cfg
	tests := []struct {
		name    string
		want    Datastore
		wantErr bool
	}{
		{
			name:    "can create initialize a local table",
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewStorage(cfg.DB)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			err = resetStorage(storage)
			if err != nil {
				t.Fatalf("Unable to reset storage err was: %s", err)
			}
			err = storage.Initialize()
			if err != nil {
				t.Errorf("storage.Initialize() err = %v, wantErr %v", err, tt.wantErr)
				return
			}
			userId := uuid.New().String()
			info := models.DomainInformation{
				DomainName: "test.edwinavalos.com",
				Verification: models.Verification{
					VerificationKey:          "thisisaverificationkey",
					VerificationZone:         "thisisthetxtrecordname",
					Verified:                 true,
					VerificationWarningStamp: time.Now(),
					VerificationExpireStamp:  time.Now(),
				},
				LEVerification: models.Verification{
					VerificationKey:          "thisistheotherverificationkey",
					VerificationZone:         "thisistheothertxtrecordname",
					Verified:                 false,
					VerificationWarningStamp: time.Now(),
					VerificationExpireStamp:  time.Now(),
				},
				Delegations: models.Delegations{
					ARecords:            []string{"test.edwinavalos.com"},
					ARecordWarningStamp: time.Now(),
					ARecordExpireStamp:  time.Now(),
					CNames:              []string{"cnametest.edwinavalos.com"},
					CNameWarningStamp:   time.Now(),
					CNameExpireStamp:    time.Now(),
				},
				UserId: userId,
			}

			err = storage.PutDomainInfo(info)
			if err != nil {
				t.Errorf("storage.PutDomainInfo() err %v, wantErr %v", err, tt.wantErr)
			}

			domainInfo, err := storage.GetDomainByUser(userId, "test.edwinavalos.com")
			if err != nil {
				t.Errorf("storage.GetDomainByUser() err %v, wantErr %v", err, tt.wantErr)
			}

			t.Logf("DomainInfo: %+v", domainInfo)

			userDomains, err := storage.GetUserDomains(userId)
			if err != nil {
				t.Errorf("storage.GetUserDomains() err %v, wantErr %v", err, tt.wantErr)
			}

			t.Logf("UserDomains: %+v", userDomains)

			err = storage.DeleteDomain(userId, "test.edwinavalos.com")
			if err != nil {
				t.Errorf("storage.DeletDomain() err %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}
