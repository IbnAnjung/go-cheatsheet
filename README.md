# 🐛 BUG REPORT: Service `log-aggregator` Mencekik CPU dengan Garbage Collector

## Laporan Insiden
Tim Platform mengeluhkan bahwa *microservice* **log-aggregator** (yang bertugas menerima dan memformat jutaan baris log dari seluruh aplikasi tiap menitnya) sering memicu *CPU Spikes* hingga 100%.

Berdasarkan hasil investigasi dari *profiling*, masalahnya BUKAN pada fungsi pemrosesan datanya, melainkan pada **Garbage Collector (GC)**. GC mengambil alih hampir 40% waktu CPU hanya untuk membersihkan "sampah" memori yang terus-menerus diproduksi secara liar oleh fungsi `ProcessLogs`. 

## Gejala
- **Tingginya angka `allocs/op`** pada saat mem-parsing JSON log dan saat menggabungkan string.
- *Memory Allocation Rate* mencapai ratusan Gigabytes per menit, meski sebagian besarnya berumur sangat pendek (hanya dipakai dalam 1 iterasi loop).
- Latensi pemrosesan melambat.

## Tujuan
Buat fungsi `ProcessLogs_Good` di `processor.go` agar:
1. Menurunkan angka alokasi memori di Heap (`allocs/op`) se-drastis mungkin.
2. Memanfaatkan teknik penggunaan ulang (*reusability*) objek memori karena *struct* ini selalu dibuat dan dibuang ribuan kali per detik.
3. Hindari pembuatan string baru secara boros.

## Clue dari Tim
- Senior Engineer (sambil tersenyum menantang): *"json.Unmarshal() itu menerima argumen bertipe `any` (interface). Jadi, struct yang kamu lempar ke sana pasti 'escape' ke Heap. Kita nggak bisa ngakalin compiler-nya, tapi... kita bisa mendaur ulang sampahnya kan? Inget kasus di modul Memory kita dulu."*
- *Hint tambahan:* *"Selain struct JSON-nya, ingat juga cara merakit string tanpa membuat objek baru yang sudah kita pelajari. Gimana kalau objek perakit stringnya juga didaur ulang sekalian?"*

## Cara Validasi
Jalankan benchmark ini dan lihat betapa hancurnya angka alokasi pada versi Bad:
```bash
go test -bench=. -benchmem -v
```
Pastikan fungsi *Good* Anda menghasilkan teks log yang identik dengan versi *Bad*.

## 🚫 Kesalahan Umum (Common Mistakes)

1. **Heap Escape karena Interface (json.Unmarshal)**
   Fungsi `json.Unmarshal(data, v any)` menerima argumen berupa `any` (interface). Di Go, melempar objek lokal (seperti `&LogEntry{}`) ke dalam parameter interface akan otomatis memicu *Escape Analysis*, memaksa struct tersebut dialokasikan ke **Heap**, bukan *Stack*.
   **❌ Berbahaya (OOM akibat jutaan objek sampah):**
   ```go
   for _, raw := range rawLogs {
       entry := &LogEntry{} // Dibuat baru, lalu jadi sampah saat loop selesai
       json.Unmarshal(raw, entry)
   }
   ```

2. **String Concatenation berantai**
   Membuat format output menggunakan `fmt.Sprintf` atau `+` untuk setiap baris log.
   **❌ Berbahaya (Alokasi string tanpa henti):**
   ```go
   formatted := fmt.Sprintf("[%s] %s", entry.Timestamp, entry.Level)
   ```

---

## ✅ Perbaikan / Solusi (Fixes / Solution)

Untuk menekan tekanan pada *Garbage Collector* (GC), kita melakukan 2 optimasi tingkat tinggi:

**1. Menggunakan `sync.Pool` untuk Struct dan Builder**
Karena kita tidak bisa mencegah `json.Unmarshal` melempar objek ke Heap, kita gunakan `sync.Pool` untuk **mendaur ulang** memori Heap tersebut. Kita membuat dua pool: `logEntryPool` dan `sbPool` (`strings.Builder`).
Dengan ini, sistem hanya membuat segelintir objek di memori, lalu memakainya berulang-ulang untuk memproses jutaan baris log.

**2. Membersihkan Objek Daur Ulang (Menghindari "Dirty State")**
Sebelum mengembalikan `*LogEntry` ke dalam pool, kita **wajib** mereset isinya. Jika tidak, data dari log lama (misal field `Service`) akan "bocor" (*data leak*) ke log baru yang kebetulan tidak memiliki field tersebut.
**✅ Pendekatan Terbaik:**
```go
*lEntry = LogEntry{} // Me-reset seluruh isi field menjadi nilai default ("")
logEntryPool.Put(lEntry)
```

**3. Micro-Optimization: `WriteByte`**
Mengganti `sb.WriteString("[")` menjadi `sb.WriteByte('[')` untuk karakter tunggal. Ini menghindari *overhead* struktur *string header* di internal Go.

**4. Chunking Parallelism (Tanpa Channel & Mutex)**
Membagikan data antar goroutine menggunakan *channel* untuk tugas yang eksekusinya sangat cepat (CPU-Bound murni) ternyata malah melambatkan performa akibat birokrasi *locking* dan *context switching* di internal OS.
**✅ Pendekatan Terbaik (Data Partitioning):**
Pre-alokasikan memori tujuan `results := make([]string, len(rawLogs))`. Bagi *slice* data menjadi beberapa blok (*chunks*) secara matematis murni (tanpa tipe float), lalu tugaskan setiap *worker* (goroutine) untuk memproses bloknya masing-masing dan menulis langsung ke *index* absolutnya: `results[start+j] = sb.String()`. Hasilnya: Tidak ada *channel*, tidak ada *mutex*, dan operasi berjalan paralel 100% bebas hambatan (Waktu eksekusi membelah dua)!

**✅ Hasil Akhir:**
Kombinasi ini sukses menurunkan beban alokasi memori puluhan persen per operasi, dan memangkas waktu eksekusi hingga **lebih dari 50%**, membebaskan CPU dari tugas berat membersihkan sampah (*GC Pauses*) maupun antrean goroutine..
