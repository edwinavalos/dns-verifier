package cert_service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var cfg *config.Config
var l logger.Logger
var externalIP net.IP

func SetConfig(conf *config.Config) {
	cfg = conf
}

func SetLogger(toSet logger.Logger) {
	l = toSet
}

func SetExternalIP(toSet net.IP) {
	externalIP = toSet
}

type certRequestUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *certRequestUser) GetEmail() string {
	return u.Email
}
func (u certRequestUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *certRequestUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func getRequestUserCert() (*ecdsa.PrivateKey, error) {
	var privateKey *ecdsa.PrivateKey
	_, err := os.Stat(cfg.LESettings.PrivateKeyLocation)
	if err != nil {
		var err2 error
		privateKey, err2 = ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
		if err2 != nil {
			return nil, fmt.Errorf("problem generating key: %w", err2)
		}

		keyBytes, err3 := x509.MarshalPKCS8PrivateKey(privateKey)
		if err3 != nil {
			return nil, fmt.Errorf("problem marshalling private key: %w", err3)
		}
		err4 := os.WriteFile(cfg.LESettings.PrivateKeyLocation, keyBytes, 0644)
		if err4 != nil {
			return nil, fmt.Errorf("problem writing private key file: %w", err4)
		}
	} else {
		dBytes, err2 := os.ReadFile(cfg.LESettings.PrivateKeyLocation)
		if err2 != nil {
			return nil, fmt.Errorf("unable to read privateKey file: %w", err2)
		}

		privateKeyVal, err3 := x509.ParsePKCS8PrivateKey(dBytes)
		if err3 != nil {
			return nil, fmt.Errorf("error parsing privateKey: %w", err3)
		}
		var ok bool
		privateKey, ok = privateKeyVal.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("unable to convert file into private key")
		}
	}

	return privateKey, nil
}

func CompleteCertificateRequest(userId string, domain string, client *lego.Client) (string, error) {
	manualProvider, err := NewDNSProviderManual()
	if err != nil {
		return "", err
	}

	err = client.Challenge.SetDNS01Provider(manualProvider)
	if err != nil {
		return "", err
	}

	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return "", err
	}
	// Need to save the cert URL to retrieve it without re-requesting if possible
	val, ok := domain_service.VerificationMap.Load(userId)
	if !ok {
		return "", fmt.Errorf("domain: %s unable to lookup user in verification map")
	}
	domainInformation, ok := val.(models.DomainInformation)
	if !ok {
		return "", fmt.Errorf("domain: %s unable to convert verification map entry to DomainInformation")
	}

	ctx := context.Background()
	domainInformation.LEVerification.Verified = true
	err = domainInformation.SaveDomainInformation(ctx)
	if err != nil {
		return "", fmt.Errorf("domain: %s unable to save DomainInformation %w", err)
	}
	// ... all done

	// Place the certificate into storage that is shared between our servers, and then send back a positive string
	l.Infof("%+v", string(certificates.Certificate))
	return "domain: %s saved to persistent storage", nil
}

func GetLEGOClient(userId string, domain string, email string) (*lego.Client, error) {
	privateKey, err := getRequestUserCert()
	if err != nil {
		return nil, fmt.Errorf("unable to requestUserCert(): %w", err)
	}

	myUser := certRequestUser{
		Email: email,
		key:   privateKey,
	}

	leConfig := lego.NewConfig(&myUser)

	// This CA URL is configured for a local dev instance of Boulder running in Docker in a VM.

	leConfig.CADirURL = cfg.LESettings.CADirURL
	leConfig.Certificate.KeyType = certcrypto.RSA2048

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(leConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func RequestCertificate(userId string, domain string, email string) (string, string, error) {
	// Create a user. New accounts need an email and private key to start.

	privateKey, err := getRequestUserCert()
	if err != nil {
		return "", "", fmt.Errorf("unable to requestUserCert(): %w", err)
	}

	myUser := certRequestUser{
		Email: email,
		key:   privateKey,
	}

	leConfig := lego.NewConfig(&myUser)

	// This CA URL is configured for a local dev instance of Boulder running in Docker in a VM.

	leConfig.CADirURL = cfg.LESettings.CADirURL
	leConfig.Certificate.KeyType = certcrypto.RSA2048

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(leConfig)
	if err != nil {
		return "", "", err
	}

	// New users will need to register
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return "", "", err
	}
	myUser.Registration = reg

	provider, err := NewDNSProviderManual()
	if err != nil {
		return "", "", fmt.Errorf("domain: %s unable to create new DNSProviderManual %w", domain, err)
	}

	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		return "", "", fmt.Errorf("domain: %s unable to set new DNSProviderManual %w", domain, err)
	}

	err = provider.Present(domain, userId, cfg.LESettings.KeyAuth)
	if err != nil {
		return "", "", fmt.Errorf("domain: %s unable to present challenge %w", domain, err)
	}

	// Because present has to satisfy an interface, we need to the information we wrote in it via DomainInformation lookups. Joy.
	domainInfo, err := domain_service.DomainInfoByUserId(userId, domain)
	if err != nil {
		return "", "", fmt.Errorf("domain: %s unable to get DomainInfo from verification map: %w", domain, err)
	}

	return domainInfo.LEVerification.VerificationZone, domainInfo.LEVerification.VerificationKey, nil
}

func contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func toStringList(ips []net.IP) []string {
	var retList []string
	for _, v := range ips {
		retList = append(retList, v.String())
	}
	return retList
}

// Saving this for later, I think I need something like it in the envoy manager, but it will not be doing certificate
// operations anymore
//func RequestCerts() error {
//
//	l.Infof("My outbound IP for this loop is %s", externalIP.String())
//
//	l.Infof("Starting the big loop")
//	networkMap := gatherer.GetNetworkMap()
//	for domain, val := range networkMap {
//		for _, port := range val {
//			if port.ContainerPort == 443 {
//				ipRecords, err := net.LookupIP(domain)
//				if err != nil {
//					l.Errorf("domain: %s failed to lookup IP err: %s", domain, err)
//					continue
//				}
//				if !contains(toStringList(ipRecords), externalIP.String()) {
//					l.Errorf("domain: %s there isnt a dns entry for this domain pointing to us records were: %+v, skipping", domain, ipRecords)
//					continue
//				}
//				parts := strings.Split(port.Service, ":")
//				exists, valid, err := CertificateExistsAndValid(domain, parts[0])
//				if err != nil {
//					l.Errorf("domain: %s error figuring out if cert for is valid or exist: %s... will not generate", domain, err)
//					continue
//				}
//				var certificates *certificate.Resource
//				if !exists {
//					l.Infof("domain: %s cert file does not exist... attempting creation", domain)
//					err = RequestCertificate(domain, cfg.LESettings.AdminEmail)
//					if err != nil {
//						l.Errorf("unable to request certificate from Let's Encrypt for domain: %s, err: %s", domain, err)
//						continue
//					}
//					exists = true
//					valid = true
//				}
//
//				if !valid {
//					l.Infof("domain %s certificate is expired... will generate", domain)
//					err = RequestCertificate(domain, cfg.LESettings.AdminEmail)
//					if err != nil {
//						l.Errorf("unable to request certificate from Let's Encrypt for domain: %s, err: %s", domain, err)
//						continue
//					}
//					valid = true
//				}
//
//				publicKeyLocation, privateKeyLocation, err := DomainToKeyLocations(domain, parts[0])
//				if err != nil {
//					l.Errorf("domain: %s unable to get locations for saving keys: %s", domain, err)
//					continue
//				}
//				basePath := filepath.Base(publicKeyLocation)
//				err = saveCertificates(publicKeyLocation, privateKeyLocation, basePath, certificates)
//				if err != nil {
//					l.Errorf("domain: %s unable to save certicates, err: %s", domain, err)
//				}
//				l.Infof("successfully generated certificates for: %s", domain)
//			}
//		}
//	}
//	l.Infof("ending the big loop")
//
//	return nil
//}

func saveCertificates(publicKeyLocation string, privateKeyLocation string, basePath string, certificates *certificate.Resource) error {
	generatedLocation := filepath.Join(basePath + "/keys/.generated")
	l.Infof("Saving public key to: %s", publicKeyLocation)
	err := os.WriteFile(publicKeyLocation, certificates.Certificate, 0644)
	if err != nil {
		return err
	}

	l.Infof("Saving private key to: %s", privateKeyLocation)
	err = os.WriteFile(privateKeyLocation, certificates.PrivateKey, 0600)
	if err != nil {
		return err
	}

	l.Infof("Touching generated file: %s", generatedLocation)
	_, err = os.Create(generatedLocation)
	if err != nil {
		return err
	}
	return nil
}

func CertificateExistsAndValid(domain string, service string) (bool, bool, error) {
	publicKeyLocation, _, err := DomainToKeyLocations(domain, service)
	if err != nil {
		return false, false, err
	}

	_, err = os.Stat(publicKeyLocation)
	if os.IsNotExist(err) {
		return false, false, nil
	}

	dBytes, err := os.ReadFile(publicKeyLocation)
	if err != nil {
		return false, true, err
	}

	certChain := decodePem(dBytes)
	var result bool
	for _, cert := range certChain.Certificate {
		x509Cert, err := x509.ParseCertificate(cert)
		if err != nil {
			return false, true, err
		}
		if x509Cert.Subject.CommonName == domain {
			result = x509Cert.NotAfter.After(time.Now())
		}
	}

	return result, true, nil
}

func decodePem(certInput []byte) tls.Certificate {
	var cert tls.Certificate
	var certDERBlock *pem.Block
	for {
		certDERBlock, certInput = pem.Decode(certInput)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		}
	}
	return cert
}

func DomainToKeyLocations(domain string, service string) (string, string, error) {
	var pathTemplate string
	switch runtime.GOOS {
	case "windows":
		pathTemplate = "C:\\mastodon\\%s_%s\\keys\\"
	case "linux":
		pathTemplate = "/root/certs/%s_%s/keys/"
	}

	domainSanitized := strings.Replace(domain, ".", "_", -1)
	path := fmt.Sprintf(pathTemplate, service, domainSanitized)
	publicKeyPath := filepath.Join(path, "/cert.crt")
	privateKeyPath := filepath.Join(path, "/cert.key")

	publicKeyPath = strings.Replace(publicKeyPath, "\\", "/", -1)
	privateKeyPath = strings.Replace(privateKeyPath, "\\", "/", -1)

	return publicKeyPath, privateKeyPath, nil
}
