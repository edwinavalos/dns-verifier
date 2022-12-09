package verification_service

import (
	"context"
	"dnsVerifier/config"
	"dnsVerifier/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"sync"

	"time"
)

var SvConfig *config.Config
var VerificationMap *sync.Map

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
	Verified                 bool
	VerificationWarningStamp time.Time
	VerificationExpireStamp  time.Time
}

type DomainInformation struct {
	DomainName   string
	Verification Verification
	Delegations  Delegations
	UserId       uuid.UUID
}

// VerifyOwnership checks the TXT record for our verification string we give people
func (di *DomainInformation) VerifyOwnership(ctx context.Context) (bool, error) {

	txtRecords, err := net.LookupTXT(di.DomainName)
	if err != nil {
		return false, err
	}

	log.Debug().Msgf("txtRecords: %+v", txtRecords)
	log.Debug().Msgf("trying to find: %s;%s;%s", SvConfig.App.VerificationTxtRecordName, di.DomainName, di.Verification.VerificationKey)
	for _, txt := range txtRecords {
		if txt == fmt.Sprintf("%s;%s;%s", SvConfig.App.VerificationTxtRecordName, di.DomainName, di.Verification.VerificationKey) {
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
		if contains(SvConfig.Network.OwnedHosts, record) {
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

	if contains(SvConfig.Network.OwnedHosts, cname) {
		return true, err
	}

	return false, nil
}

func (di *DomainInformation) LoadOrStore(ctx context.Context) (*DomainInformation, bool, error) {
	value, loaded := VerificationMap.LoadOrStore(di.DomainName, di)
	if !loaded {
		return di, false, nil
	}

	actualValue, ok := value.(*DomainInformation)
	if !ok {
		return nil, false, fmt.Errorf("unable to cast stored value to DomainInformation")
	}

	return actualValue, true, nil
}

func (di *DomainInformation) Load(ctx context.Context) (*DomainInformation, error) {
	value, ok := VerificationMap.Load(di.DomainName)
	if !ok {
		return di, fmt.Errorf("unable to find %s in verification map", di.DomainName)
	}

	actualValue, ok := value.(*DomainInformation)
	if !ok {
		return nil, fmt.Errorf("unable to convert map value of key: %s to DomainInformation", di.DomainName)
	}

	return actualValue, nil
}

func (di *DomainInformation) LoadAndDelete(ctx context.Context) (bool, error) {
	_, loaded := VerificationMap.LoadAndDelete(di.DomainName)
	if !loaded {
		return false, nil
	}
	return true, nil
}

func (di *DomainInformation) SaveDomainInformation(ctx context.Context) error {
	VerificationMap.Store(di.DomainName, di)

	err := SaveDomainInformationFile(ctx, VerificationMap)
	if err != nil {
		return err
	}

	return nil
}

func SaveDomainInformationFile(ctx context.Context, verifications *sync.Map) error {
	verificationFileName := SvConfig.Aws.VerificationFileName
	log.Debug().Msgf("creating verification file at s3://%s/%s", SvConfig.Aws.BucketName, SvConfig.Aws.VerificationFileName)
	jsonMap := utils.SyncMap2Map(verifications)
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
	_, err = SvConfig.Aws.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &SvConfig.Aws.BucketName,
		Key:           &SvConfig.Aws.VerificationFileName,
		Body:          file,
		ContentLength: stat.Size(),
	})
	if err != nil {
		log.Error().Msgf("unable to create file at s3://%s/%s", SvConfig.Aws.BucketName, SvConfig.Aws.VerificationFileName)
		return err
	}

	return nil
}

func GetOrCreateDomainInformationFile(ctx context.Context) (*sync.Map, error) {
	var verifications sync.Map
	createFile := false
	getObjectOutput, err := SvConfig.Aws.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &SvConfig.Aws.BucketName,
		Key:    &SvConfig.Aws.VerificationFileName,
	})
	if err != nil {
		var nske *types.NoSuchKey
		if errors.As(err, &nske) {
			log.Info().Msgf("Did not find key, creating file... s3://%s/%s", SvConfig.Aws.BucketName, SvConfig.Aws.VerificationFileName)
			createFile = true
		}
		var nsb *types.NoSuchBucket
		if errors.As(err, &nsb) {
			log.Error().Msgf("bucket: %s does not exit... exiting", SvConfig.Aws.BucketName)
			return &verifications, fmt.Errorf("bucket does not exist")
		}
	}
	if createFile || SvConfig.App.AlwaysRecreate {
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
	regMap := map[string]*DomainInformation{}
	err := json.NewDecoder(output.Body).Decode(&regMap)
	if err != nil {
		return err
	}

	for _, k := range regMap {
		syncMap.Store(k.DomainName, k)
	}

	return nil
}
