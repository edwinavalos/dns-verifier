package cert_service

import (
	"fmt"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/log"
	"golang.org/x/net/idna"
	"time"
)

const (
	dnsTemplate = `%s %d IN TXT %q`
)

// DNSProviderManual is an implementation of the ChallengeProvider interface.
type DNSProviderManual struct {
	UserId  string
	KeyAuth string
}

// NewDNSProviderManual returns a DNSProviderManual instance.
func NewDNSProviderManual(userId string) (*DNSProviderManual, error) {
	return &DNSProviderManual{UserId: userId}, nil
}

// Present prints instructions for manually creating the TXT record.
func (d *DNSProviderManual) Present(domain string, token string, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)

	_, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return err
	}

	di, err := storage.GetDomainByUser(d.UserId, domain)
	if err != nil {
		return err
	}
	di.LEVerification.VerificationKey = value
	di.LEVerification.VerificationZone = fqdn
	l.Infof("Add %s with value %s for domain %s to complete certification", fqdn, value, domain)
	err = storage.PutDomainInfo(di)
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

func (d *DNSProviderManual) Timeout() (duration time.Duration, interval time.Duration) {
	return time.Second, time.Second
}

func sanitizeDomain(domains []string) []string {
	var sanitizedDomains []string
	for _, domain := range domains {
		sanitizedDomain, err := idna.ToASCII(domain)
		if err != nil {
			log.Infof("skip domain %q: unable to sanitize (punnycode): %v", domain, err)
		} else {
			sanitizedDomains = append(sanitizedDomains, sanitizedDomain)
		}
	}
	return sanitizedDomains
}

//
//func ManualObtain(request certificate.ObtainRequest, core *api.Core, authzUri string) (*certificate.Resource, error) {
//	if len(request.Domains) == 0 {
//		return nil, errors.New("no domains to obtain a certificate for")
//	}
//
//	domains := sanitizeDomain(request.Domains)
//
//
//	if request.Bundle {
//		log.Infof("[%s] acme: Obtaining bundled SAN certificate", strings.Join(domains, ", "))
//	} else {
//		log.Infof("[%s] acme: Obtaining SAN certificate", strings.Join(domains, ", "))
//	}
//
//	order, err := core.Orders.New(domains)
//	if err != nil {
//		return nil, err
//	}
//
//	authz, err := core.Authorizations.Get(authzUri)
//	if err != nil {
//		// If any challenge fails, return. Do not generate partial SAN certificates.
//		err := core.Authorizations.Deactivate(authzUri)
//		if err != nil {
//			return nil, err
//		}
//		return nil, err
//	}
//
//
//	if err != nil {
//		// If any challenge fails, return. Do not generate partial SAN certificates.
//		c.deactivateAuthorizations(order, request.AlwaysDeactivateAuthorizations)
//		return nil, err
//	}
//
//	log.Infof("[%s] acme: Validations succeeded; requesting certificates", strings.Join(domains, ", "))
//
//	failures := make(obtainError)
//	cert, err := c.getForOrder(domains, order, request.Bundle, request.PrivateKey, request.MustStaple, request.PreferredChain)
//	if err != nil {
//		for _, auth := range authz {
//			failures[challenge.GetTargetedDomain(auth)] = err
//		}
//	}
//
//	if request.AlwaysDeactivateAuthorizations {
//		c.deactivateAuthorizations(order, true)
//	}
//
//	// Do not return an empty failures map, because
//	// it would still be a non-nil error value
//	if len(failures) > 0 {
//		return cert, failures
//	}
//	return cert, nil
//}
