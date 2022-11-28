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
		apiv1.POST("/generateKey", v1.GenerateKey)
		apiv1.POST("/verifyDomain", v1.VerifyDomain)
		apiv1.POST("/verifyDomains", v1.VerifyDomains)
		apiv1.GET("/verifications", v1.GetVerifications)
		apiv1.DELETE("/verification", v1.DeleteVerification)
	}
	return r
}
