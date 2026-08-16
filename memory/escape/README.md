# Case 4: Escape Analysis & Stack vs Heap Allocation

## Tujuan (Objective)
Tujuan dari studi kasus ini adalah memahami konsep **Escape Analysis** di Go compiler. Di Go, memori bisa dialokasikan di **Stack** (cepat, otomatis bersih saat fungsi selesai, 0 *Garbage Collection*) atau di **Heap** (lambat, membebani GC). Compiler Go berusaha menaruh sebanyak mungkin variabel di Stack. Namun, pola kode tertentu "memaksa" compiler untuk melempar variabel tersebut ke Heap (*Escape to Heap*).

## Kapan Pola Ini Digunakan (Real-World Use Case)
Pemahaman *Escape Analysis* sangat krusial dalam optimasi aplikasi skala industri:
1. **High-Throughput Services:** Membangun *microservice* yang memproses puluhan ribu *request* per detik (seperti *Real-Time Bidding* atau *Payment Gateway*). Jika setiap request menghasilkan alokasi Heap yang tidak perlu (hanya karena latah menggunakan *pointer* di mana-mana), GC akan sering *pause* (*Stop the World*) dan menyebabkan latensi memburuk.
2. **Library Development:** Membuat *open-source library* (seperti *JSON parser*, *Logger*, atau *Router*). *Developer* lain berekspektasi *library* Anda berlabel "Zero Allocation" (0 `allocs/op`) di jalur panas (*hot-path*).

## Mitos Terbesar di Go: "Semua Pointer Pasti Masuk Heap"
Di bahasa seperti C/C++, sintaks pembuatan pointer (seperti `new`) pasti mengalokasikan memori di Heap. Di Go, ini adalah **MITOS**. 
Go memiliki mekanisme pintar bernama **Escape Analysis**. Aturan mainnya: *"Jika sebuah pointer tidak pernah keluar (escape) dari fungsi tempat ia dibuat, maka ia akan dialokasikan di Stack yang cepat dan gratis GC."*

**Contoh 1: Pointer yang Tetap di Stack (0 Allocs)**
```go
func DoSomething() {
    u := &User{Name: "Alice"} // Ini pointer
    fmt.Println(u.Name)
} // Fungsi selesai, 'u' tidak pernah dikirim ke luar. 
  // Compiler akan menaruh 'u' di STACK!
```

**Contoh 2: Pointer yang Escape ke Heap (1 Allocs) - Kasus Kita**
```go
func CreateUser_Bad() *User {
    u := &User{Name: "Alice"}
    return u // Pointer 'u' dilempar (escape) ke luar fungsi!
} // Stack CreateUser_Bad dihancurkan, jadi 'u' HARUS dipindah ke HEAP agar tidak hangus.
```

## Kasus (Case)
Buka file `escape.go`. Di sana terdapat fungsi `CreateUser_Bad` yang membuat struct `User` dan mengembalikan *pointer* (`*User`). Karena *pointer* tersebut "keluar" (*escape*) dari ruang lingkup fungsi, compiler terpaksa mengalokasikan struct tersebut di Heap (karena Stack akan dihancurkan setelah fungsi selesai).

**Persyaratan Tugas:**
1. Pelajari fungsi `CreateUser_Bad` dan jalankan *benchmark*-nya. Anda akan melihat adanya alokasi memori (1 allocs/op).
2. Tulis implementasi fungsi `CreateUser_Good`.
3. Buatlah agar fungsi tersebut mengembalikan data *User* tanpa menyebabkan alokasi di Heap (0 allocs/op).
4. *(Hint: Gunakan Value Semantics alih-alih Pointer Semantics).*

