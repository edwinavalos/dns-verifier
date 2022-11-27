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
		apiv1.POST("/generateKey/", v1.GenerateKey)
	}
	return r
}
