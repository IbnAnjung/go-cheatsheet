# Case 6: Data Alignment & Struct Padding

## Tujuan (Objective)
Mempelajari bagaimana *Go Compiler* mengalokasikan memori untuk sebuah `struct`, dan bagaimana sekadar mengubah urutan baris variabel bisa menghemat pemakaian RAM secara signifikan (hingga 40%!) tanpa mengubah logika program sama sekali.

## Apa Itu CPU Cache dan Mengapa Penting?

Sebelum masuk ke kode, kita perlu memahami **mengapa** urutan field di dalam `struct` berpengaruh terhadap performa — dan jawabannya ada di level hardware.

### Hierarki Kecepatan Memori

CPU memproses data jauh lebih cepat daripada kecepatan RAM membacanya. Untuk menjembatani kesenjangan ini, CPU memiliki memori bertingkat yang tertanam langsung di dalam chip CPU (bukan di papan RAM):

| Tipe Memori | Kecepatan Akses | Ukuran |
|---|---|---|
| **L1 Cache** (di dalam Core CPU) | ~4 siklus CPU | 32 - 64 KB |
| **L2 Cache** (dekat Core CPU) | ~10 siklus CPU | 256 KB - 512 KB |
| **RAM Utama** (di luar Chip CPU) | ~200 siklus CPU | Gigabytes |

Artinya, membaca data dari RAM 50x lebih lambat dibandingkan dari L1 Cache.

### Cache Line: CPU Membaca Blok 64 Bytes Sekaligus

CPU **tidak pernah** membaca RAM byte-per-byte. Setiap kali CPU perlu mengambil satu variabel, ia akan mengambil seluruh blok **64 bytes berurutan** dari RAM ke dalam Cache — blok ini disebut **Cache Line**.

Inilah mengapa ukuran dan urutan *field* di dalam `struct` sangat berpengaruh:

* **Struct Kecil & Teratur (24 bytes):** Satu Cache Line (64 bytes) dapat menampung **2 objek utuh sekaligus**. Saat CPU memproses objek pertama, objek kedua sudah otomatis tersedia di L1 Cache (**Cache Hit**).
* **Struct Bengkak & Berantakan (40 bytes):** Satu Cache Line hanya mampu menampung **1 objek**. Untuk memproses objek berikutnya, CPU harus membaca ke RAM lagi (**Cache Miss** — 50x lebih lambat).

### Kapan Data Dihapus dari CPU Cache?

Karena kapasitas CPU Cache sangat kecil dan cepat penuh, CPU menggunakan algoritma **LRU (Least Recently Used)** — data yang paling lama tidak diakses akan ditimpa oleh data baru. Jika data sempat dimodifikasi di Cache sebelum ditimpa, CPU akan secara otomatis melakukan **Write-Back** (menulis perubahan tersebut kembali ke RAM) agar data tetap konsisten.

Semua proses ini terjadi sepenuhnya di level hardware, di luar jangkauan program Go Anda.

---

## Kapan Pola Ini Digunakan (Real-World Use Case)
1. **High-Density Caching:** Jika Anda membuat *in-memory cache* yang menampung puluhan juta objek `struct` di RAM. Selisih 16 bytes per struct dikalikan 10 juta objek adalah penghematan RAM langsung sebesar **~150 Megabytes**, hanya dengan mengubah urutan baris deklarasi!
2. **Game Servers & IoT:** Saat Anda harus memproses *state* ratusan ribu entitas dalam satu siklus CPU, struct yang rapat membuat seluruh *array* muat di dalam L1/L2 Cache, mempercepat kalkulasi berkali-kali lipat.

---

## Kasus (Case)
Buka file `alignment.go`. Anda akan melihat struct bernama `User_Bad`. Walaupun jika ditotal, tipe data aslinya hanya membutuhkan memori **24 bytes** (8+8+4+2+1+1), `unsafe.Sizeof` akan melaporkan bahwa struct tersebut memakan RAM sebesar **40 bytes**!

