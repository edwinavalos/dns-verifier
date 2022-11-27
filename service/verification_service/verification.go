package verification_service

import (
	"context"
	"dnsVerifier/config"
	"dnsVerifier/utils"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net"
	"net/url"
	"sync"

	"time"
)

var SvConfig *config.Config
var VerificationMap *sync.Map

type Verification struct {
	DomainName      *url.URL
	VerificationKey string
	Verified        bool
	WarningStamp    time.Time
	ExpireStamp     time.Time
	UserId          uuid.UUID
}

func (*Verification) VerifyDomain(ctx context.Context, verificationList []Verification) (bool, error) {

	for _, item := range verificationList {
		txtRecords, err := net.LookupTXT(item.DomainName.Host)
		if err != nil {
			return false, err
		}

		for _, txt := range txtRecords {
			if txt == fmt.Sprintf("%s;%s;%s", SvConfig.App.VerificationTxtRecordName, item.DomainName, item.VerificationKey) {
				log.Info().Msgf("found key: %s", txt)
				return true, nil
			}
			log.Info().Msgf("record: %s, on %s", txt, item.DomainName)
			return false, nil
		}

	}

	return false, nil
}

func (v *Verification) SaveVerification(ctx context.Context) error {
	fmt.Printf("%+v", utils.SyncMap2Map(VerificationMap))
	VerificationMap.Store(v.DomainName.Host, &v)

	err := utils.SaveVerificationFile(ctx, VerificationMap, SvConfig)
	if err != nil {
		return err
	}

	return nil
}
