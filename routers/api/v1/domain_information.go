package v1

import (
	"dnsVerifier/service/verification_service"
	"dnsVerifier/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"net/url"
)

type generateOwnershipKeyRequest struct {
	DomainName string `json:"domain_name"`
}

type generateOwnershipKeyResponse struct {
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
//	there isn't other information. StoreOrLoad probably is what I want here.
func GenerateOwnershipKey(c *gin.Context) {
	newGenerateOwnershipKeyReq := generateOwnershipKeyRequest{}
	err := c.BindJSON(&newGenerateOwnershipKeyReq)
	if err != nil {
		return
	}

	di := verification_service.DomainInformation{DomainName: newGenerateOwnershipKeyReq.DomainName}

	loadedDi, err := di.Load(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("unable to find domain name: %s in verification map err was: %s", newGenerateOwnershipKeyReq.DomainName, err)})
		return
	}

	loadedDi.Verification.VerificationKey = utils.RandomString(30)

	err = loadedDi.SaveDomainInformation(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

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
		return
	}

	err := verification_service.SaveDomainInformationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
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
		return
	}

	val, ok := verification_service.VerificationMap.Load(domainName.Path)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "could not find requested domainName in database"})
		return
	}
	verification, ok := val.(*verification_service.DomainInformation)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to convert db value to verification"})
		return
	}
	result, err := verification.VerifyOwnership(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("unable to verify domain: %s", err)})
		return
	}

	if verification.Verification.Verified != result {
		verification.Verification.Verified = result
		err := verification.SaveDomainInformation(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	}

	c.JSON(http.StatusOK, verifyDomainResponse{
		DomainName: verification.DomainName,
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
		verification.Verification.Verified = result
		verification_service.VerificationMap.Store(key, verification)
		return true
	})

	err := verification_service.SaveDomainInformationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, response)
	return
}

type createDomainInformationReq struct {
	DomainName string    `json:"domain_name"`
	UserId     uuid.UUID `json:"user_id"`
}

// CreateDomainInformation will create domain information if not present in VerificationMap
// if it is present, we return 202 and move on with our lives
func CreateDomainInformation(c *gin.Context) {

	newCreateDomainInformationReq := createDomainInformationReq{}
	err := c.BindJSON(&newCreateDomainInformationReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	domainInformation := verification_service.DomainInformation{
		DomainName: newCreateDomainInformationReq.DomainName,
	}

	loadedDi, loaded, err := domainInformation.LoadOrStore(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	if loaded {
		c.JSON(http.StatusAccepted, gin.H{"message": "Accepted and domain already present"})
		return
	}

	err = loadedDi.SaveDomainInformation(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

type deleteDomainInformationReq struct {
	DomainName string `json:"domain_name"`
}

func DeleteDomainInformation(c *gin.Context) {
	var newDeleteDomainInformationReq deleteDomainInformationReq
	if err := c.Bind(&newDeleteDomainInformationReq); err != nil {
		return
	}

	di := verification_service.DomainInformation{DomainName: newDeleteDomainInformationReq.DomainName}

	loaded, err := di.LoadAndDelete(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	if !loaded {
		c.JSON(http.StatusNotFound, gin.H{"error": "value was not in verification map, check name?"})
		return
	}

	err = verification_service.SaveDomainInformationFile(c, verification_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s deleted", di.DomainName)})
	return
}
