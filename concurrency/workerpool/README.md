# Case 1: Worker Pool & Goroutine Leak Prevention

## Tujuan (Objective)
Tujuan dari studi kasus ini adalah memahami bagaimana memproses sejumlah besar pekerjaan (jobs) secara bersamaan (concurrently) menggunakan goroutine, tetapi **membatasi jumlah goroutine yang berjalan bersamaan** menggunakan pola *Worker Pool*. Selain itu, kamu juga harus memastikan **tidak ada goroutine yang leak (bocor)** setelah semua pekerjaan selesai.

## Kasus (Case)
Kamu diminta untuk melengkapi fungsi `ProcessJobs(jobs []Job, numWorkers int) []Result` di dalam file `workerpool.go`.

**Persyaratan:**
1. Ada `N` pekerjaan yang harus diselesaikan.
2. Kamu hanya boleh menggunakan tepat sejumlah `numWorkers` goroutine yang berjalan secara konkuren.
3. Setiap pekerjaan akan menghitung nilai kuadrat dari `Value` dan menyimpannya di dalam `Result`.
4. Kembalikan semua `Result` dalam sebuah slice. Urutan tidak penting.
5. **Penting:** Pastikan semua goroutine dihentikan dengan benar. File testing akan mengecek *goroutine leak*.

## Kesalahan Umum (Common Mistakes)
1. **Data Race pada Slice:** Melakukan operasi `append` ke dalam *slice* yang sama dari beberapa goroutine secara bersamaan tanpa pengamanan. Karena `append` tidak *thread-safe*, ini menyebabkan elemen tertimpa atau *slice header* rusak.
2. **Goroutine Leak:** Lupa memanggil `close(jobChans)`. Tanpa itu, *worker* akan *hang* selamanya menunggu *channel* yang tidak akan pernah tertutup (`for job := range jobChans`).
3. **Tidak Menunggu Worker Selesai:** Lupa memanggil `wg.Wait()`, sehingga fungsi utama mengembalikan *slice* yang masih kosong (atau belum lengkap) karena *worker* belum selesai bekerja.

## Perbaikan / Solusi (Fixes / Solution)
**Solusi menggunakan Mutex (Shared Memory):**
Cara praktis memperbaiki *Data Race* adalah melindungi *slice* dengan `sync.Mutex`.

```go
var wg sync.WaitGroup
var mu sync.Mutex

for i := 0; i < numWorkers; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for job := range jobChans {
            // TIP: Lakukan komputasi di luar area yang di-lock agar lebih efisien!
            square := job.Value * job.Value
            
            mu.Lock()
            result = append(result, Result{
                JobID:  job.ID,
                Square: square,
            })
            mu.Unlock()
        }
    }()
}
```

**Solusi Alternatif (Message Passing) - Highly Recommended!:**
Dalam idiom Go (*"Share memory by communicating"*), alih-alih memakai `Mutex`, Anda mendelegasikan 1 goroutine khusus penyelia (mandor) dan membiarkan *worker* mengirim datanya ke channel. Hanya *main goroutine* yang melakukan `append`. Ini adalah versi yang paling bersih dari *Data Race*:

```go
jobChans := make(chan Job, len(jobs))
resultChans := make(chan Result)
result := make([]Result, 0, len(jobs))
var wg sync.WaitGroup

// 1. Jalankan workers
for i := 0; i < numWorkers; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for job := range jobChans {
            resultChans <- Result{JobID: job.ID, Square: job.Value * job.Value}
        }
    }()
}

// 2. Kirim jobs & tutup pintu masuk
for _, job := range jobs {
    jobChans <- job
}
close(jobChans)

// 3. Goroutine "Mandor": Tunggu semua worker beres, lalu tutup pintu keluar
go func() {
    wg.Wait()
    close(resultChans)
}()

// 4. Goroutine Utama: Fokus kumpulkan hasil dengan aman (tanpa lock)
for res := range resultChans {
    result = append(result, res)
}
return result
```

## Cara Mengerjakan
1. Buka file `workerpool.go` dan lengkapi implementasi fungsi `ProcessJobs`.
2. Jika sudah selesai atau butuh di-review, beritahu saya.
3. Untuk mengetes kode kamu, jalankan perintah di terminal:
   ```bash
   cd concurrency/workerpool
   go test -v -race
   ```
