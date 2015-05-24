package api

import (
	"fmt"
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

		{"server1.net/interface-eth0/if_packets/rx",
			Expected{dir + "server1.net/interface-eth0/if_packets.rrd"}},
	}

	api := NewAPI(rrd.Directory)
	for _, c := range cases {

		r := api.findRRDFiles(c.query)

		if !comparePaths(r, c.expected) {
			sort.Strings(c.expected)
			sort.Strings(r)
			test.Error(fmt.Sprintf("Check failed for \"%s\":\nexpected:\n\t%+v\nreal:\n\t%+v",
				c.query,
				strings.Join(c.expected, "\n\t"),
				strings.Join(r, "\n\t")))
		}
	}
}

func TestFindMetricsRecursive(test *testing.T) {
	rrd, ok := NewTestRRD()
	defer rrd.Clean()
	if !ok {
		return
	}

	type Expected []string
	cases := []struct {
		query    string
		expected Expected
	}{
		{"",
			Expected{"server1.net/interface-eth0/if_errors/rx",
				"server1.net/interface-eth0/if_errors/tx",
				"server1.net/interface-eth0/if_octets/rx",
				"server1.net/interface-eth0/if_octets/tx",
				"server1.net/interface-eth0/if_packets/rx",
				"server1.net/interface-eth0/if_packets/tx",
				"server1.net/cpu-0/cpu-system",
				"server1.net/cpu-1/cpu-system",
			}},

		{"verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/",
			Expected{}},

		{"verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string",
			Expected{}},

		{"server1.net/interface-eth0/if_errors",
			Expected{"server1.net/interface-eth0/if_errors/rx",
				"server1.net/interface-eth0/if_errors/tx",
			}},

		{"server1.net/interface-eth0/if_errors/",
			Expected{"server1.net/interface-eth0/if_errors/rx",
				"server1.net/interface-eth0/if_errors/tx",
			}},

		{"server1.net/interface-eth0/if_errors/r",
			Expected{"server1.net/interface-eth0/if_errors/rx"}},

		{"server1.net/interface-eth0/if_errors/rx",
			Expected{"server1.net/interface-eth0/if_errors/rx"}},

		{"server1.net/interface-eth0/if_errors/rxn",
			Expected{}},

		{"server1.net/cpu-",
			Expected{"server1.net/cpu-0/cpu-system",
				"server1.net/cpu-1/cpu-system"}},

		{"server1.net/cpu-0",
			Expected{"server1.net/cpu-0/cpu-system"}},

		{"server1.net/cpu-0/",
			Expected{"server1.net/cpu-0/cpu-system"}},

		{"server1.net/cpu-0/cpu-",
			Expected{"server1.net/cpu-0/cpu-system"}},

		{"server1.net/cpu-0/cpu-system",
			Expected{"server1.net/cpu-0/cpu-system"}},
	}

	api := NewAPI(rrd.Directory)
	for _, c := range cases {

		r, err := api.findMetricsRecursive(c.query)
		if err != nil {
			test.Error(fmt.Sprintf("Check failed for \"%s\":%v", err))
			continue
		}

		if !comparePaths(r, c.expected) {
			sort.Strings(c.expected)
			sort.Strings(r)
			test.Error(fmt.Sprintf("Check failed for recursive \"%s\":\nexpected:\n\t%+v\nreal:\n\t%+v\n",
				c.query,
				strings.Join(c.expected, "\n\t"),
				strings.Join(r, "\n\t")))
		}
	}
}

func TestFindMetricsNonRecursive(test *testing.T) {
	rrd, ok := NewTestRRD()
	defer rrd.Clean()
	if !ok {
		return
	}

	type Expected []string
	cases := []struct {
		query    string
		expected Expected
	}{
		{"",
			Expected{"server1.net"}},

		{"verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/",
			Expected{}},

		{"verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string/verry/long/string",
			Expected{}},

		{"s",
			Expected{"server1.net"}},

		{"server1.net/",
			Expected{"interface-eth0",
				"cpu-0",
				"cpu-1",
			}},

		{"server1.net/interface-eth0/",
			Expected{"if_errors",
				"if_octets",
				"if_packets",
			}},

		{"server1.net/interface-eth0/if_error",
			Expected{"if_errors"}},

		{"server1.net/interface-eth0/if_errors",
			Expected{"if_errors"}},

		{"server1.net/interface-eth0/if_errors/",
			Expected{"rx",
				"tx",
			}},

		{"server1.net/interface-eth0/if_errors/r",
			Expected{"rx"}},

		{"server1.net/interface-eth0/if_errors/rx",
			Expected{"rx"}},

		{"server1.net/interface-eth0/if_errors/rxn",
			Expected{}},

		{"server1.net/cpu-",
			Expected{
				"cpu-0",
				"cpu-1",
			}},

		{"server1.net/cpu-1",
			Expected{
				"cpu-1",
			}},

		{"server1.net/cpu-0/",
			Expected{"cpu-system"}},

		{"server1.net/cpu-0/cpu-",
			Expected{"cpu-system"}},

		{"server1.net/cpu-0/cpu-system",
			Expected{"cpu-system"}},

		{"server1.net/cpu-0/cpu-systemZZZ",
			Expected{""}},

		{"server1.net/cpu-0/cpu-system/",
			Expected{""}},
	}

	api := NewAPI(rrd.Directory)
	for _, c := range cases {

		r, err := api.findMetricsNonRecursive(c.query)
		if err != nil {
			test.Error(fmt.Sprintf("Check failed for \"%s\":%v", err))
			continue
		}

		if !comparePaths(r, c.expected) {
			sort.Strings(c.expected)
			sort.Strings(r)
			test.Error(fmt.Sprintf("Check failed for non recursive \"%s\":\nexpected:\n\t%+v\nreal:\n\t%+v\n",
				c.query,
				strings.Join(c.expected, "\n\t"),
				strings.Join(r, "\n\t")))
		}
	}
}
