package routers

import (
	"github.com/edwinavalos/common/config"
	v1 "github.com/edwinavalos/dns-verifier/routers/api/v1"
	"github.com/edwinavalos/dns-verifier/service/cert_service"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/gin-gonic/gin"
)

func InitRouter(conf *config.Config, domainService *domain_service.Service, certService *cert_service.Service) *gin.Engine {
	r := gin.Default()
	v1DomainHandler := v1.NewDomainHandler(domainService)
	v1CertHandler := v1.NewCertHandler(conf, certService)
	apiv1 := r.Group("/api/v1")
	apiv1.Use()
	{
		apiv1.POST("/domain", v1DomainHandler.HandleCreateDomainInformation)
		apiv1.DELETE("/domain", v1DomainHandler.HandleDeleteDomainInformation)

		apiv1.POST("/domain/verificationKey", v1DomainHandler.HandleGenerateOwnershipKey)
		//apiv1.DELETE("/domain/verification", v1.HandleDeleteVerification)
		apiv1.POST("/domain/verification", v1DomainHandler.HandleVerifyOwnership) // Cause we are using a form for this???

		//apiv1.POST("/domains/verify", v1.VerifyDomains)
		apiv1.GET("/domains", v1DomainHandler.HandleGetDomainInformation)

		apiv1.POST("/domain/delegation", v1DomainHandler.HandleVerifyDelegation)
		//apiv1.GET("/domains/delegations", v1.GetDelegations)

		apiv1.POST("/cert/request", v1CertHandler.HandleRequestCertificate)
		apiv1.POST("/cert/complete", v1CertHandler.HandleCompleteCertificateRequest)
	}
	return r
}
