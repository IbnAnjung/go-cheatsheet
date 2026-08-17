# Case 7: Zero-Allocation String Operations

## Tujuan (Objective)
Memahami mengapa operasi penggabungan string menggunakan operator `+=` di dalam *loop* adalah salah satu **anti-pattern paling boros memori** di Go, dan bagaimana menggantinya dengan teknik yang menghasilkan alokasi mendekati nol.

## Mengapa String di Go Itu Immutable?

Di Go, tipe `string` bersifat **immutable** (tidak bisa diubah setelah dibuat). Artinya, setiap kali Anda melakukan `result += "data"`, Go **tidak** menambahkan `"data"` ke belakang `result` yang sudah ada. Yang sebenarnya terjadi adalah:

1. Go membuat *array byte* **baru** di memori dengan ukuran `len(result) + len("data")`.
2. Go meng-*copy* seluruh isi `result` lama ke *array byte* baru.
3. Go menambahkan `"data"` di belakang *array byte* baru.
4. `result` lama dibuang menjadi sampah yang harus dibersihkan oleh *Garbage Collector*.

Jika Anda melakukan ini di dalam *loop* sebanyak 10.000 iterasi, Go akan membuat **10.000 string sementara yang dibuang**, masing-masing semakin besar. Total memori yang dialokasikan dan dibuang bisa mencapai ratusan Megabytes!

## Kapan Pola Ini Digunakan (Real-World Use Case)
1. **Report Generation:** Membangun laporan teks/CSV/JSON besar secara dinamis (misal: export data 100.000 baris transaksi ke file teks).
2. **Template Rendering:** Membangun HTML atau email body dari potongan-potongan string secara iteratif.
3. **Log Aggregation:** Menggabungkan ribuan baris log dari berbagai *goroutine* menjadi satu string besar untuk dikirim ke *logging service*.

---

## Kasus (Case)
Buka file `stringops.go`. Terdapat dua pasang fungsi:

### Pasangan 1: `ProcessData`
- `ProcessData_Bad`: Menggabungkan 10.000 nama menjadi satu string dipisahkan koma menggunakan operator `+=`.
- `ProcessData_Good`: Anda harus mengimplementasikan ini agar hasilnya identik, namun dengan alokasi memori yang jauh lebih rendah.

### Pasangan 2: `GenerateReport`
- `GenerateReport_Bad`: Membangun laporan transaksi menggunakan `+=` dan `fmt.Sprintf`.
- `GenerateReport_Good`: Anda harus mengimplementasikan ini agar hasilnya identik, namun jauh lebih efisien.

**Persyaratan Tugas:**
1. Implementasikan `ProcessData_Good` dan `GenerateReport_Good`.
2. Pastikan unit test lulus: `go test -v -run "Test"`.
3. Bandingkan performa dengan benchmark: `go test -bench=. -benchmem`.

## Kesalahan Umum (Common Mistakes)
1. **Menggunakan `+=` dalam Loop Terhadap String:** String di Go bersifat *immutable*. Setiap `result += "a"` tidak menambahkan `"a"`, melainkan membuat *string baru* di memori dan membuang yang lama. Di dalam loop, ini menyebabkan efek *snowball* pada alokasi memori (mencapai Gigabytes jika datanya puluhan ribu).
   **❌ Boros Memori (O(N^2) memory allocations):**
   ```go
   var report string
   for _, tx := range transactions {
       report += fmt.Sprintf("ID: %d | Status: %s\n", tx.ID, tx.Status)
   }
   ```
2. **Membuat String Perantara (Intermediary Strings):** Menggunakan `fmt.Sprintf` untuk merakit string kecil, lalu memasukkannya ke `strings.Builder` masih menyisakan beban alokasi.
   **⚠️ Setengah Optimal (Merakit memori string sementara):**
   ```go
   sb.WriteString(fmt.Sprintf("ID: %d | Status: %s\n", tx.ID, tx.Status))
   ```

## Perbaikan / Solusi (Fixes / Solution)

### 1. The Sweet Spot (Performa Tinggi + Mudah Dibaca) 🌟
Ini adalah standar industri untuk 95% kebutuhan aplikasi. Alih-alih membuat string baru, tulislah *langsung* ke dalam buffer milik `strings.Builder` menggunakan `fmt.Fprintf`.

**✅ Cepat & Bersih (10.000x lebih efisien):**
```go
var sb strings.Builder // Tidak perlu pointer &strings.Builder
// Opsional: Lakukan pre-allocation memori
// sb.Grow(len(transactions) * 80) 

for _, tx := range transactions {
    // Tulis langsung ke buffer builder
    fmt.Fprintf(&sb, "ID: %d | Status: %s\n", tx.ID, tx.Status)
}
return sb.String() // Hanya melakukan 1 kali konversi byte ke string di akhir
```

