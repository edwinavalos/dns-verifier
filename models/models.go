package models

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/edwinavalos/dns-verifier/config"
	"time"
)

var cfg *config.Config

func SetConfig(toSet *config.Config) {
	cfg = toSet
}

type Delegations struct {
	ARecords            []string  `dynamodbav:"a_records"`
	ARecordWarningStamp time.Time `dynamodbav:"a_record_warning_stamp"`
	ARecordExpireStamp  time.Time `dynamodbav:"a_record_expire_stamp"`
	CNames              []string  `dynamodbav:"c_names"`
	CNameWarningStamp   time.Time `dynamodbav:"c_name_warning_stamp"`
	CNameExpireStamp    time.Time `dynamodbav:"c_name_expire_stamp"`
}

type Verification struct {
	VerificationKey          string    `dynamodbav:"verification_key"`
	VerificationZone         string    `dynamodbav:"verification_zone"`
	Verified                 bool      `dynamodbav:"verified"`
	VerificationWarningStamp time.Time `dynamodbav:"verification_warning_stamp"`
	VerificationExpireStamp  time.Time `dynamodbav:"verification_expire_stamp"`
}

type DomainInformation struct {
	DomainName     string       `dynamodbav:"domain_name"`
	Verification   Verification `dynamodbav:"verification"`
	LEVerification Verification `dynamodbav:"le_verification"`
	Delegations    Delegations  `dynamodbav:"delegations"`
	UserId         string       `dynamodbav:"user_id"`
}

func (domainInfo *DomainInformation) GetKey() (map[string]types.AttributeValue, error) {
	userId, err := attributevalue.Marshal(domainInfo.UserId)
	if err != nil {
		return nil, err
	}

	domainName, err := attributevalue.Marshal(domainInfo.DomainName)
	if err != nil {
		return nil, err
	}
	return map[string]types.AttributeValue{"user_id": userId, "domain_name": domainName}, nil
}
