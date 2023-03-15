package v1

import "github.com/edwinavalos/dns-verifier/config"

var cfg *config.config

func SetConfig(toSet *config.config) {
	cfg = toSet
}
