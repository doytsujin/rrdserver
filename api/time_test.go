package api

import (
	"fmt"
	"testing"
	"time"
)

func relTime(t time.Time, offset string) time.Time {
	d, _ := time.ParseDuration(offset)
	return t.Add(d)
}

func TestAPIStrToDateTime(test *testing.T) {
	now := time.Now()
	y := now.Year()
	m := now.Month()
	d := now.Day()

	tzLocal, _ := time.LoadLocation("Local")
	tzUTC, _ := time.LoadLocation("UTC")

	cases := []struct {
		query    string
		expected time.Time
		delta    string
	}{
		{"", now, "1s"},
		{"now", now, "1s"},

		{"1000000000", time.Date(2001, 9, 9, 1, 46, 40, 0, tzUTC), ""},
		{"1430031094", time.Date(2015, 4, 26, 6, 51, 34, 0, tzUTC), ""},
		{"9999999999", time.Date(2286, 11, 20, 17, 46, 39, 0, tzUTC), ""},
		{"1000000000000", time.Date(2001, 9, 9, 1, 46, 40, 0, tzUTC), ""},
		{"1000000000125", time.Date(2001, 9, 9, 1, 46, 40, 125, tzUTC), ""},
		{"1430031094317", time.Date(2015, 4, 26, 6, 51, 34, 317, tzUTC), ""},
		{"9999999999000", time.Date(2286, 11, 20, 17, 46, 39, 0, tzUTC), ""},
		{"9999999999999", time.Date(2286, 11, 20, 17, 46, 39, 999, tzUTC), ""},

		{"2006.05.04 03:02:01", time.Date(2006, 5, 4, 3, 2, 1, 0, tzLocal), ""},
		{"2006.5.4 3:2:1", time.Date(2006, 5, 4, 3, 2, 1, 0, tzLocal), ""},
		{"2006.5.4 23:59:59", time.Date(2006, 5, 4, 23, 59, 59, 0, tzLocal), ""},
		{"2015.04.26 10:41:15", time.Date(2015, 4, 26, 10, 41, 15, 0, tzLocal), ""},
		{"2015.04.26 10:41", time.Date(2015, 4, 26, 10, 41, 0, 0, tzLocal), ""},
		{"2015.04.26 10", time.Date(2015, 4, 26, 10, 0, 0, 0, tzLocal), ""},
		{"2015.04.26", time.Date(2015, 4, 26, 0, 0, 0, 0, tzLocal), ""},

		{"2006/05/04 03:02:01", time.Date(2006, 5, 4, 3, 2, 1, 0, tzLocal), ""},
		{"2006/5/4 3:2:1", time.Date(2006, 5, 4, 3, 2, 1, 0, tzLocal), ""},
		{"2006/5/4 23:59:59", time.Date(2006, 5, 4, 23, 59, 59, 0, tzLocal), ""},
		{"2015/04/26 10:41:15", time.Date(2015, 4, 26, 10, 41, 15, 0, tzLocal), ""},
		{"2015/04/26 10:41", time.Date(2015, 4, 26, 10, 41, 0, 0, tzLocal), ""},
		{"2015/04/26 10", time.Date(2015, 4, 26, 10, 0, 0, 0, tzLocal), ""},
		{"2015/04/26", time.Date(2015, 4, 26, 0, 0, 0, 0, tzLocal), ""},

		{"01:02:03", time.Date(y, m, d, 1, 2, 3, 0, tzLocal), ""},
		{"1:2:3", time.Date(y, m, d, 1, 2, 3, 0, tzLocal), ""},
		{"01:02", time.Date(y, m, d, 1, 2, 0, 0, tzLocal), ""},
		{"1:2", time.Date(y, m, d, 1, 2, 0, 0, tzLocal), ""},
		{"01", time.Date(y, m, d, 1, 0, 0, 0, tzLocal), ""},
		{"1", time.Date(y, m, d, 1, 0, 0, 0, tzLocal), ""},

		{"-1s", relTime(now, "-1s"), "1s"},
		{"-1m", relTime(now, "-1m"), "1s"},
		{"-1h", relTime(now, "-1h"), "1s"},
		{"-65s", relTime(now, "-1m5s"), "1s"},
		{"-8888s", relTime(now, "-2h28m8s"), "1s"},
		{"-1h2m", relTime(now, "-1h2m"), "1s"},
		{"-1h2m3s", relTime(now, "-1h2m3s"), "1s"},
		{" -1s", relTime(now, "-1s"), "1s"},
		{"  -1m", relTime(now, "-1m"), "1s"},
		{" -1h ", relTime(now, "-1h"), "1s"},

		{"1s", relTime(now, "1s"), "1s"},
		{"+1s", relTime(now, "1s"), "1s"},
		{"+1m", relTime(now, "+1m"), "1s"},
		{"+1h", relTime(now, "+1h"), "1s"},
		{"+65s", relTime(now, "+1m5s"), "1s"},
		{"+8888s", relTime(now, "+2h28m8s"), "1s"},
		{"+1h2m", relTime(now, "+1h2m"), "1s"},
		{"+1h2m3s", relTime(now, "+1h2m3s"), "1s"},
		{" +1s", relTime(now, "1s"), "1s"},
		{"  +1m", relTime(now, "1m"), "1s"},
		{" +1h ", relTime(now, "1h"), "1s"},
		{" 1s", relTime(now, "1s"), "1s"},
		{"  1m", relTime(now, "1m"), "1s"},
		{" 1h ", relTime(now, "1h"), "1s"},
	}

	for _, c := range cases {
		t, err := TimeFromString(c.query)

		if err != nil {
			test.Error(fmt.Sprintf("Check failed for \"%s\",\n%v", c.query, err))
		} else {
			if c.delta != "" {
				dur, _ := time.ParseDuration(c.delta)
				t = t.Round(dur)
				c.expected = c.expected.Round(dur)
			}

			if t.UnixNano() != c.expected.UnixNano() {
				test.Error(fmt.Sprintf("Check failed for \"%s\":\n\t expected: %s   (%s),\n\t real:     %s   (%s)",
					c.query,
					c.expected.Local(), c.expected.UTC(),
					t.Local(), t.UTC()))
			}
		}
	}
}

func TestStringToDuration(test *testing.T) {
	cases := []struct {
		query    string
		expected string
	}{
		{"", "1s"},
		{"1", "1s"},
		{"100", "1m40s"},
		{"1s", "1s"},
		{"2m", "2m"},
		{"3h", "3h"},
		{"125s", "2m5s"},
	}

	for _, c := range cases {
		expected, _ := time.ParseDuration(c.expected)
		v, err := DurationFromString(c.query)
		if err != nil {
			test.Error(fmt.Sprintf("Check failed for \"%s\",\n%v", c.query, err))
		} else {
			if v != expected {
				test.Error(fmt.Sprintf("Check failed for \"%s\":\n\t expected: %s,\n\t real:     %s",
					c.query,
					expected,
					v))
			}
		}
	}
}
