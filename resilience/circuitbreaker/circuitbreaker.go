package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
	mu sync.RWMutex

	state            State
	failureThreshold int
	failureCount     int
	recoveryTimeout  time.Duration
	lastFailureTime  time.Time
}

func NewCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
	}
}

// Execute menjalankan fungsi 'req'.
// Jika circuit terbuka, langsung kembalikan ErrCircuitOpen.
// Jika tertutup atau setengah-terbuka, jalankan 'req'.
// Berdasarkan hasil 'req', perbarui status circuit breaker.
func (cb *CircuitBreaker) Execute(req func() error) error {
	cb.mu.Lock()
	switch cb.state {
	case StateOpen:
		if time.Since(cb.lastFailureTime) < cb.recoveryTimeout {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
		cb.state = StateHalfOpen
	case StateHalfOpen:
		cb.mu.Unlock()
		return ErrCircuitOpen
	}

	cb.mu.Unlock()
	err := req()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastFailureTime = time.Now()
		}
	} else {
		cb.failureCount = 0
		cb.state = StateClosed
	}

	return err
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
