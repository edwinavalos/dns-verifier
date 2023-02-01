package datastore

import (
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
)

type Datastore interface {
	Initialize() error
	GetUserDomains(userId string) map[string]models.DomainInformation
	GetDomainByUser(userId string, domain string) models.DomainInformation
	PutDomainInfo(information models.DomainInformation) error
	DeleteDomain(userId string, domain string)
	DeleteUser(userId string)
	DropTable() error
	GetTableName() string
}

func SetLogger(toSet *logger.Logger) {
	Log = toSet
}

func SetConfig(toSet *config.Config) {
	Config = toSet
}

var Log *logger.Logger
var Config *config.Config
