package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidBathroomsFormat = errors.New("invalid bathrooms format")

type Bathrooms int32

func (b Bathrooms) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("%d baths", b)
	return []byte(strconv.Quote(formatted)), nil
}

func (b *Bathrooms) UnmarshalJSON(jsonValue []byte) error {
	// Try to unmarshal as a plain number first
	var num int32
	if err := json.Unmarshal(jsonValue, &num); err == nil {
		*b = Bathrooms(num)
		return nil
	}

	// Try to unmarshal as a formatted string
	unquoted, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidBathroomsFormat
	}

	parts := strings.Split(unquoted, " ")
	if len(parts) != 2 || parts[1] != "baths" {
		return ErrInvalidBathroomsFormat
	}

	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidBathroomsFormat
	}

	*b = Bathrooms(i)
	return nil
}
