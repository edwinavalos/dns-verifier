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

type Verification struct {
	DomainName      *url.URL
	VerificationKey string
	Verified        bool
	WarningStamp    time.Time
	ExpireStamp     time.Time
	UserId          uuid.UUID
}

func (v *Verification) VerifyDomain(ctx context.Context) (bool, error) {

	txtRecords, err := net.LookupTXT(v.DomainName.Host)
	if err != nil {
		return false, err
	}

	log.Debug().Msgf("txtRecords: %+v", txtRecords)

	for _, txt := range txtRecords {
		if txt == fmt.Sprintf("%s;%s;%s", SvConfig.App.VerificationTxtRecordName, v.DomainName.Host, v.VerificationKey) {
			log.Info().Msgf("found key: %s", txt)
			return true, nil
		}
		log.Info().Msgf("record: %s, on %s", txt, v.DomainName)
		return false, nil
	}

	return false, nil
}

func (v *Verification) SaveVerification(ctx context.Context) error {
	VerificationMap.Store(v.DomainName.Host, &v)

	err := SaveVerificationFile(ctx, VerificationMap)
	if err != nil {
		return err
	}

	return nil
}

func SaveVerificationFile(ctx context.Context, verifications *sync.Map) error {
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

func GetOrCreateVerificationFile(ctx context.Context) (*sync.Map, error) {

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
		err2 := SaveVerificationFile(ctx, &verifications)
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
	regMap := map[string]Verification{}
	err := json.NewDecoder(output.Body).Decode(&regMap)
	if err != nil {
		return err
	}

	for _, k := range regMap {
		syncMap.Store(k.DomainName.Host, k)
	}

	return nil
}
