package v1

import (
	"fmt"
	"github.com/edwinavalos/common/config"
	"github.com/edwinavalos/common/logger"
	"github.com/edwinavalos/dns-verifier/service/cert_service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type CertificateReq struct {
	UserId uuid.UUID `json:"user_id"`
	Domain string    `json:"domain"`
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

type CertHandler struct {
	certService *cert_service.Service
	cfg         *config.Config
}

func NewCertHandler(conf *config.Config, certService *cert_service.Service) *CertHandler {
	return &CertHandler{
		certService: certService,
		cfg:         conf,
	}
}

func (h *CertHandler) HandleRequestCertificate(c *gin.Context) {
	var newCertReq CertificateReq
	err := c.BindJSON(&newCertReq)
	if err != nil {
		return
	}

	if newCertReq.UserId == uuid.Nil || newCertReq.Domain == "" {
		c.JSON(http.StatusBadRequest, RequestCertificateResp{
			Error: "missing user_id or domain or email in request",
		})
		return
	}

	recordName, recordValue, alreadyValid, err := h.certService.RequestCertificate(newCertReq.UserId, newCertReq.Domain, h.cfg.LEAdminEmail())
	if err != nil {
		if alreadyValid {
			c.JSON(http.StatusAccepted, RequestCertificateResp{
				Domain:      newCertReq.Domain,
				RecordName:  "",
				RecordValue: "",
				Error:       err.Error(),
			})
			return
		}

		logger.Error("unable to create new certificate request from Let's Encrypt: %s", err)
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

func (h *CertHandler) HandleCompleteCertificateRequest(c *gin.Context) {
	var newCertReq CertificateReq
	err := c.BindJSON(&newCertReq)
	if err != nil {
		return
	}

	domain := newCertReq.Domain
	userId := newCertReq.UserId
	if userId == uuid.Nil || domain == "" {
		c.JSON(http.StatusBadRequest, CompleteCertificateRequestResp{
			Error: "missing user_id or domain or email in request",
		})
		return
	}

	err = h.certService.CompleteCertificateRequest(userId, domain, "")
	if err != nil {
		logger.Error("ran into issue complete certificate request: %s", err)
		c.JSON(http.StatusInternalServerError, CompleteCertificateRequestResp{
			Error: fmt.Sprintf("domain: %s, unable to complete certificate request: %s", domain, err),
		})
		return
	}

	c.JSON(http.StatusOK, CompleteCertificateRequestResp{
		Domain:  domain,
		Message: "got a cert",
	})
	return
}
