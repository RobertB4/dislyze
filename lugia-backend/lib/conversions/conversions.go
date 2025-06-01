package conversions

import (
	"fmt"
	"math"
)

// SafeInt32 safely converts an int to int32, returning an error if overflow would occur
func SafeInt32(value int) (int32, error) {
	if value > math.MaxInt32 || value < math.MinInt32 {
		return 0, fmt.Errorf("value %d would overflow int32 (range: %d to %d)", value, math.MinInt32, math.MaxInt32)
	}
	return int32(value), nil
}
