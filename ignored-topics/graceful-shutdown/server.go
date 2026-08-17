package graceful

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// StartServer_Bad mensimulasikan web server yang berjalan dan berhenti secara paksa (Hard Kill).
// Jika server ini dihentikan (misal oleh OS atau Kubernetes), semua request aktif langsung terputus.
func StartServer_Bad(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulasi proses bisnis (misal: simpan DB, panggil API lain)
		fmt.Fprint(w, "Process completed successfully!")
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fmt.Println("Server running on port", port)
	// ListenAndServe akan memblokir proses selamanya (atau sampai error).
	// Tidak ada mekanisme untuk bereaksi terhadap permintaan berhenti secara elegan.
	return server.ListenAndServe()
}

// StartServer_Good menerima `ctx` yang mewakili sinyal sistem (misal: SIGINT dari Ctrl+C).
//
// TODO: Implementasikan Graceful Shutdown:
// 1. Inisiasi http.Server dan ServeMux yang sama seperti fungsi Bad.
// 2. Jalankan `server.ListenAndServe()` di dalam sebuah goroutine terpisah agar tidak memblokir.
// 3. Blokir fungsi ini dengan mendengarkan `<-ctx.Done()`.
// 4. Ketika `ctx.Done()` terpicu (artinya sinyal kill diterima), buat context baru ber-timeout (misal 5 detik).
// 5. Panggil `server.Shutdown(timeoutCtx)` dan return error-nya.
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

	servErrChan := make(chan error, 1)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			servErrChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("stopping server gracefully")
	case err := <-servErrChan:
		return fmt.Errorf("server error start :%s", err.Error())
	}

	timeOutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := server.Shutdown(timeOutCtx)
	if err != nil {
		log.Printf("fail to shutdown %s", err.Error())
	}

	return err
}
