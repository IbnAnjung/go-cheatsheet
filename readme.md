# 🐛 BUG REPORT: Service `report-export` OOM Crash di Production

## Laporan Insiden
Tim DevOps melaporkan bahwa service **report-export** mengalami crash berulang kali di production dengan error **OOM Killed** (Out of Memory). Service ini bertugas meng-generate laporan CSV dari data pesanan (order items) untuk di-download oleh tim Finance.

Crash terjadi setiap kali tim Finance melakukan export data dengan jumlah > 50.000 baris. Monitoring menunjukkan konsumsi RAM melonjak drastis dari ~100MB ke **4GB+** dalam hitungan detik sebelum container Kubernetes di-kill oleh OOM Killer.

## Gejala
- **RAM melonjak drastis** saat endpoint `/export/csv` dipanggil dengan data besar.
- **Response time sangat lambat** (~5-10 detik untuk 50.000 baris, padahal target SLA < 500ms).
- Container selalu di-restart oleh Kubernetes karena melebihi memory limit.

## Tujuan
Perbaiki kode di file `report.go` agar:
1. Konsumsi memori turun secara **signifikan** (target: di bawah 100MB untuk 50.000 baris).
2. Response time turun secara **dramatis**.
3. Output CSV yang dihasilkan **harus identik** dengan versi sebelumnya (jangan ubah format output).

## Clue dari Tim
- Senior Engineer bilang: *"Coba cek bagaimana string CSV-nya dibangun. Dan periksa juga struct-nya, rasanya ada yang bisa dihemat."*
- DevOps bilang: *"Gue lihat di profiler, GC-nya kerja gila-gilaan. Alloc/op nya tinggi banget."*

## Cara Validasi
Jalankan benchmark untuk melihat kondisi awal:
```bash
go test -bench=. -benchmem -v
```
Lalu bandingkan hasilnya setelah Anda melakukan perbaikan.

---

## 🚫 Kesalahan Umum (Common Mistakes)
Dalam *codebase* awal, terdapat 3 dosa besar performa (*anti-patterns*) yang bekerja berbarengan untuk menghancurkan server:

1. **Struct Padding yang Boros (Data Alignment):**
   Urutan tipe data pada struct `OrderItem` ditulis secara acak. Hal ini memaksa Go compiler menyisipkan memori kosong (*padding*) agar data sejajar dengan batas 8-byte CPU.
   **❌ Berbahaya (Struct memakan 56 bytes per objek):**
   ```go
   type OrderItem struct {
       IsFreeShipping bool    // 1 byte
       TotalPrice     float64 // 8 bytes
       // ... dan seterusnya dengan urutan berantakan
   }
   ```

2. **Penggunaan Pointer yang Tidak Perlu (Escape Analysis):**
   Fungsi pembantu mereturn pointer `*OrderItem`. Saat ini dipanggil 50.000 kali di production, compiler memutuskan memori tersebut "lolos" (escape) ke *Heap*, menghasilkan 50.000 objek sampah untuk dibersihkan GC.
   **❌ Berbahaya (Memicu Heap Allocation masif):**
   ```go
   func CreateOrderItem(...) *OrderItem { return &OrderItem{...} }
   ```

3. **String Concatenation dalam Loop (Zero-Allocation):**
   Membuat format CSV menggunakan `+=` dan `fmt.Sprintf`. Karena string itu *immutable*, setiap perulangan membuat string baru yang semakin membesar, lalu membuang yang lama.
   **❌ Berbahaya (O(N^2) memory allocation - 48 Gigabytes!):**
   ```go
   for _, item := range items {
       csv += fmt.Sprintf(...)
   }
   ```

---

## ✅ Perbaikan / Solusi (Fixes / Solution)
Setelah optimasi berlapis dilakukan, performa berhasil ditingkatkan secara radikal (dari memakan RAM **48.5 GB** dengan 320.000+ allocs menjadi HANYA **5 MB** dengan **1 allocs**!).

Berikut adalah 3 langkah solusinya:

1. **Hemat Memori: Data Alignment (Size: 40 bytes)**
   Urutkan tipe data dari yang terbesar (`int64`, `float64`, `string`) ke yang terkecil (`int16`, `bool`). Hal ini menghilangkan *padding* dan menghemat **28% memori RAM per objek**.
   
2. **Hemat Beban GC: Value Semantics**
   Ubah return type menjadi *Value* (langsung `OrderItem`), bukan *Pointer*. Struct 40 bytes sangat ringan untuk disalin murni di dalam area *Stack*, membebaskan *Garbage Collector* dari beban kerja tambahan.

3. **True Zero-Allocation: `strings.Builder` & `strconv` Array Stack**
   - Gunakan `strings.Builder` dengan pre-alokasi akurat `sb.Grow(len(items) * 100)` agar tidak ada proses *re-size* buffer sama sekali.
   - Singkirkan `fmt.Sprintf` sepenuhnya. Gunakan konversi manual dengan `strconv.AppendInt` dan `strconv.AppendFloat` yang disuntikkan ke dalam *byte array lokal di Stack* (`var buf [64]byte`). Pendekatan ekstrem ini sukses meraih impian **1 allocs/op**.