## Kesalahan Umum (Common Mistakes)
1. **Latah Menggunakan Pointer di Mana-Mana:** Banyak developer Go (terutama yang datang dari Java/C#) menganggap pointer selalu lebih efisien karena "tidak perlu meng-copy data". Padahal untuk struct berukuran kecil-menengah, meng-copy value jauh lebih murah daripada membebani GC dengan alokasi Heap.
2. **Benchmark yang Di-Optimize-Out oleh Compiler:** Jika hasil fungsi dalam benchmark di-assign ke `_` (*blank identifier*), compiler Go bisa saja mendeteksinya sebagai *dead code* dan menghapus seluruh eksekusi fungsi tersebut.
3. **Menganggap "Semua Pointer = Heap":** Di Go, pointer yang dibuat dan digunakan hanya di dalam satu fungsi (tanpa di-return atau disimpan ke variabel luar) tetap akan dialokasikan di Stack oleh Escape Analysis. Pointer baru escape ke Heap jika ia "keluar" dari fungsi pembuatnya.

## Perbaikan / Solusi (Fixes / Solution)
Solusi optimal adalah menggunakan **Value Semantics** — mengembalikan struct secara langsung (bukan pointer) sehingga data tetap berada di Stack dan tidak membebani Garbage Collector.

```go
// BAD: Return pointer → escape to heap → 1 allocs/op
func CreateUser_Bad(id int, name, email string) *User {
    u := User{ID: id, Name: name, Email: email}
    return &u // 'u' DIPAKSA pindah ke Heap
}

// GOOD: Return value → tetap di Stack → 0 allocs/op
func CreateUser_Good(id int, name, email string) User {
    return User{ID: id, Name: name, Email: email}
    // Data di-COPY ke Stack pemanggil, lalu Stack fungsi ini dihancurkan. Bersih!
}
```

**Cara membuktikan di terminal:**
```bash
# Lihat keputusan compiler secara transparan
go build -gcflags="-m" .
# Output: "moved to heap: u" pada fungsi Bad
# Tidak ada "moved to heap" pada fungsi Good
```

## Panduan Keputusan: Pointer vs Value (Perspektif Escape Analysis)

Tidak semua pointer itu buruk. Kadang membiarkan objek escape ke Heap adalah keputusan yang benar. Berikut panduannya:

### Kapan Sebaiknya HINDARI Pointer (Cegah Escape)

**1. Hot-Path — Fungsi yang dipanggil ribuan-jutaan kali per detik**
```go
// ❌ 10,000 req/s = 10,000 alokasi Heap/detik. GC kewalahan.
func ParseToken(raw string) *Token {
    t := Token{Value: raw}
    return &t
}

// ✅ 0 alokasi Heap. GC tidak terganggu.
func ParseToken(raw string) Token {
    return Token{Value: raw}
}
```
**Alasan:** Setiap alokasi Heap memerlukan runtime untuk mencari ruang kosong, menulis metadata objek, dan menambahkannya ke daftar tracking GC. Kalikan jutaan kali per detik = CPU habis untuk urusan memori.

**2. Struct berukuran kecil (di bawah ~256 bytes)**
```go
type Point struct { X, Y float64 } // 16 bytes

// ❌ Membuang 16 bytes ke Heap untuk menghindari copy 16 bytes? Rugi!
func NewPoint() *Point { return &Point{X: 1, Y: 2} }

// ✅ Copy 16 bytes di Stack = beberapa nanosecond, tanpa GC
func NewPoint() Point { return Point{X: 1, Y: 2} }
```
**Alasan:** Copy di Stack hanyalah operasi `memcpy` (beberapa instruksi CPU). Alokasi Heap melibatkan: (1) lock pada memory allocator, (2) cari ruang kosong, (3) tulis header objek, (4) registrasi ke GC tracker. Untuk struct kecil, overhead Heap **puluhan kali lipat** lebih mahal dari copy.

### Kapan TETAP Pakai Pointer (Biarkan Escape, GC Cost Worth It)

**1. Struct berukuran besar — biaya copy lebih mahal dari GC**
```go
type ImageBuffer struct {
    Pixels [1920 * 1080 * 4]byte // ~8 MB per struct!
}

// ✅ Meng-copy 8 MB setiap return = CPU mati. Pointer lebih baik!
func LoadImage(path string) *ImageBuffer {
    img := &ImageBuffer{}
    return img
}
```
**Alasan:** Di titik ini, trade-off berbalik. GC cost untuk 1 objek besar < biaya meng-copy 8 MB data setiap kali fungsi dipanggil.

**2. Objek yang hidupnya panjang (long-lived)**
```go
// Dibuat 1x saat startup, hidup selamanya selama aplikasi berjalan.
func LoadConfig() *AppConfig {
    return &AppConfig{DBHost: "localhost", DBPort: 5432}
}
```
**Alasan:** GC hanya membebani CPU saat **memungut** objek yang sudah mati. Objek yang hidup selamanya (config, singleton, connection pool) tidak pernah dipungut. Escape ke Heap tidak menimbulkan beban GC yang berarti.

**3. Perlu mutasi (mengubah data asli)**
```go
func (o *Order) MarkPaid() {
    o.Status = "paid" // Mengubah struct asli, bukan copy-nya
}
```
**Alasan:** Ini bukan soal performa — ini soal **kebenaran logika program**. Jika pakai value receiver `(o Order)`, yang berubah hanya copy-nya.

**4. Perlu membedakan "nil" vs "kosong" (nullable semantics)**
```go
type UpdateRequest struct {
    Name *string // nil = "field tidak dikirim", "" = "kosongkan field"
}
```
**Alasan:** Value type di Go selalu punya *zero value*. Tidak ada cara lain merepresentasikan "ketiadaan nilai" selain pointer (`nil`).

### Rumus Keputusan Cepat
```
Apakah struct > ~256 bytes?
  └─ YA  → Pakai pointer (copy terlalu mahal)
  └─ TIDAK ↓

Apakah fungsi ini di hot-path (dipanggil ribuan kali/detik)?
  └─ YA  → Hindari pointer (0 allocs = GC tenang)
  └─ TIDAK ↓

Apakah perlu mutasi data asli atau nil semantics?
  └─ YA  → Pakai pointer (tidak ada alternatif)
  └─ TIDAK → Pakai value (default terbaik di Go)
```

---

## Cara Mengerjakan
1. Buka file `escape.go` dan perbaiki fungsi `CreateUser_Good`.
2. Anda bisa membuktikan *Escape Analysis* dengan melihat keputusan *compiler* Go secara transparan. Jalankan perintah ini:
   ```bash
   cd memory/escape
   go build -gcflags="-m" .
   ```
   *(Cari baris yang mengatakan `escapes to heap`)*
3. Jalankan *benchmark* untuk membuktikan *Zero Allocation*:
   ```bash
   go test -bench="." -benchmem
   ```
