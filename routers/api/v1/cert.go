package v1

import (
	"fmt"
	"github.com/edwinavalos/dns-verifier/service/cert_service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type CertificateReq struct {
	UserId string `json:"user_id"`
	Domain string `json:"domain"`
}

type RequestCertificateResp struct {
	Domain      string `json:"domain,omitempty"`
	RecordName  string `json:"record_name,omitempty"`
	RecordValue string `json:"record_value,omitempty"`
	Error       string `json:"error,omitempty"`
}

type CompleteCertificateRequestResp struct {
	Domain  string `json:"domain,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func HandleRequestCertificate(c *gin.Context) {
	var newCertReq CertificateReq
	err := c.BindJSON(&newCertReq)
	if err != nil {
		return
	}

	if newCertReq.UserId == "" || newCertReq.Domain == "" {
		c.JSON(http.StatusBadRequest, RequestCertificateResp{
			Error: "missing user_id or domain or email in request",
		})
		return
	}

	recordName, recordValue, err := cert_service.RequestCertificate(newCertReq.UserId, newCertReq.Domain, cfg.LESettings.AdminEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, RequestCertificateResp{
			Domain: newCertReq.Domain,
			Error:  fmt.Sprintf("unable to request new certificate from Let's Encrypt: %s", err),
		})
		return
	}

	c.JSON(http.StatusOK, RequestCertificateResp{
		Domain:      newCertReq.Domain,
		RecordName:  recordName,
		RecordValue: recordValue,
	})
	return
}

func HandleCompleteCertificateRequest(c *gin.Context) {
	var newCertReq CertificateReq
	err := c.BindJSON(&newCertReq)
	if err != nil {
		return
	}

	domain := newCertReq.Domain
	userId := newCertReq.UserId
	if userId == "" || domain == "" {
		c.JSON(http.StatusBadRequest, CompleteCertificateRequestResp{
			Error: "missing user_id or domain or email in request",
		})
		return
	}

	_, err = cert_service.CompleteCertificateRequest(userId, domain, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, CompleteCertificateRequestResp{
			Error: fmt.Sprintf("domain: %s, unable to complete certificate request: %s", domain, err),
		})
		return
	}

	//for _, der := range ders {
	//	cert, err := x509.ParseCertificate(der)
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	//		return
	//	}
	//	log.Infof("cert: %s \n%s", cert.Subject, string(cert.Raw))
	//}

	c.JSON(http.StatusInternalServerError, CompleteCertificateRequestResp{
		Domain:  domain,
		Message: "got a cert",
	})
	return
}
