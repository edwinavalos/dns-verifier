package v1

import (
	"context"
	"fmt"
	"github.com/edwinavalos/common/models"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type GenerateOwnershipKeyReq struct {
	DomainName string    `json:"domain_name"`
	UserID     uuid.UUID `json:"user_id"`
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
	DomainName string    `json:"domain_name"`
	UserID     uuid.UUID `json:"user_id"`
}

type DeleteDomainInformationReq struct {
	DomainName string    `json:"domain_name"`
	UserID     uuid.UUID `json:"user_id"`
}

type DeleteDomainInformationResp struct {
	DomainName string `json:"domain_name,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

type Record string

const (
	ARecord = "arecord"
	CName   = "cname"
)

type VerifyDelegationReq struct {
	DomainName string    `json:"domain_name"`
	UserId     uuid.UUID `json:"user_id"`
	Type       Record    `json:"type"`
}

type DomainHandler struct {
	domainService *domain_service.Service
}

func NewDomainHandler(service *domain_service.Service) *DomainHandler {
	return &DomainHandler{
		domainService: service,
	}
}

func (d *DomainHandler) HandleGetDomainInformation(c *gin.Context) {
	var retRecords map[string]models.DomainInformation
	userID := c.Query("userID")
	if userID == "" {
		retRecordMap, err := d.domainService.GetAllRecords(context.TODO())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ran into error getting all records: %s", err.Error())})
			return
		}
		c.JSON(http.StatusOK, retRecordMap)
		return
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("ran into error parsing uuid: %s", err.Error())})
		return
	}
	retRecords, err = d.domainService.GetUserDomains(context.TODO(), userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ran into error creating user recordmap: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, retRecords)
	return
}

// HandleGenerateOwnershipKey
// TODO: This behavior of wiping out the verification is probably too stronk, need to make it only create one if
//
//	there isn't other information. StoreOrLoad probably is what I want here.
func (d *DomainHandler) HandleGenerateOwnershipKey(c *gin.Context) {
	var newGenerateOwnershipKeyReq = GenerateOwnershipKeyReq{}
	err := c.BindJSON(&newGenerateOwnershipKeyReq)
	if err != nil {
		return
	}

	domainName := newGenerateOwnershipKeyReq.DomainName
	userID := newGenerateOwnershipKeyReq.UserID
	verificationKey, err := d.domainService.GenerateOwnershipKey(context.TODO(), userID, domainName)
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
	DomainName string    `json:"domain_name"`
	UserID     uuid.UUID `json:"user_id"`
}

// HandleVerifyOwnership only does TXT record checks
func (d *DomainHandler) HandleVerifyOwnership(c *gin.Context) {
	var newVerifyOwnershipReq VerifyOwnershipReq
	err := c.Bind(&newVerifyOwnershipReq)
	if err != nil {
		return
	}

	domainName := newVerifyOwnershipReq.DomainName
	userID := newVerifyOwnershipReq.UserID
	domain, err := d.domainService.GetDomainByUser(context.TODO(), userID, domainName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unable to find given user: %s with domain: %s", userID, domainName)})
		return
	}

	result, err := d.domainService.VerifyTXTRecord(c, domain.Verification.Zone, domain.Verification.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("unable to verify domain: %s", err)})
		return
	}

	if domain.Verification.Verified != result {
		domain.Verification.Verified = result
	}

	err = d.domainService.PutDomain(context.TODO(), domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, VerifyDomainResp{
		DomainName: domain.DomainName,
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
func (d *DomainHandler) HandleCreateDomainInformation(c *gin.Context) {
	var newCreateDomainInformationReq = CreateDomainInformationReq{}
	err := c.BindJSON(&newCreateDomainInformationReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	domain := newCreateDomainInformationReq.DomainName
	userID := newCreateDomainInformationReq.UserID

	if domain == "" || userID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing domain_name or user_id"})
		return
	}

	newDomain := models.DomainInformation{
		DomainName: domain,
		UserID:     userID,
	}
	err = d.domainService.PutDomain(context.TODO(), newDomain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ran into error putting domain: %s err: %s", domain, err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DomainInformation created"})
	return
}

func (d *DomainHandler) HandleDeleteDomainInformation(c *gin.Context) {
	var newDeleteDomainInformationReq DeleteDomainInformationReq
	if err := c.Bind(&newDeleteDomainInformationReq); err != nil {
		return
	}

	domainName := newDeleteDomainInformationReq.DomainName
	userID := newDeleteDomainInformationReq.UserID
	err := d.domainService.DeleteDomain(context.TODO(), userID, domainName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, DeleteDomainInformationResp{Error: fmt.Sprintf("unable to delete domain: %s err %s", domainName, err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s deleted", domainName)})
	return
}

func (d *DomainHandler) HandleVerifyDelegation(c *gin.Context) {
	var newVerifyDelegationRequest VerifyDelegationReq
	err := c.BindJSON(&newVerifyDelegationRequest)
	if err != nil {
		return
	}

	userID := newVerifyDelegationRequest.UserId
	domain := newVerifyDelegationRequest.DomainName

	var verified bool
	switch newVerifyDelegationRequest.Type {
	case ARecord:
		verified, err = d.domainService.VerifyARecord(c, userID, domain)
	case CName:
		verified, err = d.domainService.VerifyCNAME(c, userID, domain)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": verified})
	return
}
