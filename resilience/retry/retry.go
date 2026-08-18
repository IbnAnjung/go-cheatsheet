package retry

import (
	"context"
	"time"
)

// DoWithRetry mengeksekusi 'operation' hingga berhasil,
// atau hingga mencapai batas 'maxRetries', atau 'ctx' dibatalkan.
func DoWithRetry(ctx context.Context, maxRetries int, initialBackoff time.Duration, operation func(ctx context.Context) error) error {
	// TODO: Implementasikan logika eksekusi operation dan pengulangannya.
	// Jika gagal, tunggu selama durasi backoff (yang terus dikali dua).
	// AWAS: Jika ctx dibatalkan SAAT fungsi ini sedang menunggu, Anda harus langsung mereturn ctx.Err()!
	// Hint: Jangan menggunakan time.Sleep() karena fungsi tersebut memblokir goroutine
	// secara statis dan tidak akan merespon sinyal channel dari ctx.Done().
	retry := 1
	var err error
	backoff := initialBackoff
	for {
		err = operation(ctx)
		if err == nil {
			break
		}

		if retry > maxRetries {
			break
		}

		retry++

		backoff *= 2
		select {
		case <-time.After(backoff):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}
