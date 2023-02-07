package cert_service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/aws/smithy-go/rand"
	"github.com/edwinavalos/dns-verifier/config"
	"github.com/edwinavalos/dns-verifier/datastore"
	"github.com/edwinavalos/dns-verifier/logger"
	"github.com/edwinavalos/dns-verifier/models"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/registration"
	"golang.org/x/crypto/acme"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var cfg *config.Config
var l *logger.Logger
var externalIP net.IP
var dbStorage datastore.Datastore
var fileStorage datastore.FileStore

func SetDBStorage(toSet datastore.Datastore) {
	dbStorage = toSet
}

func SetFileStorage(toSet datastore.FileStore) {
	fileStorage = toSet
}

func SetConfig(conf *config.Config) {
	cfg = conf
}

func SetLogger(toSet *logger.Logger) {
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

func recordInPlace(domainInfo models.DomainInformation) bool {
	if domainInfo.LEVerification.VerificationKey == "" || domainInfo.LEVerification.VerificationZone == "" {
		return false
	}

	found, err := domain_service.VerifyTXTRecord(context.TODO(), domainInfo.LEVerification.VerificationZone, domainInfo.LEVerification.VerificationKey)
	if err != nil || !found {
		return false
	}

	return true
}

func CompleteCertificateRequest(userId string, domain string, email string) error {
	// Retrieve our domain info from the database
	domainInfo, err := dbStorage.GetDomainByUser(userId, domain)
	if err != nil {
		return fmt.Errorf("domain: %s unable to get DomainInfo from database: %w", domain, err)
	}

	inPlace := recordInPlace(domainInfo)
	if !inPlace {
		return fmt.Errorf("unable to verify that your record is in place before completing request")
	}

	privateKey, err := getRequestUserCert()
	if err != nil {
		return fmt.Errorf("unable to requestUserCert(): %w", err)
	}

	// Create our client which will interact with the acme api
	client := &acme.Client{
		Key:          privateKey,
		DirectoryURL: cfg.LESettings.CADirURL,
	}

	if domainInfo.LEVerification.Verified == true && domainInfo.LEInfo.CertURL != "" {
		certs, err := client.FetchCert(context.TODO(), domainInfo.LEInfo.CertURL, true)
		if err != nil {
			return err
		}

		err2 := WriteToStorage(privateKey, domain, certs)
		if err2 != nil {
			return err2
		}
		return nil
	}

	if domainInfo.LEInfo.OrderURL == "" {
		return fmt.Errorf("missing order_url")
	}

	identifiers := acme.DomainIDs(domain)
	authOrder, err := client.GetOrder(context.TODO(), domainInfo.LEInfo.OrderURL)
	if err != nil || authOrder.Status == acme.StatusInvalid {
		return fmt.Errorf("AuthorizeOrder: %v", err)
	}
	// we set this because get order doesn't populate the URI value
	authOrder.URI = domainInfo.LEInfo.OrderURL

	if domainInfo.LEInfo.AuthzURL == "" {
		return fmt.Errorf("missing authz_url")
	}
	authz, err := client.GetAuthorization(context.TODO(), domainInfo.LEInfo.AuthzURL)
	if err != nil {
		return err
	}

	if domainInfo.LEInfo.ChallengeURL == "" {
		return fmt.Errorf("missing challenge_url")
	}
	chal, err := client.GetChallenge(context.TODO(), domainInfo.LEInfo.ChallengeURL)
	if err != nil {
		return err
	}

	err = completeDNS01(context.TODO(), client, authz, chal, authOrder)
	if err != nil {
		return err
	}

	csr, privateKey := newCSR(identifiers)
	ders, curl, err := client.CreateOrderCert(context.TODO(), authOrder.FinalizeURL, csr, true)
	if err != nil {
		return fmt.Errorf("CreateOrderCert: %v", err)
	}
	domainInfo.LEInfo.CertURL = curl
	domainInfo.LEVerification.Verified = true
	err = dbStorage.PutDomainInfo(domainInfo)
	if err != nil {
		return err
	}
	l.Infof("cert URL: %s", curl)

	err2 := WriteToStorage(privateKey, domain, ders)
	if err2 != nil {
		return err2
	}

	return nil
}

func WriteToStorage(privateKey *ecdsa.PrivateKey, domain string, ders [][]byte) error {
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}

	// Encode the private key into PEM format
	privateKeyPEM := &pem.Block{
		Type:  "ECDSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	// Encode the PEM block to a byte buffer.
	var privBuf bytes.Buffer
	err = pem.Encode(&privBuf, privateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to encode PEM block: %v\n", err)
	}

	err = fileStorage.SaveBuf(privBuf, fmt.Sprintf("mastodon_le_certs/%s/cert.key", domain))
	if err != nil {
		return err
	}

	var mergedDers []byte
	for _, slice := range ders {
		mergedDers = append(mergedDers, slice...)
	}
	secondBuf := bytes.NewBuffer(mergedDers)
	err = fileStorage.SaveBuf(*secondBuf, fmt.Sprintf("mastodon_le_certs/%s/cert.crt", domain))
	if err != nil {
		return err
	}
	return nil
}

func newCSR(identifiers []acme.AuthzID) ([]byte, *ecdsa.PrivateKey) {
	var csr x509.CertificateRequest
	for _, id := range identifiers {
		switch id.Type {
		case "dns":
			csr.DNSNames = append(csr.DNSNames, id.Value)
		case "ip":
			csr.IPAddresses = append(csr.IPAddresses, net.ParseIP(id.Value))
		default:
			panic(fmt.Sprintf("newCSR: unknown identifier type %q", id.Type))
		}
	}
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("newCSR: ecdsa.GenerateKey for a cert: %v", err))
	}
	b, err := x509.CreateCertificateRequest(rand.Reader, &csr, k)
	if err != nil {
		panic(fmt.Sprintf("newCSR: x509.CreateCertificateRequest: %v", err))
	}
	return b, k
}

