package syncpool

import (
	"bytes"
	"sync"
)

// ProcessData mensimulasikan sebuah fungsi yang memproses stream data berkecepatan tinggi.
// Fungsi ini secara terus menerus dipanggil, dan SETIAP KALI dipanggil
// ia mengalokasikan memori baru (buffer) yang menyebabkan beban Garbage Collector (GC) sangat tinggi.
//
// TODO: Optimalkan fungsi ini menggunakan `sync.Pool` agar buffer bisa digunakan kembali (reused).
func ProcessData(data []byte) string {
	// SAAT INI: Selalu mengalokasikan memori baru setiap kali dipanggil.
	// Jika dipanggil 10.000x per detik, akan ada 10.000 bytes.Buffer mati yang harus disapu oleh GC.
	buf := new(bytes.Buffer)

	// Melakukan simulasi pemrosesan (menulis data ke buffer)
	buf.Write(data)
	buf.WriteString(" - processed")

	return buf.String()
}

var bufferPool = &sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func ProcessDataWithSyncPool(data []byte) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	// Melakukan simulasi pemrosesan (menulis data ke buffer)
	buf.Write(data)
	buf.WriteString(" - processed")

	return buf.String()
}
