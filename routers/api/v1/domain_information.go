package v1

import (
	"fmt"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type GenerateOwnershipKeyReq struct {
	DomainName string `json:"domain_name"`
	UserId     string `json:"user_id"`
}

type GenerateOwnershipKeyResp struct {
	VerificationKey string `json:"verification_key,omitempty"`
	DomainName      string `json:"domain_name,omitempty"`
	Error           string `json:"error,omitempty"`
}

type VerifyDomainResp struct {
	DomainName string `json:"domain_name,omitempty"`
	Status     bool   `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
}

type DomainVerificationResp struct {
	DomainName string `json:"domain_name,omitempty"`
	Status     bool   `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
}

type VerifyDomainsResp map[string]DomainVerificationResp

type CreateDomainInformationReq struct {
	DomainName string `json:"domain_name"`
	UserId     string `json:"user_id"`
}

type DeleteDomainInformationReq struct {
	DomainName string `json:"domain_name"`
	UserId     string `json:"user_id"`
}

type DeleteDomainInformationResp struct {
	DomainName string `json:"domain_name,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

func HandleGetDomainInformation(c *gin.Context) {
	var retRecordMap map[string]map[string]models.DomainInformation
	userId := c.Query("userId")
	if userId == "" {
		retRecordMap, err := domain_service.GetAllRecords()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ran into error getting all records: %s", err.Error())})
			return
		}
		c.JSON(http.StatusOK, retRecordMap)
		return
	}

	retRecordMap, err := domain_service.GetUserDomains(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ran into error creating user recordmap: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, retRecordMap)
	return
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

	domainName := newGenerateOwnershipKeyReq.DomainName
	userId := newGenerateOwnershipKeyReq.UserId
	verificationKey, err := domain_service.GenerateOwnershipKey(userId, domainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GenerateOwnershipKeyResp{Error: fmt.Sprintf("unable to generate ownership key: %s", err)})
		return
	}

	c.JSON(http.StatusOK, GenerateOwnershipKeyResp{
		VerificationKey: verificationKey,
		DomainName:      domainName,
	})
	return
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

	domainName := newVerifyOwnershipReq.DomainName
	userId := newVerifyOwnershipReq.UserId
	di := models.DomainInformation{DomainName: domainName, UserId: userId}

	result, err := domain_service.VerifyTXTRecord(c, di.Verification.VerificationZone, di.Verification.VerificationKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("unable to verify domain: %s", err)})
		return
	}

	if di.Verification.Verified != result {
		di.Verification.Verified = result
	}

	err = domain_service.SaveDomain(di)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
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

	err = domain_service.PutDomain(userId, domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ran into error putting domain: %s err: %s", domain, err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DomainInformation created"})
	return
}

func HandleDeleteDomainInformation(c *gin.Context) {
	var newDeleteDomainInformationReq DeleteDomainInformationReq
	if err := c.Bind(&newDeleteDomainInformationReq); err != nil {
		return
	}

	domainName := newDeleteDomainInformationReq.DomainName
	userId := newDeleteDomainInformationReq.UserId
	err := domain_service.DeleteDomain(userId, domainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, DeleteDomainInformationResp{Error: fmt.Sprintf("unable to delete domain: %s err %s", domainName, err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s deleted", domainName)})
	return
}

type Record string

const (
	ARecord = "arecord"
	CName   = "cname"
)

type VerifyDelegationReq struct {
	DomainName string `json:"domain_name"`
	UserId     string `json:"user_id"`
	Type       Record `json:"type"`
}

func HandleVerifyDelegation(c *gin.Context) {
	var newVerifyDelegationRequest VerifyDelegationReq
	err := c.BindJSON(&newVerifyDelegationRequest)
	if err != nil {
		return
	}

	userId := newVerifyDelegationRequest.UserId
	domain := newVerifyDelegationRequest.DomainName

	var verified bool
	switch newVerifyDelegationRequest.Type {
	case ARecord:
		verified, err = domain_service.VerifyARecord(c, userId, domain)
	case CName:
		verified, err = domain_service.VerifyCNAME(c, userId, domain)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": verified})
	return
}
