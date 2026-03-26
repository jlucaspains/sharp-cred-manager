package services

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
)

// http.client singleton
var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	// disabling security here is fine
	// the purpose of the client is to pull certs
	// no data exchange is happening
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

var mockAzureResult []byte = nil

func GetConfigCerts() []models.CheckCertItem {
	result := []models.CheckCertItem{}
	for i := 1; true; i++ {
		rawUrl, ok := os.LookupEnv(fmt.Sprintf("SITE_%d", i))
		if !ok {
			break
		}
		site, err := url.Parse(rawUrl)

		if err != nil {
			continue
		}

		result = append(result, models.CheckCertItem{Name: site.Hostname(), Url: rawUrl, Type: models.CertCheckURL})
	}

	for i := 1; true; i++ {
		rawUrl, ok := os.LookupEnv(fmt.Sprintf("AZUREKEYVAULT_%d", i))
		if !ok {
			break
		}
		akvUrl, err := url.Parse(rawUrl)

		if err != nil {
			continue
		}

		name := akvUrl.Hostname() + "/" + strings.Split(akvUrl.Path, "/")[2]

		result = append(result, models.CheckCertItem{Name: name, Url: rawUrl, Type: models.CertCheckAzure})
	}

	return result
}

func CheckCertStatus(cert models.CheckCertItem, expirationWarningDays int) (*models.CertCheckResult, error) {
	if cert.Name == "" || cert.Url == "" {
		err := errors.New("name, url, and type are required")
		return nil, err
	}

	switch cert.Type {
	case models.CertCheckURL:
		return checkCertByUrlStatus(cert.Name, cert.Url, expirationWarningDays)
	case models.CertCheckAzure:
		return checkAzureCertStatus(cert.Name, cert.Url, expirationWarningDays)
	}

	return nil, errors.New("invalid type")
}

func checkCertByUrlStatus(name string, url string, expirationWarningDays int) (*models.CertCheckResult, error) {
	resp, err := client.Get(url)

	if err != nil || resp.TLS == nil {
		result := &models.CertCheckResult{Hostname: name, CertStartDate: time.Time{}, CertEndDate: time.Time{}, CertDnsNames: []string{}, IsValid: false, ValidityInDays: 0}
		return result, err
	}

	result := prepareResult(resp.TLS.PeerCertificates[0], resp.TLS.PeerCertificates[1:], name, expirationWarningDays, false)

	return result, nil
}

func checkAzureCertStatus(name string, rawUrl string, expirationWarningDays int) (*models.CertCheckResult, error) {
	parsedUrl, _ := url.Parse(rawUrl)
	keyVaultUrl := parsedUrl.Scheme + "://" + parsedUrl.Host
	certName := strings.Split(parsedUrl.Path, "/")[2]

	cer, err := getCertFromKeyVault(keyVaultUrl, certName, name, expirationWarningDays)

	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(cer)

	if err != nil {
		return nil, err
	}

	result := prepareResult(cert, []*x509.Certificate{}, name, expirationWarningDays, true)

	return result, nil
}

func prepareResult(certificate *x509.Certificate, peerCertificates []*x509.Certificate, name string, expirationWarningDays int, skipHostNameValidation bool) *models.CertCheckResult {
	isValid, errors := validate(certificate, name, skipHostNameValidation)
	certNotAfter := certificate.NotAfter.UTC()
	now := time.Now().UTC()
	return &models.CertCheckResult{
		Hostname:          name,
		Issuer:            certificate.Issuer.CommonName,
		Signature:         certificate.SignatureAlgorithm.String(),
		CertStartDate:     certificate.NotBefore,
		CertEndDate:       certificate.NotAfter,
		CertDnsNames:      certificate.DNSNames,
		TLSVersion:        uint16(certificate.Version),
		IsCA:              certificate.IsCA,
		CommonName:        certificate.Subject.CommonName,
		IsValid:           isValid,
		OtherCerts:        getOtherCerts(peerCertificates),
		ValidationIssues:  errors,
		ExpirationWarning: certNotAfter.Before(now.UTC().AddDate(0, 0, expirationWarningDays)),
		ValidityInDays:    getValidityInDays(now, certNotAfter),
	}
}

func getValidityInDays(startDate time.Time, endDate time.Time) int {
	if startDate.IsZero() || endDate.IsZero() {
		return 0
	}

	validity := endDate.Sub(startDate)
	if validity.Hours() < 0 {
		return 0
	}

	return int(validity.Hours() / 24)
}

func getCertFromKeyVault(keyVaultUrl string, certName string, hostName string, expirationWarningDays int) ([]byte, error) {
	if mockAzureResult != nil {
		return mockAzureResult, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)

	if err != nil {
		return nil, err
	}

	client, err := azcertificates.NewClient(keyVaultUrl, cred, nil)

	if err != nil {
		return nil, err
	}

	log.Printf("Getting certificate from Azure Key Vault: %s", certName)

	response, err := client.GetCertificate(context.Background(), certName, "", nil)

	if err != nil {
		return nil, err
	}

	return response.CER, nil
}

func validate(cert *x509.Certificate, hostName string, skipHostNameValidation bool) (bool, []string) {
	isHostNameValid := skipHostNameValidation || cert.VerifyHostname(hostName) == nil
	areDatesValid := cert.NotBefore.Before(time.Now().UTC()) && cert.NotAfter.After(time.Now().UTC())
	isSignatureValid := cert.SignatureAlgorithm.String() != "SHA1-RSA"
	isValid := isHostNameValid && areDatesValid && isSignatureValid

	errors := []string{}
	if !isHostNameValid {
		errors = append(errors, "Hostname is not valid")
	}
	if !areDatesValid {
		errors = append(errors, "Certificate is not valid yet or expired")
	}
	if !isSignatureValid {
		errors = append(errors, "SHA1 is not a secure signature algorithm")
	}

	return isValid, errors
}

func getOtherCerts(certs []*x509.Certificate) []models.OtherCert {
	results := []models.OtherCert{}
	for _, v := range certs {
		results = append(results, models.OtherCert{
			IsCA:       v.IsCA,
			CommonName: v.Subject.CommonName,
			Issuer:     v.Issuer.CommonName,
		})
	}

	return results
}
