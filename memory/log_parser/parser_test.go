package parser

import (
	"testing"
)

func TestParseLogLine(t *testing.T) {
	entry, err := ParseLogLine("[ERROR] Database connection failed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.Level != "ERROR" {
		t.Errorf("expected level ERROR, got %s", entry.Level)
	}
	if entry.Message != "Database connection failed" {
		t.Errorf("expected message 'Database connection failed', got '%s'", entry.Message)
	}

	// TODO: Uncomment baris di bawah ini jika Anda sudah membuat fungsi ReleaseLogEntry
	ReleaseLogEntry(entry)
}

func BenchmarkParseLogLine(b *testing.B) {
	line := "[WARN] CPU usage is above 90%"
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		entry, _ := ParseLogLine(line)

		// DUMMY USE agar tidak dioptimasi oleh compiler
		_ = entry.Level

		// TODO: Uncomment baris di bawah ini jika Anda sudah membuat fungsi ReleaseLogEntry
		ReleaseLogEntry(entry)
	}
}
