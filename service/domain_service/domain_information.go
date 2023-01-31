package domain_service

import (
	"context"
	"errors"
	"fmt"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/rs/zerolog/log"
	"net"
	"sync"
)

var cfg *config.Config
var VerificationMap *sync.Map
var l *logger.Logger

func SetConfig(conf *config.Config) {
	cfg = conf
}

func SetLogger(toSet *logger.Logger) {
	l = toSet
}

var (
	ErrUnableToFindUser    = errors.New("unable to find userId verification map")
	ErrUnableToCast        = errors.New("unable to cast to DomainInformation")
	ErrNoDomainInformation = errors.New("unable to find domain in userId's in verification map")
)

// VerifyOwnership checks the TXT record for our verification string we give people
func VerifyOwnership(ctx context.Context, di models.DomainInformation) (bool, error) {

	txtRecords, err := net.LookupTXT(di.DomainName)
	if err != nil {
		return false, err
	}

	log.Debug().Msgf("txtRecords: %+v", txtRecords)
	log.Debug().Msgf("trying to find: %s;%s;%s", cfg.App.VerificationTxtRecordName, di.DomainName, di.Verification.VerificationKey)
	for _, txt := range txtRecords {
		if txt == fmt.Sprintf("%s;%s;%s", cfg.App.VerificationTxtRecordName, di.DomainName, di.Verification.VerificationKey) {
			log.Info().Msgf("found key: %s", txt)
			return true, nil
		}
		log.Debug().Msgf("record: %s, on %s", txt, di.DomainName)
	}

	return false, nil
}

func contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func VerifyARecord(ctx context.Context, di models.DomainInformation) (bool, error) {

	aRecords, err := net.LookupHost(di.DomainName)
	if err != nil {
		return false, err
	}

	// This might need to become that all A records are pointing at us, which might be the correct thing to do
	for _, record := range aRecords {
		if contains(cfg.Network.OwnedHosts, record) {
			return true, nil
		}
	}

	return false, nil
}

func VerifyCNAME(ctx context.Context, di models.DomainInformation) (bool, error) {

	cname, err := net.LookupCNAME(di.DomainName)
	if err != nil {
		return false, err
	}

	if contains(cfg.Network.OwnedCNames, cname) {
		return true, err
	}

	return false, nil
}

//func DomainInfoByUserId(userId string, domain string) (models.DomainInformation, error) {
//	val, ok := VerificationMap.Load(userId)
//	if !ok {
//		return models.DomainInformation{}, fmt.Errorf("domain: %s, user: %s %w", domain, userId, ErrUnableToFindUser)
//	}
//
//	actualVal, ok := val.(map[string]models.DomainInformation)
//	if !ok {
//		return models.DomainInformation{}, fmt.Errorf("domain: %s, val: %+v %w", domain, val, ErrUnableToCast)
//	}
//
//	domainInfo := actualVal[domain]
//	if domainInfo.DomainName == "" {
//		return models.DomainInformation{}, fmt.Errorf("domain: %s, domainInfo: %+v %w", domain, domainInfo, ErrNoDomainInformation)
//	}
//	return domainInfo, nil
//}
