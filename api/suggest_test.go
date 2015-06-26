package api

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestFindRRDFiles(test *testing.T) {
	rrd, ok := NewTestRRD()
	defer rrd.Clean()
	if !ok {
		return
	}
	dir := rrd.Directory

	type Expected []string
	cases := []struct {
		query    string
		expected Expected
	}{
		{"",
			Expected{dir + "server1.net/interface-eth0/if_errors.rrd",
				dir + "server1.net/interface-eth0/if_octets.rrd",
				dir + "server1.net/interface-eth0/if_packets.rrd",
				dir + "server1.net/cpu-0/cpu-system.rrd",
				dir + "server1.net/cpu-1/cpu-system.rrd",
			}},

		{"serv",
			Expected{dir + "server1.net/interface-eth0/if_errors.rrd",
				dir + "server1.net/interface-eth0/if_octets.rrd",
				dir + "server1.net/interface-eth0/if_packets.rrd",
				dir + "server1.net/cpu-0/cpu-system.rrd",
				dir + "server1.net/cpu-1/cpu-system.rrd",
			}},

		{"notexists",
			Expected{}},

		{"server1.net",
			Expected{dir + "server1.net/interface-eth0/if_errors.rrd",
				dir + "server1.net/interface-eth0/if_octets.rrd",
				dir + "server1.net/interface-eth0/if_packets.rrd",
				dir + "server1.net/cpu-0/cpu-system.rrd",
				dir + "server1.net/cpu-1/cpu-system.rrd",
			}},

		{"server1.net/",
			Expected{dir + "server1.net/interface-eth0/if_errors.rrd",
				dir + "server1.net/interface-eth0/if_octets.rrd",
				dir + "server1.net/interface-eth0/if_packets.rrd",
				dir + "server1.net/cpu-0/cpu-system.rrd",
				dir + "server1.net/cpu-1/cpu-system.rrd",
			}},

		{"server1.net/i",
			Expected{dir + "server1.net/interface-eth0/if_errors.rrd",
				dir + "server1.net/interface-eth0/if_octets.rrd",
				dir + "server1.net/interface-eth0/if_packets.rrd",
			}},

		{"server1.net/interface-eth0",
			Expected{dir + "server1.net/interface-eth0/if_errors.rrd",
				dir + "server1.net/interface-eth0/if_octets.rrd",
				dir + "server1.net/interface-eth0/if_packets.rrd",
			}},

		{"server1.net/interface-eth0/",
			Expected{dir + "server1.net/interface-eth0/if_errors.rrd",
				dir + "server1.net/interface-eth0/if_octets.rrd",
				dir + "server1.net/interface-eth0/if_packets.rrd",
			}},

		{"server1.net/interface-eth0/if",
			Expected{dir + "server1.net/interface-eth0/if_errors.rrd",
				dir + "server1.net/interface-eth0/if_octets.rrd",
				dir + "server1.net/interface-eth0/if_packets.rrd",
			}},

		{"server1.net/interface-eth0/if_packets",
			Expected{dir + "server1.net/interface-eth0/if_packets.rrd"}},

		{"server1.net/interface-eth0/if_packets/",
			Expected{dir + "server1.net/interface-eth0/if_packets.rrd"}},
	}

	api := NewAPI(rrd.Directory)
	for _, c := range cases {

		r := api.findRRDFiles(c.query)

		if !comparePaths(r, c.expected) {
			sort.Strings(c.expected)
			sort.Strings(r)
			test.Errorf("Check failed for \"%s\":\nexpected:\n\t%+v\nreal:\n\t%+v",
				c.query,
				strings.Join(c.expected, "\n\t"),
				strings.Join(r, "\n\t"))
		}
	}
}

