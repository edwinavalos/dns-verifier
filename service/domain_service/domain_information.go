package domain_service

import (
	"context"
	"errors"
	"fmt"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/datastore"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/edwinavalos/dns-verifier/utils"
	"github.com/rs/zerolog/log"
	"net"
)

var cfg *config.Config
var l *logger.Logger
var storage datastore.Datastore

func SetStorage(toSet datastore.Datastore) {
	storage = toSet
}

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

func SaveDomain(info models.DomainInformation) error {
	err := storage.PutDomainInfo(info)
	if err != nil {
		return err
	}
	return nil
}

func GetAllRecords() (map[string]map[string]models.DomainInformation, error) {
	records, err := storage.GetAllRecords()
	if err != nil {
		return nil, err
	}

	retMap := map[string]map[string]models.DomainInformation{}
	for _, v := range records {
		_, ok := retMap[v.UserId]
		if !ok {
			retMap[v.UserId] = map[string]models.DomainInformation{}
		}
		retMap[v.UserId][v.DomainName] = v
	}

	return retMap, nil
}

func PutDomain(userId string, domainName string) error {
	return storage.PutDomainInfo(models.DomainInformation{DomainName: domainName, UserId: userId})
}

func DeleteDomain(userId string, domainName string) error {
	return storage.DeleteDomain(userId, domainName)
}

func GenerateOwnershipKey(userId string, domainName string) (string, error) {
	di, err := storage.GetDomainByUser(userId, domainName)
	if err != nil {
		return "", err
	}

	di.Verification.VerificationKey = fmt.Sprintf("%s;%s;%s", cfg.App.VerificationTxtRecordName, di.DomainName, utils.RandomString(30))
	err = storage.PutDomainInfo(di)
	if err != nil {
		return "", err
	}

	return di.Verification.VerificationKey, nil
}

// VerifyOwnership checks the TXT record for our verification string we give people
func VerifyOwnership(ctx context.Context, userId string, domainName string) (bool, error) {

	di, err := storage.GetDomainByUser(userId, domainName)
	if err != nil {
		return false, err
	}

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

func VerifyARecord(ctx context.Context, userId string, domainName string) (bool, error) {

	di, err := storage.GetDomainByUser(userId, domainName)
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

func VerifyCNAME(ctx context.Context, userId string, domainName string) (bool, error) {

	di, err := storage.GetDomainByUser(userId, domainName)

	cname, err := net.LookupCNAME(di.DomainName)
	if err != nil {
		return false, err
	}

	if contains(cfg.Network.OwnedCNames, cname) {
		return true, err
	}

	return false, nil
}
