package integrity

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var ErrInvalidSubset = errors.New("invalid data subset (use a percentage such as 10% or a part such as 2/7)")

type Scope struct {
	readData bool
	subset   string
}

func StructureOnly() Scope {
	return Scope{}
}

func AllData() Scope {
	return Scope{readData: true}
}

func DataSubset(value string) (Scope, error) {
	value = strings.TrimSpace(value)
	if !validSubset(value) {
		return Scope{}, fmt.Errorf("%w: %q", ErrInvalidSubset, value)
	}

	return Scope{subset: value}, nil
}

func validSubset(value string) bool {
	if number, ok := strings.CutSuffix(value, "%"); ok {
		percent, err := strconv.ParseFloat(number, 64)
		return err == nil && !math.IsNaN(percent) && percent > 0 && percent <= 100
	}

	partText, totalText, ok := strings.Cut(value, "/")
	if !ok {
		return false
	}

	part, partErr := strconv.ParseUint(partText, 10, 64)
	total, totalErr := strconv.ParseUint(totalText, 10, 64)

	return partErr == nil && totalErr == nil && part > 0 && part <= total
}

func (s Scope) ReadsAllData() bool {
	return s.readData
}

func (s Scope) Subset() string {
	return s.subset
}

func (s Scope) ReadsData() bool {
	return s.readData || s.subset != ""
}

func (s Scope) String() string {
	switch {
	case s.readData:
		return "structure and all data"
	case s.subset != "":
		return "structure and " + s.subset + " of the data"
	default:
		return "structure"
	}
}
