package certs

import (
	"fmt"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"time"
)

const (
	dnsTemplate = `%s %d IN TXT %q`
)

// DNSProviderManual is an implementation of the ChallengeProvider interface.
type DNSProviderManual struct{}

// NewDNSProviderManual returns a DNSProviderManual instance.
func NewDNSProviderManual() (*DNSProviderManual, error) {
	return &DNSProviderManual{}, nil
}

// Present prints instructions for manually creating the TXT record.
func (*DNSProviderManual) Present(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	authZone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return err
	}

	fmt.Printf("lego: Please create the following TXT record in your %s zone:\n", authZone)
	fmt.Printf(dnsTemplate+"\n", fqdn, dns01.DefaultTTL, value)

	return err
}

// CleanUp prints instructions for manually removing the TXT record.
func (*DNSProviderManual) CleanUp(domain, token, keyAuth string) error {
	fqdn, _ := dns01.GetRecord(domain, keyAuth)

	authZone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return err
	}

	fmt.Printf("lego: You can now remove this TXT record from your %s zone:\n", authZone)

	return nil
}

// Sequential All DNS challenges for this provider will be resolved sequentially.
// Returns the interval between each iteration.
func (d *DNSProviderManual) Sequential() time.Duration {
	return dns01.DefaultPropagationTimeout
}