func runDNS01(ctx context.Context, client *acme.Client, chal *acme.Challenge) (string, error) {
	dnsToken, err := client.DNS01ChallengeRecord(chal.Token)
	if err != nil {
		return "", fmt.Errorf("DNS01ChallengeRecord: %v", err)
	}

	return dnsToken, nil
}

func completeDNS01(ctx context.Context, client *acme.Client, z *acme.Authorization, chal *acme.Challenge, order *acme.Order) error {

	newChal, err := client.Accept(ctx, chal)
	if err != nil {
		return fmt.Errorf("accept(%q): %v", chal.URI, err)
	}
	l.Infof("%+v", newChal)

	_, err = client.WaitAuthorization(context.TODO(), z.URI)
	if err != nil {
		return err
	}

	l.Infof("all challenges are done")
	if _, err := client.WaitOrder(ctx, order.URI); err != nil {
		return fmt.Errorf("waitOrder(%q): %v", order.URI, err)
	}

	return nil
}

func request(ctx context.Context, client *acme.Client, z *acme.Authorization) (string, *acme.Challenge, error) {
	chal := getDnsChallenge(z)
	if chal == nil {
		return "", nil, fmt.Errorf("challenge type %q wasn't offered for authz %s", challenge.DNS01, z.URI)
	}

	dnsToken, err := runDNS01(ctx, client, chal)
	if err != nil {
		return "", nil, err
	}
	return dnsToken, chal, nil
}

func getDnsChallenge(z *acme.Authorization) *acme.Challenge {
	var chal *acme.Challenge
	for i, c := range z.Challenges {
		l.Infof("challenge %d: %+v", i, c)
		if c.Type == challenge.DNS01.String() {
			l.Infof("picked %s for authz %s", c.URI, z.URI)
			chal = c
		}
	}
	return chal
}

