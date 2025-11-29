package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidAreaFormat = errors.New("invalid area format")

type Area int32

func (a Area) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("%d m^2", a)
	return []byte(strconv.Quote(formatted)), nil
}

func (a *Area) UnmarshalJSON(jsonValue []byte) error {
	unquoted, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidAreaFormat
	}

	parts := strings.Split(unquoted, " ")
	if len(parts) != 2 || parts[1] != "m^2" {
		return ErrInvalidAreaFormat
	}

	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidAreaFormat
	}

	*a = Area(i)
	return nil
}
