package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ziutek/rrd"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type SuggestRequest struct {
	Query     string `json:"query"`
	Recursive bool   `json:"recursive"`
}

func (req *SuggestRequest) FromForm(form url.Values) (err error) {
	if v, ok := form["query"]; ok {
		req.Query = SafeMetric(v[0])
	}

	if v, ok := form["recursive"]; ok {
		req.Recursive, err = strconv.ParseBool(v[0])
		if err != nil {
			return
		}
	}

	return nil
}

func (api *API) suggestMetricsGetHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	req := new(SuggestRequest)
	if err := req.FromForm(r.Form); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	api.suggestMetrics(*req, w, r)
}

func (api *API) suggestMetricsPostHandler(w http.ResponseWriter, r *http.Request) {
	req := SuggestRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	api.suggestMetrics(req, w, r)
}

func (api *API) suggestMetrics(req SuggestRequest, w http.ResponseWriter, r *http.Request) {
	api.CommonHeader(w, r)
	req.Query = SafeMetric(req.Query)

	var metrics []string
	var err error
	if req.Recursive {
		metrics, err = api.findMetricsRecursive(req.Query)
	} else {
		metrics, err = api.findMetricsNonRecursive(req.Query)
	}

	if err != nil {
		InternalServerError(w, "%v", err)
		return
	}

	res, err := json.Marshal(metrics)
	if err != nil {
		InternalServerError(w, "%v", err)
		return
	}

	w.Write(res)
}

func getDataSources(rddFile string) ([]string, error) {
	res := []string{}

	inf, err := rrd.Info(rddFile)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Can't get info for %s file: %v", rddFile, err))
	}

	switch v := inf["ds.type"].(type) {
	case map[string]interface{}:
		for ds := range v {
			res = append(res, ds)
		}
	default:
		return nil, errors.New(fmt.Sprintf("Can't get ds.type from info for %s file", rddFile))
	}

	return res, nil
}

func (api *API) findRRDFiles(query string) []string {

	path := api.DataDir + SafeMetric(query)
	dir := filepath.Dir(path)

	// Query like "interface-eth0/if_packets"
	if fileInfo, err := os.Stat(path + ".rrd"); err == nil && !fileInfo.IsDir() {
		return []string{path + ".rrd"}
	}

	// Query like "interface-eth0/if_packets/rx", where rx is datasource
	if fileInfo, err := os.Stat(dir + ".rrd"); err == nil && !fileInfo.IsDir() {
		return []string{dir + ".rrd"}
	}

	var startPath string
	if fileInfo, err := os.Stat(path); err == nil && fileInfo.IsDir() {
		startPath = path
	} else {
		startPath = dir
	}

	res := make([]string, 0)
	filepath.Walk(startPath, func(file string, f os.FileInfo, e error) error {

		if e != nil {
			return e
		}

		if !f.IsDir() && strings.HasSuffix(file, ".rrd") && strings.HasPrefix(file, path) {
			res = append(res, file)
		}
		return nil
	})

	return res
}

func (api *API) findMetricsRecursive(query string) ([]string, error) {
	files := api.findRRDFiles(query)
	res := make([]string, 0)

	dir := filepath.Dir(api.DataDir + query)
	dataDirLen := len(api.DataDir)

	for _, f := range files {
		if !strings.HasPrefix(f, dir) {
			continue
		}

		dss, err := getDataSources(f)
		if err != nil {
			return nil, err
		}

		if len(dss) == 1 {
			res = append(res, f[dataDirLen:len(f)-4])
		} else {
			for _, ds := range dss {
				metric := f[dataDirLen:len(f)-4] + "/" + ds
				if strings.HasPrefix(metric, query) {
					res = append(res, metric)
				}
			}
		}
	}

	sort.Strings(res)
	return res, nil
}

func (api *API) findMetricsNonRecursive(query string) ([]string, error) {
	files := api.findRRDFiles(query)
	m := map[string]bool{}

	dir, base := filepath.Split(api.DataDir + query)
	dir = strings.TrimRight(dir, "/")
	dirLen := len(dir)

	for _, f := range files {

		if !strings.HasPrefix(f, dir) {
			continue
		}

		if f == dir+".rrd" {

			dss, err := getDataSources(f)
			if err != nil {
				return nil, err
			}

			if len(dss) != 1 {
				for _, ds := range dss {
					if strings.HasPrefix(ds, base) {
						m[ds] = true
					}
				}
			}
		} else {

			metric := f[dirLen+1 : len(f)-4]

			if e := strings.Index(metric, "/"); e > -1 {
				m[metric[:e]] = true
			} else {
				m[metric] = true
			}
		}
	}

	res := make([]string, 0)

	for k := range m {
		res = append(res, k)
	}

	sort.Strings(res)
	return res, nil
}
