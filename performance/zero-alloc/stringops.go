package pprof

import (
	"fmt"
	"strconv"
	"strings"
)

// ProcessData_Bad menerima slice berisi ribuan string pendek
// dan menggabungkannya menjadi satu string besar yang dipisahkan koma.
//
// Contoh: ["alice", "bob", "charlie"] -> "alice,bob,charlie"
//
// Fungsi ini menggunakan pendekatan yang umum dilakukan pemula,
// namun sangat boros memori dan CPU di skala besar.
func ProcessData_Bad(names []string) string {
	result := ""
	for i, name := range names {
		if i > 0 {
			result += ","
		}
		result += name
	}
	return result
}

// ProcessData_Good harus melakukan hal yang persis sama dengan ProcessData_Bad,
// namun tanpa alokasi memori berlebihan.
//
// TODO: Implementasikan fungsi ini menggunakan teknik yang efisien
// agar benchmark menunjukkan alokasi memori (B/op) yang jauh lebih rendah.
//
// Hint: Gunakan package "strings" dari standard library.
func ProcessData_Good(names []string) string {
	return strings.Join(names, ",")
}

// GenerateReport_Bad membuat laporan teks dari data transaksi.
// Ia membangun string besar secara iteratif menggunakan operator +=
func GenerateReport_Bad(transactions []Transaction) string {
	report := "=== TRANSACTION REPORT ===\n"
	for _, tx := range transactions {
		report += fmt.Sprintf("ID: %d | User: %s | Amount: %.2f | Status: %s\n",
			tx.ID, tx.User, tx.Amount, tx.Status)
	}
	report += "=== END OF REPORT ==="
	return report
}

// GenerateReport_Good harus menghasilkan output yang persis sama dengan GenerateReport_Bad,
// namun menggunakan teknik yang jauh lebih efisien.
//
// TODO: Implementasikan fungsi ini.
// Hint: Gunakan strings.Builder.
func GenerateReport_Good(transactions []Transaction) string {
	sb := &strings.Builder{}
	sb.WriteString("=== TRANSACTION REPORT ===\n")
	for _, tx := range transactions {
		fmt.Fprintf(sb, "ID: %d | User: %s | Amount: %.2f | Status: %s\n",
			tx.ID, tx.User, tx.Amount, tx.Status)
	}
	sb.WriteString("=== END OF REPORT ===")

	return sb.String()
}

// GenerateReport_Extreme melakukan hal yang persis sama, namun menggunakan
// pendekatan "True Zero-Allocation" dengan strconv dan sb.Grow().
// Pendekatan ini sangat cepat tapi mengorbankan readability (keterbacaan kode).
func GenerateReport_Extreme(transactions []Transaction) string {
	var sb strings.Builder

	// Pre-allocate: 1 baris transaksi butuh kira-kira ~80 bytes.
	sb.Grow(len(transactions) * 80)

	// Rahasia True Zero-Allocation: 
	// Siapkan array kecil di Stack. Array tidak akan membebani Garbage Collector.
	var buf [64]byte

	sb.WriteString("=== TRANSACTION REPORT ===\n")
	for _, tx := range transactions {
		sb.WriteString("ID: ")
		
		// Konversi angka langsung ke array byte di Stack (0 alloc)
		b := strconv.AppendInt(buf[:0], int64(tx.ID), 10)
		sb.Write(b) // Tulis byte ke Builder
		
		sb.WriteString(" | User: ")
		sb.WriteString(tx.User)
		sb.WriteString(" | Amount: ")
		
		// Konversi float langsung ke array byte di Stack (0 alloc)
		b = strconv.AppendFloat(buf[:0], tx.Amount, 'f', 2, 64)
		sb.Write(b)

		sb.WriteString(" | Status: ")
		sb.WriteString(tx.Status)
		sb.WriteString("\n")
	}
	sb.WriteString("=== END OF REPORT ===")

	return sb.String()
}

// Transaction mewakili satu baris data transaksi.
type Transaction struct {
	User   string
	Amount float64
	Status string
	ID     int
}

// ignore
var _ = strings.NewReader

