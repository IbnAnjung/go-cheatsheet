package ratelimiter

import (
	"sync"
	"time"
)

// RateLimiter membatasi jumlah request yang dapat diproses dalam suatu periode.
// Menggunakan algoritma Token Bucket.
type RateLimiter struct {
	mu sync.Mutex

	capacity       int
	tokens         int
	refillAmount   int
	refillInterval time.Duration
	stopCh         chan struct{}
	stopOnce       sync.Once
}

// NewRateLimiter membuat RateLimiter baru.
// - capacity: Jumlah maksimum token di bucket.
// - refillAmount: Jumlah token yang ditambahkan setiap kali refill.
// - refillInterval: Seberapa sering token diisi ulang.
// Bucket dimulai dengan penuh (tokens = capacity).
// Harus menjalankan goroutine background untuk refill token secara periodik.
func NewRateLimiter(capacity, refillAmount int, refillInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		capacity:       capacity,
		tokens:         capacity,
		refillAmount:   refillAmount,
		refillInterval: refillInterval,
		stopCh:         make(chan struct{}),
	}

	// TODO: Jalankan goroutine background untuk refill token secara periodik.
	// Goroutine harus berhenti ketika menerima sinyal dari stopCh.
	go func() {
		ticker := time.NewTicker(rl.refillInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.mu.Lock()
				rl.tokens += rl.refillAmount
				if rl.tokens > rl.capacity {
					rl.tokens = rl.capacity
				}
				rl.mu.Unlock()
			case <-rl.stopCh:
				return
			}
		}
	}()

	return rl
}

// Allow mengecek apakah request diizinkan.
// Return true jika ada token tersedia (dan kurangi 1 token).
// Return false jika bucket kosong.
func (rl *RateLimiter) Allow() bool {
	// TODO: Implementasi pengecekan dan pengurangan token secara thread-safe..
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.tokens == 0 {
		return false
	}

	rl.tokens -= 1
	return true
}

// Stop menghentikan goroutine background refill.
func (rl *RateLimiter) Stop() {
	// TODO: Kirim sinyal untuk menghentikan goroutine refill.
	rl.stopOnce.Do(func() {
		close(rl.stopCh)
	})
}
