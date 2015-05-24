package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Time time.Time

func (t Time) Unix() int64 {
	return time.Time(t).Unix()
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var err error
	var n int64
	if json.Unmarshal(data, &n) == nil {
		*t, err = TimeFromString(fmt.Sprintf("%v", n))
		return err
	}

	var s string
	if json.Unmarshal(data, &s) == nil {
		*t, err = TimeFromString(s)
		return err
	}

	return errors.New(fmt.Sprintf("Incorrect time string '%v'", data))
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%v"`, time.Time(*t).Unix())), nil
}

func TimeFromString(str string) (Time, error) {
	str = strings.Trim(str, " \t\n")

	res, err := parseAbsTime(str)
	if err == nil {
		return res, nil
	}

	res = Time(time.Now())

	if d, err := time.ParseDuration(str); err != nil {
		return Time{}, errors.New(fmt.Sprintf("Incorrect time string '%v': %s\n", str, err))
	} else {
		res = Time(time.Time(res).Add(d))
	}

	return res, nil
}

func parseAbsTime(s string) (Time, error) {
	tz, _ := time.LoadLocation("Local")

	// Literals .......................
	if s == "" ||
		strings.ToUpper(s) == "NOW" {
		return Time(time.Now()), nil
	}

	// Unix timestamp .................
	n, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		// Any integers with 13 (or 14) characters will be treated as a millisecond timestamp.
		if n >= 1000000000000 {
			return Time(time.Unix(n/1000, n%1000)), nil
		}

		// Anything 9 characters or less represent seconds.
		if n >= 100000000 {
			return Time(time.Unix(n, 0)), nil
		}
	}

	// Date and Time or only Date .....
	dLayouts := []string{
		"20060102",

		"2006.01.02 15:04:05",
		"2006.01.02 15:04",
		"2006.01.02 15",
		"2006.01.02",

		"2006.01.02-15:04:05",
		"2006.01.02-15:04",
		"2006.01.02-15",

		"2006.1.2 15:4:5",
		"2006.1.2 15:4",
		"2006.1.2 15",
		"2006.1.2",

		"2006.1.2-15:4:5",
		"2006.1.2-15:4",
		"2006.1.2-15",

		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02 15",
		"2006/01/02",

		"2006/01/02-15:04:05",
		"2006/01/02-15:04",
		"2006/01/02-15",

		"2006/1/2 15:4:5",
		"2006/1/2 15:4",
		"2006/1/2 15",
		"2006/1/2",

		"2006/1/2-15:4:5",
		"2006/1/2-15:4",
		"2006/1/2-15",
	}

	for _, l := range dLayouts {
		res, err := time.ParseInLocation(l, s, tz)
		if err == nil {
			return Time(res), nil
		}
	}

	// Only Time ......................
	tLayouts := []string{"15:04:05", "15:4:5",
		"15:04", "15:4",
		"15",
	}

	for _, l := range tLayouts {
		res, err := time.ParseInLocation(l, s, tz)
		if err == nil {
			now := time.Now()
			return Time(time.Time(res).AddDate(now.Year(), int(now.Month()-1), now.Day()-1)), nil
		}
	}

	return Time{}, errors.New(fmt.Sprintf("Incorrect date string '%v'", s))
}

type Duration time.Duration

func DurationFromString(s string) (Duration, error) {
	if s == "" {
		return Duration(time.Second), nil
	}

	n, err := strconv.ParseInt(s, 10, 0)
	if err == nil {
		return Duration(n * int64(time.Second)), nil
	}

	res, err := time.ParseDuration(s)
	return Duration(res), err
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var n int
	if json.Unmarshal(data, &n) == nil {
		*d = Duration(n / 1000000000)
		return nil
	}

	var s string
	if json.Unmarshal(data, &s) == nil {
		v, err := time.ParseDuration(s)
		*d = Duration(v)
		return err
	}

	return errors.New(fmt.Sprintf("Invalid duration string '%v'", data))
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`%v`, time.Duration(*d).Seconds())), nil
}
