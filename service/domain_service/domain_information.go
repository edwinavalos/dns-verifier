package domain_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"sync"

	"time"
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

type Delegations struct {
	ARecords            []string
	ARecordWarningStamp time.Time
	ARecordExpireStamp  time.Time
	CNames              []string
	CNameWarningStamp   time.Time
	CNameExpireStamp    time.Time
}

type Verification struct {
	VerificationKey          string
	VerificationZone         string
	Verified                 bool
	VerificationWarningStamp time.Time
	VerificationExpireStamp  time.Time
}

type DomainInformation struct {
	DomainName     string
	Verification   Verification
	LEVerification Verification
	Delegations    Delegations
	UserId         string
}

// VerifyOwnership checks the TXT record for our verification string we give people
func (di *DomainInformation) VerifyOwnership(ctx context.Context) (bool, error) {

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

func (di *DomainInformation) VerifyARecord(ctx context.Context) (bool, error) {

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

func (di *DomainInformation) VerifyCNAME(ctx context.Context) (bool, error) {

	cname, err := net.LookupCNAME(di.DomainName)
	if err != nil {
		return false, err
	}

	if contains(cfg.Network.OwnedCnames, cname) {
		return true, err
	}

	return false, nil
}

func (di *DomainInformation) LoadOrStore(ctx context.Context) (map[string]DomainInformation, bool, error) {
	var user map[string]DomainInformation
	val, ok := VerificationMap.Load(di.UserId)
	if !ok {
		// We didn't load it, so we create a new map and stick it into the sync.Map
		newMap := map[string]DomainInformation{di.DomainName: *di}
		VerificationMap.Store(di.UserId, newMap)
		return user, false, nil
	}

	user, ok = val.(map[string]DomainInformation)
	if !ok {
		return user, false, fmt.Errorf("ran into error casting value into map[string]DomainInformation")
	}

	_, ok = user[di.DomainName]
	if !ok {
		user[di.DomainName] = *di
		return user, false, nil
	}

	return user, true, nil
}

func (di *DomainInformation) Load(ctx context.Context) (*DomainInformation, error) {
	value, ok := VerificationMap.Load(di.UserId)
	if !ok {
		return di, fmt.Errorf("unable to find %s in verification map", di.DomainName)
	}

	actualValue, ok := value.(map[string]DomainInformation)
	if !ok {
		return nil, fmt.Errorf("unable to convert map value of key: %s to DomainInformation", di.DomainName)
	}

	information := actualValue[di.DomainName]
	return &information, nil
}

func (di *DomainInformation) LoadAndDelete(ctx context.Context) (bool, error) {
	val, ok := VerificationMap.Load(di.UserId)
	if !ok {
		return false, fmt.Errorf("unable to load domain information to delete it")
	}
	actualVal, ok := val.(map[string]DomainInformation)
	if !ok {
		return false, fmt.Errorf("unable to cast to map[string]DomainInformation to delete it")
	}

	delete(actualVal, di.DomainName)

	if len(actualVal) == 0 {
		VerificationMap.Delete(di.UserId)
	}
	return true, nil
}

func (di *DomainInformation) SaveDomainInformation(ctx context.Context) error {
	val, ok := VerificationMap.Load(di.UserId)
	if !ok {
		l.Infof("did not load userId: %s", di.UserId)
		newMap := map[string]DomainInformation{di.DomainName: *di}
		VerificationMap.Store(di.UserId, newMap)
		return nil
	}
	actualVal, ok := val.(map[string]DomainInformation)
	if !ok {
		l.Infof("did not load domain: %s", di.DomainName)
		newMap := map[string]DomainInformation{di.DomainName: *di}
		VerificationMap.Store(di.UserId, newMap)
		return nil
	}
	_, ok = actualVal[di.DomainName]
	if !ok {
		actualVal = make(map[string]DomainInformation)
	}
	actualVal[di.DomainName] = *di
	VerificationMap.Store(di.UserId, actualVal)

	err := SaveDomainInformationFile(ctx, VerificationMap)
	if err != nil {
		return err
	}

	return nil
}

var (
	ErrUnableToFindUser    = errors.New("unable to find userId verification map")
	ErrUnableToCast        = errors.New("unable to cast to DomainInformation")
	ErrNoDomainInformation = errors.New("unable to find domain in userId's in verification map")
)

func DomainInfoByUserId(userId string, domain string) (DomainInformation, error) {
	val, ok := VerificationMap.Load(userId)
	if !ok {
		return DomainInformation{}, fmt.Errorf("domain: %s, user: %s %w", domain, userId, ErrUnableToFindUser)
	}

	actualVal, ok := val.(map[string]DomainInformation)
	if !ok {
		return DomainInformation{}, fmt.Errorf("domain: %s, val: %+v %w", domain, val, ErrUnableToCast)
	}

	domainInfo := actualVal[domain]
	if domainInfo.DomainName == "" {
		return DomainInformation{}, fmt.Errorf("domain: %s, domainInfo: %+v %w", domain, domainInfo, ErrNoDomainInformation)
	}
	return domainInfo, nil
}

func SaveDomainInformationFile(ctx context.Context, verifications *sync.Map) error {
	verificationFileName := cfg.Aws.VerificationFileName
	log.Debug().Msgf("creating verification file at s3://%s/%s", cfg.Aws.BucketName, cfg.Aws.VerificationFileName)
	jsonMap := SyncMap2Map(verifications)
	content, _ := json.MarshalIndent(jsonMap, "", " ")
	err := os.WriteFile(verificationFileName, content, 0644)
	if err != nil {
		return err
	}
	file, err := os.Open(verificationFileName)
	if err != nil {
		return err
	}
	stat, err := os.Stat(verificationFileName)
	if err != nil {
		return err
	}
	_, err = cfg.Aws.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &cfg.Aws.BucketName,
		Key:           &cfg.Aws.VerificationFileName,
		Body:          file,
		ContentLength: stat.Size(),
	})
	if err != nil {
		log.Error().Msgf("unable to create file at s3://%s/%s", cfg.Aws.BucketName, cfg.Aws.VerificationFileName)
		return err
	}

	return nil
}

