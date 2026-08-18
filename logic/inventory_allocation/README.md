# Objective
Melatih logika pemrograman dasar, pemrosesan struktur data (map, slice), pengurutan (sorting), dan penanganan *edge cases* dalam algoritma alokasi bersyarat.

# Real-World / Production Use Case
Dalam sistem *E-Commerce* atau manajemen rantai pasok (Supply Chain Management), ketika pengguna melakukan *checkout* pesanan yang terdiri dari beberapa macam produk, sistem harus memutuskan dari gudang mana produk-seproduk tersebut akan dikirim. Algoritma harus meminimalkan pengiriman terpisah (*split shipment*) untuk menekan biaya logistik, namun tetap memastikan barang bisa terpenuhi jika stok tersebar di banyak gudang.

# Case
Anda diminta untuk mengimplementasikan fungsi `AllocateOrder(order Order, warehouses []Warehouse) ([]Fulfillment, error)`.

**Aturan Bisnis (Business Rules):**
1. **Single Fulfillment (Prioritas Utama):** 
   Cari SATU gudang yang dapat memenuhi SELURUH pesanan. Jika ada lebih dari satu gudang yang mampu, pilih gudang dengan nilai `Priority` paling kecil (Prioritas 1 lebih tinggi dari 2).
2. **Split Fulfillment (Prioritas Kedua):**
   Jika tidak ada satupun gudang yang bisa memenuhi seluruh pesanan sendirian, Anda harus memecah pesanan (Split Order) ke beberapa gudang.
3. Saat melakukan Split Fulfillment, Anda harus memprioritaskan pengambilan stok dari gudang dengan nilai `Priority` paling kecil. Ambil stok semaksimal mungkin dari gudang prioritas tertinggi tersebut, lalu ambil sisanya dari gudang prioritas berikutnya.
4. **Insufficient Stock:**
   Jika total gabungan stok dari SEMUA gudang tidak mencukupi untuk memenuhi pesanan, kembalikan error `ErrInsufficientStock`.
5. **Clean Result:**
   Gudang yang tidak berkontribusi dalam pemenuhan pesanan (stok yang dialokasikan 0) tidak boleh masuk ke dalam *slice* `[]Fulfillment` yang direturn.

# Common Mistakes

### 1. Mutasi Parameter Bertipe Map (Reference Type)
Map di Golang adalah *reference type*. Jika fungsi Anda mengubah isi map yang dikirim sebagai parameter, map di sisi pemanggil (*caller*) juga akan ikut berubah tanpa disadari. Ini bisa menyebabkan bug yang sulit dilacak jika data pesanan masih digunakan untuk proses selanjutnya (seperti notifikasi atau riwayat transaksi).

❌ **Berbahaya (Map Parameter Termutasi):**
```go
func AllocateOrder(order Order, warehouses []Warehouse) ([]Fulfillment, error) {
    // ...
    for item, qty := range order.Items {
        // ...
        order.Items[item] -= qty // MENGUBAH MAP MILIK CALLER!
        if order.Items[item] == 0 {
            delete(order.Items, item) // MENGHAPUS DATA MILIK CALLER!
        }
    }
}
```

✅ **Pendekatan Terbaik (Gunakan Map Baru):**
```go
func AllocateOrder(order Order, warehouses []Warehouse) ([]Fulfillment, error) {
    // Buat salinan map untuk melacak sisa barang yang perlu dipenuhi
    remainingItems := make(map[string]int, len(order.Items))
    for k, v := range order.Items {
        remainingItems[k] = v
    }
    
    // Gunakan remainingItems untuk logika pengurangan stok, 
    // sehingga order.Items milik caller tetap utuh.
    // ...
}
```

### 2. Mengalokasikan Gudang Tanpa Kontribusi (Gudang Kosong)
Menambahkan semua gudang ke dalam daftar `fulfillments` walaupun gudang tersebut tidak berkontribusi sama sekali mengalokasikan stok (stok yang diberikan `0`), yang melanggar aturan kebersihan data output (*Clean Result*).

❌ **Berbahaya (Memasukkan Gudang Kosong & Alokasi Map Tidak Efisien):**
```go
for _, wh := range warehouses {
    var whItems = map[string]int{} // Selalu alokasi map baru
    for item, qty := range remainingItems {
        // ... alokasi stok
    }
    fulfillments = append(fulfillments, Fulfillment{ // ❌ Selalu memasukkan meskipun whItems kosong
        WarehouseID: wh.ID,
        Items:       whItems,
    })
}
```

✅ **Pendekatan Terbaik (Lazy Map Initialization & Conditional Append):**
```go
for _, wh := range warehouses {
    var whItems map[string]int // nil secara default, tidak memakan alokasi memori map
    for item, qty := range remainingItems {
        // ... cari ketersediaan stok
        if whItems == nil {
            whItems = make(map[string]int) // ✅ Di-inisialisasi HANYA saat ada item yang dialokasikan
        }
        whItems[item] = take
        // ...
    }
    if whItems != nil { // ✅ Hanya masukkan gudang jika benar-benar berkontribusi
        fulfillments = append(fulfillments, Fulfillment{
            WarehouseID: wh.ID,
            Items:       whItems,
        })
    }
}
```

# Fixes / Solution

1. **Pengurutan (Sorting):** Kita wajib mengurutkan `warehouses` berdasarkan `Priority` terlebih dahulu dengan `sort.Slice()` agar gudang dengan prioritas tertinggi (angka paling kecil) diproses terlebih dahulu.
2. **Phase 1 (Single Fulfillment Check):** 
   - Lakukan pengecekan menyeluruh pada setiap gudang.
   - Menggunakan perulangan bersyarat: jika ditemukan satu item dengan stok kurang di gudang tersebut, set `isFullfillment = false` dan lakukan `break` untuk lanjut ke gudang berikutnya.
   - Jika `isFullfillment` tetap `true` setelah memeriksa seluruh item, segera kembalikan *fulfillment* tunggal dari gudang tersebut (karena pasti yang paling prioritas).
3. **Phase 2 (Split Fulfillment):**
   - Jika tidak ada yang mampu memenuhi sendirian, kloning `order.Items` ke `remainingItems` menggunakan `maps.Clone`.
   - Iterasi gudang berdasarkan prioritas, dan cari item yang masih tersisa di `remainingItems`.
   - Terapkan **Lazy Initialization** untuk map alokasi gudang (`whItems`) agar menghindari alokasi memori sia-sia pada gudang yang tidak berkontribusi.
   - Gunakan fungsi `min(qty, whQty)` bawaan Go (mulai Go 1.21) untuk menentukan jumlah alokasi barang secara aman dan efisien.
   - Hapus item dari `remainingItems` jika sisa kebutuhan item tersebut telah mencapai `0`.
   - Jika `remainingItems` sudah kosong di tengah loop gudang, gunakan `break` untuk *early exit*.
4. **Validasi Akhir:** Di akhir fungsi, periksa apakah `remainingItems` masih memiliki sisa barang. Jika masih ada sisa, kembalikan `nil, ErrInsufficientStock`. Jika tidak, kembalikan daftar `fulfillments` dan `nil`.
