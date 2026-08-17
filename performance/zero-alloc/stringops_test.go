package pprof

import (
	"testing"
)

// generateNames membuat slice berisi N string pendek untuk testing.
func generateNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = "user_" + string(rune('A'+i%26))
	}
	return names
}

// generateTransactions membuat slice berisi N transaksi untuk testing.
func generateTransactions(n int) []Transaction {
	txs := make([]Transaction, n)
	for i := range txs {
		txs[i] = Transaction{
			ID:     i + 1,
			User:   "user_" + string(rune('A'+i%26)),
			Amount: float64(i) * 99.99,
			Status: "completed",
		}
	}
	return txs
}

// --- Unit Tests ---

func TestProcessData_Good(t *testing.T) {
	input := []string{"alice", "bob", "charlie"}
	expected := "alice,bob,charlie"

	got := ProcessData_Good(input)
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestProcessData_Good_Empty(t *testing.T) {
	got := ProcessData_Good([]string{})
	if got != "" {
		t.Errorf("Expected empty string, got %q", got)
	}
}

func TestGenerateReport_Good(t *testing.T) {
	txs := []Transaction{
		{ID: 1, User: "Alice", Amount: 100.50, Status: "completed"},
	}

	expected := GenerateReport_Bad(txs)
	got := GenerateReport_Good(txs)

	if got != expected {
		t.Errorf("Output mismatch.\nExpected:\n%s\nGot:\n%s", expected, got)
	}
}

// --- Benchmarks ---

const benchSize = 10_000

func BenchmarkProcessData_Bad(b *testing.B) {
	names := generateNames(benchSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ProcessData_Bad(names)
	}
}

func BenchmarkProcessData_Good(b *testing.B) {
	names := generateNames(benchSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ProcessData_Good(names)
	}
}

func BenchmarkGenerateReport_Bad(b *testing.B) {
	txs := generateTransactions(benchSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateReport_Bad(txs)
	}
}

func BenchmarkGenerateReport_Good(b *testing.B) {
	txs := generateTransactions(benchSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateReport_Good(txs)
	}
}

func BenchmarkGenerateReport_Extreme(b *testing.B) {
	txs := generateTransactions(benchSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateReport_Extreme(txs)
	}
}

