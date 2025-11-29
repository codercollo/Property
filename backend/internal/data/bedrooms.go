package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidBedroomsFormat = errors.New("invalid bedrooms format")

type Bedrooms int32

func (b Bedrooms) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("%d beds", b)
	return []byte(strconv.Quote(formatted)), nil
}

func (b *Bedrooms) UnmarshalJSON(jsonValue []byte) error {
	unquoted, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidBedroomsFormat
	}

	parts := strings.Split(unquoted, " ")
	if len(parts) != 2 || parts[1] != "beds" {
		return ErrInvalidBedroomsFormat
	}

	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidBedroomsFormat
	}

	*b = Bedrooms(i)
	return nil
}
