package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	stdnet "net"
	"strings"
	"time"
)

const (
	maxSelfSignedNames       = 16
	maxSelfSignedNamesLength = 1024
)

// SelfSignedCertificate contains a PEM encoded certificate and private key.
type SelfSignedCertificate struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

// GetNewSelfSignedCert creates a short-lived server certificate for the requested SANs.
func (s *ServerService) GetNewSelfSignedCert(serverNames string) (*SelfSignedCertificate, error) {
	names, err := normalizeSelfSignedServerNames(serverNames)
	if err != nil {
		return nil, err
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, fmt.Errorf("generate certificate serial: %w", err)
	}
	if serialNumber.Sign() == 0 {
		serialNumber = big.NewInt(1)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: names[0]},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, name := range names {
		if ip := stdnet.ParseIP(name); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, name)
		}
	}

	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("encode private key: %w", err)
	}

	return &SelfSignedCertificate{
		Cert: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})),
		Key:  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER})),
	}, nil
}

func normalizeSelfSignedServerNames(serverNames string) ([]string, error) {
	if len(serverNames) > maxSelfSignedNamesLength {
		return nil, fmt.Errorf("server names exceed %d characters", maxSelfSignedNamesLength)
	}

	if strings.TrimSpace(serverNames) == "" {
		return []string{"localhost", "127.0.0.1", "::1"}, nil
	}

	seen := make(map[string]struct{})
	names := make([]string, 0, 4)
	for _, rawName := range strings.Split(serverNames, ",") {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		if stdnet.ParseIP(name) == nil {
			name = strings.TrimSuffix(strings.ToLower(name), ".")
			if err := validateCertificateDNSName(name); err != nil {
				return nil, err
			}
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
		if len(names) > maxSelfSignedNames {
			return nil, fmt.Errorf("server names exceed %d entries", maxSelfSignedNames)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("at least one valid server name is required")
	}
	return names, nil
}

func validateCertificateDNSName(name string) error {
	if name == "" || len(name) > 253 {
		return fmt.Errorf("invalid DNS server name %q", name)
	}
	labels := strings.Split(name, ".")
	for index, label := range labels {
		if label == "*" && index == 0 && len(labels) > 1 {
			continue
		}
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("invalid DNS server name %q", name)
		}
		for _, char := range label {
			if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
				continue
			}
			return fmt.Errorf("invalid DNS server name %q", name)
		}
	}
	return nil
}
