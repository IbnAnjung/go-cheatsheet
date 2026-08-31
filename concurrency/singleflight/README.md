# Concurrency: Singleflight (Cache Stampede Prevention)

## Objective
Mengimplementasikan pattern **Singleflight** untuk menggabungkan (*coalescing*) beberapa request identik yang terjadi secara bersamaan menjadi satu eksekusi tunggal. 

## Real-World / Production Use Case
Dalam sistem berkinerja tinggi yang memiliki *Cache*, sering kali terjadi fenomena **Cache Stampede** (atau *Thundering Herd*). Bayangkan sebuah *key* di Redis baru saja *expired*. Pada detik yang sama, ada 1.000 user yang mengakses *key* tersebut. Karena cache kosong, ke-1.000 request ini akan langsung meneruskan kueri ke Database. Akibatnya, Database langsung kewalahan dan berpotensi tumbang (*down*).

Dengan **Singleflight**, jika ada 1.000 request datang bersamaan untuk *key* yang sama, **hanya 1 request pertama** yang diizinkan untuk jalan ke Database. Ke-999 request sisanya akan **menunggu** request pertama selesai, dan kemudian berbagi (*share*) hasil yang sama. Ini membuat beban Database tetap stabil (hanya memproses 1 kueri alih-alih 1.000 kueri).

## Case
Lengkapi implementasi di `singleflight.go`.
Anda diminta membuat struct `Group` yang memiliki metode `Do(key string, fn func() (interface{}, error))`.

Alur yang diharapkan:
1. Ketika `Do` dipanggil, cek apakah eksekusi untuk `key` tersebut sedang berjalan (sedang *in-flight*).
2. **Jika sedang berjalan:** Goroutine ini tidak boleh mengeksekusi `fn`, melainkan harus **menunggu** (bisa menggunakan `sync.WaitGroup`) eksekusi yang sedang berjalan selesai, lalu me-return hasil dari eksekusi tersebut.
3. **Jika belum ada:** Goroutine ini menjadi pengeksekusi utama. Ia harus mencatat bahwa `key` tersebut sedang diproses, lalu mengeksekusi `fn`.
4. Setelah eksekusi `fn` selesai, hasilnya (baik nilai maupun error) harus disimpan agar bisa dibaca oleh goroutine lain yang ikut menunggu.
5. Pengeksekusi utama harus memberi sinyal bahwa ia sudah selesai, dan menghapus `key` tersebut dari pencatatan agar pemanggilan `Do` di masa depan (setelah ini selesai) akan mengeksekusi `fn` kembali.

Tingkat kesulitan ini lebih tinggi karena Anda harus menggabungkan `sync.Mutex` (untuk melindungi map pencatatan) dan `sync.WaitGroup` (untuk mekanisme tunggu) dengan tepat.

## Common Mistakes
1. **Menimpa (*Overwrite*) Map Saat Inisialisasi**
   Kesalahan yang sangat fatal adalah mengalokasikan ulang map secara langsung setiap kali ada *key* baru. Ini akan menghapus semua *key* lain yang sedang berjalan (*in-flight*).

   ❌ Berbahaya (Menghapus data goroutine lain):
   ```go
   g.m = make(map[string]*call) // Fatal! Map lama beserta isinya terhapus
   g.m[key] = &call{}
   ```

   ✅ Pendekatan Terbaik (Inisialisasi Lazy):
   ```go
   if g.m == nil {
       g.m = make(map[string]*call) // Hanya buat jika belum pernah ada
   }
   ```

2. **Mengakses Map Setelah Melepas Mutex (Data Race / Panic)**
   Mencoba menulis hasil ke dalam map `g.m[key]` setelah fungsi selesai tereksekusi tanpa memikirkan kemungkinan *key* tersebut sudah dihapus atau *map* sudah berubah.

   ❌ Berbahaya (Panic Nil Pointer):
   ```go
   val, err := fn()
   g.mu.Lock()
   g.m[key].val = val // Bisa panic jika g.m[key] sudah tidak ada!
   ```

   ✅ Pendekatan Terbaik (Gunakan Variabel Lokal):
   ```go
   c := new(call)
   g.m[key] = c
   g.mu.Unlock()
   
   c.val, c.err = fn() // Aman, karena 'c' adalah pointer lokal milik goroutine ini
   ```

## Fixes / Solution
Implementasi **Singleflight** yang tangguh menggabungkan penggunaan `sync.Mutex` (untuk state internal) dan `sync.WaitGroup` (untuk pemblokiran eksternal). Alur yang tepat adalah:

1. Kunci Mutex `g.mu.Lock()`.
2. **Cek ketersediaan (In-Flight Check):**
   - Jika `g.m[key]` ADA, tangkap pointernya (misal variabel `c`), lepas Mutex `g.mu.Unlock()`. Goroutine ini harus menunggu `c.wg.Wait()`, lalu me-return hasil dari `c.val` dan `c.err`.
3. **Pendaftaran (Register):**
   - Jika `g.m[key]` BELUM ADA, artinya ini adalah goroutine pengeksekusi utama.
   - Lakukan *lazy init* map jika `g.m == nil`.
   - Buat objek `c := new(call)`, lakukan `c.wg.Add(1)`.
   - Masukkan `c` ke dalam map: `g.m[key] = c`.
   - Lepas Mutex `g.mu.Unlock()`.
4. **Eksekusi:**
   - Jalankan fungsi mahal secara bebas tanpa mengunci map: `c.val, c.err = fn()`.
   - Setelah selesai, beri tahu goroutine penunggu dengan `c.wg.Done()`.
5. **Pembersihan (Cleanup):**
   - Kunci Mutex lagi `g.mu.Lock()`.
   - Hapus key dari map agar request berikutnya di masa depan bisa memicu eksekusi baru: `delete(g.m, key)`.
   - Lepas Mutex `g.mu.Unlock()`.

Dengan cara ini, ribuan request bersamaan hanya akan memicu 1 kali eksekusi `fn()`, dan sisanya menunggu dengan sangat efisien berkat `WaitGroup`.
