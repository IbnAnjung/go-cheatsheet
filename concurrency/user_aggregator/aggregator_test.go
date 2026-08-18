package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFetchUserProfile_Success(t *testing.T) {
	services := Services{
		FetchUser: func(ctx context.Context, id string) (User, error) {
			time.Sleep(100 * time.Millisecond) // Simulasi latensi
			return User{ID: id, Name: "Budi", Email: "budi@example.com"}, nil
		},
		FetchOrders: func(ctx context.Context, id string) ([]Order, error) {
			time.Sleep(150 * time.Millisecond)
			return []Order{{ID: "O-1", Amount: 50000}}, nil
		},
		FetchLoyalty: func(ctx context.Context, id string) (Loyalty, error) {
			time.Sleep(50 * time.Millisecond)
			return Loyalty{Points: 100, Tier: "Gold"}, nil
		},
	}

	// Gunakan timeout untuk memastikan eksekusinya paralel, bukan sekuensial.
	// Total sequential time = 100 + 150 + 50 = 300ms
	// Jika dijalankan concurrent, max time sekitar = ~150ms
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	profile, err := FetchUserProfile(ctx, "U-1", services)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if duration > 250*time.Millisecond {
		t.Errorf("Execution took too long (%v), it is likely not concurrent", duration)
	}

	if profile.User.Name != "Budi" || len(profile.Orders) != 1 || profile.Loyalty.Points != 100 {
		t.Errorf("Incomplete data mapped to profile: %s - %d - %d - %+v", profile.User.Name, len(profile.Orders), profile.Loyalty.Points, profile)
	}
}

func TestFetchUserProfile_Error(t *testing.T) {
	errAPI := errors.New("API failed")
	services := Services{
		FetchUser: func(ctx context.Context, id string) (User, error) {
			time.Sleep(50 * time.Millisecond)
			return User{}, errAPI
		},
		FetchOrders: func(ctx context.Context, id string) ([]Order, error) {
			time.Sleep(100 * time.Millisecond)
			return []Order{}, nil
		},
		FetchLoyalty: func(ctx context.Context, id string) (Loyalty, error) {
			time.Sleep(100 * time.Millisecond)
			return Loyalty{}, nil
		},
	}

	ctx := context.Background()
	_, err := FetchUserProfile(ctx, "U-1", services)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != errAPI {
		t.Errorf("Expected errAPI, got: %v", err)
	}
}

func TestFetchUserProfile_ContextTimeout(t *testing.T) {
	services := Services{
		FetchUser: func(ctx context.Context, id string) (User, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return User{}, nil
			case <-ctx.Done():
				return User{}, ctx.Err()
			}
		},
		FetchOrders: func(ctx context.Context, id string) ([]Order, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return []Order{}, nil
			case <-ctx.Done():
				return []Order{}, ctx.Err()
			}
		},
		FetchLoyalty: func(ctx context.Context, id string) (Loyalty, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return Loyalty{}, nil
			case <-ctx.Done():
				return Loyalty{}, ctx.Err()
			}
		},
	}

	// Timeout di set 50ms, fungsi butuh 200ms, jadi HARUS terjadi timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := FetchUserProfile(ctx, "U-1", services)

	if err == nil {
		t.Fatal("Expected error due to context timeout, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
	}
}
