package datastore

import (
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/models"
)

type Datastore interface {
	Initialize() error
	GetUserDomains(userId string) (map[string]models.DomainInformation, error)
	GetDomainByUser(userId string, domain string) (models.DomainInformation, error)
	PutDomainInfo(information models.DomainInformation) error
	DeleteDomain(userId string, domain string) error
	DropTable() error
	GetTableName() string
	GetAllRecords() ([]models.DomainInformation, error)
}

func SetConfig(toSet *config.Config) {
	Config = toSet
}

var Config *config.Config
