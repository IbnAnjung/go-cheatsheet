# Fan-In / Fan-Out Pattern

## Objective
Mempelajari pola *Fan-Out* (mendistribusikan satu antrean pekerjaan ke banyak *worker*) dan *Fan-In* (menggabungkan kembali hasil dari banyak channel *worker* menjadi satu aliran channel tunggal).

## Real-World / Production Use Case

**Contoh 1: Sistem Video Transcoder**
Bayangkan Anda memiliki 1 channel yang berisi daftar video mentah untuk diproses. Anda melakukan *Fan-Out* dengan menjalankan 5 *worker* (secara *concurrent*) yang semuanya berebut membaca dari 1 channel sumber tersebut. Setelah diproses, ke-5 *worker* ini akan menghasilkan channel *output*-nya masing-masing. Anda membutuhkan pola **Fan-In** untuk "mengalirkan" ke-5 channel output tersebut kembali menjadi 1 channel tunggal agar bisa dibaca dengan mudah oleh satu fungsi penyimpan ke *Database*.

**Contoh 2: High-Speed Web Scraper (Crawler)**
Anda ingin mengambil data dari 1.000 URL secara bersamaan. Anda menaruh 1.000 URL tersebut di satu channel awal. Lalu Anda membuat 50 *worker* untuk men-download halamannya (*Fan-Out*). Setiap worker mengirimkan hasil teks HTML ke *channel* miliknya sendiri. Anda menggunakan *Fan-In* untuk menggabungkan 50 *channel* tersebut menjadi 1 *channel* akhir. Satu goroutine terakhir akan membaca *channel* gabungan ini untuk menuliskannya ke dalam satu file CSV secara berurutan, sehingga terhindar dari *race condition* saat menulis file.

## Case
Di file `pipeline.go`, Anda diberikan fungsi primitif:
- `FetchAPI(source)`: Mengunduh data yang butuh waktu (simulasi 10ms).
- `SaveToDatabase(data)`: Menyimpan data ke database secara sekuensial (juga 10ms).

**Tugas Anda:** 
Bangun fungsi orkestrator `RunPipeline()` yang mengombinasikan pola **Fan-Out** (memanggil API secara paralel) dan **Fan-In** (menggabungkan channel hasil) agar total 10 data dari 3 Social Media (Twitter, Facebook, Instagram) bisa diambil serentak, lalu digabungkan menjadi 1 aliran (channel), dan akhirnya dibaca secara sekuensial oleh `SaveToDatabase()`.

## Common Mistakes
1. **Memblokir (Blocking) saat wg.Wait() pada Merge**
   Pemula sering menaruh `wg.Wait()` dan `close(out)` langsung di akhir fungsi tanpa membungkusnya dalam *goroutine*. Akibatnya, fungsi `Merge` tidak akan pernah me-return `out`, dan program pemanggil yang menunggunya akan mengalami *Deadlock*.
   
2. **Jebakan "Loop Variable Closure" (Go versi < 1.22)**
   Saat iterasi pembaca di dalam `Merge` (`for _, c := range cs`), jika Anda tidak mem-*passing* variabel `c` ke dalam argumen goroutine, semua pembaca bisa-bisa hanya akan menyedot data dari channel terakhir saja! (Bug ini sudah otomatis aman jika memakai Go 1.22+).

3. **Salah Menghitung Waktu Konsumen (Bottleneck)**
   Jika *Fan-In* Anda luar biasa cepat namun proses penyimpanan ke database (`SaveToDatabase`) dilakukan secara **sekuensial (satu-satu)** dan memakan waktu (misal 10ms per data), waktu total sistem Anda akan tetap tertahan oleh kecepatan simpan (100ms untuk 10 data).

## Fixes / Solution (The Architect's Way)

**1. Pola Fan-Out (Menyebar Tugas):**
Membungkus fungsi *Fetch* ke dalam Goroutine yang mengembalikan *channel*.
```go
func ProsesData(sourceNames []string) <-chan string {
	out := make(chan string)
	go func() {
		for _, sourceName := range sourceNames {
			out <- FetchAPI(sourceName) // Pekerjaan dilakukan di background!
		}
		close(out)
	}()
	return out
}
```

**2. Pola Fan-In (Multiplexer):**
Menerima variadic channel `cs`, lalu menyatukannya ke satu channel `in`.
```go
func Merge(cs ...<-chan string) <-chan string {
	in := make(chan string)
	var wg sync.WaitGroup
	for _, c := range cs {
		wg.Add(1)
		go func(c <-chan string) {
			defer wg.Done()
			for v := range c {
				in <- v
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(in)
	}()
	return in
}
```

## Real-World Production Notes ⚠️
Di skala *Enterprise* raksasa, ada dua aturan tambahan saat memakai pola ini:
1. **Pencegahan Rate Limit:** Jangan meluncurkan 1.000 goroutine *Fan-Out* sekaligus jika memanggil API pihak ketiga. Gunakan `time.Ticker` atau *Rate Limiter* agar IP Anda tidak di-blokir (HTTP 429).
2. **Database Batching:** Di sisi ujung konsumen, jangan pernah mengeksekusi `INSERT` satu per satu. Tampung aliran channel ke dalam *Array Slice* sementara. Jika sudah terkumpul 100 data, lakukan **Bulk Insert** sekaligus untuk meringankan beban I/O database.
