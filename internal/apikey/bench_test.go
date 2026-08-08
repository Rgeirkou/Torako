package apikey

import "testing"

func BenchmarkHash(b *testing.B) {
	key := "w0rBkFJj6OasjVqsLK9mZ7vXqxSqIoCFY4lYoTjSWhY"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Hash(key)
	}
}

func BenchmarkGenerate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Generate()
	}
}
