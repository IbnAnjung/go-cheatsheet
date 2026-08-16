package errgroup

import (
	"context"
	"errors"
	"fmt"
	"time"

	eg "golang.org/x/sync/errgroup"
)

// MockFetch mensimulasikan pemanggilan HTTP/Database yang memakan waktu.
// Jika URL adalah "error.com", ia akan langsung mengembalikan error.
// Jika URL adalah "slow.com", ia akan memakan waktu 3 detik, KECUALI dibatalkan (cancelled) oleh Context.
func MockFetch(ctx context.Context, url string) (string, error) {
	if url == "error.com" {
		return "", errors.New("failed to fetch")
	}

	delay := 100 * time.Millisecond
	if url == "slow.com" {
		delay = 3 * time.Second
	}

	// Simulasi proses I/O yang mematuhi Context Cancellation
	// PENTING: Gunakan time.NewTimer, BUKAN time.After!
	// time.After membuat timer internal yang tidak di-GC sampai expired,
	// menyebabkan memory leak di hot-path production.
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return "", ctx.Err() // Segera berhenti jika context dibatalkan
	case <-timer.C:
		return fmt.Sprintf("Data from %s", url), nil
	}
}

// FetchAll memanggil MockFetch untuk setiap URL secara konkuren.
// Jika ADA SATU SAJA URL yang mengembalikan error, fungsi ini harus SEGERA membatalkan sisa pemanggilan lain (Fail-Fast)
// dan langsung mengembalikan error tersebut.
// Jika semuanya sukses, kembalikan semua datanya (urutan bebas).
func FetchAll(urls []string) ([]string, error) {
	g, ctx := eg.WithContext(context.Background())
	results := make([]string, len(urls))
	for i, url := range urls {
		url := url
		i := i

		g.Go(func() error {
			result, err := MockFetch(ctx, url)
			if err != nil {
				return err
			}

			results[i] = result
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}
