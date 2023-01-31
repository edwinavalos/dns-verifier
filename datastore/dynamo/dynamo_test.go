package dynamo

import (
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/datastore"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"testing"
	"time"
)

func TestLocalDynamoStorage(t *testing.T) {
	cfg := config.Config{
		DB: config.DatabaseSettings{
			TableName: "dns-verifier",
			Region:    "us-east-1",
			IsLocal:   true,
		},
	}
	datastore.Log = &logger.Logger{
		Logger: zerolog.Logger{},
	}

	datastore.Config = &cfg
	tests := []struct {
		name    string
		want    datastore.Datastore
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
			storage, err := NewStorage()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			err = storage.Initialize()
			if err != nil {
				t.Errorf("storage.Initialize() err = %v, wantErr %v", err, tt.wantErr)
				return
			}

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
				UserId: uuid.New().String(),
			}

			err = storage.PutDomainInfo(info)
			if err != nil {
				t.Errorf("storage.PutDomainInfo() err %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}
