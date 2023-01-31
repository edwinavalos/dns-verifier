package routers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/models"
	v1 "github.com/edwinavalos/dns-verifier/routers/api/v1"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestAllOfIt this is an ugly happy path test through registration to the website
func TestAllOfIt(t *testing.T) {

	// Configure the application
	domainName := "test.edwinavalos.com"

	viper.SetConfigName("config-test")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./../resources")
	err := viper.ReadInConfig()
	if err != nil {
		t.Fatal("unable to read configuration file, exiting.")
	}
	ctx := context.Background()

	appConfig := config.NewConfig()
	appConfig.RootCtx = ctx
	appConfig.ReadConfig()

	if appConfig.Aws.BucketName == "" || appConfig.Aws.VerificationFileName == "" {
		log.Fatal().Msgf("did not have enough information to get or create domain_service file")
		log.Debug().Msgf("bucketName: {%s}, verificationFileName {%s}", appConfig.Aws.BucketName, appConfig.Aws.VerificationFileName)
		panic(fmt.Errorf("missing aws configuration"))
	}

	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(appConfig.Aws.Region))
	if err != nil {
		log.Panic().Msg("unable to load default aws appConfig")
		panic(err)
	}

	awsS3Client := s3.NewFromConfig(cfg)
	appConfig.Aws.S3Client = awsS3Client
	domain_service.svConfig = appConfig
	domain_service.VerificationMap = &sync.Map{}

	// Create our router
	r := InitRouter()

	// First we create a new domain in our service
	w := httptest.NewRecorder()
	newDomainRequest := v1.CreateDomainInformationReq{
		DomainName: domainName,
		UserId:     uuid.New(),
	}
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(newDomainRequest)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("POST", "/api/v1/domain", &buf)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)

	// Then we generate an ownership key for verification
	buf = bytes.Buffer{}
	w = httptest.NewRecorder()
	newGenerateOwnershipKey := v1.GenerateOwnershipKeyReq{DomainName: domainName}
	err = json.NewEncoder(&buf).Encode(newGenerateOwnershipKey)
	req, err = http.NewRequest("POST", "/api/v1/domain/verificationKey", &buf)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)

	// Now we need to change the domain information we just wrote to be one that we can verify with our
	// edwinavalos.com domain
	di := models.DomainInformation{DomainName: domainName}
	diToUpdate, err := di.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}

	diToUpdate.Verification.VerificationKey = "111122223333"
	diToUpdate.Delegations.ARecords = []string{"34.217.225.52"}
	diToUpdate.Delegations.CNames = []string{"spons.us"}
	// Save it so the services will have access to it
	err = diToUpdate.SaveDomainInformation(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the domains various records
	buf = bytes.Buffer{}
	w = httptest.NewRecorder()

	// Verify that we own the domain

	req, err = http.NewRequest("POST", fmt.Sprintf("/api/v1/domain/verification?domain_name=%s", domainName), &buf)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)

	buf = bytes.Buffer{}
	w = httptest.NewRecorder()

	// Verify that our ARecord points to a service node
	delegationReq1 := v1.VerifyDelegationReq{
		DomainName: domainName,
		Type:       v1.ARecord,
	}

	err = json.NewEncoder(&buf).Encode(delegationReq1)

	req, err = http.NewRequest("POST", "/api/v1/domain/verification", &buf)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusOK)
	// need to write one that does CNames with another name loaded in to the di object

}
