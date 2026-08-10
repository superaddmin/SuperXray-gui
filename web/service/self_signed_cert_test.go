package service

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"reflect"
	"testing"
)

func TestGetNewSelfSignedCertCreatesServerCertificate(t *testing.T) {
	service := &ServerService{}
	result, err := service.GetNewSelfSignedCert("Example.COM,127.0.0.1,example.com")
	if err != nil {
		t.Fatalf("GetNewSelfSignedCert failed: %v", err)
	}

	certificateBlock, _ := pem.Decode([]byte(result.Cert))
	if certificateBlock == nil || certificateBlock.Type != "CERTIFICATE" {
		t.Fatal("certificate response is not PEM encoded")
	}
	certificate, err := x509.ParseCertificate(certificateBlock.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	if certificate.IsCA {
		t.Fatal("generated server certificate must not be a CA")
	}
	if !reflect.DeepEqual(certificate.DNSNames, []string{"example.com"}) {
		t.Fatalf("DNSNames = %#v", certificate.DNSNames)
	}
	if len(certificate.IPAddresses) != 1 || !certificate.IPAddresses[0].Equal(net.ParseIP("127.0.0.1")) {
		t.Fatalf("IPAddresses = %#v", certificate.IPAddresses)
	}

	privateKeyBlock, _ := pem.Decode([]byte(result.Key))
	if privateKeyBlock == nil || privateKeyBlock.Type != "PRIVATE KEY" {
		t.Fatal("private key response is not PKCS#8 PEM encoded")
	}
	privateKeyValue, err := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}
	privateKey, ok := privateKeyValue.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatalf("private key type = %T", privateKeyValue)
	}
	publicKey, ok := certificate.PublicKey.(*ecdsa.PublicKey)
	if !ok || privateKey.PublicKey.X.Cmp(publicKey.X) != 0 || privateKey.PublicKey.Y.Cmp(publicKey.Y) != 0 {
		t.Fatal("certificate public key does not match the private key")
	}
}

func TestNormalizeSelfSignedServerNamesUsesLocalDefaults(t *testing.T) {
	names, err := normalizeSelfSignedServerNames("  ")
	if err != nil {
		t.Fatalf("normalize defaults: %v", err)
	}
	if !reflect.DeepEqual(names, []string{"localhost", "127.0.0.1", "::1"}) {
		t.Fatalf("names = %#v", names)
	}
}

func TestNormalizeSelfSignedServerNamesRejectsInvalidDNSNames(t *testing.T) {
	for _, value := range []string{"bad name", "-example.com", "example..com", "foo.*.example.com"} {
		if _, err := normalizeSelfSignedServerNames(value); err == nil {
			t.Fatalf("accepted invalid server name %q", value)
		}
	}
}
