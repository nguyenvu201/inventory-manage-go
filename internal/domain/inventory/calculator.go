package inventory

import (
	"math"
)

// InventoryCalculator computes inventory metrics from net weight.
type InventoryCalculator struct{}

// NewInventoryCalculator creates a new instance of InventoryCalculator
func NewInventoryCalculator() *InventoryCalculator {
	return &InventoryCalculator{}
}

// Calculate logic to compute Qty and Percentage from NetWeight.
// Formula:
// Qty = floor(net_weight / unit_weight_kg)
// Percentage = clamp((net_weight / full_capacity_kg) * 100, 0, 100)
func (ic *InventoryCalculator) Calculate(netWeightKg, unitWeightKg, fullCapacityKg float64) (int, float64) {
	var qty int
	var percentage float64

	if unitWeightKg > 0 && netWeightKg > 0 {
		qty = int(math.Floor(netWeightKg / unitWeightKg))
	} else {
		qty = 0
	}

	if fullCapacityKg > 0 {
		percentage = (netWeightKg / fullCapacityKg) * 100
		
		// Clamp between 0 and 100
		if percentage < 0 {
			percentage = 0
		} else if percentage > 100 {
			percentage = 100
		}

		// Round to 2 decimal places
		percentage = math.Round(percentage*100) / 100
	} else {
		percentage = 0
	}

	return qty, percentage
}
