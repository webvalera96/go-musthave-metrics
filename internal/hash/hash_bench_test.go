package hash

import (
	"testing"
)

func BenchmarkCalculateHash_Small(b *testing.B) {
	body := []byte("small body")
	key := "secret"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateHash(body, key)
	}
}

func BenchmarkCalculateHash_Medium(b *testing.B) {
	body := make([]byte, 1024)
	for i := range body {
		body[i] = byte(i)
	}
	key := "secret"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateHash(body, key)
	}
}

func BenchmarkCalculateHash_Large(b *testing.B) {
	body := make([]byte, 64*1024)
	for i := range body {
		body[i] = byte(i)
	}
	key := "secret"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateHash(body, key)
	}
}
