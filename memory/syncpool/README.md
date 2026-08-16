# Case 3: Object Reuse & Garbage Collection Optimization (`sync.Pool`)

## Tujuan (Objective)
Tujuan dari studi kasus ini adalah memahami bagaimana cara mengurangi beban *Garbage Collector* (GC) dengan menggunakan `sync.Pool`. Di Go, objek-objek berumur pendek (*short-lived objects*) yang terus-menerus dibuat dan dibuang akan memicu siklus GC yang sering, yang dapat membuat CPU sibuk memungut sampah (GC overhead) daripada menjalankan *business logic*.

## Kapan Pola Ini Digunakan (Real-World Use Case)
Pola **`sync.Pool`** wajib digunakan pada skenario di mana alokasi memori terjadi dengan frekuensi yang sangat ekstrim, contohnya:
1. **HTTP Server / Router:** Framework seperti *Gin* atau *Fiber* menggunakan `sync.Pool` untuk menggunakan kembali *Context* request. Alih-alih membuat struct `gin.Context` baru untuk setiap *incoming request*, mereka mengambilnya dari *pool*, mereset nilainya, lalu memasukannya kembali setelah *request* selesai.
2. **JSON Serialization / Deserialization:** Menggunakan ulang `bytes.Buffer` atau `[]byte` untuk membangun string JSON agar memori tidak terus-menerus membesar dan dibuang saat melayani puluhan ribu QPS (Queries Per Second).

## Kasus (Case)
Buka file `syncpool.go`. Di sana terdapat fungsi `ProcessData` yang saat ini selalu membuat `new(bytes.Buffer)` baru setiap kali dipanggil. Jika fungsi ini dipanggil 10,000 kali per detik, ini akan menyebabkan puluhan ribu alokasi memori membuang-buang siklus CPU untuk dicleanup oleh GC.

**Persyaratan Tugas:**
1. Deklarasikan sebuah *global variable* `var bufferPool sync.Pool`.
2. Modifikasi `bufferPool` dengan mendefinisikan *method* `New` yang bertugas membuat `bytes.Buffer` baru jika di dalam *pool* sedang kosong.
3. Ubah implementasi fungsi `ProcessData`:
   - Ambil (Get) *buffer* dari `bufferPool` menggunakan *Type Assertion*.
   - **PENTING:** Reset (bersihkan) isi *buffer* (Gunakan `buf.Reset()`) sebelum digunakan agar data dari eksekusi sebelumnya tidak ikut tercetak.
   - Kembalikan (Put) *buffer* kembali ke dalam *pool* di akhir eksekusi (sebaiknya gunakan `defer`).
4. Cek seberapa jauh beban memorinya berkurang!

## Kesalahan Umum (Common Mistakes)
1. **Lupa Memanggil `buf.Reset()`:** Jika Anda mengambil buffer dari kolam dan tidak membersihkannya, data dari *request* atau eksekusi sebelumnya akan ikut tercetak (Data Corruption).
2. **Urutan `defer` yang Salah:** Menggunakan `defer bufferPool.Put(buf)` lalu `defer buf.Reset()` secara terpisah sangat berbahaya. Karena `defer` dieksekusi secara LIFO (mundur), buffer yang masih kotor akan dimasukkan ke kolam terlebih dahulu sebelum dibersihkan. Jika goroutine lain mengambilnya di jeda waktu tersebut, datanya akan korup.
3. **Menggunakan `sync.Pool` untuk State Persisten:** Menganggap `sync.Pool` sama seperti koneksi database. `sync.Pool` adalah *cache* yang sangat agresif dibersihkan oleh *Garbage Collector*. Jangan pernah menyimpan objek yang tidak boleh hilang di dalamnya.

## Perbaikan / Solusi (Fixes / Solution)
Solusi optimal untuk menekan beban *Garbage Collector* (GC) adalah menggunakan `sync.Pool` dengan tipe data *Pointer* dan membungkus logika pembersihannya dalam satu fungsi anonim `defer` agar dijamin aman dari *Race Condition*.

```go
var bufferPool = &sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func ProcessDataWithSyncPool(data []byte) string {
	// Ambil dari pool, ubah tipenya dengan Type Assertion
	buf := bufferPool.Get().(*bytes.Buffer)
	
	// Pastikan Reset dilakukan SEBELUM dikembalikan ke pool
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	buf.Write(data)
	buf.WriteString(" - processed")

	// Pemanggilan buf.String() membuat string baru, 
	// namun beban alokasi dari pembuatan 'bytes.Buffer' itu sendiri sudah 0 allocs.
	return buf.String()
}
```

---

## Cara Mengerjakan
1. Jalankan benchmark awal (Sebelum dioptimasi) untuk melihat beban memori saat ini:
   ```bash
   cd memory/syncpool
   go test -bench=. -benchmem
   ```
   *Catat angka `B/op` (Bytes per operation) dan `allocs/op` (Memory allocations per operation).*
2. Edit file `syncpool.go` dan terapkan `sync.Pool`.
3. Jalankan kembali benchmark yang sama dan lihat seberapa drastis penurunannya!
