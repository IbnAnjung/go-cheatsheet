# Resilience: Circuit Breaker

## Objective
Memahami dan mengimplementasikan pattern Circuit Breaker untuk mencegah request berulang pada service yang sedang down/bermasalah, sehingga sistem dapat melakukan _fail-fast_ dan memulihkan diri (self-healing).

## Real-World / Production Use Case
Dalam arsitektur microservices, sebuah service seringkali bergantung pada service lain atau database. Jika service dependensi tersebut mengalami gangguan (timeout, error), terus-menerus mengirimkan request ke service tersebut dapat menyebabkan:
1. Habisnya resource (koneksi, thread, memory) di service pemanggil (cascading failure).
2. Membebani service dependensi yang sedang berusaha pulih.

Circuit Breaker berfungsi seperti sekring listrik. Jika mendeteksi banyak kegagalan beruntun, ia akan "terbuka" (Open) dan langsung menolak request baru secara internal (_fail-fast_), memberikan waktu bagi dependensi untuk pulih. Setelah waktu tertentu, ia akan mencoba "setengah terbuka" (Half-Open) dengan mengizinkan beberapa request uji coba. Jika berhasil, ia "tertutup" (Closed) kembali; jika gagal, kembali ke status "terbuka".

## Case
Anda diminta untuk melengkapi implementasi `CircuitBreaker`.
Terdapat 3 status (State) dalam Circuit Breaker:
- `StateClosed`: Request diizinkan. Jika terjadi error beruntun melebihi `failureThreshold`, ubah status menjadi `StateOpen` dan catat waktu terakhir kegagalan.
- `StateOpen`: Request langsung ditolak dan me-return `ErrCircuitOpen`. Namun, jika waktu saat ini sudah melewati `lastFailureTime + recoveryTimeout`, ubah status menjadi `StateHalfOpen` dan izinkan request tersebut untuk mencoba (trial).
- `StateHalfOpen`: Mengizinkan 1 request trial.
  - Jika trial gagal, kembalikan ke `StateOpen` dan perbarui `lastFailureTime`.
  - Jika trial berhasil, ubah ke `StateClosed` dan reset counter kegagalan.

Lengkapi kerangka di `circuitbreaker.go` agar dapat melewati semua unit test di `circuitbreaker_test.go`. Pastikan implementasi aman terhadap _concurrent access_ (gunakan `sync.Mutex` atau `sync.RWMutex`).

## Common Mistakes
1. **Data Race / Lock Leak (Menulis tanpa proteksi / Lupa RUnlock)**
   Kesalahan yang sering terjadi adalah mengubah `state` atau membaca data secara concurrent tanpa sinkronisasi yang baik. Misalnya, me-*return* *error* namun lupa melepaskan *read lock* (RUnlock) sehingga memicu *Deadlock*.

   ❌ Berbahaya (Lupa Melepas Lock memicu Deadlock):
   ```go
   cb.mu.RLock()
   if cb.state == StateOpen && time.Since(cb.lastFailureTime) < cb.recoveryTimeout {
       return ErrCircuitOpen // Lock bocor! Goroutine lain akan nyangkut (stuck)
   }
   cb.mu.RUnlock()
   ```

   ✅ Pendekatan Terbaik (Gunakan 1 Lock dan Pastikan Terlepas):
   ```go
   cb.mu.Lock()
   if cb.state == StateOpen && time.Since(cb.lastFailureTime) < cb.recoveryTimeout {
       cb.mu.Unlock() // Aman, Lock selalu dilepas
       return ErrCircuitOpen
   }
   ```

2. **Menahan Lock Terlalu Lama (Blocking I/O)**
   Jangan pernah mengeksekusi request eksternal (I/O, HTTP, Database) di dalam perlindungan `Mutex`, karena goroutine lain tidak akan bisa mengecek status circuit dan aplikasi menjadi *hang*.

   ❌ Berbahaya (Eksekusi dengan menahan Lock):
   ```go
   cb.mu.Lock()
   defer cb.mu.Unlock() // Semua goroutine lain akan nge-block menunggu ini!
   err := req() // Misal req memakan waktu 5 detik
   ```

   ✅ Pendekatan Terbaik (Bebaskan Lock sebelum request):
   ```go
   cb.mu.Unlock() 
   err := req() // Goroutine lain bisa bebas membaca status CB
   cb.mu.Lock() // Kunci lagi saat mau update status
   ```

3. **Trial Leak di StateHalfOpen**
   Saat `StateHalfOpen`, jika pengecekan *state* tidak membatasi jumlah eksekusi, ribuan request bisa masuk bersamaan untuk melakukan *trial*, yang mana justru merusak dependensi yang sedang *recovery*.

## Fixes / Solution
Solusi optimalnya adalah memisahkan antara **Fase Pengecekan Izin** dan **Fase Pembaruan Status** dengan `sync.Mutex` (`Lock` dan `Unlock`), serta membiarkan eksekusi `req()` tanpa ditahan oleh lock apa pun:

1. Gunakan 1 blok `cb.mu.Lock()` di awal untuk mengecek status (Closed, Open, HalfOpen) menggunakan `switch`.
2. Jika `StateOpen` namun batas waktu telah lewat, ubah menjadi `StateHalfOpen`, lalu segera `cb.mu.Unlock()`.
3. Jika sudah `StateHalfOpen` sejak awal pengecekan, artinya ada request lain yang sedang menjadi "sukarelawan trial", maka tolak request yang datang (return `ErrCircuitOpen`) lalu `Unlock()`.
4. Eksekusi `req()` **tanpa** lock.
5. Setelah selesai eksekusi, lakukan `cb.mu.Lock()` lagi untuk memproses hasil (jika error, tambah `failureCount`, jika sukses reset status ke `StateClosed`), lalu akhiri dengan `cb.mu.Unlock()`.

Pendekatan ini menjamin **Thread-Safety**, **Performa Tinggi** (karena I/O tidak menahan *mutex*), dan secara efektif membatasi jumlah eksekusi selama fase *recovery*.
