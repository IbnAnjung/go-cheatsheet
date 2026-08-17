package reportexport

import (
	"testing"
	"unsafe"
)

func generateTestItems(n int) []OrderItem {
	items := make([]OrderItem, n)
	for i := range items {
		items[i] = OrderItem{
			OrderID:        int64(1000 + i),
			SKU:            "SKU-" + string(rune('A'+i%26)),
			Qty:            int16(1 + i%10),
			TotalPrice:     float64(i) * 99.99,
			DiscountPct:    int32(i % 50),
			IsFreeShipping: i%3 == 0,
			IsReturned:     i%7 == 0,
		}
	}
	return items
}

// === BENCHMARKS ===

const benchSize = 50_000

func BenchmarkGenerateCSVReport(b *testing.B) {
	items := generateTestItems(benchSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateCSVReport(items)
	}
}

// Tambahkan benchmark untuk versi perbaikan Anda di bawah ini.
// Pastikan outputnya identik dengan GenerateCSVReport.

// === STRUCT SIZE ===

func TestOrderItemSize(t *testing.T) {
	size := unsafe.Sizeof(OrderItem{})
	t.Logf("OrderItem size: %d bytes", size)
}
