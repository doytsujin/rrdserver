package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Consolidation uint8

const (
	CFAVERAGE Consolidation = iota
	CFMIN     Consolidation = iota
	CFMAX     Consolidation = iota
	CFLAST    Consolidation = iota
)

func (c *Consolidation) String() string {
	switch *c {
	case CFAVERAGE:
		return "AVERAGE"
	case CFMAX:
		return "MAX"
	case CFMIN:
		return "MIN"
	case CFLAST:
		return "LASTMAX"
	}
	return "AVERAGE"
}

func ConsolidationFromString(s string) (Consolidation, error) {
	switch strings.ToUpper(s) {
	case "", "AVERAGE":
		return Consolidation(CFAVERAGE), nil

	case "MIN":
		return Consolidation(CFMIN), nil

	case "MAX":
		return Consolidation(CFMAX), nil

	case "LAST":
		return Consolidation(CFLAST), nil
	}

	return Consolidation(CFAVERAGE), fmt.Errorf("Invalid consolidation '%v'", s)
}

func (c *Consolidation) UnmarshalJSON(data []byte) error {
	var s string
	if json.Unmarshal(data, &s) == nil {
		r, err := ConsolidationFromString(s)
		*c = r
		return err
	}

	return fmt.Errorf("Invalid consolidation '%v'", data)
}

func (c *Consolidation) MarshalJSON() ([]byte, error) {
	return []byte(`"` + c.String() + `"`), nil
}
