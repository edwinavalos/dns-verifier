package routers

import (
	"github.com/edwinavalos/dns-verifier/logger"
	v1 "github.com/edwinavalos/dns-verifier/routers/api/v1"
	"github.com/gin-gonic/gin"
)

var Log *logger.Logger

func SetLogger(toSet *logger.Logger) {
	Log = toSet
}

func InitRouter() *gin.Engine {
	r := gin.Default()
	apiv1 := r.Group("/api/v1")
	apiv1.Use()
	{
		apiv1.POST("/domain", v1.HandleCreateDomainInformation)
		apiv1.DELETE("/domain", v1.HandleDeleteDomainInformation)

		apiv1.POST("/domain/verificationKey", v1.HandleGenerateOwnershipKey)
		//apiv1.DELETE("/domain/verification", v1.HandleDeleteVerification)
		apiv1.GET("/domain/verification", v1.HandleVerifyOwnership)

		//apiv1.POST("/domains/verify", v1.VerifyDomains)
		apiv1.GET("/domains", v1.HandleGetDomainInformation)

		apiv1.POST("/domain/delegation", v1.HandleVerifyDelegation)
		//apiv1.GET("/domains/delegations", v1.GetDelegations)

		apiv1.POST("/cert/request", v1.HandleRequestCertificate)
		apiv1.POST("/cert/complete", v1.HandleCompleteCertificateRequest)
	}
	return r
}
