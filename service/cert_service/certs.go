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

func CompleteCertificateRequest(userId string, domain string, email string) ([][]byte, error) {
	privateKey, err := getRequestUserCert()
	if err != nil {
		return nil, fmt.Errorf("unable to requestUserCert(): %w", err)
	}

	// Create our client which will interact with the acme api
	client := &acme.Client{
		Key:          privateKey,
		DirectoryURL: cfg.LESettings.CADirURL,
	}

	// Retrieve our domain info from the database
	domainInfo, err := dbStorage.GetDomainByUser(userId, domain)
	if err != nil {
		return nil, fmt.Errorf("domain: %s unable to get DomainInfo from database: %w", domain, err)
	}

	identifiers := acme.DomainIDs(domain)
	authOrder, err := client.AuthorizeOrder(context.TODO(), identifiers)
	if err != nil {
		return nil, fmt.Errorf("AuthorizeOrder: %v", err)
	}

	authz, err := client.GetAuthorization(context.TODO(), domainInfo.OrderURL)
	if err != nil {
		return nil, err
	}

	var chal *acme.Challenge
	for i, c := range authz.Challenges {
		l.Infof("challenge %d: %+v", i, c)
		if c.Type == challenge.DNS01.String() {
			l.Infof("picked %s for authz %s", c.URI, authz.URI)
			chal = c
		}
	}
	if chal == nil {
		return nil, fmt.Errorf("challenge type %q wasn't offered for authz %s", challenge.DNS01, authz.URI)
	}

	err = completeDNS01(context.TODO(), client, authz, chal, authOrder)
	if err != nil {
		return nil, err
	}

	csr, privateKey := newCSR(identifiers)
	ders, curl, err := client.CreateOrderCert(context.TODO(), authOrder.FinalizeURL, csr, true)
	if err != nil {
		return nil, fmt.Errorf("CreateOrderCert: %v", err)
	}
	domainInfo.CertURL = curl
	err = dbStorage.PutDomainInfo(domainInfo)
	if err != nil {
		return nil, err
	}
	l.Infof("cert URL: %s", curl)

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to encode PEM block: %v\n", err)
	}

	err = fileStorage.SaveBuf(privBuf, fmt.Sprintf("mastodon_le_certs/%s/cert.key", domain))
	if err != nil {
		return nil, err
	}

	var mergedDers []byte
	for _, slice := range ders {
		mergedDers = append(mergedDers, slice...)
	}
	secondBuf := bytes.NewBuffer(mergedDers)
	err = fileStorage.SaveBuf(*secondBuf, fmt.Sprintf("mastodon_le_certs/%s/cert.crt", domain))
	if err != nil {
		return nil, err
	}

	return ders, nil
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

func runDNS01(ctx context.Context, client *acme.Client, z *acme.Authorization, chal *acme.Challenge) (string, string, error) {
	token, err := client.DNS01ChallengeRecord(chal.Token)
	if err != nil {
		return "", "", fmt.Errorf("DNS01ChallengeRecord: %v", err)
	}

	//if _, err := client.Accept(ctx, chal); err != nil {
	//	return "", "", fmt.Errorf("accept(%q): %v", chal.URI, err)
	//}

	return fmt.Sprintf("_acme-challenge.%s", z.Identifier.Value), token, nil
}

func completeDNS01(ctx context.Context, client *acme.Client, z *acme.Authorization, chal *acme.Challenge, order *acme.Order) error {
	if _, err := client.Accept(ctx, chal); err != nil {
		return fmt.Errorf("accept(%q): %v", chal.URI, err)
	}

	_, err := client.WaitAuthorization(context.TODO(), z.URI)
	if err != nil {
		return err
	}

	l.Infof("all challenges are done")
	if _, err := client.WaitOrder(ctx, order.URI); err != nil {
		return fmt.Errorf("waitOrder(%q): %v", order.URI, err)
	}

	return nil
}

func request(ctx context.Context, client *acme.Client, z *acme.Authorization) (string, string, error) {
	var chal *acme.Challenge
	for i, c := range z.Challenges {
		l.Infof("challenge %d: %+v", i, c)
		if c.Type == challenge.DNS01.String() {
			l.Infof("picked %s for authz %s", c.URI, z.URI)
			chal = c
		}
	}
	if chal == nil {
		return "", "", fmt.Errorf("challenge type %q wasn't offered for authz %s", challenge.DNS01, z.URI)
	}

	zone, token, err := runDNS01(ctx, client, z, chal)
	if err != nil {
		return "", "", err
	}
	return zone, token, nil
}

func RequestCertificate(userId string, domain string, email string) (string, string, error) {

	// Get the private key for the cert administrator account
	privateKey, err := getRequestUserCert()
	if err != nil {
		return "", "", fmt.Errorf("unable to requestUserCert(): %w", err)
	}

	// Create our client which will interact with the acme api
	client := &acme.Client{
		Key:          privateKey,
		DirectoryURL: cfg.LESettings.CADirURL,
	}

	// Retrieve our domain info from the database
	domainInfo, err := dbStorage.GetDomainByUser(userId, domain)
	if err != nil {
		return "", "", fmt.Errorf("domain: %s unable to get DomainInfo from database: %w", domain, err)
	}

	// Log in and get our account information
	//var account *acme.Account
	if domainInfo.CertURL != "" {
		// the url parameter is legacy and not used
		_, err = client.GetReg(context.TODO(), "")
		if err != nil {
			return "", "", err
		}
	} else {
		_, err = client.UpdateReg(context.TODO(), &acme.Account{})
		if err != nil {
			return "", "", err
		}
	}

	// Identifiers is an acme construct for our domain names
	identifiers := acme.DomainIDs(domain)

	// We create an order for our domain name, this is key to authorizing everything
	// this kicks off the process
	authOrder, err := client.AuthorizeOrder(context.TODO(), identifiers)
	if err != nil {
		return "", "", err
	}

	var zurls []string
	for _, u := range authOrder.AuthzURLs {
		z, err := client.GetAuthorization(context.TODO(), u)
		if err != nil {
			return "", "", fmt.Errorf("GetAuthorization(%q): %v", u, err)
		}
		l.Infof("Authorizations: %+v", z)
		if z.Status != acme.StatusPending {
			l.Infof("authz status is %q; skipping", z.Status)
			continue
		}
		zone, token, err := request(context.TODO(), client, z)
		domainInfo.LEVerification.VerificationZone = zone
		domainInfo.LEVerification.VerificationKey = token
		domainInfo.OrderURL = u
		domainInfo.CertURL = z.URI
		err = dbStorage.PutDomainInfo(domainInfo)
		if err != nil {
			return "", "", err
		}
		zurls = append(zurls, z.URI)
		l.Infof("authorized for %+v", z.Identifier)
		return domainInfo.LEVerification.VerificationZone, domainInfo.LEVerification.VerificationKey, nil
	}

	return "", "", fmt.Errorf("no dns01 challenges to complete")

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
