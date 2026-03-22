package securepayload

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	formatVersion = uint8(1)
	nonceSize     = 12
	aesKeySize    = 32
)

// HTTPHeaderEncrypted — заголовок запроса: тело зашифровано (RSA-OAEP ключа AES + AES-GCM).
const HTTPHeaderEncrypted = "X-Encrypted"

// Encrypt hybrid: RSA-OAEP encrypts random AES-256 key, AES-GCM encrypts plaintext.
func Encrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	if pub == nil {
		return nil, errors.New("public key is nil")
	}
	aesKey := make([]byte, aesKeySize)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	rsaCipher, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, err
	}

	rsaLen := uint32(len(rsaCipher))
	if int(rsaLen) != len(rsaCipher) {
		return nil, errors.New("rsa ciphertext length overflow")
	}

	var out []byte
	out = append(out, formatVersion)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, rsaLen)
	out = append(out, buf...)
	out = append(out, rsaCipher...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// Decrypt reverses Encrypt.
func Decrypt(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	if priv == nil {
		return nil, errors.New("private key is nil")
	}
	if len(data) < 1+4 {
		return nil, errors.New("payload too short")
	}
	if data[0] != formatVersion {
		return nil, fmt.Errorf("unsupported format version: %d", data[0])
	}
	rsaLen := binary.BigEndian.Uint32(data[1:5])
	offset := 5
	if len(data) < offset+int(rsaLen)+nonceSize {
		return nil, errors.New("truncated rsa segment")
	}
	rsaCipher := data[offset : offset+int(rsaLen)]
	offset += int(rsaLen)
	nonce := data[offset : offset+nonceSize]
	offset += nonceSize
	aesCiphertext := data[offset:]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, rsaCipher, nil)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, aesCiphertext, nil)
}
