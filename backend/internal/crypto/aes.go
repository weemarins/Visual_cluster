package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// EncryptAES cifra dados usando AES-256-GCM.
func EncryptAES(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("chave AES deve ter 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAES decifra dados usando AES-256-GCM.
func DecryptAES(key, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("chave AES deve ter 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aesGCM.NonceSize() {
		return nil, errors.New("ciphertext inválido")
	}
	nonce := ciphertext[:aesGCM.NonceSize()]
	data := ciphertext[aesGCM.NonceSize():]
	plaintext, err := aesGCM.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

