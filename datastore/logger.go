package datastore

import "github.com/edwinavalos/dns-verifier/logger"

var Log *logger.Logger

func SetLogger(toSet *logger.Logger) {
	Log = toSet
}
