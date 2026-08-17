package campaign

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// Variabel global untuk keperluan testing (jangan diubah)
var ProcessedCount int32

// SendEmail mensimulasikan pengiriman email yang butuh waktu 50ms ke SMTP.
// Jika Context dibatalkan dari luar, I/O delay ini akan otomatis return error.
func SendEmail(ctx context.Context, email string) error {
	// Simulasi delay jaringan (SMTP server delay)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}

	// Menghitung berapa banyak email yang benar-benar coba dikirim
	atomic.AddInt32(&ProcessedCount, 1)

	if email == "invalid@example.com" {
		return errors.New("failed to send: invalid email address")
	}
	return nil
}

// SendCampaign_Bad mensimulasikan kode production yang saat ini berjalan.
// Menggunakan Unbounded Concurrency (tanpa batas).
func SendCampaign_Bad(emails []string) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	// Karena kode lama tidak punya fitur fail-fast, kita beri context.Background() biasa
	ctx := context.Background()

	for _, email := range emails {
		wg.Add(1)
		go func(e string) {
			defer wg.Done()

			// Memanggil fungsi pengirim email
			err := SendEmail(ctx, e)

			// Menangkap error pertama yang terjadi secara manual pakai Mutex
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(email)
	}

	// Menunggu semua 50.000 goroutine selesai walaupun sudah ada yang error
	wg.Wait()
	return firstErr
}

// TODO: Buat fungsi SendCampaign_Good di bawah ini untuk memperbaiki masalah.
var totalWorker = 10

func SendCampaign_Good(emails []string) error {
	mailChan := make(chan string, len(emails))
	g, ctx := errgroup.WithContext(context.Background())

	for i := 0; i < totalWorker; i++ {
		g.Go(func() error {
			for email := range mailChan {
				if err := SendEmail(ctx, email); err != nil {
					return err
				}
			}

			return nil
		})
	}

	for _, email := range emails {
		mailChan <- email
	}
	close(mailChan)

	return g.Wait()
}

func SendCampaign_Clean(emails []string) error {
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(totalWorker)

	for _, email := range emails {
		e := email

		g.Go(func() error {
			return SendEmail(ctx, e)
		})
	}

	return g.Wait()
}
