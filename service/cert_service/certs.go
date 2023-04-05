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
	"github.com/edwinavalos/common/config"
	"github.com/edwinavalos/common/logger"
	"github.com/edwinavalos/common/models"
	"github.com/edwinavalos/dns-verifier/service/domain_service"
	"github.com/edwinavalos/dns-verifier/storage"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/registration"
	"github.com/google/uuid"
	"golang.org/x/crypto/acme"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Service struct {
	fileStorage   *storage.VerifierFileStore
	domainService *domain_service.Service
	cfg           *config.Config
}

func New(conf *config.Config, fileStorage *storage.VerifierFileStore, domainService *domain_service.Service) *Service {
	return &Service{
		fileStorage:   fileStorage,
		domainService: domainService,
		cfg:           conf,
	}
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

func (s *Service) getRequestUserCert() (*ecdsa.PrivateKey, error) {
	var privateKey *ecdsa.PrivateKey
	_, err := os.Stat(s.cfg.LEPrivateKeyLocation())
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
		err4 := os.WriteFile(s.cfg.LEPrivateKeyLocation(), keyBytes, 0644)
		if err4 != nil {
			return nil, fmt.Errorf("problem writing private key file: %w", err4)
		}
	} else {
		dBytes, err2 := os.ReadFile(s.cfg.LEPrivateKeyLocation())
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

func (s *Service) recordInPlace(domainInfo models.DomainInformation) bool {
	if domainInfo.Verification.Key == "" || domainInfo.Verification.Zone == "" {
		return false
	}

	found, err := s.domainService.VerifyTXTRecord(context.TODO(), domainInfo.Verification.Zone, domainInfo.Verification.Key)
	if err != nil || !found {
		return false
	}

	return true
}

func (s *Service) CompleteCertificateRequest(userID uuid.UUID, domain string, email string) error {
	// Retrieve our domain info from the database
	domainInfo, err := s.domainService.GetDomainByUser(context.TODO(), userID, domain)
	if err != nil {
		return fmt.Errorf("domain: %s unable to get DomainInfo from database: %w", domain, err)
	}

	inPlace := s.recordInPlace(domainInfo)
	if !inPlace {
		return fmt.Errorf("unable to verify that your record is in place before completing request")
	}

	privateKey, err := s.getRequestUserCert()
	if err != nil {
		return fmt.Errorf("unable to requestUserCert(): %w", err)
	}

	// Create our client which will interact with the acme api
	client := &acme.Client{
		Key:          privateKey,
		DirectoryURL: s.cfg.LECADirURL(),
	}

	certInfo := domainInfo.Verification.CertInfo
	if domainInfo.Verification.Verified == true && certInfo.CertURL != "" {
		certs, err := client.FetchCert(context.TODO(), certInfo.CertURL, true)
		if err != nil {
			return err
		}

		err2 := s.WriteToStorage(privateKey, domain, certs)
		if err2 != nil {
			return err2
		}
		return nil
	}

	if certInfo.OrderURL == "" {
		return fmt.Errorf("missing order_url")
	}

	identifiers := acme.DomainIDs(domain)
	authOrder, err := client.GetOrder(context.TODO(), certInfo.OrderURL)
	if err != nil || authOrder.Status == acme.StatusInvalid {
		return fmt.Errorf("AuthorizeOrder: %v", err)
	}
	// we set this because get order doesn't populate the URI value
	authOrder.URI = certInfo.OrderURL

	if certInfo.AuthzURL == "" {
		return fmt.Errorf("missing authz_url")
	}
	authz, err := client.GetAuthorization(context.TODO(), certInfo.AuthzURL)
	if err != nil {
		return err
	}

	if certInfo.ChallengeURL == "" {
		return fmt.Errorf("missing challenge_url")
	}
	chal, err := client.GetChallenge(context.TODO(), certInfo.ChallengeURL)
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
	certInfo.CertURL = curl
	domainInfo.Verification.Verified = true
	domainInfo.Verification.CertInfo = certInfo
	err = s.domainService.PutDomain(context.TODO(), domainInfo)
	if err != nil {
		return err
	}
	logger.Info("cert URL: %s", curl)

	err2 := s.WriteToStorage(privateKey, domain, ders)
	if err2 != nil {
		return err2
	}

	return nil
}

func (s *Service) WriteToStorage(privateKey *ecdsa.PrivateKey, domain string, ders [][]byte) error {
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

	err = s.fileStorage.SaveBuf(privBuf, fmt.Sprintf("mastodon_le_certs/%s/cert.key", domain))
	if err != nil {
		return err
	}

	var mergedDers []byte
	for _, slice := range ders {
		mergedDers = append(mergedDers, slice...)
	}
	secondBuf := bytes.NewBuffer(mergedDers)
	err = s.fileStorage.SaveBuf(*secondBuf, fmt.Sprintf("mastodon_le_certs/%s/cert.crt", domain))
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
	logger.Info("%+v", newChal)

	_, err = client.WaitAuthorization(context.TODO(), z.URI)
	if err != nil {
		return err
	}

	logger.Info("all challenges are done")
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
		logger.Info("challenge %d: %+v", i, c)
		if c.Type == challenge.DNS01.String() {
			logger.Info("picked %s for authz %s", c.URI, z.URI)
			chal = c
		}
	}
	return chal
}

func (s *Service) RequestCertificate(userId uuid.UUID, domain string, email string) (string, string, bool, error) {

	// Get the private key for the cert administrator account
	privateKey, err := s.getRequestUserCert()
	if err != nil {
		return "", "", false, fmt.Errorf("unable to requestUserCert(): %w", err)
	}

	// Create our client which will interact with the acme api
	client := &acme.Client{
		Key:          privateKey,
		DirectoryURL: s.cfg.LECADirURL(),
	}

	// Retrieve our domain info from the database
	domainInfo, err := s.domainService.GetDomainByUser(context.TODO(), userId, domain)
	if err != nil {
		return "", "", false, fmt.Errorf("domain: %s unable to get DomainInfo from database: %w", domain, err)
	}

	// Log in and get our account information
	//var account *acme.Account
	certInfo := domainInfo.Verification.CertInfo
	if certInfo.CertURL != "" {
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
	if certInfo.OrderURL == "" {
		authOrder, err = client.AuthorizeOrder(context.TODO(), identifiers)
		if err != nil {
			return "", "", false, err
		}
	} else {
		authOrder, err = client.GetOrder(context.TODO(), certInfo.OrderURL)
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
			authOrder.URI = certInfo.OrderURL
		}
	}
	if authOrder == nil {
		return "", "", false, fmt.Errorf("unable to find auth order in common or unable to authorize new order")
	}

	var zurls []string
	zone := fmt.Sprintf("_acme-challenge.%s", domain)
	for _, u := range authOrder.AuthzURLs {
		z, err := client.GetAuthorization(context.TODO(), u)
		if err != nil {
			return "", "", false, fmt.Errorf("GetAuthorization(%q): %v", u, err)
		}
		logger.Info("Authorizations: %+v", z)
		if z.Status == acme.StatusValid {
			chal := getDnsChallenge(z)
			dnsToken, err := client.DNS01ChallengeRecord(chal.Token)
			if err != nil {
				return "", "", true, fmt.Errorf("unable to get dns token from challenge")
			}
			certInfo.ChallengeURL = chal.URI
			domainInfo.Verification.Key = dnsToken
			domainInfo.Verification.Zone = zone
			domainInfo.Verification.CertInfo = certInfo
			err = s.domainService.PutDomain(context.TODO(), domainInfo)
			if err != nil {
				return "", "", true, err
			}
			return "", "", true, fmt.Errorf("dns name is always validated, not creating a new request")
		}
		if z.Status != acme.StatusPending && z.Status != acme.StatusInvalid {
			logger.Info("authz status is %q; skipping", z.Status)
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
		domainInfo.Verification.Zone = zone
		domainInfo.Verification.Key = dnsToken
		certInfo.OrderURL = authOrder.URI
		certInfo.ChallengeURL = chal.URI
		certInfo.AuthzURL = z.URI
		certInfo.FinalizeURL = authOrder.FinalizeURL
		domainInfo.Verification.CertInfo = certInfo
		err = s.domainService.PutDomain(context.TODO(), domainInfo)
		if err != nil {
			return "", "", false, err
		}
		zurls = append(zurls, z.URI)
		logger.Info("authorized for %+v", z.Identifier)
		return domainInfo.Verification.Zone, domainInfo.Verification.Key, false, nil
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
	logger.Info("Saving public key to: %s", publicKeyLocation)
	err := os.WriteFile(publicKeyLocation, certificates.Certificate, 0644)
	if err != nil {
		return err
	}

	logger.Info("Saving private key to: %s", privateKeyLocation)
	err = os.WriteFile(privateKeyLocation, certificates.PrivateKey, 0600)
	if err != nil {
		return err
	}

	logger.Info("Touching generated file: %s", generatedLocation)
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