Mengapa bengkak? Karena CPU modern (64-bit) membaca memori dalam blok sebesar **8 bytes**. Jika sebuah variabel kecil (seperti `bool` berukuran 1 byte) diletakkan di antara dua variabel 8 bytes, compiler Go akan **menyisipkan ruang kosong (padding)** agar variabel berikutnya sejajar (*aligned*) dengan blok CPU. Ruang kosong ini murni sampah RAM yang terbuang.

**Persyaratan Tugas:**
1. Pelajari ukuran tipe data berikut:
   - `int64`, `float64` = 8 bytes
   - `int32`, `float32` = 4 bytes
   - `int16` = 2 bytes
   - `bool`, `int8`, `byte` = 1 byte
2. Implementasikan struct `User_Good` dengan *field* yang persis sama dengan `User_Bad`.
3. Namun, **urutkan field tersebut dari ukuran tipe data yang paling besar ke yang paling kecil**.
4. Pastikan `User_Good` ukurannya turun menjadi **24 bytes** (menghemat 40% memori!).

---

## Kesalahan Umum (Common Mistakes)
1. **Mengabaikan Urutan Deklarasi Field Struct:** Developer sering menulis urutan field berdasarkan pengelompokan logika bisnis tanpa sadar bahwa compiler akan menyisipkan *padding* tersembunyi di antaranya.

   **❌ Boros Memori (40 bytes karena padding berantakan):**
   ```go
   type User_Bad struct {
       IsActive  bool    // 1 byte  -> compiler sisipkan (+7 bytes padding di sini)
       Salary    float64 // 8 bytes
       Age       int16   // 2 bytes -> compiler sisipkan (+6 bytes padding di sini)
       ID        int64   // 8 bytes
       RoleID    int32   // 4 bytes
       IsPremium bool    // 1 byte  -> compiler sisipkan (+7 bytes padding di sini)
   } // Total: 40 bytes (padahal data aslinya hanya 24 bytes!)
   ```

   **✅ Hemat Memori (24 bytes, diurutkan dari terbesar ke terkecil):**
   ```go
   type User_Good struct {
       ID        int64   // 8 bytes
       Salary    float64 // 8 bytes
       RoleID    int32   // 4 bytes
       Age       int16   // 2 bytes
       IsPremium bool    // 1 byte
       IsActive  bool    // 1 byte
       //                   + 2 bytes padding di akhir (minimal, hanya untuk sejajar batas 8 bytes)
   } // Total: 24 bytes (menghemat 40%!)
   ```

2. **Pointer Chasing (Heap Fragmentation):** Membuat banyak objek yang dialokasikan secara acak di Heap menggunakan pointer alih-alih menyimpannya bersebelahan di dalam `[]Slice`. Setiap lompatan pointer ke alamat RAM yang acak hampir selalu memicu **Cache Miss**, karena data yang dibutuhkan tidak berada dalam blok Cache Line yang sudah dimuat.

---

## Perbaikan / Solusi (Fixes / Solution)
Susun *field* di dalam `struct` dari **ukuran tipe data terbesar ke terkecil**. Aturan praktisnya:

```
int64 / float64 (8 bytes) → int32 / float32 (4 bytes) → int16 (2 bytes) → bool / byte (1 byte)
```

Tindakan ini meminimalkan *padding* kosong yang disisipkan compiler, memaksimalkan penggunaan setiap Cache Line oleh CPU, dan meningkatkan efisiensi RAM hingga **40%** per objek — tanpa mengubah satu baris logika program pun.

---

## Cara Mengerjakan
1. Buka file `alignment.go` dan ubah urutan field di struct `User_Good`.
2. Uji keberhasilan Anda dengan menjalankan:
   ```bash
   go test -v -run TestStructSize
   ```
   *Test akan memberitahu Anda berapa bytes memori yang berhasil Anda hemat!*
