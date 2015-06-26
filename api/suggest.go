package api

import (
	"encoding/json"
	"fmt"
	"github.com/ziutek/rrd"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type SuggestMetricsRequest struct {
	Query  string `json:"query"`
	WithDS bool   `json:"withds"`
}

type SuggestMetric struct {
	Metric string   `json:"metric"`
	DS     []string `json:"ds"`
}

type SuggestMetricsResponse []SuggestMetric

func (api *API) SuggestMetricsGetHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	api.CommonHeader(w, r)

	if err = r.ParseForm(); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	req := SuggestMetricsRequest{WithDS: true}

	if v, ok := r.Form["query"]; ok {
		req.Query, err = Unquote(v[0])
		if err != nil {
			BadRequest(w, "Incorrect query '%v': %v\n", v[0], err)
			return
		}
	}

	if v, ok := r.Form["withds"]; ok {
		req.WithDS, err = strconv.ParseBool(v[0])
		if err != nil {
			BadRequest(w, "%v", err)
			return
		}
	}

	res, err := api.SuggestMetrics(req)

	if err != nil {
		InternalServerError(w, "%v", err)
		return
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		InternalServerError(w, "%v", err)
		return
	}
}

func (api *API) SuggestMetricsPostHandler(w http.ResponseWriter, r *http.Request) {
	api.CommonHeader(w, r)

	if r.Body == nil {
		BadRequest(w, "Empty request.")
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		BadRequest(w, "%v", err)
		return
	}

	req := SuggestMetricsRequest{WithDS: true}

	if len(body) > 0 {
		if err = json.Unmarshal(body, &req); err != nil {
			BadRequest(w, "%v", err)
			return
		}
	}

	res, err := api.SuggestMetrics(req)

	if err != nil {
		InternalServerError(w, "%v", err)
		return
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		InternalServerError(w, "%v", err)
		return
	}
}

func splitSuggestMetricsRequestQuery(query string) (metric, ds string) {
	items := strings.SplitN(query, ":", 2)
	metric = items[0]
	if len(items) > 1 {
		ds = items[1]
	}
	return metric, ds
}

func (api API) SuggestMetrics(req SuggestMetricsRequest) (SuggestMetricsResponse, error) {
	dataDirLen := len(api.DataDir)

	reqMetric, reqDS := splitSuggestMetricsRequestQuery(req.Query)

	res := SuggestMetricsResponse{}

	for _, f := range api.findRRDFiles(reqMetric) {

		item := SuggestMetric{
			Metric: f[dataDirLen : len(f)-4],
			DS:     []string{},
		}

		if req.WithDS {
			ds, err := api.GetDataSources(f)
			if err != nil {
				return SuggestMetricsResponse{}, err
			}

			for _, d := range ds {
				if strings.HasPrefix(d, reqDS) {
					item.DS = append(item.DS, d)
				}
			}

		}

		if !req.WithDS || len(item.DS) > 0 {
			res = append(res, item)
		}
	}

	return res, nil
}

func (api API) GetDataSources(rddFile string) ([]string, error) {
	res := []string{}

	inf, err := rrd.Info(rddFile)
	if err != nil {
		return nil, fmt.Errorf("Can't get info for %s file: %v", rddFile, err)
	}

	switch v := inf["ds.type"].(type) {
	case map[string]interface{}:
		for ds := range v {
			res = append(res, ds)
		}
	default:
		return nil, fmt.Errorf("Can't get ds.type from info for %s file", rddFile)
	}

	sort.Strings(res)
	return res, nil
}

func (api *API) findRRDFiles(query string) []string {
	path := api.DataDir + SafeMetric(strings.TrimRight(query, "/"))
	dir := filepath.Dir(path)

	if isFile(path + ".rrd") {
		return []string{path + ".rrd"}
	}

	var startPath string
	if isDir(path) {
		startPath = path
	} else {
		startPath = dir
	}

	res := []string{}
	filepath.Walk(startPath, func(file string, f os.FileInfo, e error) error {

		if e != nil {
			return e
		}

		if !f.IsDir() && strings.HasSuffix(file, ".rrd") && strings.HasPrefix(file, path) {
			res = append(res, file)
		}
		return nil
	})

	sort.Strings(res)
	return res
}
