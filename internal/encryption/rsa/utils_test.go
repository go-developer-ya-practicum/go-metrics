package rsa

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodePublicKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	var publicKeyPEM bytes.Buffer
	err = pem.Encode(&publicKeyPEM, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})
	require.NoError(t, err)

	publicKey, err := DecodePublicKey(publicKeyPEM.Bytes())
	require.NoError(t, err)
	require.Equal(t, &privateKey.PublicKey, publicKey)
}

func TestDecodePrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	var privateKeyPEM bytes.Buffer
	err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	require.NoError(t, err)

	privateKeyDecoded, err := DecodePrivateKey(privateKeyPEM.Bytes())
	require.NoError(t, err)
	require.Equal(t, privateKey, privateKeyDecoded)
}