**Pertimbangan menggunakan `sb.Grow(n)`:**
Penggunaan `sb.Grow(len(transactions) * 80)` digunakan untuk memesan kapasitas memori di awal secara sekaligus agar `Builder` tidak perlu berulang kali me-*resize* memori di tengah-tengah *loop*. Angka `80` adalah estimasi kasar bahwa 1 baris string memakan ~80 bytes.

**Dampak Negatifnya:**
* **Over-allocation (Pemesanan Berlebih):** Jika estimasi Anda terlalu besar (misal dikalikan 1000 padahal teks aslinya pendek), aplikasi Anda akan "merampas" memori yang sebenarnya tidak terpakai dari OS.
* **Under-allocation (Pemesanan Kurang):** Jika estimasi Anda terlalu kecil, tidak menyebabkan error, namun `strings.Builder` akan kembali melakukan operasi lambat berupa *re-size* memori secara otomatis (sehingga keuntungannya berkurang).

*Praktik Industri:* Gunakan `Grow()` hanya jika Anda tahu **persis** atau bisa mengestimasi dengan sangat akurat ukuran akhirnya. Jika datanya sangat dinamis dan sulit ditebak panjangnya, lebih baik biarkan `strings.Builder` bekerja otomatis.

### 2. The Extreme Path (True Zero-Allocation)
Meski `fmt.Fprintf` sangat efisien, ia masih melakukan alokasi kecil di *Heap* karena parameter `...any` (proses *Interface Boxing* saat menerima `int` atau `float`). Jika Anda membutuhkan optimasi absolut (misal: membangun *logger* atau *framework*), Anda harus merakitnya manual dengan `strconv`.

**⚠️ Sangat Cepat, Tapi Sulit Dibaca (Hanya untuk Hot-Paths spesifik):**
```go
for _, tx := range transactions {
    sb.WriteString("ID: ")
    sb.WriteString(strconv.Itoa(tx.ID)) // 0 alloc int to string
    sb.WriteString(" | Status: ")
    sb.WriteString(tx.Status)
    sb.WriteString("\n")
}
```

**Kapan menggunakan yang mana?**
Sebagai *Software Engineer*, utamakan Keterbacaan (*Readability*). Gunakan **Pendekatan 1** kecuali angka *benchmark* Anda membuktikan bahwa Anda butuh **Pendekatan 2**.

---

## 📊 Appendix: Membaca Hasil Benchmark Go

Jika Anda menjalankan perintah `go test -bench="." -benchmem`, Anda akan mendapatkan output dengan 5 kolom penting. Berikut adalah cara membacanya ala *Senior Engineer*:

```text
BenchmarkGenerateReport_Bad-8          2   1003098900 ns/op    3222160284 B/op    67747 allocs/op
BenchmarkGenerateReport_Good-8       285      4302276 ns/op       3721315 B/op    39779 allocs/op
BenchmarkGenerateReport_Extreme-8   1641       782034 ns/op        802816 B/op        1 allocs/op
```

1. **`BenchmarkName-8`**: Nama fungsi tes. Angka `-8` menunjukkan jumlah *Logical CPU Core* yang digunakan (berasal dari `GOMAXPROCS`).
2. **`Iterasi (1641)`**: Berapa kali fungsi berhasil dijalankan oleh Go dalam 1 detik. **(Semakin BESAR semakin BAGUS)**.
3. **`ns/op` (Nanoseconds per Operation)**: Kecepatan murni. Berapa nanodetik waktu yang dibutuhkan untuk mengeksekusi fungsi 1 kali (1 ms = 1.000.000 ns). **(Semakin KECIL semakin CEPAT)**.
4. **`B/op` (Bytes per Operation)**: Konsumsi RAM murni. Berapa bytes memori Heap yang diminta saat fungsi dieksekusi 1 kali. **(Semakin KECIL semakin HEMAT RAM)**.
5. **`allocs/op` (Allocations per Operation)**: Beban *Garbage Collector*. Berapa kali fungsi berteriak meminta memori ke OS. Angka `1 allocs/op` pada versi *Extreme* menandakan *True Zero-Allocation* di dalam *loop* (1 alloc hanya berasal dari pemanggilan fungsi `sb.Grow()` di awal). **(Semakin KECIL semakin RINGAN BEBAN GC)**.

---

## Cara Mengerjakan
1. Buka file `stringops.go` dan implementasikan kedua fungsi `Good`.
2. Jalankan *unit test*:
   ```bash
   cd performance/zero-alloc
   go test -v -run "Test"
   ```
3. Jalankan *benchmark* untuk melihat perbedaan performa:
   ```bash
   go test -bench=. -benchmem
   ```
   *Perhatikan perbedaan dramatis pada kolom `B/op` (bytes allocated) dan `allocs/op`!*
