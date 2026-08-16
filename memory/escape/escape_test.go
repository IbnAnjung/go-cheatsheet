package escape

import (
	"testing"
)

func TestCreateUser_Good(t *testing.T) {
	u := CreateUser_Good(1, "Alice", "alice@example.com")
	if u.ID != 1 || u.Name != "Alice" {
		t.Errorf("Unexpected user data: %+v", u)
	}
}

// Gunakan variabel global agar hasil fungsi tidak dibuang (optimized out) oleh compiler.
var GlobalBadUser *User
var GlobalGoodUser User

// BenchmarkCreateUser_Bad akan menunjukkan adanya 1 heap allocation per operasi.
// Jalankan dengan: go test -bench=. -benchmem
func BenchmarkCreateUser_Bad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Meng-assign ke variabel global memaksa compiler untuk benar-benar melakukan alokasi
		GlobalBadUser = CreateUser_Bad(i, "Alice", "alice@example.com")
	}
}

// BenchmarkCreateUser_Good seharusnya menunjukkan 0 heap allocations.
func BenchmarkCreateUser_Good(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		GlobalGoodUser = CreateUser_Good(i, "Alice", "alice@example.com")
	}
}
