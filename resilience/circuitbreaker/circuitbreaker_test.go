package circuitbreaker

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond)

	mockReqSuccess := func() error { return nil }
	mockReqFail := func() error { return errors.New("service unavailable") }

	// 1. Awalnya harus Closed
	if state := cb.State(); state != StateClosed {
		t.Fatalf("expected state Closed, got %v", state)
	}

	// 2. Berhasil mengeksekusi request di StateClosed
	err := cb.Execute(mockReqSuccess)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 3. Simulasi kegagalan hingga threshold (3)
	for i := 0; i < 3; i++ {
		_ = cb.Execute(mockReqFail)
	}

	// 4. Seharusnya sekarang StateOpen
	if state := cb.State(); state != StateOpen {
		t.Fatalf("expected state Open after 3 failures, got %v", state)
	}

	// 5. Request di StateOpen harus langsung fail-fast dengan ErrCircuitOpen
	err = cb.Execute(mockReqSuccess) // asumsikan request ini aslinya success, tapi breaker nge-block
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	// 6. Tunggu hingga melewati recoveryTimeout
	time.Sleep(60 * time.Millisecond)

	// 7. Request berikutnya (meskipun gagal) harus mengubah ke HalfOpen terlebih dahulu saat dicoba,
	// namun hasil akhirnya akan menentukan state selanjutnya.
	// Kita coba berikan request sukses agar pindah ke Closed.
	err = cb.Execute(mockReqSuccess)
	if err != nil {
		t.Fatalf("expected trial request to succeed, got %v", err)
	}

	if state := cb.State(); state != StateClosed {
		t.Fatalf("expected state to return to Closed after successful trial, got %v", state)
	}

	// 8. Pindah lagi ke Open untuk test trial yang gagal
	for i := 0; i < 3; i++ {
		_ = cb.Execute(mockReqFail)
	}
	if state := cb.State(); state != StateOpen {
		t.Fatalf("expected state to be Open again, got %v", state)
	}

	// 9. Tunggu timeout lagi
	time.Sleep(60 * time.Millisecond)

	// 10. Lakukan trial request yang gagal
	err = cb.Execute(mockReqFail)
	if err == nil {
		t.Fatalf("expected error from failed trial")
	}

	// 11. State harus kembali menjadi Open
	if state := cb.State(); state != StateOpen {
		t.Fatalf("expected state to revert to Open after failed trial, got %v", state)
	}
}

func TestCircuitBreaker_Concurrency(t *testing.T) {
	cb := NewCircuitBreaker(5, 100*time.Millisecond)
	
	// Simulasi 50 request konkuren yang gagal
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.Execute(func() error { return errors.New("fail") })
		}()
	}
	wg.Wait()
	
	// Harusnya state menjadi Open
	if state := cb.State(); state != StateOpen {
		t.Fatalf("expected StateOpen, got %v", state)
	}
	
	// Request ke-51 harus langsung ErrCircuitOpen
	err := cb.Execute(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}