func TestSuggestMetricsHandler(test *testing.T) {
	cases := []struct {
		Get  string
		Post string
		Want string
	}{
		{
			``,
			``,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/cpu-1/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": ["rx", "tx"]},
			{"metric": "server1.net/interface-eth0/if_octets",  "ds": ["rx", "tx"]},
			{"metric": "server1.net/interface-eth0/if_packets", "ds": ["rx", "tx"]}
		]`,
		},
		// ***********************************
		{
			`""`,
			``,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/cpu-1/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": ["rx", "tx"]},
			{"metric": "server1.net/interface-eth0/if_octets",  "ds": ["rx", "tx"]},
			{"metric": "server1.net/interface-eth0/if_packets", "ds": ["rx", "tx"]}
		]`,
		},
		// ***********************************
		{
			`?query=`,
			`{"query":""}`,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/cpu-1/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": ["rx", "tx"]},
			{"metric": "server1.net/interface-eth0/if_octets",  "ds": ["rx", "tx"]},
			{"metric": "server1.net/interface-eth0/if_packets", "ds": ["rx", "tx"]}
		]`,
		},
		// ***********************************
		{
			`?query=&withds=true`,
			`{"query":"", "withds":true}`,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/cpu-1/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": ["rx", "tx"]},
			{"metric": "server1.net/interface-eth0/if_octets",  "ds": ["rx", "tx"]},
			{"metric": "server1.net/interface-eth0/if_packets", "ds": ["rx", "tx"]}
		]`,
		},
		// ***********************************
		{
			`?query=&withds=false`,
			`{"query":"", "withds":false}`,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": []},
			{"metric": "server1.net/cpu-1/cpu-system", "ds": []},
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": []},
			{"metric": "server1.net/interface-eth0/if_octets",  "ds": []},
			{"metric": "server1.net/interface-eth0/if_packets", "ds": []}
		]`,
		},
		// ***********************************
		{
			`?query=verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/&withds=true`,
			`{"query":"verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/", "withds":true}`,
			`[]`,
		},
		// ***********************************
		{
			`?query=verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string&withds=true`,
			`{"query":"verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string", "withds":true}`,
			`[]`,
		},
		// ***********************************
		{
			`?query=server1.net/interface-eth0/if_errors&withds=true`,
			`{"query":"server1.net/interface-eth0/if_errors", "withds":true}`,
			`[
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": ["rx", "tx"]}
		]`,
		},
		// ***********************************
		{
			`?query=server1.net/interface-eth0/if_errors&withds=false`,
			`{"query":"server1.net/interface-eth0/if_errors", "withds":false}`,
			`[
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": []}
		]`,
		},
		// ***********************************
		{
			`?query=server1.net/interface-eth0/if_errors/&withds=true`,
			`{"query":"server1.net/interface-eth0/if_errors/", "withds":true}`,
			`[
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": ["rx", "tx"]}
		 ]`,
		},
		// ***********************************
		{
			`?query=server1.net/interface-eth0/if_errors:r&withds=true`,
			`{"query":"server1.net/interface-eth0/if_errors:r", "withds":true}`,
			`[
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": ["rx"]}
		]`,
		},
		// ***********************************
		{
			`?query=server1.net/interface-eth0/if_errors:rx&withds=true`,
			`{"query":"server1.net/interface-eth0/if_errors:rx", "withds":true}`,
			`[
			{"metric": "server1.net/interface-eth0/if_errors",  "ds": ["rx"]}
		]`,
		},
		// ***********************************
		{
			`?query=server1.net/interface-eth0/if_errors:rxn&withds=true`,
			`{"query":"server1.net/interface-eth0/if_errors:rxn", "withds":true}`,
			`[]`,
		},
		// ***********************************
		{
			`?query=server1.net/cpu-&withds=true`,
			`{"query":"server1.net/cpu-", "withds":true}`,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]},
			{"metric": "server1.net/cpu-1/cpu-system", "ds": ["value"]}
		]`,
		},
		// ***********************************
		{
			`?query=server1.net/cpu-0&withds=true`,
			`{"query":"server1.net/cpu-0", "withds":true}`,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]}
		]`,
		},
		// ***********************************
		{
			`?query=server1.net/cpu-0/&withds=true`,
			`{"query":"server1.net/cpu-0/", "withds":true}`,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]}
		]`,
		},
		// ***********************************
		{
			`?query=server1.net/cpu-0/cpu-&withds=true`,
			`{"query":"server1.net/cpu-0/cpu-", "withds":true}`,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]}
		]`,
		},
		// ***********************************
		{
			`?query=server1.net/cpu-0/cpu-system&withds=true`,
			`{"query":"server1.net/cpu-0/cpu-system", "withds":true}`,
			`[
			{"metric": "server1.net/cpu-0/cpu-system", "ds": ["value"]}
		]`,
		},
		// ***********************************
	}

	rrd, ok := NewTestRRD()
	defer rrd.Clean()
	if !ok {
		return
	}

	api := NewAPI(rrd.Directory)

	check := func(method string, query, wantJSON, respJSON string) {
		want := SuggestMetricsResponse{}
		resp := SuggestMetricsResponse{}

		if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
			test.Errorf("%v query: '%s'\nIncorrect want '%v':\nError: %v\n", method, query, want, err)
			return
		}

		if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
			test.Errorf("%v query: '%s'\nIncorrect response '%v':\nError: %v\n", method, query, respJSON, err)
			return
		}

		if !reflect.DeepEqual(resp, want) {
			test.Errorf("%v query: '%s'\n\nResult: %v\n\nWant:   %v\n", method, query, resp, want)
		}
	}

	for _, c := range cases {
		j := MakePostRequest(test, api.SuggestMetricsPostHandler, c.Post)
		check("POST", c.Post, c.Want, j)
	}

	for _, c := range cases {
		j := MakeGetRequest(test, api.SuggestMetricsGetHandler, c.Get)
		check("GET", c.Get, c.Want, j)
	}
}
