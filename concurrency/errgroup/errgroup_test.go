package errgroup

import (
	"strings"
	"testing"
	"time"
)

func TestFetchAll_Success(t *testing.T) {
	urls := []string{"google.com", "github.com", "golang.org"}
	results, err := FetchAll(urls)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != len(urls) {
		t.Fatalf("Expected %d results, got %d", len(urls), len(results))
	}

	// Verify all returned
	for _, url := range urls {
		found := false
		for _, res := range results {
			if strings.Contains(res, url) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Result for %s not found", url)
		}
	}
}

func TestFetchAll_ErrorAndCancellation(t *testing.T) {
	// "error.com" akan menghasilkan error dalam waktu instan.
	// "slow.com" butuh waktu 3 detik.
	// Karena ini Fail-Fast, operasi harus langsung berhenti dan dibatalkan,
	// tidak perlu menunggu slow.com selesai.
	urls := []string{"google.com", "error.com", "golang.org", "slow.com"}

	start := time.Now()
	_, err := FetchAll(urls)
	duration := time.Since(start)

	if err == nil {
		t.Fatalf("Expected error due to 'error.com', but got nil")
	}

	if err.Error() != "failed to fetch" {
		t.Errorf("Expected 'failed to fetch', got: %v", err)
	}

	// Operasi harus fail-fast, tidak perlu menunggu proses yang lambat
	// Jika memakan waktu > 1 detik, berarti context cancellation gagal
	if duration > 1*time.Second {
		t.Errorf("Operation took too long (%v), context cancellation (Fail-Fast) tidak bekerja!", duration)
	}
}
