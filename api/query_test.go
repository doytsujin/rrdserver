package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

var cases = []struct {
	get  string
	post string
	want string
}{
	{
		`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&resolution=1s&metric=server1.net/interface-eth0/if_packets/rx&consolidation=AVERAGE`,
		`{
		"start":  		"2000.01.02 00:59:59",
		"end":    		"2000.01.02 01:05:00",
		"resolution":   "1s",
		"queries": [{
			"metric": "server1.net/interface-eth0/if_packets/rx",
			"consolidation":"AVERAGE"
		}]
	}`,
		`
	2000.01.02-01:00:00  100
	2000.01.02-01:01:00  110
	2000.01.02-01:02:00  120
	2000.01.02-01:03:00  130
	2000.01.02-01:04:00  140
	2000.01.02-01:05:00  150
	`},
	// ***********************************
	{
		`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&resolution=1s&metric=server1.net/interface-eth0/if_packets/rx&consolidation=AVERAGE&metric=server1.net/interface-eth0/if_packets/tx`,
		`{
		"start":  		"2000.01.02 00:59:59",
		"end":    		"2000.01.02 01:05:00",
		"resolution":   "1s",
		"queries": [{
			"metric": "server1.net/interface-eth0/if_packets/rx",
			"consolidation":"AVERAGE"
		},{
			"metric": "server1.net/interface-eth0/if_packets/tx",
			"consolidation":"AVERAGE"
		}]
	}`,
		`
	2000.01.02-01:00:00  100 320
	2000.01.02-01:01:00  110 320
	2000.01.02-01:02:00  120 340
	2000.01.02-01:03:00  130 360
	2000.01.02-01:04:00  140 380
	2000.01.02-01:05:00  150 400
	`},
	// ***********************************
	{
		`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&resolution=1s&metric=server1.net/cpu-0/cpu-system/value&consolidation=AVERAGE`,
		`{
		"start":  		"2000.01.02 00:59:59",
		"end":    		"2000.01.02 01:05:00",
		"resolution":   "1s",
		"queries": [{
			"metric": "server1.net/cpu-0/cpu-system/value",
			"consolidation":"AVERAGE"
		}]
	}`,
		`
	2000.01.02-01:00:00  0
	2000.01.02-01:01:00  1
	2000.01.02-01:02:00  2
	2000.01.02-01:03:00  3
	2000.01.02-01:04:00  4
	2000.01.02-01:05:00  5
	`},
	// ***********************************
	{
		`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&resolution=1s&metric=server1.net/cpu-0/cpu-system&consolidation=AVERAGE`,
		`{
		"start":  		"2000.01.02 00:59:59",
		"end":    		"2000.01.02 01:05:00",
		"resolution":   "1s",
		"queries": [{
			"metric": "server1.net/cpu-0/cpu-system",
			"consolidation":"AVERAGE"
		}]
	}`,
		`
	2000.01.02-01:00:00  0
	2000.01.02-01:01:00  1
	2000.01.02-01:02:00  2
	2000.01.02-01:03:00  3
	2000.01.02-01:04:00  4
	2000.01.02-01:05:00  5
	`},
	// ***********************************
}

func createRRDFiles(rrd *TestRRD) {
	//                                                                                   rx   tx
	rrd.InsertValues("server1.net/interface-eth0/if_packets.rrd", "2000.01.02 00:59:00", 100, 300)
	rrd.InsertValues("server1.net/interface-eth0/if_packets.rrd", "2000.01.02 01:00:00", 100, 320)
	rrd.InsertValues("server1.net/interface-eth0/if_packets.rrd", "2000.01.02 01:01:00", 110, 320)
	rrd.InsertValues("server1.net/interface-eth0/if_packets.rrd", "2000.01.02 01:02:00", 120, 340)
	rrd.InsertValues("server1.net/interface-eth0/if_packets.rrd", "2000.01.02 01:03:00", 130, 360)
	rrd.InsertValues("server1.net/interface-eth0/if_packets.rrd", "2000.01.02 01:04:00", 140, 380)
	rrd.InsertValues("server1.net/interface-eth0/if_packets.rrd", "2000.01.02 01:05:00", 150, 400)

	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 00:59:00", 0)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:00:00", 0)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:01:00", 1)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:02:00", 2)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:03:00", 3)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:04:00", 4)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:05:00", 5)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:06:00", 6)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:07:00", 7)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:08:00", 8)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:09:00", 9)
	rrd.InsertValues("server1.net/cpu-0/cpu-system.rrd", "2000.01.02 01:10:00", 10)
}

