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

type QueryRequest struct {
	Start         time.Time     `json:"start"`
	End           time.Time     `json:"end"`
	Metric        string        `json:"metric"`
	Consolidation Consolidation `json:"consolidation"`
	Resolution    time.Duration `json:"resolution"`
	file          string
	ds            string
}

type QueryResponseValues map[time.Time]float64

type QueryResponse struct {
	Query  QueryRequest        `json:"query"`
	Values QueryResponseValues `json:"result"`
}

func (req *QueryRequest) FromForm(form url.Values) (err error) {
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

	if v, ok := form["metric"]; ok {
		req.Metric = SafeMetric(v[0])
	}

	if v, ok := form["consolidation"]; ok {
		req.Consolidation, err = ConsolidationFromString(v[0])
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

	return req.Check()
}

func (req *QueryRequest) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Start         Time          `json:"start"`
		End           Time          `json:"end"`
		Metric        string        `json:"metric"`
		Consolidation Consolidation `json:"consolidation"`
		Resolution    Duration      `json:"resolution"`
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	req.Start = time.Time(tmp.Start)
	req.End = time.Time(tmp.End)
	req.Metric = SafeMetric(tmp.Metric)
	req.Consolidation = tmp.Consolidation
	req.Resolution = time.Duration(tmp.Resolution)

	return req.Check()
}

func (req *QueryRequest) Check() error {
	if req.Metric == "" {
		return errors.New("Missing parameter 'metric' in the request.")
	}

	if req.Start.IsZero() {
		return errors.New("Missing parameter 'start' in the request.")
	}

	if req.End.IsZero() {
		req.End = time.Now()
	}

	if req.End.Before(req.Start) {
		return errors.New("Parameter 'end' should be grater then 'start'")
	}

	if req.Resolution == 0 {
		req.Resolution = time.Second
	}

	return nil
}

func (req QueryRequest) MarshalJSON() ([]byte, error) {
	metric, _ := json.Marshal(req.Metric)
	res := "{"
	res += fmt.Sprintf("\"start\": %v,\n", req.Start.Unix())
	res += fmt.Sprintf("\"end\": %v,\n", req.End.Unix())
	res += fmt.Sprintf("\"metric\": %s,\n", string(metric))
	res += fmt.Sprintf("\"consolidation\": \"%s\",\n", req.Consolidation.String())
	res += fmt.Sprintf("\"resolution\": %v\n", req.Resolution.Seconds())
	res += "}"
	return []byte(res), nil
}

func (vals QueryResponseValues) MarshalJSON() ([]byte, error) {
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

func NewQueryResponse(req QueryRequest, count int) QueryResponse {
	return QueryResponse{req, make(QueryResponseValues, count)}
}

func isArray(data []byte) bool {
	var v []interface{}
	return json.Unmarshal(data, &v) == nil
}

func (api *API) QueryPostHandler(w http.ResponseWriter, r *http.Request) {
	api.CommonHeader(w, r)
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		BadRequest(w, "%v", err)
		return
	}

	var req []*QueryRequest

	if isArray(body) {

		if err = json.Unmarshal(body, &req); err != nil {
			BadRequest(w, "%v", err)
			return
		}

	} else {
		var r QueryRequest
		if err = json.Unmarshal(body, &r); err != nil {
			BadRequest(w, "%v", err)
			return
		}
		req = []*QueryRequest{&r}
	}

	res, err := api.query(req)
	if err != nil {
		InternalServerError(w, "%v", err)
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		InternalServerError(w, "%v", err)
	}
}

func (api *API) QueryGetHandler(w http.ResponseWriter, r *http.Request) {
	api.CommonHeader(w, r)
	if err := r.ParseForm(); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	req := QueryRequest{}
	if err := (&req).FromForm(r.Form); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	res, err := api.query([]*QueryRequest{&req})
	if err != nil {
		InternalServerError(w, "%v", err)
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		InternalServerError(w, "%v", err)
	}

}

func (api *API) query(queryReq []*QueryRequest) ([]QueryResponse, error) {
	for _, req := range queryReq {
		err := req.Check()
		if err != nil {
			return nil, err
		}

		if isFile(api.DataDir + req.Metric + ".rrd") {
			req.file = api.DataDir + req.Metric + ".rrd"
			req.ds = ""
		} else if isFile(api.DataDir + filepath.Dir(req.Metric) + ".rrd") {
			req.file = api.DataDir + filepath.Dir(req.Metric) + ".rrd"
			req.ds = filepath.Base(req.Metric)
		}

		if req.file == "" {
			return nil, errors.New(fmt.Sprintf("Metric '%v' not exists", req.Metric))
		}
	}

	res := []QueryResponse{}
	processed := map[int]bool{}
	for i, base := range queryReq {
		if processed[i] {
			continue
		}

		fetchRes, err := rrd.Fetch(base.file,
			base.Consolidation.String(),
			base.Start,
			base.End,
			base.Resolution)
		if err != nil {
			return nil, err
		}
		defer fetchRes.FreeValues()

		for j := i; j < len(queryReq); j++ {
			if processed[j] {
				continue
			}
			r := queryReq[j]

			if base.file == r.file &&
				base.Consolidation == r.Consolidation &&
				base.Start == r.Start &&
				base.End == r.End &&
				base.Resolution == r.Resolution {

				processed[j] = true
				idx := indexOf(fetchRes.DsNames, r.ds)
				if idx < 0 {
					return nil, errors.New(fmt.Sprintf("ds '%v' not found in %v", r.ds, r.file))
				}

				data := NewQueryResponse(*r, fetchRes.RowCnt)
				for k, t := 0, fetchRes.Start.Add(fetchRes.Step); t.Before(r.End) || t.Equal(r.End); k, t = k+1, t.Add(fetchRes.Step) {
					data.Values[t] = fetchRes.ValueAt(idx, k)
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
