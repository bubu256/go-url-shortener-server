package shortener

import (
	_ "net/http/pprof"
	"testing"
)

func BenchmarkGetNewKey(b *testing.B) {
	shortener := &Shortener{
		rndSymbolsEnd: 3,
		lastID:        NewCounter(0),
	}
	shortener.lastID.Run()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shortener.getNewKey()
	}
}

func BenchmarkV2GetNewKey(b *testing.B) {
	shortener := &Shortener{
		rndSymbolsEnd: 3,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shortener.getNewKeyV2()
	}
}
