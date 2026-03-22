package securepayload

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub := &priv.PublicKey

	plaintext := []byte("hello gzip-like payload " + string(bytes.Repeat([]byte("x"), 500)))
	enc, err := Encrypt(pub, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := Decrypt(priv, enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, dec) {
		t.Fatalf("plaintext mismatch")
	}
}
