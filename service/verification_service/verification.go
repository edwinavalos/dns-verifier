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
	"net/url"
	"os"
	"sync"

	"time"
)

var SvConfig *config.Config
var VerificationMap *sync.Map

type Delegations struct {
	ARecords []string
	CNames   []string
}
type DomainInformation struct {
	DomainName      *url.URL
	VerificationKey string
	Verified        bool
	Delegations     Delegations
	WarningStamp    time.Time
	ExpireStamp     time.Time
	UserId          uuid.UUID
}

// VerifyOwnership checks the TXT record for our verification string we give people
func (di *DomainInformation) VerifyOwnership(ctx context.Context) (bool, error) {

	txtRecords, err := net.LookupTXT(di.DomainName.Host)
	if err != nil {
		return false, err
	}

	log.Debug().Msgf("txtRecords: %+v", txtRecords)

	for _, txt := range txtRecords {
		if txt == fmt.Sprintf("%s;%s;%s", SvConfig.App.VerificationTxtRecordName, di.DomainName.Host, di.VerificationKey) {
			log.Info().Msgf("found key: %s", txt)
			return true, nil
		}
		log.Debug().Msgf("record: %s, on %s", txt, di.DomainName)
		return false, nil
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

	aRecords, err := net.LookupHost(di.DomainName.Host)
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

	cname, err := net.LookupCNAME(di.DomainName.Host)
	if err != nil {
		return false, err
	}

	if contains(SvConfig.Network.OwnedHosts, cname) {
		return true, err
	}

	return false, nil
}

func (di *DomainInformation) LoadOrStoreDomainInformation(ctx context.Context) (*DomainInformation, bool, error) {
	actual, loaded := VerificationMap.LoadOrStore(di.DomainName.Host, &di)
	if !loaded {
		return di, false, nil
	}

	actualVal, ok := actual.(*DomainInformation)
	if !ok {
		return nil, false, fmt.Errorf("unable to cast stored value to DomainInformation")
	}

	return actualVal, true, nil
}

func (di *DomainInformation) SaveDomainInformation(ctx context.Context) error {
	VerificationMap.Store(di.DomainName.Host, &di)

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
	regMap := map[string]DomainInformation{}
	err := json.NewDecoder(output.Body).Decode(&regMap)
	if err != nil {
		return err
	}

	for _, k := range regMap {
		syncMap.Store(k.DomainName.Host, k)
	}

	return nil
}
