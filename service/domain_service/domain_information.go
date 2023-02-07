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
	"net"
)

var cfg *config.Config
var l *logger.Logger
var dbStorage datastore.Datastore

func SetDBStorage(toSet datastore.Datastore) {
	dbStorage = toSet
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
	err := dbStorage.PutDomainInfo(info)
	if err != nil {
		return err
	}
	return nil
}

func GetAllRecords() (map[string]map[string]models.DomainInformation, error) {
	records, err := dbStorage.GetAllRecords()
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

func GetUserDomains(userId string) (map[string]map[string]models.DomainInformation, error) {
	domains, err := dbStorage.GetUserDomains(userId)
	if err != nil {
		return nil, fmt.Errorf("ran into issue getting user's domain from dbstorage: %w", err)
	}
	return map[string]map[string]models.DomainInformation{userId: domains}, nil
}

func PutDomain(userId string, domainName string) error {
	return dbStorage.PutDomainInfo(models.DomainInformation{DomainName: domainName, UserId: userId})
}

func DeleteDomain(userId string, domainName string) error {
	return dbStorage.DeleteDomain(userId, domainName)
}

func GenerateOwnershipKey(userId string, domainName string) (string, error) {
	di, err := dbStorage.GetDomainByUser(userId, domainName)
	if err != nil {
		return "", err
	}

	di.Verification.VerificationKey = fmt.Sprintf("%s;%s;%s", cfg.App.VerificationTxtRecordName, di.DomainName, utils.RandomString(30))
	err = dbStorage.PutDomainInfo(di)
	if err != nil {
		return "", err
	}

	return di.Verification.VerificationKey, nil
}

func VerifyTXTRecord(ctx context.Context, verificationZone string, verificationKey string) (bool, error) {
	txtRecords, err := net.LookupTXT(verificationZone)
	if err != nil {
		return false, err
	}

	l.Infof("txtRecords: %+v", txtRecords)
	l.Infof("trying to find: %s", verificationKey)
	for _, txt := range txtRecords {
		if txt == verificationKey {
			l.Infof("found key: %s", txt)
			return true, nil
		}
		l.Infof("record: %s, on %s", txt, verificationZone)
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

	di, err := dbStorage.GetDomainByUser(userId, domainName)
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

	di, err := dbStorage.GetDomainByUser(userId, domainName)

	cname, err := net.LookupCNAME(di.DomainName)
	if err != nil {
		return false, err
	}

	if contains(cfg.Network.OwnedCNames, cname) {
		return true, err
	}

	return false, nil
}
