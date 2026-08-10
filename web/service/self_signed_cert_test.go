package service

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"reflect"
	"strings"
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
	if !ok || !privateKey.PublicKey.Equal(publicKey) {
		t.Fatal("certificate public key does not match the private key")
	}
	if _, err := tls.X509KeyPair([]byte(result.Cert), []byte(result.Key)); err != nil {
		t.Fatalf("load generated TLS key pair: %v", err)
	}
	if err := certificate.VerifyHostname("example.com"); err != nil {
		t.Fatalf("verify generated certificate hostname: %v", err)
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

func TestNormalizeSelfSignedServerNamesEnforcesEntryAndLengthLimits(t *testing.T) {
	validNames := make([]string, maxSelfSignedNames)
	for index := range validNames {
		validNames[index] = fmt.Sprintf("host-%d.example.com", index)
	}
	if _, err := normalizeSelfSignedServerNames(strings.Join(validNames, ",")); err != nil {
		t.Fatalf("accepted names at entry limit: %v", err)
	}

	tooManyNames := append(validNames, "extra.example.com")
	if _, err := normalizeSelfSignedServerNames(strings.Join(tooManyNames, ",")); err == nil {
		t.Fatal("expected names above entry limit to be rejected")
	}

	labels := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." +
		strings.Repeat("c", 63) + "." + strings.Repeat("d", 61)
	exactLimit := strings.Join([]string{labels, labels, labels, labels}, ",") + strings.Repeat(" ", 9)
	if len(exactLimit) != maxSelfSignedNamesLength {
		t.Fatalf("test input length = %d", len(exactLimit))
	}
	if _, err := normalizeSelfSignedServerNames(exactLimit); err != nil {
		t.Fatalf("accepted names at length limit: %v", err)
	}
	if _, err := normalizeSelfSignedServerNames(exactLimit + " "); err == nil {
		t.Fatal("expected names above length limit to be rejected")
	}
}
