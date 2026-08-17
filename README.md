# 🐛 BUG REPORT: Service `email-campaign` Sering Kena Banned SMTP & CPU Thrashing

## Laporan Insiden
Tim Marketing melaporkan bahwa fitur "Blast Email Promo" sering gagal di tengah jalan. 
Saat mereka mengupload 50.000 email pengguna, CPU server kita langsung melonjak ke 100%, dan server SMTP (pihak ketiga) langsung mem-banned IP kita sementara karena *Rate Limiting* (terlalu banyak koneksi bersamaan). 

Parahnya lagi, jika ada 1 email invalid di urutan awal, sistem tetap melanjutkan mengirim sisa 49.999 email lainnya (buang-buang kuota), padahal seharusnya langsung berhenti (*fail-fast*) agar tim Marketing bisa memperbaiki datanya dulu.

## Gejala
- **CPU Thrashing & Out of Memory** karena membuat puluhan ribu *goroutine* tanpa batas (*unbounded concurrency*).
- **Kena Rate Limit API pihak ketiga** karena koneksi tidak dibatasi.
- **Tidak ada mekanisme *Fail-Fast***. Error handling saat ini ditulis manual pakai `sync.Mutex` dan fungsi tetap memproses semua *goroutine* sampai selesai walaupun sudah ada error di awal.

## Tujuan
Buat fungsi baru bernama `SendCampaign_Good` di `campaign.go` agar:
1. Concurrency dibatasi maksimal **100 *worker* aktif** secara bersamaan.
2. Jika ada 1 email gagal dikirim, sisa email yang belum diproses **harus dibatalkan saat itu juga** (*Fail-Fast* via Context).
3. Kode harus lebih bersih dan elegan tanpa `sync.WaitGroup` dan `sync.Mutex` manual untuk menangkap error.

## Clue dari Tim
- Senior Engineer bilang: *"Ngapain bikin WaitGroup dan Mutex manual buat nangkep error? Bukannya Go punya package standar spesifik buat jalanin goroutine, nangkap error pertama, dan otomatis batalin sisa task pakai Context? Oh ya, di Go 1.20+, package itu juga bisa di-set limit worker-nya lho."*

## Cara Validasi
```bash
go test -v
```
Pastikan `TestSendCampaign_FailFast` **PASS**.
Test ini didesain akan *FAIL* jika fungsi Anda gagal membatalkan proses dengan cepat saat menemukan error.

---

## 🚫 Kesalahan Umum (Common Mistakes)

1. **Unbounded Concurrency (Tanpa Batas Pekerja)**
   Secara naif menggunakan `go func()` di dalam perulangan sebanyak jumlah data. Ini adalah resep pasti untuk memicu *CPU Thrashing*, *Out of Memory* (OOM), dan pemblokiran API pihak ketiga (*Rate Limiting*).
   **❌ Berbahaya (OOM & Rate Limit):**
   ```go
   for _, email := range emails {
       go SendEmail(email) // 50.000 goroutine meledak bersamaan!
   }
   ```

2. **Error Handling Manual & Tidak Ada Fail-Fast**
   Menggunakan `sync.WaitGroup` dan `sync.Mutex` secara manual untuk menangkap error, lalu tetap ngotot mengeksekusi sisa puluhan ribu task meskipun error fatal sudah terjadi di awal.
   **❌ Berbahaya (Buang-buang *resource* komputasi):**
   ```go
   var mu sync.Mutex
   var firstErr error
   // ... di dalam goroutine:
   if err != nil {
       mu.Lock(); firstErr = err; mu.Unlock()
   }
   wg.Wait() // Tetap menunggu 50.000 selesai semua
   ```

---

## ✅ Perbaikan / Solusi (Fixes / Solution)

Gunakan *package* standar `golang.org/x/sync/errgroup`. *Package* ini menyelesaikan 3 masalah besar sekaligus secara elegan:

**1. Menangkap Error Pertama & Membatalkan Sisa Task (Fail-Fast)**
Fungsi `errgroup.WithContext` akan menghasilkan sebuah `Context`. Jika ada satu saja goroutine yang me-return *error*, maka `Context` tersebut akan di-*cancel*. Fungsi pekerja (`SendEmail`) yang mendengarkan `ctx.Done()` akan langsung dibatalkan (Fail-Fast).
Dari hasil tes, alih-alih mengirim 1000 email sia-sia, sistem langsung membatalkan pengiriman tepat setelah ~57 email terkirim.

**2. Membatasi Concurrency (Worker Pool)**
Gunakan `SetLimit(n)` (tersedia sejak Go 1.20) untuk membatasi jumlah *goroutine* yang berjalan bersamaan. Tidak perlu lagi membuat logika *channel* dan *worker loop* manual yang rawan *deadlock*.

**✅ Pendekatan Terbaik (Bersih & Aman):**
```go
func SendCampaign_Clean(emails []string) error {
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(10) // Maksimal 10 goroutine aktif bersamaan (aman dari Rate Limit)

	for _, email := range emails {
		e := email // (Penting untuk Go < 1.22 untuk mencegah loop pointer bug)
		g.Go(func() error {
			// Jika ctx dibatalkan akibat error goroutine lain, ini akan langsung diinterupsi
			return SendEmail(ctx, e) 
		})
	}

	return g.Wait() // Tunggu semua selesai atau kembalikan error pertama
}
```
