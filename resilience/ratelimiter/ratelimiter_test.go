package ratelimiter

import (
	"sync"
	"testing"
	"time"
)

func TestAllow_Basic(t *testing.T) {
	rl := NewRateLimiter(3, 1, 100*time.Millisecond)
	defer rl.Stop()

	// Bucket penuh (3 token), ketiga request harus diizinkan
	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// Token habis, request ke-4 harus ditolak
	if rl.Allow() {
		t.Fatal("request should be rejected when bucket is empty")
	}
}

func TestAllow_Refill(t *testing.T) {
	rl := NewRateLimiter(2, 1, 50*time.Millisecond)
	defer rl.Stop()

	// Habiskan semua token
	rl.Allow()
	rl.Allow()

	// Pastikan bucket kosong
	if rl.Allow() {
		t.Fatal("bucket should be empty")
	}

	// Tunggu refill (1 token setiap 50ms)
	time.Sleep(70 * time.Millisecond)

	// Seharusnya ada 1 token setelah refill
	if !rl.Allow() {
		t.Fatal("should have 1 token after refill")
	}

	// Dan kembali kosong
	if rl.Allow() {
		t.Fatal("should be empty again after consuming refilled token")
	}
}

func TestAllow_RefillDoesNotExceedCapacity(t *testing.T) {
	rl := NewRateLimiter(3, 5, 50*time.Millisecond)
	defer rl.Stop()

	// Habiskan semua token
	for i := 0; i < 3; i++ {
		rl.Allow()
	}

	// Tunggu refill. refillAmount=5 tapi capacity=3, jadi token harus = 3, bukan 5
	time.Sleep(70 * time.Millisecond)

	allowed := 0
	for i := 0; i < 5; i++ {
		if rl.Allow() {
			allowed++
		}
	}

	if allowed != 3 {
		t.Fatalf("expected exactly 3 requests allowed (capacity cap), got %d", allowed)
	}
}

func TestAllow_Concurrency(t *testing.T) {
	rl := NewRateLimiter(100, 0, time.Hour) // refill=0 agar tidak ada tambahan token
	defer rl.Stop()

	var mu sync.Mutex
	allowed := 0

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow() {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed != 100 {
		t.Fatalf("expected exactly 100 allowed requests, got %d", allowed)
	}
}

func TestStop(t *testing.T) {
	rl := NewRateLimiter(1, 1, 30*time.Millisecond)

	// Habiskan token
	rl.Allow()

	// Stop refill goroutine
	rl.Stop()

	// Tunggu waktu yang seharusnya cukup untuk refill
	time.Sleep(80 * time.Millisecond)

	// Token tidak boleh bertambah karena refill sudah dihentikan
	if rl.Allow() {
		t.Fatal("refill should have stopped, no new tokens expected")
	}
}
