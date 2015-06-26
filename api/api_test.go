package api

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type TestRRD struct {
	Directory string
}

func NewTestRRD() (res *TestRRD, ok bool) {
	res = &TestRRD{}

	if dir, err := ioutil.TempDir("", "rrdserver"); err == nil {
		res.Directory = dir + "/"
	} else {
		log.Fatal(fmt.Sprintf("Can't create temporary directory: %v\n", err))
		return res, false
	}

	files := []struct {
		file string
		args []string
	}{
		{"server1.net/interface-eth0/if_errors.rrd",
			[]string{"DS:rx:GAUGE:600:U:U",
				"DS:tx:GAUGE:600:U:U",
				"RRA:AVERAGE:0.5:1:100"}},

		{"server1.net/interface-eth0/if_octets.rrd",
			[]string{"DS:rx:GAUGE:600:U:U",
				"DS:tx:GAUGE:600:U:U",
				"RRA:AVERAGE:0.5:1:100"}},

		{"server1.net/interface-eth0/if_packets.rrd",
			[]string{"DS:rx:GAUGE:600:U:U",
				"DS:tx:GAUGE:600:U:U",
				"RRA:AVERAGE:0.5:1:100"}},

		{"server1.net/cpu-0/cpu-system.rrd",
			[]string{"DS:value:GAUGE:120:U:U",
				"RRA:AVERAGE:0.5:1:100"}},

		{"server1.net/cpu-1/cpu-system.rrd",
			[]string{"DS:value:GAUGE:600:U:U",
				"RRA:AVERAGE:0.5:1:100"}},
	}

	for _, q := range files {
		file := res.Directory + q.file

		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			res.Clean()
			log.Fatal(fmt.Sprintf("Can't create directory '%v' for RRD file: %v\n", filepath.Dir(file), err))
			return res, false
		}

		args := []string{"create", file, "--start", "943920000", "--step", "60"}
		args = append(args, q.args...)

		cmd := exec.Command("rrdtool", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			res.Clean()
			log.Fatal(fmt.Sprintf("Can't create RRD file '%s':\n%s", file, out))
			return res, false
		}
	}

	return res, true
}

func (r *TestRRD) Clean() {
	os.RemoveAll(r.Directory)
}

func (r *TestRRD) Fatal(msg string) {
	defer r.Clean()
	log.Fatal("Error: " + msg)
}

func (r *TestRRD) Fatalf(format string, args ...interface{}) {
	r.Fatal(fmt.Sprintf(format, args...))
}

func (r TestRRD) InsertValues(File string, Time string, Values ...float64) {
	tz, _ := time.LoadLocation("Local")
	t, err := time.ParseInLocation("2006.01.02 15:04:05", Time, tz)
	if err != nil {
		r.Fatal(fmt.Sprintf("%v", err))
	}

	counts := fmt.Sprintf("%v", uint64(t.Unix()))
	for _, v := range Values {
		counts += fmt.Sprintf(":%f", v)
	}

	args := make([]string, 3)
	args[0] = "update"
	args[1] = r.Directory + File
	args[2] = counts

	cmd := exec.Command("rrdtool", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.Fatal(fmt.Sprintf("can't insert values to RRD file '%s':\n%s", File, out))
	}
}

func comparePaths(s1, s2 []string) bool {
	sort.Strings(s1)
	sort.Strings(s2)
	return fmt.Sprintf("%+v", s1) == fmt.Sprintf("%+v", s2)
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

func ParseWantTable(table string) []DataPoints {
	res := []DataPoints{}

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

		items = items[1:]
		for i := len(res); i < len(items); i++ {
			res = append(res, DataPoints{})
		}
		n := 0
		for _, s := range items {
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

func MakeGetRequest(t *testing.T, f http.HandlerFunc, query string) string {
	if query != "" && !strings.HasPrefix(query, "?") {
		query = "?" + query
	}

	req, err := http.NewRequest("GET", "http://127.0.0.1"+query, nil)

	if err != nil {
		log.Fatal(fmt.Sprintf("Incorrect get query '%v':\nError: %v", query, err))
	}

	w := httptest.NewRecorder()
	f(w, req)

	body := w.Body.String()
	if w.Code != 200 {
		t.Errorf("Get query: '%s'\n HTTP Error: %v", query, body)
		return ""
	}

	return body
}

func MakePostRequest(t *testing.T, f http.HandlerFunc, query string) string {
	req, err := http.NewRequest("POST", "http://127.0.0.1", strings.NewReader(query))

	if err != nil {
		log.Fatal(fmt.Sprintf("Incorrect post query '%v':\nError: %v", query, err))
	}

	w := httptest.NewRecorder()
	f(w, req)

	body := w.Body.String()
	if w.Code != 200 {
		t.Errorf("Post query: '%s'\n HTTP Error: %v", query, body)
		return ""
	}

	return body
}
