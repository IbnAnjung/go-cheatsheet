# Case 2: Error Groups & Context Cancellation

## Tujuan (Objective)
Tujuan dari studi kasus ini adalah mempelajari **Error Groups** menggunakan library standar-de-facto yaitu `golang.org/x/sync/errgroup`, dan menguasai teknik penyebaran pembatalan (*cancellation propagation*) melalui `context.Context`. Mengelola banyak *goroutine* yang bisa menghasilkan *error* adalah keterampilan esensial di Go.

## Kapan Pola Ini Digunakan (Real-World Use Case)
Pola **Error Group** dengan fitur *Fail-Fast Cancellation* sangat krusial digunakan dalam skala *production*, contohnya:
1. **API Aggregator (BFF Layer):** Saat *backend* Anda perlu mengambil data dari 5 *microservices* berbeda secara bersamaan. Jika 1 layanan *down* (misalnya layanan Autentikasi), sistem harus langsung membatalkan 4 request ke layanan lainnya untuk menghemat memori, *network bandwidth*, dan CPU.
2. **Batch / Parallel Processing:** Saat memproses atau mengunduh ribuan file dari *Cloud Storage* (AWS S3/GCP). Jika direktori ternyata tidak ditemukan atau *credential invalid* di *request* pertama, kita tidak perlu menunggu 999 *request* sisanya untuk *timeout*, proses dibatalkan seketika itu juga.

## Kasus (Case)
Kamu diminta untuk melengkapi fungsi `FetchAll(urls []string) ([]string, error)` di dalam file `errgroup.go`.

**Persyaratan:**
1. Ambil data secara bersamaan dari berbagai `urls` menggunakan fungsi `MockFetch`.
2. Jika semua berhasil, kumpulkan datanya ke dalam sebuah slice.
3. **Penting (Fail-Fast):** Jika ada *satu saja* panggilan yang menghasilkan error (misal dari `"error.com"`), maka:
   - Keseluruhan proses harus langsung mengembalikan error tersebut secepatnya.
   - Panggilan ke URL lain yang belum selesai (seperti `"slow.com"`) harus otomatis dibatalkan (dibatalkan lewat *Context Cancellation* yang diteruskan ke `MockFetch`).
4. Pastikan tidak ada *Data Race* saat mengumpulkan hasil sukses.

## Kesalahan Umum (Common Mistakes)
1. **Lupa Melempar `ctx` dari Errgroup:** Seringkali developer membuat `errgroup.WithContext(ctx)`, namun di dalam goroutine mereka malah memanggil fungsi lain dengan context yang lama (bukan context milik errgroup). Akibatnya, *fail-fast cancellation* tidak berfungsi.
2. **Data Race pada Slice Hasil:** Jika tidak mengalokasikan slice di awal dan menggunakan `append()`, goroutine akan bertabrakan.
3. **Lupa Variable Capture (Go <1.22):** Lupa melakukan `url := url` atau `i := i` di dalam *for-loop* sebelum menjalankan *goroutine*, sehingga semua *goroutine* akan memproses URL yang sama (elemen terakhir dari *array*).

## Perbaikan / Solusi (Fixes / Solution)
Solusi paling optimal untuk kasus ini adalah memadukan **Errgroup** dan pola **Index-based Parallel Assignment** untuk menghindari *Data Race* tanpa memakai *Mutex*.

```go
func FetchAll(urls []string) ([]string, error) {
    g, ctx := errgroup.WithContext(context.Background())
    
    // Alokasi memori penuh sejak awal
    results := make([]string, len(urls))
    
    for i, url := range urls {
        // Variable capture (Wajib untuk Go 1.21 ke bawah)
        i, url := i, url

        g.Go(func() error {
            // TERUSKAN 'ctx' milik errgroup ke dalam fungsi. 
            // Jika errgroup dicancel, 'ctx' ini otomatis mati.
            result, err := MockFetch(ctx, url)
            if err != nil {
                return err // Mengembalikan error akan memicu pembatalan errgroup
            }

            // AMAN DARI DATA RACE! 
            // Karena setiap goroutine menulis ke index 'i' yang berbeda-beda, 
            // kita tidak butuh Mutex atau Channel.
            results[i] = result
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }

    return results, nil
}
```

---

## Cara Mengerjakan
1. Buka file `errgroup.go` dan lengkapi implementasi fungsi `FetchAll`.
2. Gunakan package `golang.org/x/sync/errgroup`.
3. Tes pekerjaanmu dengan menjalankan:
   ```bash
   cd concurrency/errgroup
   go test -v -race
   ```
