package verfication_service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net"
	"net/url"
	"time"
)

type Verification struct {
	DomainName      url.URL
	VerificationKey string
	Verified        bool
	WarningStamp    time.Time
	ExpireStamp     time.Time
	UserId          uuid.UUID
}

func (*Verification) VerifyDomain(ctx context.Context, verificationList []Verification, config *config.Config) (bool, error) {

	for _, item := range verificationList {
		txtRecords, err := net.LookupTXT(item.DomainName.Host)
		if err != nil {
			return false, err
		}

		for _, txt := range txtRecords {
			if txt == fmt.Sprintf("%s;%s;%s", config.App.VerificationTxtRecordName, item.Domain.Host, item.VerificationKey) {
				log.Info().Msgf("found key: %s", txt)
				return true, nil
			}
			log.Info().Msgf("record: %s, on %s", txt, item.Domain.Host)
			return false, nil
		}

	}

	return false, nil
}

func (v *Verification) SaveVerification() {
	return
}
