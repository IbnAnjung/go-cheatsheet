# Objective
Melatih pemahaman tentang *Memory Management*, *Escape Analysis*, dan teknik *Zero-Allocation* di Go menggunakan `sync.Pool` untuk meminimalkan beban Garbage Collector (GC).

# Real-World / Production Use Case
Dalam sistem *High-Throughput* seperti *Log Aggregator* atau *API Gateway*, aplikasi menerima puluhan ribu baris teks per detik. Jika setiap kali aplikasi memproses satu baris teks, ia membuat struct/pointer baru di heap, maka laju alokasi memori (*allocation rate*) akan meroket tajam. Hal ini akan memaksa Garbage Collector berjalan terus-menerus, memakan CPU yang seharusnya digunakan untuk memproses request, dan meningkatkan latensi.

Dengan memanfaatkan `sync.Pool` (untuk daur ulang memori pointer struct), kita dapat mencapai kinerja *zero-allocation* di *hot path*.

# Case
Anda diminta mengoptimasi fungsi `ParseLogLine(line string) (*LogEntry, error)`.

Log berbentuk sederhana: `[LEVEL] MESSAGE`
Contoh: `[INFO] User logged in`

Saat ini implementasinya (di `parser.go`) berfungsi dengan benar. Namun, karena ia me-return pointer `*LogEntry`, objek tersebut "escape ke heap". Jika dijalankan jutaan kali, ini akan memicu alokasi heap yang masif.

**Tugas Anda:**
1. Implementasikan `sync.Pool` secara global di package tersebut untuk mendaur ulang objek `LogEntry`. Jangan alokasikan struct baru menggunakan `&LogEntry{}` setiap kali fungsi dipanggil, melainkan ambil dari *pool* dan kembalikan ke *pool* saat selesai.
2. Sediakan fungsi pendamping `ReleaseLogEntry(entry *LogEntry)` agar pengguna (pemanggil) bisa mengembalikan objek tersebut ke dalam *pool* setelah selesai digunakan.

**Petunjuk:**
- Anda bisa menjalankan benchmark dengan: `go test -bench=. -benchmem ./memory/log_parser`
- Saat ini benchmark akan mencetak `1 allocs/op`.
- Target akhir Anda adalah membuat benchmark tersebut mencapai **0 allocs/op**.
- Untuk mencapai ini, Anda harus meng-uncomment pemanggilan `ReleaseLogEntry` di dalam file test nanti.

# Common Mistakes

### 1. Lupa Me-reset State Objek Sebelum Dikembalikan ke Pool
Objek yang ditarik dari `sync.Pool` adalah objek bekas pakai. Jika Anda tidak me-reset isi datanya (terutama yang berupa pointer, slice, atau map) sebelum melakukan `Put`, data lama bisa "bocor" (Data Leak) ke proses berikutnya, atau mencegah Garbage Collector membersihkan memori yang direferensikan oleh objek tersebut (Memory Leak).

❌ **Berbahaya (Langsung Memasukkan Objek Kotor):**
```go
func ReleaseLogEntry(entry *LogEntry) {
    // ❌ Berbahaya jika struct memiliki field berupa pointer/slice berukuran besar.
    // Data lama akan menetap di memori pool dan memblokir GC.
    logEntryPool.Put(entry) 
}
```

✅ **Pendekatan Terbaik (Reset/Bersihkan State):**
```go
func ReleaseLogEntry(entry *LogEntry) {
    // ✅ Bersihkan data sebelum dikembalikan (Zeroing)
    entry.Level = ""
    entry.Message = ""
    logEntryPool.Put(entry)
}
```
*(Catatan: Untuk kasus string sederhana seperti pada `LogEntry` ini, dampaknya tidak fatal karena field selalu ditimpa ulang di fungsi pemanggil, tetapi me-reset data adalah *best practice* wajib saat bekerja dengan `sync.Pool`).*

### 2. Menggunakan `sync.Pool` Secara Membabi-buta
`sync.Pool` dirancang untuk meringankan beban GC dari objek yang **pasti escape ke heap** (seperti mengembalikan pointer `*LogEntry` keluar dari fungsi). Jika objek Anda sebenarnya hanya digunakan di dalam satu fungsi secara lokal (tersimpan di Stack), memasukkannya ke dalam `sync.Pool` justru akan **memperlambat** kinerja karena proses penarikan dari Pool (`Get/Put`) memiliki *overhead* (penguncian internal/spinlock) yang lebih lambat dibanding alokasi lokal Stack.

# Fixes / Solution

Dengan mengimplementasikan `sync.Pool`, kita berhasil mencapai kinerja luar biasa yaitu **0 allocs/op (Zero-Allocation)**! 
Berikut adalah ringkasan solusinya:

1. **Deklarasi Pool Global:** Kita mendefinisikan `var logEntryPool = sync.Pool{ New: func() any { return new(LogEntry) } }` yang bertugas menyuplai objek baru jika pool sedang kosong (terkena *flush* oleh siklus GC).
2. **Mengambil Objek (Get):** Di dalam `ParseLogLine`, alih-alih melakukan `entry := &LogEntry{}`, kita menariknya dari pool menggunakan `entry := logEntryPool.Get().(*LogEntry)`. Ini menyelamatkan kita dari alokasi heap baru.
3. **Mengembalikan Objek (Put):** Pengguna diwajibkan memanggil `ReleaseLogEntry(entry)` setelah selesai memproses data untuk mendaur ulangnya.
4. **Hasil Kinerja:** Objek `LogEntry` yang sama akan diputar ulang secara ekstrem. Buktinya, benchmark yang awalnya berukuran **32 B/op (1 allocs/op)** memakan waktu **42 ns/op** kini turun drastis menjadi **0 B/op (0 allocs/op)** dan hanya **16 ns/op**. Kita berhasil menghemat 60% waktu eksekusi (CPU) murni hanya dari teknik penggunaan ulang memori!
