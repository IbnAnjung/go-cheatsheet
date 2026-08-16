package escape

// User mewakili entitas pengguna
type User struct {
	ID    int
	Name  string
	Email string
}

// CreateUser_Bad mensimulasikan fungsi yang secara tidak sengaja menyebabkan
// objek `User` "escape to heap" (terlempar ke heap memori), sehingga membebani GC.
//
// TODO: Buat fungsi `CreateUser_Good` yang melakukan hal yang persis sama,
// namun memastikan objek `User` tetap berada di Stack (0 heap allocations).
func CreateUser_Bad(id int, name, email string) *User {
	u := User{
		ID:    id,
		Name:  name,
		Email: email,
	}
	// Mengembalikan pointer ke lokal variabel memaksa variabel ini "escape to Heap"
	// karena memori variabel ini harus tetap hidup setelah fungsi CreateUser_Bad selesai.
	return &u
}

func CreateUser_Good(id int, name, email string) User {
	// Implementasikan pembuatan user di sini,
	// pastikan ia dialokasikan di Stack, bukan di Heap! (Gunakan Value Semantics)
	return User{
		ID:    id,
		Name:  name,
		Email: email,
	}
}
