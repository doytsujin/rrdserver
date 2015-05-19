package api

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

type TestRRD struct {
	Directory string
	Test      *testing.T
}

func NewTestRRD(test *testing.T) (res *TestRRD, ok bool) {
	res = &TestRRD{Test: test}

	if dir, err := ioutil.TempDir("", "rrdserver"); err == nil {
		res.Directory = dir + "/"
	} else {
		test.Error(err)
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
			test.Error(fmt.Sprintf("Can't create directory '%v' for RRD file: %v\n", filepath.Dir(file), err))
			return res, false
		}

		args := make([]string, 4)
		args[0] = "create"
		args[1] = file
		args[2] = "--step"
		args[3] = "60"
		args = append(args, q.args...)

		cmd := exec.Command("rrdtool", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			res.Clean()
			test.Error(fmt.Sprintf("Can't create RRD file '%s':\n%s", file, out))
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

	counts := fmt.Sprintf("%v", t.Unix())
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

func isFile(file string) bool {
	fileInfo, err := os.Stat(file)
	return err == nil && !fileInfo.IsDir()
}
