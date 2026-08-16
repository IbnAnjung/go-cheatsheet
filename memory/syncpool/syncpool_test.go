package syncpool

import (
	"testing"
)

func TestProcessData(t *testing.T) {
	data := []byte("hello")
	expected := "hello - processed"

	result := ProcessData(data)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// BenchmarkProcessData digunakan untuk melihat perbedaan beban alokasi memori (Memory Allocation).
// Jalankan dengan: go test -bench=. -benchmem
func BenchmarkProcessData(b *testing.B) {
	data := []byte("benchmark data for testing sync pool")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessData(data)
	}
}

func BenchmarkProcessDataWithSyncPool(b *testing.B) {
	data := []byte("benchmark data for testing sync pool")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProccesDataWithSyncPool(data)
	}
}
