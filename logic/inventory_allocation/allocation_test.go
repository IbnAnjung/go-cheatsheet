package inventory

import (
	"reflect"
	"testing"
)

func TestAllocateOrder(t *testing.T) {
	tests := []struct {
		name       string
		order      Order
		warehouses []Warehouse
		wantResult []Fulfillment
		wantErr    error
	}{
		{
			name: "Single fulfillment possible from one warehouse",
			order: Order{
				ID: "ORD-001",
				Items: map[string]int{
					"A": 2,
					"B": 1,
				},
			},
			warehouses: []Warehouse{
				{
					ID:       "W-Jakarta",
					Priority: 2,
					Inventory: map[string]int{
						"A": 5,
						"B": 5,
					},
				},
				{
					ID:       "W-Surabaya",
					Priority: 1,
					Inventory: map[string]int{
						"A": 1, // Not enough A
						"B": 10,
					},
				},
			},
			wantResult: []Fulfillment{
				{
					WarehouseID: "W-Jakarta",
					Items: map[string]int{
						"A": 2,
						"B": 1,
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "Multiple warehouses can fulfill, choose highest priority (smallest number)",
			order: Order{
				ID: "ORD-002",
				Items: map[string]int{
					"A": 2,
				},
			},
			warehouses: []Warehouse{
				{
					ID:       "W-LowPri",
					Priority: 5,
					Inventory: map[string]int{
						"A": 10,
					},
				},
				{
					ID:       "W-HighPri",
					Priority: 1,
					Inventory: map[string]int{
						"A": 10,
					},
				},
				{
					ID:       "W-MidPri",
					Priority: 3,
					Inventory: map[string]int{
						"A": 10,
					},
				},
			},
			wantResult: []Fulfillment{
				{
					WarehouseID: "W-HighPri",
					Items: map[string]int{
						"A": 2,
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "Split fulfillment required, respect priority",
			order: Order{
				ID: "ORD-003",
				Items: map[string]int{
					"A": 5,
					"B": 3,
				},
			},
			warehouses: []Warehouse{
				{
					ID:       "W1",
					Priority: 2,
					Inventory: map[string]int{
						"A": 2,
						"B": 1, // Will take all this (Pri 2)
					},
				},
				{
					ID:       "W2",
					Priority: 1,
					Inventory: map[string]int{
						"A": 1,
						"B": 0, // Will take all this (Pri 1)
					},
				},
				{
					ID:       "W3",
					Priority: 3,
					Inventory: map[string]int{
						"A": 3, // Changed from 5 so it cannot fulfill the entire A:5
						"B": 5, 
					},
				},
			},
			wantResult: []Fulfillment{
				{
					WarehouseID: "W2",
					Items: map[string]int{
						"A": 1,
					},
				},
				{
					WarehouseID: "W1",
					Items: map[string]int{
						"A": 2,
						"B": 1,
					},
				},
				{
					WarehouseID: "W3",
					Items: map[string]int{
						"A": 2,
						"B": 2,
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "Insufficient total stock",
			order: Order{
				ID: "ORD-004",
				Items: map[string]int{
					"A": 10,
				},
			},
			warehouses: []Warehouse{
				{
					ID:       "W1",
					Priority: 1,
					Inventory: map[string]int{
						"A": 5,
					},
				},
				{
					ID:       "W2",
					Priority: 2,
					Inventory: map[string]int{
						"A": 4, // Total is only 9
					},
				},
			},
			wantResult: nil,
			wantErr:    ErrInsufficientStock,
		},
		{
			name: "Partial item missing entirely",
			order: Order{
				ID: "ORD-005",
				Items: map[string]int{
					"A": 1,
					"Z": 1, // Product Z doesn't exist anywhere
				},
			},
			warehouses: []Warehouse{
				{
					ID:       "W1",
					Priority: 1,
					Inventory: map[string]int{
						"A": 5,
					},
				},
			},
			wantResult: nil,
			wantErr:    ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotErr := AllocateOrder(tt.order, tt.warehouses)

			if gotErr != tt.wantErr {
				t.Errorf("AllocateOrder() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			// We need to compare ignoring the order of fulfillment results
			// since depending on implementation the slice order might vary slightly, 
			// though it *should* ideally be ordered by priority for split shipments.
			// Let's do a strict match on contents.
			
			if len(gotResult) != len(tt.wantResult) {
				t.Fatalf("AllocateOrder() returned %d fulfillments, want %d. Got: %v", len(gotResult), len(tt.wantResult), gotResult)
			}

			// Helper to check if a fulfillment exists in wantResult
			for _, res := range gotResult {
				found := false
				for _, want := range tt.wantResult {
					if res.WarehouseID == want.WarehouseID {
						if reflect.DeepEqual(res.Items, want.Items) {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("Unexpected fulfillment found or items mismatch: %v. Expected among: %v", res, tt.wantResult)
				}
			}
		})
	}
}
