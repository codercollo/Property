package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidPriceFormat = errors.New("invalid price format")

// Price represents a property's price and formats it as a JSON string(KSH)
type Price float64

// Marshal Formats the Price as a string
func (p Price) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("KSh %.2f", p)
	return []byte(strconv.Quote(formatted)), nil
}

func (p *Price) UnmarshalJSON(jsonValue []byte) error {
	// Try to unmarshal as a plain number first
	var num float64
	if err := json.Unmarshal(jsonValue, &num); err == nil {
		*p = Price(num)
		return nil
	}

	// Try to unmarshal as a formatted string
	unquoted, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidPriceFormat
	}

	if !strings.HasPrefix(unquoted, "KSh ") {
		return ErrInvalidPriceFormat
	}

	valStr := strings.TrimPrefix(unquoted, "KSh ")
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return ErrInvalidPriceFormat
	}

	*p = Price(val)
	return nil
}
