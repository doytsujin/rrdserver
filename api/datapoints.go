package api

import (
	"encoding/json"
	"fmt"
	"math"
)

type DataPoints map[Time]float64

func (dps DataPoints) MarshalJSON() ([]byte, error) {
	res := "{"
	n := 0
	for t, v := range dps {

		if n > 0 {
			res += ",\n"
		}
		n++

		if math.IsNaN(v) {
			res += fmt.Sprintf("\"%v\":null", t.Unix())
		} else {
			res += fmt.Sprintf("\"%v\":%f", t.Unix(), v)
		}
	}
	res += "}"
	return []byte(res), nil
}

func (dps *DataPoints) UnmarshalJSON(data []byte) error {
	var tmp map[string]float64

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	vls := make(DataPoints, len(tmp))
	for k, v := range tmp {
		t, err := TimeFromString(k)
		if err != nil {
			return err
		}
		vls[t] = v
	}

	*dps = vls
	return nil
}
