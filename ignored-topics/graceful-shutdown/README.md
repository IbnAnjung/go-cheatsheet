# Case 5: Graceful Shutdown

## Tujuan (Objective)
Mempelajari cara mematikan server atau proses Go secara elegan (*Graceful Shutdown*). Secara default, jika Anda menekan Ctrl+C atau men-kill proses, sistem operasi akan langsung memenggal aplikasi Anda. Jika saat itu aplikasi sedang melayani request HTTP atau sedang menyimpan data ke *database*, transaksi tersebut akan terputus dan datanya berpotensi korup.

*Graceful Shutdown* memastikan server:
1. Berhenti menerima *request* baru.
2. Menunggu *request* lama yang sedang berjalan hingga selesai (atau *timeout* tercapai).
3. Baru setelah itu mematikan program.

## Kapan Pola Ini Digunakan (Real-World Use Case)
Pola ini **WAJIB** digunakan di seluruh aplikasi Go skala produksi, khususnya di arsitektur cloud-native:
1. **Kubernetes Deployments (Rolling Updates):** Saat Anda melakukan *deploy* versi baru, Kubernetes akan mengirim sinyal `SIGTERM` ke pod lama dan memberinya jeda waktu (default 30 detik) sebelum menembakkan `SIGKILL`. Jika aplikasi Anda tidak mendengarkan `SIGTERM` ini, aplikasi Anda akan di-kill seketika, dan pengguna yang sedang menekan tombol "Bayar" di detik tersebut akan mendapati layar *error 502 Bad Gateway*.
2. **Database Integrity:** Mencegah transaksi tertinggal dalam state yang menggantung (*hanging connection*) yang bisa membuat *connection pool database* kehabisan koneksi.

## Kasus (Case)
Buka file `server.go`. Terdapat fungsi `StartServer_Bad` yang menggunakan metode standar pemula: `server.ListenAndServe()`. Fungsi ini akan memblokir (*block*) selamanya sampai server *crash* atau di-kill oleh OS. Ia tidak memiliki mekanisme untuk bersiap-siap mati.

**Persyaratan Tugas:**
1. Pelajari fungsi `StartServer_Bad` di `server.go`.
2. Implementasikan fungsi `StartServer_Good(ctx context.Context, port string) error`:
   - Deklarasikan objek `http.Server` dan pasang *routing* yang sama dengan fungsi Bad.
   - Jalankan pemanggilan `server.ListenAndServe()` di dalam sebuah **goroutine**.
   - Blokir (*block*) fungsi utama dengan mendengarkan saluran pembatalan dari argumen `ctx`, yaitu: `<-ctx.Done()`.
   - Begitu sinyal pembatalan diterima, buat `context` baru dengan timeout (misal 5 detik) menggunakan `context.WithTimeout`.
   - Panggil `server.Shutdown(timeoutCtx)` dan kembalikan hasilnya.

## Kesalahan Umum (Common Mistakes)
1. **Menggunakan `log.Fatalf` di Goroutine Server:** Pemula sering menggunakan `log.Fatalf` atau `log.Fatal` di dalam goroutine. `log.Fatalf` langsung membunuh aplikasi (memanggil `os.Exit(1)`), sehingga *Graceful Shutdown* dan semua `defer` tidak akan pernah berjalan.

   **❌ Berbahaya (Aplikasi terbunuh seketika):**
   ```go
   go func() {
       if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
           log.Fatalf("error: %v", err) // BERBAHAYA! Akan mengabaikan seluruh defer()
       }
   }()
   ```

   **✅ Pendekatan Terbaik (Kirim error ke channel):**
   ```go
   servErrChan := make(chan error, 1)
   go func() {
       if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
           servErrChan <- err // Kirim error, jangan matikan program sendiri
       }
   }()

   select {
   // ...
   case err := <-servErrChan: // Listen error
       return fmt.Errorf("server error start: %s", err.Error()) // Akan keluar dengan elegan dan menjalankan seluruh defer
   }
   ```

2. **Panic karena `defer close(chan)`:** Menutup *channel* error dari goroutine dengan `defer close(errChan)` bisa menyebabkan panic jika server telat mati dan mencoba mengirim error ke *channel* yang sudah ditutup. *Channel* di Go akan di-*garbage collect* secara otomatis tanpa perlu di-*close* jika sudah tidak dipakai.

   **❌ Berbahaya (Bisa Panic):**
   ```go
   servErrChan := make(chan error, 1)
   defer close(servErrChan) // JANGAN LAKUKAN INI
   ```

3. **Mengabaikan Error dari `server.Shutdown`:** `Shutdown` bisa gagal (misal karena *timeout* terlampaui sebelum semua request selesai). Anda harus selalu memeriksa dan me-*return* error dari `Shutdown` agar aplikasi tahu ada request yang terputus secara paksa.

## Perbaikan / Solusi (Fixes / Solution)
Solusi paling kuat (Production-Grade) untuk *Graceful Shutdown* adalah menggunakan goroutine untuk `ListenAndServe`, lalu memblokir *main thread* dengan `select` untuk menunggu sinyal pembatalan ATAU error saat server gagal start.

```go
func StartServer_Good(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		fmt.Fprint(w, "Process completed successfully!")
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Channel untuk menangkap error saat server start
	servErrChan := make(chan error, 1)

	go func() {
		// Filter http.ErrServerClosed karena itu bukan error yang sesungguhnya (normal saat Shutdown)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			servErrChan <- err
		}
	}()

	// Tunggu sinyal mati dari OS (ctx.Done) ATAU error gagal start
	select {
	case <-ctx.Done():
		fmt.Println("stopping server gracefully")
	case err := <-servErrChan:
		return fmt.Errorf("server error start: %s", err.Error())
	}

	// Beri server waktu maksimal 30 detik untuk menyelesaikan request yang masih berjalan
	timeOutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown akan memblokir sampai semua request selesai atau timeout
	err := server.Shutdown(timeOutCtx)
	if err != nil {
		log.Printf("fail to shutdown %s", err.Error())
	}

	return err
}
```

---

## Cara Mengerjakan
1. Buka file `server.go` dan perbaiki fungsi `StartServer_Good`.
2. Jalankan *unit test* untuk menyimulasikan "bencana" (Pod Kubernetes dimatikan mendadak saat request sedang diproses):
   ```bash
   cd ignored-topics/graceful-shutdown
   go test -v -run TestStartServer_Good
   ```
   *Jika kode Anda benar, test akan sukses karena server menunggu request klien selesai sebelum menghentikan dirinya sendiri.*
