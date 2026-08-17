package pipeline

import (
	"fmt"
	"sync"
	"time"
)

// 1. TAHAP FAN-OUT (PRODUCER)
// FetchAPI mensimulasikan layanan eksternal yang mengirimkan data terus-menerus.
// Berjalan secara paralel jika dipanggil berkali-kali.
// Jangan ubah code ini, anggap ini adalah proses dari pihak external
func FetchAPI(sourceName string) string {
	time.Sleep(10 * time.Millisecond) // Simulasi latensi jaringan / delay
	return fmt.Sprintf("GET [%s] Data", sourceName)
}

// 3. TAHAP KONSUMEN (SEQUENTIAL)
// SaveToDatabase mensimulasikan proses insert ke database.
// Sengaja dibuat membaca dari channel (sekuensial) untuk menghindari Race Condition.
// Jangan ubah code ini, code ini adalah hanya untuk jalankan query ke database
func SaveToDatabase(q string) {
	time.Sleep(10 * time.Millisecond)
	fmt.Printf("📥 Menyimpan ke DB: %s\n", q)
}

func ProsesData(sourceNames []string) <-chan string {
	out := make(chan string)
	go func() {
		for _, sourceName := range sourceNames {
			out <- FetchAPI(sourceName)
		}
		close(out)
	}()

	return out
}

func Merge(cs ...<-chan string) <-chan string {
	out := make(chan string)
	var wg sync.WaitGroup
	for _, c := range cs {
		wg.Add(1)
		go func(c <-chan string) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// 4. ORKESTRATOR UTAMA
// RunPipeline menyatukan seluruh pola Fan-Out dan Fan-In di atas.
func RunPipeline() int {
	// A. kita perlu mengambil data dari ke tiga url in "twitter, facebook, instagram"
	// B. simpan result pada SaveToDatbase
	// C. Simpan ke database dengan aman secara sekuensial
	// D kembalikan berapa data yang di simpan
	twitter := ProsesData([]string{"Twitter 1", "Twitter 2", "Twitter 3"})
	instagram := ProsesData([]string{"Instagram 1", "Instagram 2", "Instagram 3", "Instagram 4"})
	facebook := ProsesData([]string{"Facebook 1", "Facebook 2", "Facebook 3"})

	in := Merge(twitter, instagram, facebook)

	res := 0
	for v := range in {
		SaveToDatabase(v)
		res++
	}
	return res
}