func RequestCertificate(userId string, domain string, email string) (string, string, bool, error) {

	// Get the private key for the cert administrator account
	privateKey, err := getRequestUserCert()
	if err != nil {
		return "", "", false, fmt.Errorf("unable to requestUserCert(): %w", err)
	}

	// Create our client which will interact with the acme api
	client := &acme.Client{
		Key:          privateKey,
		DirectoryURL: cfg.LESettings.CADirURL,
	}

	// Retrieve our domain info from the database
	domainInfo, err := dbStorage.GetDomainByUser(userId, domain)
	if err != nil {
		return "", "", false, fmt.Errorf("domain: %s unable to get DomainInfo from database: %w", domain, err)
	}

	// Log in and get our account information
	//var account *acme.Account
	if domainInfo.LEInfo.CertURL != "" {
		// the url parameter is legacy and not used
		_, err = client.GetReg(context.TODO(), "")
		if err != nil {
			return "", "", false, err
		}
	} else {
		_, err = client.UpdateReg(context.TODO(), &acme.Account{})
		if err != nil {
			return "", "", false, err
		}
	}

	// Identifiers is an acme construct for our domain names
	identifiers := acme.DomainIDs(domain)

	// We create an order for our domain name, this is key to authorizing everything
	// this kicks off the process
	var authOrder *acme.Order
	if domainInfo.LEInfo.OrderURL == "" {
		authOrder, err = client.AuthorizeOrder(context.TODO(), identifiers)
		if err != nil {
			return "", "", false, err
		}
	} else {
		authOrder, err = client.GetOrder(context.TODO(), domainInfo.LEInfo.OrderURL)
		if err != nil {
			return "", "", false, err
		}
		if authOrder.Status == acme.StatusInvalid {
			authOrder, err = client.AuthorizeOrder(context.TODO(), identifiers)
			if err != nil {
				return "", "", false, err
			}
		}
		// We need to set this because it doesn't get populated by GetOrder
		if authOrder.URI == "" {
			authOrder.URI = domainInfo.LEInfo.OrderURL
		}
	}
	if authOrder == nil {
		return "", "", false, fmt.Errorf("unable to find auth order in db or unable to authorize new order")
	}

	var zurls []string
	zone := fmt.Sprintf("_acme-challenge.%s", domain)
	for _, u := range authOrder.AuthzURLs {
		z, err := client.GetAuthorization(context.TODO(), u)
		if err != nil {
			return "", "", false, fmt.Errorf("GetAuthorization(%q): %v", u, err)
		}
		l.Infof("Authorizations: %+v", z)
		if z.Status == acme.StatusValid {
			chal := getDnsChallenge(z)
			dnsToken, err := client.DNS01ChallengeRecord(chal.Token)
			if err != nil {
				return "", "", true, fmt.Errorf("unable to get dns token from challenge")
			}
			domainInfo.LEInfo.ChallengeURL = chal.URI
			domainInfo.LEVerification.VerificationKey = dnsToken
			domainInfo.LEVerification.VerificationZone = zone
			err = dbStorage.PutDomainInfo(domainInfo)
			if err != nil {
				return "", "", true, err
			}
			return "", "", true, fmt.Errorf("dns name is always validated, not creating a new request")
		}
		if z.Status != acme.StatusPending && z.Status != acme.StatusInvalid {
			l.Infof("authz status is %q; skipping", z.Status)
			continue
		}
		var dnsToken string
		var chal *acme.Challenge
		if z.Status == acme.StatusPending {
			dnsToken, chal, err = request(context.TODO(), client, z)
			if err != nil {
				return "", "", false, fmt.Errorf("unable to request certificate: %w", err)
			}
		}
		if chal == nil || dnsToken == "" {
			return "", "", false, fmt.Errorf("unable to find dns challenge")
		}
		domainInfo.LEVerification.VerificationZone = zone
		domainInfo.LEVerification.VerificationKey = dnsToken
		domainInfo.LEInfo.OrderURL = authOrder.URI
		domainInfo.LEInfo.ChallengeURL = chal.URI
		domainInfo.LEInfo.AuthzURL = z.URI
		domainInfo.LEInfo.FinalizeURL = authOrder.FinalizeURL
		err = dbStorage.PutDomainInfo(domainInfo)
		if err != nil {
			return "", "", false, err
		}
		zurls = append(zurls, z.URI)
		l.Infof("authorized for %+v", z.Identifier)
		return domainInfo.LEVerification.VerificationZone, domainInfo.LEVerification.VerificationKey, false, nil
	}

	return "", "", false, fmt.Errorf("no dns01 challenges to complete")

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
