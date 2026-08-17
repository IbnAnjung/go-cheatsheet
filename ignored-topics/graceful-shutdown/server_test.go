package graceful

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStartServer_Good_GracefulShutdown(t *testing.T) {
	port := "8081"
	
	// Context untuk menyimulasikan sinyal mati dari OS (Ctrl+C / SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	
	// 1. Jalankan server Good
	errChan := make(chan error, 1)
	go func() {
		errChan <- StartServer_Good(ctx, port)
	}()

	// Beri waktu server untuk binding port dan start
	time.Sleep(100 * time.Millisecond)

	// 2. Tembak request HTTP sebagai client
	// Request ini dirancang butuh waktu 2 detik untuk selesai di server
	clientChan := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://localhost:" + port + "/process")
		if err != nil {
			clientChan <- err.Error()
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		clientChan <- strings.TrimSpace(string(body))
	}()

	// 3. Beri waktu agar request sampai ke server dan mulai diproses
	time.Sleep(500 * time.Millisecond)

	// 4. BENCANA DATANG! SIMULASI SERVER DI-KILL (Misal: Kubernetes mematikan Pod)
	// Jika servernya "Bad", proses akan langsung mati dan request client terputus (EOF).
	// Jika servernya "Good", ia akan menunggu maksimal timeout sampai request selesai.
	cancel()

	// 5. Verifikasi hasil di sisi client
	select {
	case res := <-clientChan:
		if res != "Process completed successfully!" {
			t.Errorf("Client gagal menerima response utuh. Didapat: %s", res)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Test timed out! Server hang dan tidak mati dengan benar.")
	}

	// 6. Verifikasi log exit dari server
	err := <-errChan
	if err != nil && err != http.ErrServerClosed {
		t.Errorf("Expected nil or http.ErrServerClosed, got: %v", err)
	}
}
