package singleflight

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo_Basic(t *testing.T) {
	var g Group

	v, err := g.Do("key1", func() (interface{}, error) {
		return "bar", nil
	})
	if err != nil {
		t.Fatalf("Do error: %v", err)
	}
	if v != "bar" {
		t.Errorf("got %q; want %q", v, "bar")
	}
}

func TestDo_Concurrency(t *testing.T) {
	var g Group
	var calls int32

	// Simulasi fungsi berat yang memakan waktu 100ms
	fn := func() (interface{}, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(100 * time.Millisecond)
		return "result", nil
	}

	var wg sync.WaitGroup
	const n = 100 // 100 concurrent requests
	results := make([]interface{}, n)

	// Luncurkan 100 goroutine secara bersamaan
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			val, _ := g.Do("cache_key", fn)
			results[idx] = val
		}(i)
	}

	// Tunggu semuanya selesai
	wg.Wait()

	// Fungsi fn seharusnya hanya dieksekusi 1 kali!
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("fn executed %d times; want 1 time", got)
	}

	// Semua goroutine harus menerima hasil yang sama
	for i := 0; i < n; i++ {
		if results[i] != "result" {
			t.Errorf("goroutine %d got %q; want %q", i, results[i], "result")
		}
	}
}

func TestDo_ErrorPropagation(t *testing.T) {
	var g Group
	expectedErr := errors.New("some error")

	fn := func() (interface{}, error) {
		time.Sleep(50 * time.Millisecond)
		return nil, expectedErr
	}

	var wg sync.WaitGroup
	var err1, err2 error

	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err1 = g.Do("error_key", fn)
	}()
	go func() {
		defer wg.Done()
		_, err2 = g.Do("error_key", fn)
	}()
	wg.Wait()

	if err1 != expectedErr || err2 != expectedErr {
		t.Errorf("expected both to get %v, got %v and %v", expectedErr, err1, err2)
	}
}

func TestDo_SubsequentCalls(t *testing.T) {
	var g Group
	var calls int32

	fn := func() (interface{}, error) {
		atomic.AddInt32(&calls, 1)
		return "val", nil
	}

	// Panggilan pertama
	v1, _ := g.Do("key1", fn)
	if v1 != "val" {
		t.Errorf("got %q; want %q", v1, "val")
	}

	// Panggilan kedua (setelah panggilan pertama selesai)
	// Harus mengeksekusi ulang fungsi fn
	v2, _ := g.Do("key1", fn)
	if v2 != "val" {
		t.Errorf("got %q; want %q", v2, "val")
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("fn executed %d times; want 2 times", got)
	}
}
