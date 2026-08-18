package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoWithRetry_SuccessOnFirstTry(t *testing.T) {
	tries := 0
	operation := func(ctx context.Context) error {
		tries++
		return nil
	}

	err := DoWithRetry(context.Background(), 3, 10*time.Millisecond, operation)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if tries != 1 {
		t.Fatalf("Expected exactly 1 try, got %d", tries)
	}
}

func TestDoWithRetry_SuccessOnLaterTry(t *testing.T) {
	tries := 0
	operation := func(ctx context.Context) error {
		tries++
		if tries < 3 {
			return errors.New("temporary error")
		}
		return nil // Berhasil pada percobaan ke-3 (eksekusi ke-1, lalu retry 1, lalu retry 2)
	}

	start := time.Now()
	err := DoWithRetry(context.Background(), 5, 10*time.Millisecond, operation)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if tries != 3 {
		t.Fatalf("Expected exactly 3 tries, got %d", tries)
	}

	// Backoff eksponensial: 
	// Setelah gagal pertama: nunggu 10ms
	// Setelah gagal kedua: nunggu 20ms
	// Total waktu tunggu minimal >= 30ms
	if duration < 30*time.Millisecond {
		t.Errorf("Expected execution to take at least 30ms due to backoff, took %v", duration)
	}
}

func TestDoWithRetry_MaxRetriesExhausted(t *testing.T) {
	tries := 0
	lastErr := errors.New("fatal error")
	operation := func(ctx context.Context) error {
		tries++
		return lastErr
	}

	// maxRetries = 3
	err := DoWithRetry(context.Background(), 3, 5*time.Millisecond, operation)
	
	// Eksekusi pertama = 1
	// Retry ke-1 = 1
	// Retry ke-2 = 1
	// Retry ke-3 = 1
	// Total eksekusi = 4
	if err != lastErr {
		t.Fatalf("Expected error '%v', got '%v'", lastErr, err)
	}
	if tries != 4 {
		t.Fatalf("Expected exactly 4 tries (1 initial + 3 retries), got %d", tries)
	}
}

func TestDoWithRetry_ContextCancelledDuringBackoff(t *testing.T) {
	tries := 0
	operation := func(ctx context.Context) error {
		tries++
		return errors.New("temp error")
	}

	// Waktu backoff awal dibuat sangat lama (1 Jam) agar fungsi pasti "menunggu" panjang
	ctx, cancel := context.WithCancel(context.Background())
	
	// Simulasikan pembatalan oleh sistem (misal: user menekan tombol 'Cancel') setelah 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	// Eksekusi awal -> gagal -> fungsi Anda akan mencoba menunggu 1 JAM
	err := DoWithRetry(ctx, 3, 1*time.Hour, operation)
	duration := time.Since(start)

	if err != context.Canceled {
		t.Fatalf("Expected context.Canceled error, got %v", err)
	}

	if tries != 1 {
		t.Fatalf("Expected exactly 1 try, got %d", tries)
	}

	// Jika fungsi Anda menggunakan time.Sleep(), ini akan membutuhkan waktu 1 Jam!
	// Fungsi yang benar akan terinterupsi saat channel context di-close (~50ms).
	if duration > 100*time.Millisecond {
		t.Fatalf("DoWithRetry did not respect context cancellation gracefully, took %v", duration)
	}
}
