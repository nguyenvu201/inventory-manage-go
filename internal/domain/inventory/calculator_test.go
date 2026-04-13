package inventory_test

import (
	"inventory-manage/internal/domain/inventory"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestInventoryCalculator_Calculate(t *testing.T) {
	calc := inventory.NewInventoryCalculator()

	tests := []struct {
		name           string
		netWeightKg    float64
		unitWeightKg   float64
		fullCapacityKg float64
		wantQty        int
		wantPercentage float64
	}{
		{
			name:           "AC-02: Normal calculation",
			netWeightKg:    15.0,
			unitWeightKg:   2.5,
			fullCapacityKg: 50.0,
			wantQty:        6,
			wantPercentage: 30.0,
		},
		{
			name:           "AC-02: Floor logic",
			netWeightKg:    16.0,
			unitWeightKg:   2.5,
			fullCapacityKg: 50.0,
			// Floor(16.0 / 2.5) = Floor(6.4) = 6
			wantQty:        6,
			wantPercentage: 32.0,
		},
		{
			name:           "Empty variables",
			netWeightKg:    15.0,
			unitWeightKg:   0,
			fullCapacityKg: 0,
			wantQty:        0,
			wantPercentage: 0,
		},
		{
			name:           "Negative weight (clamp 0)",
			netWeightKg:    -5.0,
			unitWeightKg:   2.5,
			fullCapacityKg: 50.0,
			wantQty:        0,
			wantPercentage: 0.0,
		},
		{
			name:           "Overweight (clamp 100)",
			netWeightKg:    55.0,
			unitWeightKg:   2.5,
			fullCapacityKg: 50.0,
			wantQty:        22,
			wantPercentage: 100.0,
		},
		{
			name:           "Percentage rounding",
			netWeightKg:    10.0,
			unitWeightKg:   2.5,
			fullCapacityKg: 30.0,
			wantQty:        4,
			wantPercentage: 33.33,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qty, percentage := calc.Calculate(tt.netWeightKg, tt.unitWeightKg, tt.fullCapacityKg)
			assert.Equal(t, tt.wantQty, qty)
			assert.InDelta(t, tt.wantPercentage, percentage, 0.01)
		})
	}
}
