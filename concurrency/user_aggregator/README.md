# Objective
Melatih penggunaan *Concurrency Patterns* di Golang (Goroutine, WaitGroup, Channel) dan manajemen pembatalan / *timeout* menggunakan `context.Context`.

# Real-World / Production Use Case
Dalam arsitektur *Microservices*, seringkali satu *request* dari *client* (misal: memuat halaman profil pengguna) membutuhkan data dari berbagai layanan berbeda (Service Profil, Service Pesanan, Service Poin/Loyalti). Jika kita mengambil data ini secara berurutan (*sequential*), waktu respons (latency) akan menjadi sangat lambat (jumlah dari latensi semua layanan). 
Dengan pola *Fan-Out* (menjalankan banyak request secara paralel) dan *Fan-In* (menggabungkan hasilnya), latensi keseluruhan hanyalah sebatas latensi layanan yang paling lambat. Selain itu, kita wajib membatasi waktu eksekusi dengan `context.Context` agar tidak terjadi penumpukan *request* (Goroutine Leak) jika salah satu layanan *down* atau *hang*.

# Case
Anda diminta untuk mengimplementasikan fungsi `FetchUserProfile(ctx context.Context, userID string, services Services) (*UserProfile, error)`.

Fungsi ini harus mengambil data dari tiga fungsi yang ada di struct `Services`:
1. `FetchUser(ctx context.Context, id string) (User, error)`
2. `FetchOrders(ctx context.Context, id string) ([]Order, error)`
3. `FetchLoyalty(ctx context.Context, id string) (Loyalty, error)`

**Aturan Bisnis (Business Rules):**
1. **Concurrent Execution:** Ketiga pemanggilan fungsi tersebut HARUS dijalankan secara bersamaan (paralel) menggunakan Goroutine.
2. **Aggregation:** Gabungkan ketiga hasil kembalian menjadi satu struct `UserProfile`.
3. **Error Handling (Fail-Fast):** Jika SALAH SATU dari ketiga layanan tersebut mengembalikan `error`, maka fungsi utama harus langsung mengembalikan `error` tersebut tanpa harus menunggu layanan lain selesai (jika memungkinkan), atau paling lambat mengembalikan error di akhir penggabungan.
4. **Context Propagation & Timeout:** Anda wajib meneruskan `ctx` ke ketiga fungsi tersebut. Jika `ctx` dibatalkan (misal karena *timeout* dari sistem penguji), fungsi Anda harus bisa mendeteksinya dan tidak bocor (Goroutine Leak).
5. **No Race Conditions:** Pastikan penggabungan data ke struct `UserProfile` aman dari *race condition* jika Anda menggunakan shared variable (atau lebih baik, hindari shared variable dengan memanfaatkan pattern sinkronisasi yang tepat).

# Common Mistakes

### 1. Race Condition saat menggunakan Channel dan WaitGroup
Sering terjadi ketika developer menggunakan `sync.WaitGroup` atau `errgroup.Group` untuk menunggu para "pengirim" (producer), namun lupa menyinkronkan "penerima" (consumer). Jika `Wait()` selesai dan fungsi langsung *return*, ada kemungkinan consumer masih sedang melakukan *assignment* nilai ke dalam objek hasil.

❌ **Berbahaya (Assignment Terpotong oleh Return):**
```go
go func() {
    for i := 0; i < 3; i++ {
        select {
        case u := <-userChan:
            userProfile.User = u // ❌ Race Condition: Eksekusi ini bisa saja belum selesai ketika fungsi utama sudah return
        }
    }
}()

if err := g.Wait(); err != nil { return nil, err }
return &userProfile, nil
```

✅ **Pendekatan Terbaik (Hindari Channel jika tidak perlu):**
```go
g.Go(func() error {
    user, err := services.FetchUser(ctx, userID)
    if err != nil { return err }
    
    // ✅ Benar: Karena setiap goroutine mengisi field yang BERBEDA pada struct yang sama, 
    // operasi ini aman dari concurrent write, dan dijamin selesai oleh g.Wait().
    userProfile.User = user 
    return nil
})
```

### 2. Goroutine Leak akibat Unbuffered Channel saat terjadi Error
Jika menggunakan *unbuffered channel* `make(chan User)` untuk menggabungkan hasil, pengirim akan **terkunci (block)** sampai ada penerima yang mengambil datanya. Jika terjadi suatu *error* di tengah jalan dan fungsi utama melakukan *return* lebih awal (*fail-fast*), goroutine pengirim yang lain tidak akan pernah bisa mengirim datanya (karena penerimanya sudah berhenti). Goroutine tersebut akan tersangkut selamanya (Goroutine Leak).

❌ **Berbahaya (Unbuffered Channel):**
```go
userChan := make(chan User) // ❌ Akan tersangkut (deadlock) jika fungsi utama return lebih awal karena error dari goroutine lain
```

✅ **Pendekatan Terbaik (Buffered Channel):**
```go
userChan := make(chan User, 1) // ✅ Aman: Kapasitas 1 menjamin pengirim tidak pernah tersangkut meskipun datanya tidak dibaca
```

# Fixes / Solution

Untuk masalah Fan-Out / Fan-In sederhana dengan jumlah task yang terprediksi (misal 3 API call), terdapat dua solusi elegan:

### Solusi 1: Menggunakan `errgroup` (Paling Direkomendasikan)
Pendekatan ini adalah yang paling idiomatis dan bersih. Karena kita mengetahui bahwa kita akan mengisi field-field spesifik yang berbeda-beda dari struktur data gabungan, kita tidak memerlukan sinkronisasi yang berlebihan menggunakan channel. `errgroup` secara otomatis menyediakan *cancellation context* jika terjadi kegagalan dan `Wait()`-nya menjamin seluruh *assignment* struct tuntas sebelum direturn.

### Solusi 2: Murni `Channel` (Raw Fan-In)
Bila ingin murni menggunakan *channel*:
1. Gunakan **Buffered Channel** dengan `capacity = 1` pada masing-masing hasil kembalian, dan `capacity = N` pada *channel error*. Ini sangat penting untuk menghindari *goroutine leak* saat skenario *fail-fast*.
2. Lakukan perulangan `select` secara sinkron langsung di fungsi utama (*Main Thread*), bukan di Goroutine tambahan. Ini akan menjamin tidak ada *Race Condition* di fase *assignment* akhir.
