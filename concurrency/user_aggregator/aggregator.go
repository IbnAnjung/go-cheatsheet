package aggregator

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type User struct {
	ID    string
	Name  string
	Email string
}

type Order struct {
	ID     string
	Amount float64
}

type Loyalty struct {
	Points int
	Tier   string
}

type UserProfile struct {
	User    User
	Orders  []Order
	Loyalty Loyalty
}

// Services berisi dependency untuk memanggil API pihak ketiga / service lain.
type Services struct {
	FetchUser    func(ctx context.Context, id string) (User, error)
	FetchOrders  func(ctx context.Context, id string) ([]Order, error)
	FetchLoyalty func(ctx context.Context, id string) (Loyalty, error)
}

// FetchUserProfile mengambil data dari ketiga fungsi di dalam Services secara konkuren.
// Jika salah satu gagal, fungsi ini harus mereturn error.
func FetchUserProfile(ctx context.Context, userID string, services Services) (*UserProfile, error) {
	// TODO: Implementasikan logika Fan-Out Fan-In di sini.
	// Wajib menggunakan Goroutine untuk memanggil fungsi-fungsi di dalam 'services'.
	// Wajib meneruskan 'ctx' agar mendukung pembatalan (cancellation) dan timeout.

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(3)

	var userProfile = UserProfile{}

	g.Go(func() error {
		user, err := services.FetchUser(ctx, userID)
		if err != nil {
			return err
		}

		userProfile.User = user
		return nil
	})

	g.Go(func() error {
		orders, err := services.FetchOrders(ctx, userID)
		if err != nil {
			return err
		}
		userProfile.Orders = orders
		return nil
	})

	g.Go(func() error {
		loyalty, err := services.FetchLoyalty(ctx, userID)
		if err != nil {
			return err
		}
		userProfile.Loyalty = loyalty
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &userProfile, nil
}
