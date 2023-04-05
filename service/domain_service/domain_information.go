package domain_service

import (
	"context"
	"errors"
	"fmt"
	"github.com/edwinavalos/common/config"
	"github.com/edwinavalos/common/logger"
	"github.com/edwinavalos/common/models"
	"github.com/edwinavalos/dns-verifier/storage"
	"github.com/edwinavalos/dns-verifier/utils"
	"github.com/google/uuid"
	"net"
)

var (
	ErrUnableToFindUser    = errors.New("unable to find userId verification map")
	ErrUnableToCast        = errors.New("unable to cast to DomainInformation")
	ErrNoDomainInformation = errors.New("unable to find domain in userId's in verification map")
)

type Service struct {
	verifierStore *storage.VerifierDataStore
	cfg           *config.Config
}

type ServiceOpt func(s *Service)

func New(conf *config.Config, store *storage.VerifierDataStore, opts ...ServiceOpt) *Service {
	s := &Service{
		verifierStore: store,
		cfg:           conf,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) GetAllRecords(ctx context.Context) (map[string]models.User, error) {
	records, err := s.verifierStore.GetAllRecords(ctx)
	if err != nil {
		return map[string]models.User{}, err
	}

	retMap := map[string]models.User{}
	for _, v := range records {
		retMap[v.ID] = v
	}

	return retMap, nil
}

func (s *Service) GetUserDomains(ctx context.Context, userId uuid.UUID) (map[string]models.DomainInformation, error) {
	return s.verifierStore.GetUserDomains(ctx, userId)
}

func (s *Service) GetDomainByUser(ctx context.Context, userID uuid.UUID, domain string) (models.DomainInformation, error) {
	return s.verifierStore.GetDomainByUser(ctx, userID, domain)
}

func (s *Service) PutDomain(ctx context.Context, domainInfo models.DomainInformation) error {
	return s.verifierStore.PutDomainInfo(ctx, domainInfo)
}

func (s *Service) DeleteDomain(ctx context.Context, userID uuid.UUID, domainName string) error {
	return s.verifierStore.DeleteDomain(ctx, userID, domainName)
}

func (s *Service) GenerateOwnershipKey(ctx context.Context, userID uuid.UUID, domainName string) (string, error) {
	di, err := s.verifierStore.GetDomainByUser(ctx, userID, domainName)
	if err != nil {
		return "", err
	}

	di.Verification.Key = fmt.Sprintf("%s;%s;%s", s.cfg.VerificationTxtRecordName(), di.DomainName, utils.RandomString(30))
	di.Verification.Zone = fmt.Sprintf(domainName + ".")
	err = s.PutDomain(ctx, di)
	if err != nil {
		return "", err
	}

	return di.Verification.Key, nil
}

func (s *Service) VerifyTXTRecord(ctx context.Context, verificationZone string, verificationKey string) (bool, error) {
	txtRecords, err := net.LookupTXT(verificationZone)
	if err != nil {
		return false, err
	}

	logger.Info("txtRecords: %+v", txtRecords)
	logger.Info("trying to find: %s", verificationKey)
	for _, txt := range txtRecords {
		if txt == verificationKey {
			logger.Info("found key: %s", txt)
			return true, nil
		}
		logger.Info("record: %s, on %s", txt, verificationZone)
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

func (s *Service) VerifyARecord(ctx context.Context, userID uuid.UUID, domainName string) (bool, error) {

	di, err := s.verifierStore.GetDomainByUser(ctx, userID, domainName)
	aRecords, err := net.LookupHost(di.DomainName)
	if err != nil {
		return false, err
	}

	// This might need to become that all A records are pointing at us, which might be the correct thing to do
	for _, record := range aRecords {
		if contains(s.cfg.OwnedHosts(), record) {
			return true, nil
		}
	}

	return false, nil
}

func (s *Service) VerifyCNAME(ctx context.Context, userID uuid.UUID, domainName string) (bool, error) {

	di, err := s.verifierStore.GetDomainByUser(ctx, userID, domainName)

	cname, err := net.LookupCNAME(di.DomainName)
	if err != nil {
		return false, err
	}

	if contains(s.cfg.OwnedCNames(), cname) {
		return true, err
	}

	return false, nil
}
