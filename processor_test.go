package aggregator

import (
	"encoding/json"
	"fmt"
	"testing"
)

func generateRawLogs(n int) [][]byte {
	logs := make([][]byte, n)
	for i := range logs {
		entry := LogEntry{
			Timestamp: "2026-08-17T10:00:00Z",
			Level:     "ERROR",
			Service:   "billing-svc",
			Message:   fmt.Sprintf("Transaction %d failed to process", i),
		}
		raw, _ := json.Marshal(entry)
		logs[i] = raw
	}
	return logs
}

func TestProcessLogs_OutputMatches(t *testing.T) {
	rawLogs := generateRawLogs(5)

	badOutput := ProcessLogs_Bad(rawLogs)

	// Uncomment ini jika fungsi ProcessLogs_Good sudah Anda buat
	/*
		goodOutput := ProcessLogs_Good(rawLogs)

		for i := range badOutput {
			if badOutput[i] != goodOutput[i] {
				t.Errorf("Mismatch at index %d:\nBad : %s\nGood: %s", i, badOutput[i], goodOutput[i])
			}
		}
	*/

	// Cetak contoh hasil
	for _, out := range badOutput {
		t.Log(out)
	}
}

// === Benchmark Perbandingan ===

const benchSize = 10_000

func BenchmarkProcessLogs_Bad(b *testing.B) {
	rawLogs := generateRawLogs(benchSize)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ProcessLogs_Bad(rawLogs)
	}
}

// TODO: Buat BenchmarkProcessLogs_Good
func BenchmarkProcessLogs_Good(b *testing.B) {
	rawLogs := generateRawLogs(benchSize)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ProcessLogs_Good(rawLogs)
	}
}
