package routers

import (
	v1 "dnsVerifier/routers/api/v1"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	apiv1 := r.Group("/api/v1")
	apiv1.Use()
	{
		apiv1.POST("/domain", v1.CreateDomainInformation)
		apiv1.DELETE("/domain", v1.DeleteDomainInformation)

		apiv1.POST("/domain/verificationKey", v1.GenerateOwnershipKey)
		apiv1.DELETE("/domain/verification", v1.DeleteVerification)
		apiv1.POST("/domain/verification", v1.VerifyOwnership)

		apiv1.POST("/domains/verify", v1.VerifyDomains)
		apiv1.GET("/domains", v1.GetDomainInformation)

		//apiv1.POST("/domain/delegation", v1.CreateDelegation)
		//apiv1.POST("/domains/delegation", v1.VerifyDelegation)
		//apiv1.DELETE("/domains/delegation", v1.DeleteDelegation)
		//apiv1.GET("/domains/delegations", v1.GetDelegations)

	}
	return r
}