func Want(args ...interface{}) (res QueryResponseValues) {
	res = make(QueryResponseValues, len(args)/2)
	tz, _ := time.LoadLocation("Local")

	for i := 0; i < len(args); i += 2 {
		t, err := time.ParseInLocation("2006.01.02 15:04:05", args[i].(string), tz)
		if err != nil {
			log.Fatal(fmt.Sprintf("Error: incorrect time %v", args[i]))
		}

		switch n := args[i+1].(type) {
		case float64:
			res[Time(t)] = n

		case float32:
			res[Time(t)] = float64(n)

		case int:
			res[Time(t)] = float64(n)

		case int32:
			res[Time(t)] = float64(n)

		case int64:
			res[Time(t)] = float64(n)

		default:
			log.Fatal(fmt.Sprintf("can't convert '%v' (%T) to flaot64", args[i+1], args[i+1]))
		}
	}
	return
}

func TestQuery(test *testing.T) {
	rrd, ok := NewTestRRD()
	defer rrd.Clean()
	if !ok {
		return
	}
	createRRDFiles(rrd)

	// ::::::::::::::::::::::::::::::::::::::::::
	api := NewAPI(rrd.Directory)
	for _, c := range cases {
		var err error

		req := QueryRequest{}
		if err = json.Unmarshal([]byte(c.post), &req); err != nil {
			log.Fatal(fmt.Sprintf("Incorrect query '%v':\nError: %v", c.post, err))
		}

		want := ParseWantTable(c.want)

		resp, err := api.query(req)
		if err != nil {
			test.Error(fmt.Sprintf("Query: %s\n Error: %v", c.post, err))
			continue
		}

		if len(resp) < 1 {
			test.Error(fmt.Sprintf("Query: %s\n Empty result.", c.post))
			continue
		}

		for i, r := range resp {
			if !reflect.DeepEqual(r.Values, want[i]) {
				test.Error(fmt.Sprintf("Query: %s\n\nMetric: %v\n\n%v", c.post, req.Queries[i].Metric, PrintQueryResponseValues(r.Values, want[i])))
			}
		}
	}
}

func TestQueryPostHandler(test *testing.T) {
	rrd, ok := NewTestRRD()
	defer rrd.Clean()
	if !ok {
		return
	}
	createRRDFiles(rrd)

	// ::::::::::::::::::::::::::::::::::::::::::
	api := NewAPI(rrd.Directory)
	for _, c := range cases {
		var err error

		want := ParseWantTable(c.want)

		req, err := http.NewRequest("POST", "http://127.0.0.1", strings.NewReader(c.post))
		if err != nil {
			log.Fatal(fmt.Sprintf("Incorrect query '%v':\nError: %v", c.post, err))
		}

		w := httptest.NewRecorder()
		api.QueryPostHandler(w, req)
		body := w.Body.String()

		if w.Code != 200 {
			test.Error(fmt.Sprintf("Query: %s\n HTTP Error: %v", c.post, body))
			continue
		}

		resp := []QueryResponse{}
		if err = json.Unmarshal([]byte(body), &resp); err != nil {
			test.Error(fmt.Sprintf("Query: %s\nIncorrect response '%v':\nError: %v", c.post, body, err))
			continue
		}

		if len(resp) < 1 {
			test.Error(fmt.Sprintf("Query: %s\n Empty result.", c.post))
			continue
		}

		for i, r := range resp {
			if !reflect.DeepEqual(r.Values, want[i]) {
				test.Error(fmt.Sprintf("Query: %s\n\nMetric: %v:%s\n\n%v", c.post, r.Metric, r.Consolidation.String(), PrintQueryResponseValues(r.Values, want[i])))
			}
		}
	}
}

