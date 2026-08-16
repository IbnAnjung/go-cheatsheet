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
		ProcessDataWithSyncPool(data)
	}
}

func TestProcessDataWithSyncPool(t *testing.T) {
	data := []byte("hello")
	expected := "hello - processed"

	result := ProcessDataWithSyncPool(data)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestProcessDataWithSyncPool_Reuse memastikan buffer yang di-reuse dari pool
// tidak menyebabkan data corruption (data dari eksekusi sebelumnya ikut tercetak).
func TestProcessDataWithSyncPool_Reuse(t *testing.T) {
	first := ProcessDataWithSyncPool([]byte("first"))
	second := ProcessDataWithSyncPool([]byte("second"))

	if first != "first - processed" {
		t.Errorf("Expected %q, got %q", "first - processed", first)
	}
	if second != "second - processed" {
		t.Errorf("Expected %q, got %q", "second - processed", second)
	}
}
