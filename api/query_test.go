package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestQueryDefFromString(test *testing.T) {
	AVERAGE, _ := ConsolidationFromString("AVERAGE")
	//MIN, _ := ConsolidationFromString("MIN")
	MAX, _ := ConsolidationFromString("MAX")
	var cases = []struct {
		str  string
		want QueryDef
	}{
		{
			`DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE`,
			QueryDef{
				Type:    "DEF",
				Name:    "A",
				Metric:  "server1.net/interface-eth0/if_packets",
				DS:      "rx",
				CF:      AVERAGE,
				Options: "",
			},
		},
		// *******************************
		{
			`DEF:A=server1.net/interface-eth0/if_packets:rx:MAX:start=e-3600:end=n`,
			QueryDef{
				Type:    "DEF",
				Name:    "A",
				Metric:  "server1.net/interface-eth0/if_packets",
				DS:      "rx",
				CF:      MAX,
				Options: "start=e-3600:end=n",
			},
		},
		// *******************************
		{
			`CDEF:SUM=RX,TX,+`,
			QueryDef{
				Type:    "CDEF",
				Name:    "SUM",
				Metric:  "RX,TX,+",
				DS:      "",
				CF:      0,
				Options: "",
			},
		},
		// *******************************
	}

	for _, c := range cases {
		res, err := QueryDefFromString(c.str)
		if err != nil {
			test.Errorf("String: %s\nError parse: %v\n", c.str, err)
			continue
		}

		if !reflect.DeepEqual(res, c.want) {
			test.Errorf("String: %s\nResult: %+v\nWant:   %+v\n", c.str, res, c.want)
		}
	}

}