func TestQueryGetHandler(test *testing.T) {
	rrd, ok := NewTestRRD()
	defer rrd.Clean()
	if !ok {
		return
	}
	createRRDFiles(rrd)

	// ::::::::::::::::::::::::::::::::::::::::::
	api := NewAPI(rrd.Directory)
	for _, c := range cases {
		var err error

		want := ParseWantTable(c.want)

		req, err := http.NewRequest("GET", "http://127.0.0.1/api/v1/query?"+c.get, nil)
		if err != nil {
			log.Fatal(fmt.Sprintf("Incorrect query '%v':\nError: %v", c.get, err))
		}

		w := httptest.NewRecorder()
		api.QueryGetHandler(w, req)
		body := w.Body.String()

		if w.Code != 200 {
			test.Error(fmt.Sprintf("Query: %s\n HTTP Error: %v", c.get, body))
			continue
		}

		resp := []QueryResponse{}
		if err = json.Unmarshal([]byte(body), &resp); err != nil {
			test.Error(fmt.Sprintf("Query: %s\nIncorrect response '%v':\nError: %v", c.get, body, err))
			continue
		}

		if len(resp) < 1 {
			test.Error(fmt.Sprintf("Query: %s\n Empty result.", c.get))
			continue
		}

		for i, r := range resp {
			if !reflect.DeepEqual(r.Values, want[i]) {
				test.Error(fmt.Sprintf("Query: %s\n\nMetric: %v:%s\n\n%v", c.get, r.Metric, r.Consolidation.String(), PrintQueryResponseValues(r.Values, want[i])))
			}
		}
	}
}

func ParseWantTable(table string) []QueryResponseValues {
	res := []QueryResponseValues{}

	for _, line := range strings.Split(table, "\n") {
		line = strings.Trim(line, " \t")
		if line == "" {
			continue
		}

		items := regexp.MustCompile("[ \t]+").Split(line, -1)
		if len(items) < 0 {
			log.Fatal(fmt.Sprintf("Incorrect table row '%v'", line))
		}

		t, err := TimeFromString(items[0])
		if err != nil {
			log.Fatal(fmt.Sprintf("Incorrect start time '%v' in line '%v'", items[0], line))
		}

		for i := len(res); i < len(items); i++ {
			res = append(res, QueryResponseValues{})
		}
		n := 0
		for _, s := range items[1:] {
			s = strings.Trim(s, " \t")
			if s == "" {
				continue
			}

			res[n][t], err = strconv.ParseFloat(s, 64)
			if err != nil {
				log.Fatal(fmt.Sprintf("Incorrect value '%v' in line '%v'", items[0], line))
			}
			n++
		}
	}

	return res
}

type Keys []Time

func (keys Keys) Len() int {
	return len(keys)
}

func (keys Keys) Swap(i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
}

func (keys Keys) Less(i, j int) bool {
	return time.Time(keys[i]).Unix() < time.Time(keys[j]).Unix()
}

func (keys Keys) indexOf(t Time) int {
	for n, k := range keys {
		if k == t {
			return n
		}
	}

	return -1
}

func PrintQueryResponseValues(res, want QueryResponseValues) string {
	keys := make(Keys, len(res))
	i := 0
	for k := range res {
		keys[i] = k
		i++
	}

	for k := range want {
		if _, ok := res[k]; !ok {
			keys = append(keys, k)
		}
	}

	sort.Sort(keys)
	const format = "%30v | %10s | %10s |%s\n"
	out := fmt.Sprintf(format, "Time", "Result", "Want", "")
	out += strings.Repeat("-", len(out)) + "\n"

	for _, t := range keys {
		s1, s2, s3 := "", "", ""
		if v, ok := res[t]; ok {
			s1 = fmt.Sprintf("%v", v)
		}

		if v, ok := want[t]; ok {
			s2 = fmt.Sprintf("%v", v)
		}

		if s1 != s2 {
			s3 = " <==="
		}
		out += fmt.Sprintf(format, time.Time(t), s1, s2, s3)
	}

	return out
}
