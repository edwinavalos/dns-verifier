package v1

import (
	"dnsVerifier/config"
	"dnsVerifier/service/verfication_service"
	"dnsVerifier/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"net/url"
	"time"
)

type generateKeyRequest struct {
	domainName url.URL
	userId     uuid.UUID
}

type generateKeyResponse struct {
	verificationKey string
	domainName      url.URL
}

func GenerateKey(config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var newKeyRequest generateKeyRequest
		if err := c.BindJSON(&newKeyRequest); err != nil {
			return
		}
		// Check if we have a key for the domain yet

		verification := verfication_service.Verification{
			DomainName:      newKeyRequest.domainName,
			VerificationKey: utils.RandomString(15),
			Verified:        false,
			WarningStamp:    time.Time{},
			ExpireStamp:     time.Now().Add(24 * time.Hour),
			UserId:          newKeyRequest.userId,
		}

		verification.SaveVerification(config.App.VerificationTxtRecordName)
		response := generateKeyResponse{
			verificationKey: verification.VerificationKey,
			domainName:      verification.DomainName,
		}
		c.JSON(http.StatusOK, response)

		return
	}

}
