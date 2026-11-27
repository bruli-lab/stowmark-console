package store

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

var (
	errNoPEMBlock        = errors.New("no PEM block found")
	errUnsupportedKeyPEM = errors.New("unsupported key PEM type")
	errNotAPublicKey     = errors.New("the key has no public part")
)

func keyFingerprint(publicKey, privateKey *string) string {
	for _, path := range []*string{publicKey, privateKey} {
		if path == nil {
			continue
		}
		if fingerprint, err := fingerprintOf(*path); err == nil {
			return fingerprint
		}
	}

	return ""
}

func fingerprintOf(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	public, err := parsePublicKey(content)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(der)

	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), nil
}

func parsePublicKey(content []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, errNoPEMBlock
	}

	switch block.Type {
	case "PUBLIC KEY":
		return x509.ParsePKIXPublicKey(block.Bytes)
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "PRIVATE KEY":
		private, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		signer, ok := private.(crypto.Signer)
		if !ok {
			return nil, errNotAPublicKey
		}
		return signer.Public(), nil
	}

	return nil, fmt.Errorf("%w %q", errUnsupportedKeyPEM, block.Type)
}
