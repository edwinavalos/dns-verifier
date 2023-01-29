package v1

import (
	"errors"
	"fmt"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/edwinavalos/dns-verifier/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

var l *logger.Logger

func SetLogger(toSet *logger.Logger) {
	l = toSet
}

type GenerateOwnershipKeyReq struct {
	DomainName string `json:"domain_name"`
	UserId     string `json:"user_id"`
}

type GenerateOwnershipKeyResp struct {
	VerificationKey string `json:"verification_key"`
	DomainName      string `json:"domain_name"`
	Error           string `json:"error,omitempty"`
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
	DomainName string `json:"domain_name"`
	UserId     string `json:"user_id"`
}

type DeleteDomainInformationReq struct {
	DomainName string `json:"domain_name"`
	UserId     string `json:"user_id"`
}

func HandleGetDomainInformation(c *gin.Context) {
	c.JSON(http.StatusOK, domain_service.SyncMap2Map(domain_service.VerificationMap))
}

// HandleGenerateOwnershipKey
// TODO: This behavior of wiping out the verification is probably too stronk, need to make it only create one if
//
//	there isn't other information. StoreOrLoad probably is what I want here.
func HandleGenerateOwnershipKey(c *gin.Context) {
	var newGenerateOwnershipKeyReq = GenerateOwnershipKeyReq{}
	err := c.BindJSON(&newGenerateOwnershipKeyReq)
	if err != nil {
		return
	}

	di := domain_service.DomainInformation{DomainName: newGenerateOwnershipKeyReq.DomainName, UserId: newGenerateOwnershipKeyReq.UserId}

	loadedDi, err := di.Load(c)
	if err != nil {
		c.JSON(http.StatusNotFound, GenerateOwnershipKeyResp{
			Error: fmt.Sprintf("unable to find domain name: %s in verification map err was: %s", newGenerateOwnershipKeyReq.DomainName, err),
		})
		return
	}

	loadedDi.Verification.VerificationKey = utils.RandomString(30)

	err = loadedDi.SaveDomainInformation(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GenerateOwnershipKeyResp{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GenerateOwnershipKeyResp{
		VerificationKey: fmt.Sprintf("%s;%s;%s", cfg.App.VerificationTxtRecordName, loadedDi.DomainName, loadedDi.Verification.VerificationKey),
		DomainName:      loadedDi.DomainName,
	})
	return
}

func HandleDeleteVerification(c *gin.Context) {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

type VerifyOwnershipReq struct {
	DomainName string `form:"domain_name"`
	UserId     string `form:"user_id"`
}

// HandleVerifyOwnership only does TXT record checks
func HandleVerifyOwnership(c *gin.Context) {
	var newVerifyOwnershipReq VerifyOwnershipReq
	err := c.BindQuery(&newVerifyOwnershipReq)
	if err != nil {
		return
	}

	val, ok := domain_service.VerificationMap.Load(newVerifyOwnershipReq.UserId)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "could not find requested domainName by userId in database"})
		return
	}
	userDomainNames, ok := val.(map[string]domain_service.DomainInformation)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to convert db value to map[string]DomainInformation"})
		return
	}

	di := userDomainNames[newVerifyOwnershipReq.DomainName]
	result, err := di.VerifyOwnership(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("unable to verify domain: %s", err)})
		return
	}

	if di.Verification.Verified != result {
		di.Verification.Verified = result
		err := di.SaveDomainInformation(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
	}

	c.JSON(http.StatusOK, VerifyDomainResp{
		DomainName: di.DomainName,
		Status:     result,
	})

	return
}

// Needs to be refactored to do all verifications for a certain userId, at the moment it doesn't go deep enough
// into the map[string]map[string]DomainInformation
//// VerifyDomains only does TXT checks for everyone
//func VerifyDomains(c *gin.Context) {
//	var response = VerifyDomainsResp{}
//	domain_service.VerificationMap.Range(func(k interface{}, v interface{}) bool {
//		verification, ok := v.(domain_service.DomainInformation)
//		if !ok {
//			return false
//		}
//		result, err := verification.HandleVerifyOwnership(c)
//		if err != nil {
//			return false
//		}
//		key, ok := k.(string)
//		if !ok {
//			return false
//		}
//		response[key] = DomainVerificationResp{
//			DomainName: key,
//			Status:     result,
//		}
//		verification.Verification.Verified = result
//		domain_service.VerificationMap.Store(key, verification)
//		return true
//	})
//
//	err := domain_service.SaveDomainInformationFile(c, domain_service.VerificationMap)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
//		return
//	}
//
//	c.JSON(http.StatusOK, response)
//	return
//}

// HandleCreateDomainInformation will create domain information if not present in VerificationMap
// if it is present, we return 202 and move on with our lives
func HandleCreateDomainInformation(c *gin.Context) {
	var newCreateDomainInformationReq = CreateDomainInformationReq{}
	err := c.BindJSON(&newCreateDomainInformationReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	domain := newCreateDomainInformationReq.DomainName
	userId := newCreateDomainInformationReq.UserId

	if domain == "" || userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing domain_name or user_id"})
		return
	}

	_, err = domain_service.DomainInfoByUserId(userId, domain)
	if err != nil {
		if errors.Is(err, domain_service.ErrUnableToFindUser) {
			err := createDomainInformation(c, domain, userId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "did not find user, created it and the domain"})
			return
		}
		if errors.Is(err, domain_service.ErrNoDomainInformation) {
			err := createDomainInformation(c, domain, userId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "found user, but not domain, created it"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "domain already existed"})
}

func createDomainInformation(c *gin.Context, domain string, userId string) error {
	domainInformation := domain_service.DomainInformation{
		DomainName: domain,
		UserId:     userId,
	}
	user, _, err := domainInformation.LoadOrStore(c)
	if err != nil {
		return err
	}

	domain_service.VerificationMap.Store(userId, user)
	err = domainInformation.SaveDomainInformation(c)
	if err != nil {
		return err
	}
	return nil
}

func HandleDeleteDomainInformation(c *gin.Context) {
	var newDeleteDomainInformationReq DeleteDomainInformationReq
	if err := c.Bind(&newDeleteDomainInformationReq); err != nil {
		return
	}

	di := domain_service.DomainInformation{DomainName: newDeleteDomainInformationReq.DomainName, UserId: newDeleteDomainInformationReq.UserId}

	loaded, err := di.LoadAndDelete(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !loaded {
		c.JSON(http.StatusNotFound, gin.H{"error": "value was not in verification map, check name?"})
		return
	}

	err = domain_service.SaveDomainInformationFile(c, domain_service.VerificationMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

func HandleVerifyDelegation(c *gin.Context) {
	var newCreateDelegationRequest VerifyDelegationReq
	err := c.BindJSON(&newCreateDelegationRequest)
	if err != nil {
		return
	}

	di := domain_service.DomainInformation{DomainName: newCreateDelegationRequest.DomainName}
	loadedDi, err := di.Load(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var verified bool
	switch newCreateDelegationRequest.Type {
	case ARecord:
		verified, err = loadedDi.VerifyARecord(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case CName:
		verified, err = loadedDi.VerifyCNAME(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"verified": verified})

}
