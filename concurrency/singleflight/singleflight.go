package singleflight

import "sync"

// call merepresentasikan eksekusi sebuah fungsi fn yang sedang berjalan
// atau sudah selesai (jika hasilnya sedang dibaca oleh goroutine penunggu).
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group mengatur eksekusi dari fungsi-fungsi agar request duplikat
// bisa ditekan (suppressed) menjadi satu eksekusi tunggal.
type Group struct {
	mu sync.Mutex       // melindungi map 'm'
	m  map[string]*call // map akan diinisialisasi secara lazy (saat dibutuhkan)
}

// Do mengeksekusi fungsi 'fn' berdasarkan 'key'.
// Jika ada pemanggilan Do dengan key yang sama sementara pemanggilan sebelumnya
// belum selesai, pemanggil kedua harus menunggu dan mengembalikan hasil yang sama
// dengan pemanggil pertama.
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	// TODO: Implementasikan logika singleflight di sini

	// 1. Lindungi akses ke map 'm' dengan Mutex.
	// 2. Inisialisasi map 'm' jika belum ada (lazy init).
	// 3. Cek apakah sudah ada 'call' yang berjalan untuk key ini.
	// 4a. Jika ADA:
	//     - Lepas Mutex map.
	//     - Tunggu (Wait) hingga 'call' tersebut selesai.
	//     - Return nilai (val, err) dari 'call' tersebut.
	// 4b. Jika BELUM ADA:
	//     - Buat objek 'call' baru.
	//     - Tambahkan '1' ke WaitGroup dari 'call' tersebut.
	//     - Masukkan 'call' ke map 'm' untuk key ini.
	//     - Lepas Mutex map.
	//     - Eksekusi fungsi 'fn()'.
	//     - Simpan hasil (val, err) eksekusi ke objek 'call'.
	//     - Beri sinyal selesai (Done) pada WaitGroup.
	//     - Hapus key dari map 'm' (jangan lupa pakai Mutex saat menghapus!).
	//     - Return hasil eksekusi.
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	if g.m == nil {
		g.m = make(map[string]*call)
	}

	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