func GetOrCreateDomainInformationFile(ctx context.Context) (*sync.Map, error) {
	var verifications sync.Map
	createFile := false
	getObjectOutput, err := cfg.Aws.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &cfg.Aws.BucketName,
		Key:    &cfg.Aws.VerificationFileName,
	})
	if err != nil {
		var nske *types.NoSuchKey
		if errors.As(err, &nske) {
			log.Info().Msgf("Did not find key, creating file... s3://%s/%s", cfg.Aws.BucketName, cfg.Aws.VerificationFileName)
			createFile = true
		}
		var nsb *types.NoSuchBucket
		if errors.As(err, &nsb) {
			log.Error().Msgf("bucket: %s does not exit... exiting", cfg.Aws.BucketName)
			return &verifications, fmt.Errorf("bucket does not exist")
		}
	}
	if createFile || cfg.App.AlwaysRecreate {
		err2 := SaveDomainInformationFile(ctx, &verifications)
		if err2 != nil {
			return &verifications, err2
		}
		return &verifications, nil
	}

	if getObjectOutput.ContentLength != 0 {
		err = PopulateVerifications(&verifications, getObjectOutput)
		if err != nil {
			return &verifications, nil
		}
	}

	return &verifications, nil

}

func PopulateVerifications(syncMap *sync.Map, output *s3.GetObjectOutput) error {
	// Refactoring note: This will be map[userId]map[url]domainInformation
	regMap := map[string]map[string]DomainInformation{}
	err := json.NewDecoder(output.Body).Decode(&regMap)
	if err != nil {
		return err
	}

	for userId := range regMap {
		val, _ := syncMap.LoadOrStore(userId, map[string]DomainInformation{})
		actualVal, _ := val.(map[string]DomainInformation)
		for domainName := range regMap[userId] {
			actualVal[domainName] = regMap[userId][domainName]
			syncMap.Store(userId, actualVal)
		}
	}

	return nil
}

func SyncMap2Map(syncMap *sync.Map) map[string]interface{} {
	regMap := make(map[string]interface{})
	if syncMap != nil {
		syncMap.Range(func(k interface{}, v interface{}) bool {
			usersDomains, ok := v.(map[string]DomainInformation)
			if !ok {
				fmt.Print("err: unable to cast to DomainInformation")
				return false
			}
			fmt.Printf("%+v", usersDomains)
			for key, val := range usersDomains {
				regMap[k.(string)] = map[string]DomainInformation{key: val}
			}
			return true
		})
	}
	return regMap
}
