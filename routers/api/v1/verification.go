package v1

import (
	"dnsVerifier/service/verification_service"
	"dnsVerifier/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"net/url"
	"time"
)

type generateKeyRequest struct {
	domainName string
	userId     uuid.UUID
}

type generateKeyResponse struct {
	verificationKey string
	domainName      *url.URL
}

func GenerateKey(c *gin.Context) {
	var newKeyRequest generateKeyRequest
	if err := c.BindJSON(&newKeyRequest); err != nil {
		return
	}

	domainName, err := url.Parse(newKeyRequest.domainName)
	verification := verification_service.Verification{
		DomainName:      domainName,
		VerificationKey: utils.RandomString(15),
		Verified:        false,
		WarningStamp:    time.Time{},
		ExpireStamp:     time.Now().Add(24 * time.Hour),
		UserId:          newKeyRequest.userId,
	}

	err = verification.SaveVerification(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}

	//verification.SaveVerification()
	response := generateKeyResponse{
		verificationKey: verification.VerificationKey,
		domainName:      domainName,
	}
	c.JSON(http.StatusOK, response)

	return
}
