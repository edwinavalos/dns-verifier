package v1

import (
	"dnsVerifier/service/verification_service"
	"dnsVerifier/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"net/url"
	"time"
)

type generateKeyRequest struct {
	DomainName string    `json:"domain_name"`
	UserId     uuid.UUID `json:"user_id"`
}

type generateKeyResponse struct {
	VerificationKey string   `json:"verification_key"`
	DomainName      *url.URL `json:"domain_name"`
}

type verifyDomainRequest struct {
	DomainName string `json:"domain_name"`
}

type verifyDomainResponse struct {
	DomainName string `json:"domain_name"`
	Status     bool   `json:"status"`
}

type DomainVerificationResult struct {
	DomainName string `json:"domain_name"`
	Status     bool   `json:"status"`
}

type verifyDomainsResponse map[string]DomainVerificationResult

type deleteVerificationRequest struct {
	DomainName string `json:"domain_name"`
}

// GenerateOwnershipKey
// TODO: This behavior of wiping out the verification is probably too stronk, need to make it only create one if
//
//	there isnt other information. StoreOrLoad probably is what I want here.
func GenerateOwnershipKey(c *gin.Context) {

	return
}

func GetDomainInformation(c *gin.Context) {
	c.JSON(http.StatusOK, utils.SyncMap2Map(verification_service.VerificationMap))
}

func DeleteVerification(c *gin.Context) {
	newDeleteVerificationRequest := deleteVerificationRequest{}
	if err := c.BindJSON(&newDeleteVerificationRequest); err != nil {
		return
	}
	_, loaded := verification_service.VerificationMap.LoadAndDelete(newDeleteVerificationRequest.DomainName)
	if !loaded {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain verification was not present"})
	}

	err := verification_service.SaveDomainInformationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{})
}

// VerifyOwnership only does TXT record checks
func VerifyOwnership(c *gin.Context) {
	var newVerifyDomainRequest verifyDomainRequest
	if err := c.BindJSON(&newVerifyDomainRequest); err != nil {
		return
	}

	domainName, err := url.Parse(newVerifyDomainRequest.DomainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}
	val, ok := verification_service.VerificationMap.Load(domainName.Host)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "could not find requested domainName in database"})
	}
	verification, ok := val.(verification_service.DomainInformation)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to convert db value to verification"})
	}
	result, err := verification.VerifyOwnership(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("unable to verify domain: %s", err)})
	}

	if verification.Verified != result {
		verification.Verified = result
		err := verification.SaveDomainInformation(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
	}

	c.JSON(http.StatusOK, verifyDomainResponse{
		DomainName: verification.DomainName.Host,
		Status:     result,
	})

	return
}

// VerifyDomains only does TXT checks for everyone
func VerifyDomains(c *gin.Context) {
	response := verifyDomainsResponse{}
	verification_service.VerificationMap.Range(func(k interface{}, v interface{}) bool {
		verification, ok := v.(verification_service.DomainInformation)
		if !ok {
			return false
		}
		result, err := verification.VerifyOwnership(c)
		if err != nil {
			return false
		}
		key, ok := k.(string)
		if !ok {
			return false
		}
		response[key] = DomainVerificationResult{
			DomainName: key,
			Status:     result,
		}
		verification.Verified = result
		verification_service.VerificationMap.Store(key, verification)
		return true
	})

	err := verification_service.SaveDomainInformationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, response)
	return
}

type createDomainInformationReq struct {
	domainName string    `json:"domain_name"`
	userId     uuid.UUID `json:"user_id"`
}

func CreateDomainInformation(c *gin.Context) {

	var newCreateDomainInformationReq createDomainInformationReq
	err := c.BindJSON(&newCreateDomainInformationReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	domainName, err := url.Parse(newCreateDomainInformationReq.domainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	domainInformation := verification_service.DomainInformation{
		DomainName:      domainName,
		VerificationKey: utils.RandomString(30),
		Verified:        false,
		Delegations: verification_service.Delegations{
			ARecords: nil,
			CNames:   nil,
		},
		WarningStamp: time.Time{},
		ExpireStamp:  time.Now().Add(24 * time.Hour),
		UserId:       newCreateDomainInformationReq.userId,
	}

	_, loaded, err := domainInformation.LoadOrStoreDomainInformation(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}
	if loaded {
		c.JSON(http.StatusAccepted, gin.H{"message": "Accepted and domain already present"})
	}

	err = verification_service.SaveDomainInformationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}

	c.JSON(http.StatusOK, gin.H{})
}
