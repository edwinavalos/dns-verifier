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

func GenerateKey(c *gin.Context) {
	var newKeyRequest generateKeyRequest
	if err := c.BindJSON(&newKeyRequest); err != nil {
		return
	}

	domainName, err := url.Parse(newKeyRequest.DomainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	verification := verification_service.Verification{
		DomainName:      domainName,
		VerificationKey: utils.RandomString(15),
		Verified:        false,
		WarningStamp:    time.Time{},
		ExpireStamp:     time.Now().Add(24 * time.Hour),
		UserId:          newKeyRequest.UserId,
	}

	err = verification.SaveVerification(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	err = verification_service.SaveVerificationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	response := generateKeyResponse{
		VerificationKey: verification.VerificationKey,
		DomainName:      verification.DomainName,
	}
	c.JSON(http.StatusOK, response)

	return
}

func GetVerifications(c *gin.Context) {
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
		return
	}

	err := verification_service.SaveVerificationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func VerifyDomain(c *gin.Context) {
	var newVerifyDomainRequest verifyDomainRequest
	if err := c.BindJSON(&newVerifyDomainRequest); err != nil {
		return
	}

	domainName, err := url.Parse(newVerifyDomainRequest.DomainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	val, ok := verification_service.VerificationMap.Load(domainName.Path)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "could not find requested domainName in database"})
		return
	}
	verification, ok := val.(*verification_service.Verification)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to convert db value to verification"})
		return
	}
	result, err := verification.VerifyDomain(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("unable to verify domain: %s", err)})
		return
	}

	if verification.Verified != result {
		verification.Verified = result
		err := verification.SaveVerification(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	}

	c.JSON(http.StatusOK, verifyDomainResponse{
		DomainName: verification.DomainName.Path,
		Status:     result,
	})

	return
}

func VerifyDomains(c *gin.Context) {
	response := verifyDomainsResponse{}
	verification_service.VerificationMap.Range(func(k interface{}, v interface{}) bool {
		verification, ok := v.(verification_service.Verification)
		if !ok {
			return false
		}
		result, err := verification.VerifyDomain(c)
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

	err := verification_service.SaveVerificationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, response)
	return
}
