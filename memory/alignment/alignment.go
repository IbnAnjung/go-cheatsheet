package alignment

// User_Bad mensimulasikan struct yang penulisannya tidak memperhatikan
// Data Alignment. Perhatikan urutan acak tipe datanya.
//
// TODO: Buat struct `User_Good` yang memiliki field yang persis sama,
// namun urutkan field tersebut dari ukuran byte terbesar ke terkecil
// agar tidak ada "lubang" memori (padding) yang terbuang sia-sia.
type User_Bad struct {
	IsActive  bool    // 1 byte
	Salary    float64 // 8 bytes
	Age       int16   // 2 bytes
	ID        int64   // 8 bytes
	RoleID    int32   // 4 bytes
	IsPremium bool    // 1 byte
}

// User_Good adalah versi yang seharusnya dioptimasi.
// Pindahkan urutan field di bawah ini agar hemat memori!
type User_Good struct {
	ID        int64
	Salary    float64
	RoleID    int32
	Age       int16
	IsPremium bool
	IsActive  bool
}
