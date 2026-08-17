package reportexport

import (
	"strconv"
	"strings"
)

// OrderItem mewakili satu item pesanan dari database.
type OrderItem struct {
	OrderID        int64
	TotalPrice     float64
	SKU            string
	DiscountPct    int32
	Qty            int16
	IsReturned     bool
	IsFreeShipping bool
}

// CreateOrderItem membuat satu OrderItem baru dari parameter yang diberikan.
func CreateOrderItem(orderID int64, sku string, qty int16, totalPrice float64, discountPct int32, isFreeShipping bool, isReturned bool) OrderItem {
	return OrderItem{
		OrderID:        orderID,
		SKU:            sku,
		Qty:            qty,
		TotalPrice:     totalPrice,
		DiscountPct:    discountPct,
		IsFreeShipping: isFreeShipping,
		IsReturned:     isReturned,
	}
}

// GenerateCSVReport menerima slice berisi puluhan ribu OrderItem
// dan menghasilkan string CSV lengkap dengan header.
//
// Format output:
// order_id,sku,qty,total_price,discount_pct,free_shipping,returned
// 1001,SKU-A,2,150000.00,10,true,false
// 1002,SKU-B,1,75000.50,0,false,false
// ...
func GenerateCSVReport(items []OrderItem) string {
	sb := &strings.Builder{}
	sb.Grow(len(items) * 100)

	sb.WriteString("order_id,sku,qty,total_price,discount_pct,free_shipping,returned\n")
	var buf [64]byte

	for _, item := range items {
		b := strconv.AppendInt(buf[:0], item.OrderID, 10)
		sb.Write(b)
		sb.WriteByte(',')

		sb.WriteString(item.SKU)
		sb.WriteByte(',')

		b = strconv.AppendInt(buf[:0], int64(item.Qty), 10)
		sb.Write(b)
		sb.WriteByte(',')

		b = strconv.AppendFloat(buf[:0], item.TotalPrice, 'f', 2, 64)
		sb.Write(b)
		sb.WriteByte(',')

		b = strconv.AppendInt(buf[:0], int64(item.DiscountPct), 10)
		sb.Write(b)
		sb.WriteByte(',')

		b = strconv.AppendBool(buf[:0], item.IsFreeShipping)
		sb.Write(b)
		sb.WriteByte(',')

		b = strconv.AppendBool(buf[:0], item.IsReturned)
		sb.Write(b)
		sb.WriteByte('\n')
	}

	return sb.String()
}