func TestQueryHandlers(test *testing.T) {
	cases := []struct {
		Get  string
		Post string
		Want string
	}{
		{
			`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&step=1s&query=DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE`,
			`{
		"start":  	"2000.01.02 00:59:59",
		"end":    	"2000.01.02 01:05:00",
		"step":   	"1s",
		"queries": [{
			"query": "DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE"
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
			`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&step=2m&query=DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE`,
			`{
		"start":  	"2000.01.02 00:59:59",
		"end":    	"2000.01.02 01:05:00",
		"step":   	"2m",
		"queries": [{
			"query": "DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE"
		}]
	}`,
			`
	2000.01.02-01:00:00  0
	2000.01.02-01:02:00  115
	2000.01.02-01:04:00  135
	`},
		// ***********************************
		{
			`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&step=2m&query="DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE"`,
			`{
		"start":  	"2000.01.02 00:59:59",
		"end":    	"2000.01.02 01:05:00",
		"step":   	"2m",
		"queries": [{
			"query": "DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE"
		}]
	}`,
			`
	2000.01.02-01:00:00  0
	2000.01.02-01:02:00  115
	2000.01.02-01:04:00  135
	`},
		// ***********************************
		{
			"start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&step=2m&query=`DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE`",
			`{
		"start":  	"2000.01.02 00:59:59",
		"end":    	"2000.01.02 01:05:00",
		"step":   	"2m",
		"queries": [{
			"query": "DEF:A=server1.net/interface-eth0/if_packets:rx:AVERAGE"
		}]
	}`,
			`
	2000.01.02-01:00:00  0
	2000.01.02-01:02:00  115
	2000.01.02-01:04:00  135
	`},
		// ***********************************

		{
			`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&step=1s&query=DEF:RX=server1.net/interface-eth0/if_packets:rx:AVERAGE&query=DEF:TX=server1.net/interface-eth0/if_packets:tx:AVERAGE`,
			`{
		"start":  	"2000.01.02 00:59:59",
		"end":    	"2000.01.02 01:05:00",
		"step":   	"1s",
		"queries": [{
			"query": "DEF:RX=server1.net/interface-eth0/if_packets:rx:AVERAGE"
		},{
			"query": "DEF:TX=server1.net/interface-eth0/if_packets:tx:AVERAGE"
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
			`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&step=1s&query=DEF:RX=server1.net/interface-eth0/if_packets:rx:AVERAGE&query=DEF:TX=server1.net/interface-eth0/if_packets:tx:AVERAGE&hidden=true&hidden=false`,
			`{
		"start":  	"2000.01.02 00:59:59",
		"end":    	"2000.01.02 01:05:00",
		"step":   	"1s",
		"queries": [{
			"query": "DEF:RX=server1.net/interface-eth0/if_packets:rx:AVERAGE",
			"hidden": true
		},{
			"query": "DEF:TX=server1.net/interface-eth0/if_packets:tx:AVERAGE"
		}]
	}`,
			`
	2000.01.02-01:00:00  320
	2000.01.02-01:01:00  320
	2000.01.02-01:02:00  340
	2000.01.02-01:03:00  360
	2000.01.02-01:04:00  380
	2000.01.02-01:05:00  400
	`},
		// ***********************************
		{
			`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&step=1s&query=DEF:RX=server1.net/interface-eth0/if_packets:rx:AVERAGE&query=DEF:TX=server1.net/interface-eth0/if_packets:tx:AVERAGE&hidden=false&hidden=true`,
			`{
		"start":  	"2000.01.02 00:59:59",
		"end":    	"2000.01.02 01:05:00",
		"step":   	"1s",
		"queries": [{
			"query": "DEF:RX=server1.net/interface-eth0/if_packets:rx:AVERAGE"
		},{
			"query": "DEF:TX=server1.net/interface-eth0/if_packets:tx:AVERAGE",
			"hidden": true			
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
			`start=2000.01.02-00:59:59&end=2000.01.02-01:05:00&step=1s&` +
				`query=DEF:RX=server1.net/interface-eth0/if_packets:rx:AVERAGE&` +
				`query=DEF:TX=server1.net/interface-eth0/if_packets:tx:AVERAGE&` +
				`query=CDEF:RXBit=RX,8,*&` +
				`query=CDEF:TXBit=TX,8,*&` +
				`query=CDEF:RTX=RX,TX,%2B&` +
				`query=CDEF:RTXBit=RXBit,TXBit,%2B`,
			`{
		"start":  	"2000.01.02 00:59:59",
		"end":    	"2000.01.02 01:05:00",
		"step":   	"1s",
		"queries": [
			{ "query": "DEF:RX=server1.net/interface-eth0/if_packets:rx:AVERAGE" },
			{ "query": "DEF:TX=server1.net/interface-eth0/if_packets:tx:AVERAGE" },
			{ "query": "CDEF:RXBit=RX,8,*" },
			{ "query": "CDEF:TXBit=TX,8,*" },
			{ "query": "CDEF:RTX=RX,TX,+" },
			{ "query": "CDEF:RTXBit=RXBit,TXBit,+" }
		]
	}`,
			`
	2000.01.02-01:00:00  100 320  800 2560 420 3360
	2000.01.02-01:01:00  110 320  880 2560 430 3440
	2000.01.02-01:02:00  120 340  960 2720 460 3680
	2000.01.02-01:03:00  130 360 1040 2880 490 3920
	2000.01.02-01:04:00  140 380 1120 3040 520 4160
	2000.01.02-01:05:00  150 400 1200 3200 550 4400
	`},
		// ***********************************
	}

	// ::::::::::::::::::::::::::::::::::::::::::
	rrd, ok := NewTestRRD()
	defer rrd.Clean()
	if !ok {
		return
	}
	createRRDFiles(rrd)
	api := NewAPI(rrd.Directory)

	check := func(method string, query, wantStr, respJSON string) {
		resp := QueryResponse{}

		if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
			test.Errorf("%v query: '%s'\nIncorrect response '%v':\nError: %v\n", method, respJSON, query, err)
			return
		}

		want := ParseWantTable(wantStr)

		if len(resp.Result) != len(want) {
			test.Errorf("%v query: %s\nDiffrent count of results:\nResult - %v, want - %v\n", method, query, len(resp.Result), len(want))
			return
		}

		for i, r := range resp.Result {
			if !reflect.DeepEqual(r.Values, want[i]) {
				test.Errorf("%v query: %s\n\nMetric: %v\n\n%v\n", method, query, r.Name, PrintDataPoints(r.Values, want[i]))
			}
		}
	}

	for _, c := range cases {
		j := MakePostRequest(test, api.QueryPostHandler, c.Post)
		check("POST", c.Post, c.Want, j)
	}

	for _, c := range cases {
		j := MakeGetRequest(test, api.QueryGetHandler, c.Get)
		check("GET", c.Get, c.Want, j)
	}

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

func PrintDataPoints(res, want DataPoints) string {
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
