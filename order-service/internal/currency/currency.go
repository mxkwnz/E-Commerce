package currency

import (
	"fmt"

	"order-service/internal/models"
)

func Normalize(code string) string {
	if code == "" {
		return "USD"
	}
	return code
}

func ValidateCartUniform(items []models.CartItem) (string, error) {
	if len(items) == 0 {
		return "USD", nil
	}
	base := Normalize(items[0].Currency)
	for _, it := range items[1:] {
		if Normalize(it.Currency) != base {
			return "", fmt.Errorf("cart contains mixed currencies (%s and %s)", base, Normalize(it.Currency))
		}
	}
	return base, nil
}

func SumLineTotals(items []models.CartItem) float64 {
	var total float64
	for _, it := range items {
		total += it.UnitPrice * float64(it.Quantity)
	}
	return total
}
