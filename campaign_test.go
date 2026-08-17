package campaign

import (
	"fmt"
	"strings"
	"testing"
)

func generateEmails(n int) []string {
	emails := make([]string, n)
	for i := range emails {
		emails[i] = fmt.Sprintf("user%d@example.com", i)
	}
	return emails
}

func testFailFast(t *testing.T, name string, fn func([]string) error, expectFailFast bool) {
	t.Run(name, func(t *testing.T) {
		emails := generateEmails(1000)
		emails[50] = "invalid@example.com" // error di awal

		ProcessedCount = 0
		err := fn(emails)

		if err == nil || !strings.Contains(err.Error(), "invalid email") {
			t.Fatalf("Expected invalid email error, got: %v", err)
		}

		t.Logf("[%s] Total processed emails: %d", name, ProcessedCount)

		if expectFailFast {
			// Jika dibatasi max 10 worker, harusnya berhenti setelah memproses sekitar ~60 email
			if ProcessedCount > 200 {
				t.Errorf("❌ FAIL-FAST TIDAK BEKERJA pada %s!\n%d email terproses sia-sia padahal seharusnya berhenti seketika.", name, ProcessedCount)
			}
		} else {
			// Fungsi Bad akan mengeksekusi semua 1000 goroutine secara liar
			if ProcessedCount < 1000 {
				t.Errorf("🤔 Aneh, fungsi %s harusnya memproses semua email.", name)
			}
		}
	})
}

func TestSendCampaigns(t *testing.T) {
	testFailFast(t, "Bad", SendCampaign_Bad, false) // Bad = Tidak punya fail fast
	testFailFast(t, "Good (Channel Worker)", SendCampaign_Good, true)
	testFailFast(t, "Clean (SetLimit)", SendCampaign_Clean, true)
}

// === Benchmark Perbandingan ===

// Karena benchmark memanggil fungsi berulang kali (ribuan kali),
// kita tidak menggunakan limit 50.000 agar tidak memakan waktu berjam-jam.
func BenchmarkSendCampaigns(b *testing.B) {
	emails := generateEmails(500)
	
	b.Run("Bad (Unbounded)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = SendCampaign_Bad(emails)
		}
	})
	
	b.Run("Good (Channel)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = SendCampaign_Good(emails)
		}
	})
	
	b.Run("Clean (SetLimit)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = SendCampaign_Clean(emails)
		}
	})
}
