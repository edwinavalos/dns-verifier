package cert_service

import (
	"context"
	"fmt"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
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
func (*DNSProviderManual) Present(domain string, userId string, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	authZone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return err
	}

	val, ok := domain_service.VerificationMap.Load(userId)
	if !ok {
		return err
	}
	domainInformation, ok := val.(models.DomainInformation)
	if !ok {
		return fmt.Errorf("unable to cast value to DomainInformation")
	}

	domainInformation.Verification.VerificationKey = value
	domainInformation.Verification.VerificationZone = authZone
	ctx := context.Background()
	err = domainInformation.SaveDomainInformation(ctx)
	if err != nil {
		return err
	}

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
