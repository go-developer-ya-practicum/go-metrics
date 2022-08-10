// Package encryption предназначен для шифрования данных
package encryption

type Encrypter interface {
	Encrypt(plaintext []byte) (ciphertext []byte, err error)
}

type Decrypter interface {
	Decrypt(ciphertext []byte) (plaintext []byte, err error)
}
