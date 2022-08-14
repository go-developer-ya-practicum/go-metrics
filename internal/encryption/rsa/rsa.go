// Package rsa предназначен для шифрования данных с помощью алгоритма RSA
package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"io/ioutil"
)

type Encrypter struct {
	publicKey *rsa.PublicKey
}

func NewEncrypter(path string) (*Encrypter, error) {
	if path == "" {
		return nil, nil
	}

	keyData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	publicKey, err := DecodePublicKey(keyData)
	if err != nil {
		return nil, err
	}
	return &Encrypter{publicKey: publicKey}, nil
}

func (e *Encrypter) Encrypt(plaintext []byte) ([]byte, error) {
	if e == nil {
		return plaintext, nil
	}

	hash := sha512.New()
	step := e.publicKey.Size() - 2*hash.Size() - 2
	var encryptedBytes []byte
	for start := 0; start < len(plaintext); start += step {
		finish := start + step
		if finish > len(plaintext) {
			finish = len(plaintext)
		}

		encryptedBlockBytes, err := rsa.EncryptOAEP(
			hash, rand.Reader, e.publicKey, plaintext[start:finish], nil)
		if err != nil {
			return nil, err
		}

		encryptedBytes = append(encryptedBytes, encryptedBlockBytes...)
	}
	return encryptedBytes, nil
}

type Decrypter struct {
	privateKey *rsa.PrivateKey
}

func NewDecrypter(path string) (*Decrypter, error) {
	if path == "" {
		return nil, nil
	}

	keyData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	privateKey, err := DecodePrivateKey(keyData)
	if err != nil {
		return nil, err
	}

	return &Decrypter{privateKey: privateKey}, nil
}

func (d *Decrypter) Decrypt(ciphertext []byte) ([]byte, error) {
	if d == nil {
		return ciphertext, nil
	}

	hash := sha512.New()
	step := d.privateKey.PublicKey.Size()
	var decryptedBytes []byte
	for start := 0; start < len(ciphertext); start += step {
		finish := start + step
		if finish > len(ciphertext) {
			finish = len(ciphertext)
		}

		decryptedBlockBytes, err := rsa.DecryptOAEP(
			hash, rand.Reader, d.privateKey, ciphertext[start:finish], nil)
		if err != nil {
			return nil, err
		}
		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}
	return decryptedBytes, nil
}
