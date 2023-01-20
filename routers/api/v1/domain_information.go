package v1

import (
	"fmt"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/edwinavalos/dns-verifier/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"net/url"
)

type GenerateOwnershipKeyReq struct {
	DomainName string `json:"domain_name"`
}

type GenerateOwnershipKeyResp struct {
	VerificationKey string   `json:"verification_key"`
	DomainName      *url.URL `json:"domain_name"`
	Error           string   `json:"error,omitempty"`
}

type VerifyDomainReq struct {
	DomainName string `json:"domain_name"`
}

type VerifyDomainResp struct {
	DomainName string `json:"domain_name"`
	Status     bool   `json:"status"`
	Error      string `json:"error,omitempty"`
}

type DomainVerificationResp struct {
	DomainName string `json:"domain_name"`
	Status     bool   `json:"status"`
	Error      string `json:"error,omitempty"`
}

type VerifyDomainsResp map[string]DomainVerificationResp

type DeleteVerificationReq struct {
	DomainName string `json:"domain_name"`
}

type CreateDomainInformationReq struct {
	DomainName string    `json:"domain_name"`
	UserId     uuid.UUID `json:"user_id"`
}

type DeleteDomainInformationReq struct {
	DomainName string `json:"domain_name"`
}

func GetDomainInformation(c *gin.Context) {
	c.JSON(http.StatusOK, utils.SyncMap2Map(domain_service.VerificationMap))
}

// GenerateOwnershipKey
// TODO: This behavior of wiping out the verification is probably too stronk, need to make it only create one if
//	there isn't other information. StoreOrLoad probably is what I want here.
func GenerateOwnershipKey(c *gin.Context) {
	var newGenerateOwnershipKeyReq = GenerateOwnershipKeyReq{}
	err := c.BindJSON(&newGenerateOwnershipKeyReq)
	if err != nil {
		return
	}

	di := domain_service.DomainInformation{DomainName: newGenerateOwnershipKeyReq.DomainName}

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

func DeleteVerification(c *gin.Context) {
	var newDeleteVerificationRequest = DeleteVerificationReq{}
	if err := c.BindJSON(&newDeleteVerificationRequest); err != nil {
		return
	}
	_, loaded := domain_service.VerificationMap.LoadAndDelete(newDeleteVerificationRequest.DomainName)
	if !loaded {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain verification was not present"})
		return
	}

	err := domain_service.SaveDomainInformationFile(c, domain_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

// VerifyOwnership only does TXT record checks
func VerifyOwnership(c *gin.Context) {
	domainNameParam := c.Param("domain_name")
	if domainNameParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing domain_name url parameter"})
		return
	}

	domainName, err := url.Parse(domainNameParam)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	val, ok := domain_service.VerificationMap.Load(domainName.Path)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "could not find requested domainName in database"})
		return
	}
	verification, ok := val.(*domain_service.DomainInformation)
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

	c.JSON(http.StatusOK, VerifyDomainResp{
		DomainName: verification.DomainName,
		Status:     result,
	})

	return
}

// VerifyDomains only does TXT checks for everyone
func VerifyDomains(c *gin.Context) {
	var response = VerifyDomainsResp{}
	domain_service.VerificationMap.Range(func(k interface{}, v interface{}) bool {
		verification, ok := v.(domain_service.DomainInformation)
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
		response[key] = DomainVerificationResp{
			DomainName: key,
			Status:     result,
		}
		verification.Verification.Verified = result
		domain_service.VerificationMap.Store(key, verification)
		return true
	})

	err := domain_service.SaveDomainInformationFile(c, domain_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, response)
	return
}

// CreateDomainInformation will create domain information if not present in VerificationMap
// if it is present, we return 202 and move on with our lives
func CreateDomainInformation(c *gin.Context) {
	var newCreateDomainInformationReq = CreateDomainInformationReq{}
	err := c.BindJSON(&newCreateDomainInformationReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	domainInformation := domain_service.DomainInformation{
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

func DeleteDomainInformation(c *gin.Context) {
	var newDeleteDomainInformationReq DeleteDomainInformationReq
	if err := c.Bind(&newDeleteDomainInformationReq); err != nil {
		return
	}

	di := domain_service.DomainInformation{DomainName: newDeleteDomainInformationReq.DomainName}

	loaded, err := di.LoadAndDelete(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	if !loaded {
		c.JSON(http.StatusNotFound, gin.H{"error": "value was not in verification map, check name?"})
		return
	}

	err = domain_service.SaveDomainInformationFile(c, domain_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s deleted", di.DomainName)})
	return
}

type Record string

const (
	ARecord = "arecord"
	CName   = "cname"
)

type VerifyDelegationReq struct {
	DomainName string `json:"domain_name"`
	Type       Record `json:"type"`
}

func VerifyDelegation(c *gin.Context) {
	var newCreateDelegationRequest VerifyDelegationReq
	err := c.BindJSON(&newCreateDelegationRequest)
	if err != nil {
		return
	}

	di := domain_service.DomainInformation{DomainName: newCreateDelegationRequest.DomainName}
	loadedDi, err := di.Load(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err})
		return
	}

	var verified bool
	switch newCreateDelegationRequest.Type {
	case ARecord:
		verified, err = loadedDi.VerifyARecord(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	case CName:
		verified, err = loadedDi.VerifyCNAME(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"verified": verified})

}
