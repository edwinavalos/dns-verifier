package v1

import "github.com/edwinavalos/dns-verifier/config"

var cfg *config.Config

func SetConfig(toSet *config.Config) {
	cfg = toSet
}
