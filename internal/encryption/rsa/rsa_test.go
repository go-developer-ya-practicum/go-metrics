package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRSAEncryption(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	plaintext := []byte("secret message")

	e := &Encrypter{publicKey: &privateKey.PublicKey}
	d := &Decrypter{privateKey: privateKey}

	ciphertext, err := e.Encrypt(plaintext)
	require.NoError(t, err)

	plaintextDecrypted, err := d.Decrypt(ciphertext)
	require.NoError(t, err)

	require.Equal(t, plaintext, plaintextDecrypted)
}
