# Objective
Melatih implementasi pola *Resilience* (Ketahanan) pada sistem terdistribusi, khususnya **Exponential Backoff Retry** yang dipadukan dengan penghormatan terhadap pembatalan *context* (`context.Context`).

# Real-World / Production Use Case
Dalam arsitektur *Microservices*, pemanggilan API pihak ketiga atau *query* ke database sering kali mengalami kegagalan sementara (*transient errors*) akibat *network blink*, *rate limiting*, atau beban server yang tiba-tiba berlebih. 
Jika kita langsung menyerah dan me-return error, UX (pengalaman pengguna) akan buruk. Namun, jika kita melakukan *retry* terus-menerus secepat mungkin tanpa jeda (atau jeda konstan), kita justru bisa membuat server tujuan yang sedang kewalahan menjadi mati total karena serangan bertubi-tubi dari layanan kita (fenomena ini disebut *Thundering Herd Problem*).

Standar industri untuk mengatasi hal ini adalah **Exponential Backoff**: Jeda antar-retry dilipatgandakan (misal: 1 detik, 2 detik, 4 detik, dst). 
Selain itu, fungsi penunda (jeda) tersebut **wajib** memantau `context.Context` secara aktif. Hal ini guna mencegah aplikasi kita tetap membuang-buang resource menunggu antrean *retry*, padahal *client* yang me-request datanya sudah memutus koneksi/timeout.

# Case
Implementasikan fungsi `DoWithRetry(ctx context.Context, maxRetries int, initialBackoff time.Duration, operation func(ctx context.Context) error) error` yang ada di dalam `retry.go`.

**Aturan Bisnis (Business Rules):**
1. Eksekusi fungsi `operation(ctx)`. Jika berhasil (mereturn `nil`), fungsi utama harus segera selesai dan mereturn `nil`.
2. Jika fungsi mengembalikan error, aplikasi harus menunggu (*jeda*) selama durasi `backoff`, lalu mencoba mengeksekusinya lagi.
3. Durasi `backoff` harus **dikalikan dua (`* 2`)** setiap kali terjadi kegagalan (secara Eksponensial).
4. Batas Pengulangan: Fungsi berhenti melakukan percobaan dan me-return error terakhir jika total pengulangan (*retry*) sudah mencapai batas `maxRetries`. (Contoh: maxRetries 3 berarti 1 eksekusi normal + 3 kali pengulangan = total maksimal 4 kali panggil).
5. **ATURAN KRITIS (Context Cancellation):** Saat fungsi sedang "berjeda" (*sleeping* menunggu antrean retry), fungsi tersebut HARUS peka terhadap pembatalan *context*. Jika `ctx.Done()` tertrigger (*misal karena timeout dari parent*) sebelum waktu jeda habis, fungsi harus **langsung batal dan me-return `ctx.Err()`**, tanpa harus menunggu waktu jeda habis. *(Hint: Jangan pernah menggunakan `time.Sleep` secara statis, cari cara menggunakan `select` dan `time.After`).*

# Common Mistakes

### 1. `defer` di dalam Loop (Memory Leak Tersembunyi)
Kesalahan fatal yang sering dilakukan Go developer pemula adalah memanggil `defer` di dalam sebuah `for` loop (misalnya `defer ticker.Stop()`). Fungsi yang di-`defer` **hanya akan dieksekusi saat fungsi pembungkusnya me-return**, bukan di akhir setiap iterasi loop. Jika loop berjalan ratusan kali, ratusan panggilan `defer` akan menumpuk di *Stack* memori dan menyebabkan kebocoran memori sementara (*memory leak*).

❌ **Berbahaya (Penumpukan Defer):**
```go
for {
    tick := time.NewTicker(backoff)
    defer tick.Stop() // ❌ SALAH! Ini akan menumpuk di memori setiap kali loop berputar.
    // ...
}
```

✅ **Pendekatan Terbaik (Hindari defer di loop, atau gunakan time.After):**
```go
for {
    // ✅ Gunakan time.After untuk one-shot jeda (tidak butuh Stop atau defer)
    select {
    case <-time.After(backoff):
    // ...
    }
}
```

### 2. Memblokir Goroutine dengan `time.Sleep` secara Kaku
Menggunakan `time.Sleep(backoff)` saat proses *retry* sangat berbahaya karena ia memblokir eksekusi goroutine secara mutlak. Jika *client* (user) membatalkan koneksinya di tengah-tengah waktu tidur tersebut, aplikasi Anda tidak akan menyadarinya dan tetap melanjutkan tidurnya, membuang-buang *resource* sistem secara percuma.

❌ **Berbahaya (Mengabaikan Context Cancellation):**
```go
if gagal {
    time.Sleep(backoff) // ❌ SALAH! Tidak peduli jika ctx sudah di-cancel, aplikasi tetap tidur.
}
```

✅ **Pendekatan Terbaik (Peka terhadap Sinyal Batal):**
```go
select {
case <-time.After(backoff):
    // ✅ Waktu jeda habis secara natural, lanjutkan retry
case <-ctx.Done():
    return ctx.Err() // ✅ Context dibatalkan, langsung batal dan keluar!
}
```

### 3. Backoff Linear vs Eksponensial
Linear backoff (misal: 10ms, 20ms, 30ms, 40ms) dengan pola pengalian urutan tidak cukup meredam *Thundering Herd Problem* pada sistem skala besar. Harus dipastikan pengali *backoff* adalah perkalian kuadrat/pangkat dari *backoff* sebelumnya, sehingga grafik jeda merenggang tajam (10ms, 20ms, 40ms, 80ms, 160ms, dst).

# Fixes / Solution

Dengan memperbaiki logika pewaktuan (*timing*) dan kepekaan *Context*, kita berhasil meracik fungsi *retry* yang memenuhi standar *Production-Grade*:

1. **Inisialisasi Base Backoff:** Variabel `backoff := initialBackoff` di-set di luar *loop* agar bisa di-mutasi dengan aman.
2. **Eksekusi dan Return Cepat (Fail/Success-Fast):** Memanggil `operation(ctx)`. Jika sukses (`err == nil`), segera keluar dari loop dan return `nil`.
3. **Pengecekan Batas Maksimal:** Jika iterasi `retry` sudah melebihi batas `maxRetries`, keluar dari loop dan kembalikan `err` dari percobaan terakhir.
4. **Mutasi Exponential Backoff:** Mengalikan `backoff *= 2` untuk melipatgandakan waktu jeda.
5. **Context-Aware Waiting:** Menggunakan kombinasi `select`, `time.After(backoff)`, dan `ctx.Done()`. Teknik ini adalah *Holy Grail* di dalam pemrograman *backend* Go yang *asynchronous*. Cara ini menjamin bahwa jeda antar *request* ditaati, **tanpa mengorbankan** daya tanggap aplikasi (*responsiveness*) ketika sistem meminta proses untuk segera dibatalkan.
