package core

import "fmt"

func issueCost(valueCents, quantity, onHand int64) (int64, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("%w: issue quantity must be positive", ErrInvalid)
	}
	if onHand <= 0 || valueCents < 0 {
		return 0, fmt.Errorf("%w: inventory has no value to issue", ErrConflict)
	}
	if quantity > onHand {
		return 0, fmt.Errorf("%w: issue exceeds on-hand stock", ErrConflict)
	}
	if quantity == onHand {
		return valueCents, nil
	}
	cost := (valueCents*quantity + onHand/2) / onHand
	if cost > valueCents {
		return valueCents, nil
	}
	return cost, nil
}
