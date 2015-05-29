package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ziutek/rrd"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"time"
)

type MetricInfo struct {
	Metric        string        `json:"metric"`
	Consolidation Consolidation `json:"consolidation"`
}

type FetchRequest struct {
	Start      Time         `json:"start"`
	End        Time         `json:"end"`
	Resolution Duration     `json:"resolution"`
	Queries    []MetricInfo `json:"queries"`
}

type FetchResponseValues map[Time]float64

type FetchResponse struct {
	Start         Time                `json:"start"`
	End           Time                `json:"end"`
	Resolution    Duration            `json:"resolution"`
	Metric        string              `json:"metric"`
	Consolidation Consolidation       `json:"consolidation"`
	Values        FetchResponseValues `json:"result"`
}

func (req *FetchRequest) FromForm(form url.Values) (err error) {
	if v, ok := form["start"]; ok {
		req.Start, err = TimeFromString(v[0])
		if err != nil {
			return
		}
	}

	if v, ok := form["end"]; ok {
		req.End, err = TimeFromString(v[0])
		if err != nil {
			return
		}
	}

	if v, ok := form["resolution"]; ok {
		req.Resolution, err = DurationFromString(v[0])
		if err != nil {
			return
		}
	}

	metrics, _ := form["metric"]
	cons, _ := form["consolidation"]

	for i, m := range metrics {
		mi := MetricInfo{Metric: m}
		if i < len(cons) {
			mi.Consolidation, err = ConsolidationFromString(cons[i])
			if err != nil {
				return
			}
		}

		req.Queries = append(req.Queries, mi)
	}

	return req.Check()
}

func (req *FetchRequest) Check() error {
	if time.Time(req.Start).IsZero() {
		return errors.New("Missing parameter 'start' in the request.")
	}

	if time.Time(req.End).IsZero() {
		req.End = Time(time.Now())
	}

	if time.Time(req.End).Before(time.Time(req.Start)) {
		return errors.New("Parameter 'end' should be grater then 'start'")
	}

	if req.Resolution == 0 {
		req.Resolution = Duration(time.Second)
	}

	if len(req.Queries) == 0 {
		return errors.New("Missing parameter 'metric' in the request.")
	}

	for _, q := range req.Queries {
		if q.Metric == "" {
			return errors.New("Missing parameter 'metric' in the request.")
		}
	}

	return nil
}

func (vals FetchResponseValues) MarshalJSON() ([]byte, error) {
	res := "{"
	n := 0
	for t, v := range vals {

		if n > 0 {
			res += ",\n"
		}
		n++

		if math.IsNaN(v) {
			res += fmt.Sprintf("\"%v\":null", t.Unix())
		} else {
			res += fmt.Sprintf("\"%v\":%f", t.Unix(), v)
		}
	}
	res += "}"
	return []byte(res), nil
}

func (vals *FetchResponseValues) UnmarshalJSON(data []byte) error {
	var tmp map[string]float64

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	vls := make(FetchResponseValues, len(tmp))
	for k, v := range tmp {
		t, err := TimeFromString(k)
		if err != nil {
			return err
		}
		vls[t] = v
	}

	*vals = vls
	return nil
}

func isArray(data []byte) bool {
	var v []interface{}
	return json.Unmarshal(data, &v) == nil
}

func (api API) FetchPostHandler(w http.ResponseWriter, r *http.Request) {
	api.CommonHeader(w, r)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		BadRequest(w, "%v", err)
		return
	}

	req := FetchRequest{}
	if err = json.Unmarshal(body, &req); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	res, err := api.fetch(req)
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

func (api API) FetchGetHandler(w http.ResponseWriter, r *http.Request) {
	api.CommonHeader(w, r)
	if err := r.ParseForm(); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	req := FetchRequest{}
	if err := (&req).FromForm(r.Form); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	res, err := api.fetch(req)
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

func (api *API) fetch(queryReq FetchRequest) ([]FetchResponse, error) {
	if err := queryReq.Check(); err != nil {
		return nil, err
	}
	/*
	fmt.Printf("Request =====================\n")
	fmt.Printf("  Start: %v\n", time.Time(queryReq.Start))
	fmt.Printf("  End:   %v\n", time.Time(queryReq.End))
	fmt.Printf("  Resolution: %v\n", time.Duration(queryReq.Resolution).Seconds())
	for _, q := range queryReq.Queries {
		fmt.Printf("    - Metric: %v\t%v\n", q.Metric, q.Consolidation.String())
	}
	fmt.Printf("=============================\n")
	*/
	type Job struct {
		metric string
		cons   Consolidation
		file   string
		ds     string
	}

	jobs := []Job{}

	for _, q := range queryReq.Queries {
		job := Job{metric: q.Metric, cons: q.Consolidation}
		if isFile(api.DataDir + q.Metric + ".rrd") {
			job.file = api.DataDir + q.Metric + ".rrd"
			job.ds = ""
		} else if isFile(api.DataDir + filepath.Dir(q.Metric) + ".rrd") {
			job.file = api.DataDir + filepath.Dir(q.Metric) + ".rrd"
			job.ds = filepath.Base(q.Metric)
		}

		if job.file == "" {
			return nil, errors.New(fmt.Sprintf("Metric '%v' not exists", q.Metric))
		}

		jobs = append(jobs, job)
	}

	res := []FetchResponse{}
	processed := map[int]bool{}
	for i, job := range jobs {
		if processed[i] {
			continue
		}

		fetchRes, err := rrd.Fetch(job.file,
			job.cons.String(),
			time.Time(queryReq.Start),
			time.Time(queryReq.End),
			time.Duration(queryReq.Resolution))

		if err != nil {
			return nil, err
		}
		defer fetchRes.FreeValues()

		for k := i; k < len(jobs); k++ {
			if processed[k] {
				continue
			}
			j := jobs[k]

			if job.file == j.file && job.cons == j.cons {
				processed[k] = true
				idx := indexOf(fetchRes.DsNames, j.ds)
				if idx < 0 {
					return nil, errors.New(fmt.Sprintf("ds '%v' not found in %v", j.ds, j.file))
				}

				data := FetchResponse{
					Start:         queryReq.Start,
					End:           queryReq.End,
					Resolution:    queryReq.Resolution,
					Metric:        j.metric,
					Consolidation: j.cons,
					Values:        make(FetchResponseValues, fetchRes.RowCnt),
				}

				end := time.Time(queryReq.End)
				for k, t := 0, fetchRes.Start.Add(fetchRes.Step); t.Before(end) || t.Equal(end); k, t = k+1, t.Add(fetchRes.Step) {
					data.Values[Time(t)] = fetchRes.ValueAt(idx, k)
				}
				res = append(res, data)
			}

		}
	}

	return res, nil
}

func indexOf(dss []string, ds string) int {
	if ds == "" {
		return 0
	}

	for n, d := range dss {
		if d == ds {
			return n
		}
	}

	return -1
}
