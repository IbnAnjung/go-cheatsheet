# Resilience: Rate Limiter (Token Bucket)

## Objective
Memahami dan mengimplementasikan **Rate Limiter** menggunakan algoritma **Token Bucket** untuk membatasi jumlah request yang dapat diproses dalam suatu periode waktu.

## Real-World / Production Use Case
Hampir semua API publik dan internal di skala industri menggunakan Rate Limiter untuk:
1. **Melindungi server** dari lonjakan traffic yang berlebihan (DDoS, bot, atau bug di client yang mengirimkan request berulang tanpa jeda).
2. **Fair usage**: Memastikan satu client tidak memonopoli semua resource server, sehingga client lain tetap bisa dilayani.
3. **Cost control**: Mencegah penggunaan berlebihan pada layanan downstream berbayar (misalnya API pihak ketiga yang dikenakan biaya per-request).

Algoritma **Token Bucket** bekerja sebagai berikut:
- Bucket memiliki kapasitas maksimum (`capacity`) token.
- Setiap interval waktu tertentu (`refillInterval`), sejumlah token (`refillAmount`) ditambahkan ke dalam bucket, namun tidak boleh melebihi `capacity`.
- Setiap kali ada request masuk, 1 token diambil dari bucket. Jika bucket kosong (token = 0), request ditolak.

## Case
Lengkapi implementasi `RateLimiter` di `ratelimiter.go` agar:

1. **`NewRateLimiter(capacity, refillAmount int, refillInterval time.Duration)`**: Membuat `RateLimiter` baru dengan bucket penuh (`tokens = capacity`), dan **menjalankan goroutine background** yang akan mengisi ulang token sebanyak `refillAmount` setiap `refillInterval`. Token tidak boleh melebihi `capacity`.
2. **`Allow() bool`**: Mengecek apakah request diizinkan. Jika tersedia token (> 0), kurangi 1 token dan return `true`. Jika tidak ada token, return `false`.
3. **`Stop()`**: Menghentikan goroutine background refill.

Pastikan implementasi **thread-safe** (aman terhadap concurrent access) karena `Allow()` dan goroutine refill akan mengakses `tokens` secara bersamaan.

## Common Mistakes
1. **`Stop()` Memiliki Side Effect (Mereset Token)**
   Fungsi `Stop()` seharusnya hanya menghentikan goroutine refill, bukan mengubah state internal lainnya.

   ❌ Berbahaya (Side Effect Tersembunyi):
   ```go
   case <-rl.stopCh:
       rl.mu.Lock()
       rl.tokens = 0   // Mereset token bukan tanggung jawab Stop()!
       rl.mu.Unlock()
       return
   ```
   Jika `Stop()` dipanggil saat *graceful shutdown*, request yang masih dalam antrian dan seharusnya masih memiliki kuota akan langsung gagal karena token tiba-tiba di-nolkan.

   ✅ Pendekatan Terbaik (Single Responsibility):
   ```go
   case <-rl.stopCh:
       return // Cukup hentikan goroutine, jangan ubah state lain
   ```

2. **`Stop()` Hanya Bisa Dipanggil Sekali (Unbuffered Channel Send)**
   Menggunakan `rl.stopCh <- struct{}{}` pada *unbuffered channel* berarti pemanggilan `Stop()` kedua kalinya akan *block* selamanya (karena goroutine penerima sudah mati).

   ❌ Berbahaya (Bisa Stuck / Panic):
   ```go
   func (rl *RateLimiter) Stop() {
       rl.stopCh <- struct{}{} // Panggilan kedua akan stuck selamanya
   }
   ```

   ✅ Pendekatan Terbaik (Gunakan `close`):
   ```go
   func (rl *RateLimiter) Stop() {
       close(rl.stopCh) // Aman, semua listener langsung menerima sinyal
   }
   ```
   Menutup channel (`close`) akan langsung membuat semua `select` yang mendengarkan channel tersebut menerima *zero value*, sehingga goroutine bisa berhenti. Namun perlu diperhatikan bahwa `close()` pada channel yang sudah ditutup akan menyebabkan **panic**. Untuk production, sebaiknya di-*guard* menggunakan `sync.Once`.

## Fixes / Solution
Implementasi yang benar memiliki 3 komponen utama:

1. **Goroutine Background Refill** dengan `time.Ticker` + `select`:
   - Menggunakan `time.NewTicker` untuk *periodic refill*.
   - `select` mendengarkan 2 channel: `ticker.C` untuk refill dan `stopCh` untuk sinyal berhenti.
   - Saat refill, gunakan `Lock()` pendek hanya untuk mengubah `tokens` (dan pastikan tidak melebihi `capacity`).
   - Jangan lupa `defer ticker.Stop()` untuk membersihkan resource ticker.

2. **`Allow()` dengan Lock Pendek dan Presisi**:
   - `Lock()` → cek token → kurangi jika ada → `Unlock()`.
   - Karena operasinya murni in-memory (tanpa I/O), menggunakan `defer Unlock()` di sini sudah aman dan tepat.

3. **`Stop()` dengan `close(stopCh)`**:
   - Gunakan `close()` alih-alih mengirim value ke channel, agar sinyal diterima oleh semua listener dan aman jika goroutine sudah selesai.

