package v1

import "github.com/edwinavalos/dns-verifier/logger"

var log *logger.Logger

func SetLogger(toSet *logger.Logger) {
	log = toSet
}
