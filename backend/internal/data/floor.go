package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidFloorFormat = errors.New("invalid floor format")

type Floor int32

func (f Floor) MarshalJSON() ([]byte, error) {
	var formatted string
	switch f {
	case 0:
		formatted = "Ground"
	case 1:
		formatted = "1st"
	case 2:
		formatted = "2nd"
	case 3:
		formatted = "3rd"
	default:
		formatted = fmt.Sprintf("%dth", f)
	}
	return []byte(strconv.Quote(formatted)), nil
}

func (f *Floor) UnmarshalJSON(jsonValue []byte) error {
	unquoted, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidFloorFormat
	}

	switch unquoted {
	case "Ground":
		*f = 0
	case "1st":
		*f = 1
	case "2nd":
		*f = 2
	case "3rd":
		*f = 3
	default:
		val, err := strconv.Atoi(strings.TrimSuffix(unquoted, "th"))
		if err != nil {
			return ErrInvalidFloorFormat
		}
		*f = Floor(val)
	}

	return nil
}
