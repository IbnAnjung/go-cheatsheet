package pipeline

import (
	"testing"
	"time"
)

func TestFanInRealWorld(t *testing.T) {
	start := time.Now()

	// 1. Jalankan seluruh orkestrasi Fan-Out & Fan-In
	totalData := RunPipeline()

	duration := time.Since(start)

	// 2. Validasi Hasil
	if totalData != 10 {
		t.Fatalf("❌ Gagal! Diharapkan 10 data total, tapi hanya dapat %d", totalData)
	}

	// 3. Validasi Kecepatan Paralelisme
	// Waktu terlama adalah Facebook (50ms). Kita beri batas toleransi sistem sebesar 80ms.
	// Jika kode tidak berjalan paralel (sekuensial), akan memakan waktu > 100ms.
	if duration > 150*time.Millisecond {
		t.Fatalf("❌ Fan-In Anda terlalu lambat! Waktu eksekusi %v (Seharusnya < 80ms)", duration)
	}

	t.Logf("✅ Sukses Luar Biasa! Keseluruhan Pipeline selesai hanya dalam %v", duration)
}
