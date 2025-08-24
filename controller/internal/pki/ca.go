package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

func GeneratePrivateKeyForCertificate() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)

	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func GenerateRootCerificate(privateKey *rsa.PrivateKey) (*x509.Certificate, error) {
	serialNumber, err := generateSerialNumber()

	if err != nil {
		return nil, err
	}

	now := time.Now()

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "my-postgres-operator-ca",
		},
		NotBefore:             now.Add(time.Hour * -1),
		NotAfter:              now.Add(time.Hour * 24 * 365 * 10),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true, // no intermediate certificates
		SignatureAlgorithm:    x509.SHA256WithRSA,
	}

	rawCertificate, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)

	if err != nil {
		return nil, err
	}

	certificate, err := x509.ParseCertificate(rawCertificate)

	if err != nil {
		return nil, err
	}

	return certificate, nil

}

func GenerateLeafCertificate(
	signerCertificate *x509.Certificate, signerPrivateKey *rsa.PrivateKey,
	signeePublicKey *rsa.PublicKey,
	commanName string, dnsNames []string,
) (*x509.Certificate, error) {
	serialNumber, err := generateSerialNumber()

	if err != nil {
		return nil, err
	}
	now := time.Now()

	template := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		NotBefore:             now.Add(time.Hour * -1),
		NotAfter:              now.Add(time.Hour * 24 * 365),
		SerialNumber:          serialNumber,
		Subject: pkix.Name{
			CommonName: commanName,
		},
		DNSNames: dnsNames,
	}

	certificateRaw, err := x509.CreateCertificate(
		rand.Reader,
		template,
		signerCertificate,
		signeePublicKey,
		signerPrivateKey,
	)

	if err != nil {
		return nil, err
	}

	cerficate, err := x509.ParseCertificate(certificateRaw)

	if err != nil {
		return nil, err
	}

	return cerficate, nil
}

func generateSerialNumber() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func MarshalCertificateToPEM(cert *x509.Certificate) ([]byte, error) {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}), nil
}

func MarshalPrivateKeyToPEM(key *rsa.PrivateKey) ([]byte, error) {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}), nil
}
