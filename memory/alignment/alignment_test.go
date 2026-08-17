package alignment

import (
	"testing"
	"unsafe"
)

// TestStructSize membandingkan ukuran memori aktual dari User_Bad dan User_Good di memori.
func TestStructSize(t *testing.T) {
	badSize := unsafe.Sizeof(User_Bad{})
	goodSize := unsafe.Sizeof(User_Good{})

	t.Logf("Ukuran memori User_Bad : %d bytes", badSize)
	t.Logf("Ukuran memori User_Good: %d bytes", goodSize)

	// User_Bad memakan 40 bytes karena CPU padding (ada 16 bytes ruang kosong yang terbuang!).
	// Jika diurutkan dengan benar (terbesar ke terkecil), User_Good seharusnya memakan maksimal 24 bytes.
	// Perhitungan data asli: Int64(8) + Float64(8) + Int32(4) + Int16(2) + Bool(1) + Bool(1) = 24 bytes.
	
	if goodSize >= badSize {
		t.Errorf("Gagal! User_Good (%d bytes) belum lebih hemat memori dibanding User_Bad (%d bytes). Urutkan dari tipe data terbesar ke terkecil!", goodSize, badSize)
	} else if goodSize > 24 {
		t.Errorf("Hampir! User_Good (%d bytes) sudah lebih kecil, tapi masih bisa ditekan menjadi 24 bytes! Kumpulkan semua field berukuran kecil (bool, int16) di bagian paling bawah.", goodSize)
	} else {
		t.Logf("Sukses! Anda berhasil menghemat memori sebesar %d bytes (%.1f%%) per struct tanpa mengubah logika!", badSize-goodSize, float64(badSize-goodSize)/float64(badSize)*100)
	}
}
