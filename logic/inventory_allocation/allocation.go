package inventory

import (
	"errors"
	"maps"
	"sort"
)

var ErrInsufficientStock = errors.New("insufficient stock to fulfill the order")

type Order struct {
	ID    string
	Items map[string]int // key: ProductID, value: Quantity requested
}

type Warehouse struct {
	ID        string
	Priority  int
	Inventory map[string]int // key: ProductID, value: Quantity available
}

type Fulfillment struct {
	WarehouseID string
	Items       map[string]int // key: ProductID, value: Quantity allocated
}

// AllocateOrder mengalokasikan stok dari gudang-gudang untuk memenuhi pesanan.
func AllocateOrder(order Order, warehouses []Warehouse) ([]Fulfillment, error) {
	sort.Slice(warehouses, func(i, j int) bool {
		return warehouses[i].Priority < warehouses[j].Priority
	})

	for _, wh := range warehouses {
		isFullfillment := true
		for item, qty := range order.Items {
			if wh.Inventory[item] < qty {
				isFullfillment = false
				break
			}
		}

		if isFullfillment {
			return []Fulfillment{
				{
					WarehouseID: wh.ID,
					Items:       maps.Clone(order.Items),
				},
			}, nil
		}
	}

	remainingItems := maps.Clone(order.Items)
	var fulfillments []Fulfillment

	for _, wh := range warehouses {
		var whItems map[string]int
		for item, qty := range remainingItems {
			whQty, ok := wh.Inventory[item]
			if !ok || whQty == 0 {
				continue
			}

			take := min(qty, whQty)

			if whItems == nil {
				whItems = make(map[string]int)
			}

			whItems[item] = take
			remainingItems[item] -= take
			if remainingItems[item] == 0 {
				delete(remainingItems, item)
			}
		}

		if whItems != nil {
			fulfillments = append(fulfillments, Fulfillment{
				WarehouseID: wh.ID,
				Items:       whItems,
			})
		}

		if len(remainingItems) == 0 {
			break
		}
	}

	if len(remainingItems) > 0 {
		return nil, ErrInsufficientStock
	}

	return fulfillments, nil
}
