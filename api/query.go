package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ziutek/rrd"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type QueryRequestQuery struct {
	Query  string `json:"query"`
	Hidden bool   `json:"hidden"`
}

type QueryRequest struct {
	Start   Time                `json:"start"`
	End     Time                `json:"end"`
	Step    Duration            `json:"step"`
	Queries []QueryRequestQuery `json:"queries"`
}

type QueryResponse struct {
	Start  Time                  `json:"start"`
	End    Time                  `json:"end"`
	Step   Duration              `json:"step"`
	Result []QueryRespDataPoints `json:"result"`
}

type QueryRespDataPoints struct {
	Name   string     `json:"name"`
	Values DataPoints `json:"dps"`
}

func (req *QueryRequest) Check() error {
	if err := CheckStartEndTime(&req.Start, &req.End, time.Time{}, time.Now()); err != nil {
		return fmt.Errorf("Incorrect request: %v", err)
	}

	if req.Step == 0 {
		req.Step = Duration(time.Second)
	}

	if len(req.Queries) == 0 {
		return errors.New("Missing parameter 'queries' in the request.")
	}

	for _, q := range req.Queries {
		if q.Query == "" {
			return errors.New("Missing parameter 'queries' in the request.")
		}
	}

	return nil
}

type QueryDef struct {
	Type    string
	Name    string
	Metric  string
	DS      string
	CF      Consolidation
	Options string
}

func QueryDefFromString(s string) (QueryDef, error) {

	items := strings.SplitN(s, ":", 5)
	if len(items) < 2 {
		return QueryDef{}, fmt.Errorf("Incorrect query '%v'", s)
	}

	res := QueryDef{}

	switch strings.ToUpper(items[0]) {
	case "DEF":
		// DEF:<vname>=<rrdfile>:<ds-name>:<CF>[:step=<step>][:start=<time>][:end=<time>][:reduce=<CF>]
		res.Type = "DEF"
		res.DS = items[2]

		var err error
		if res.CF, err = ConsolidationFromString(items[3]); err != nil {
			return QueryDef{}, fmt.Errorf("Incorrect consolidation in the query '%v'", s)
		}

		if len(items) > 4 {
			res.Options = items[4]
		}

	case "CDEF":
		// CDEF:vname=RPN expression
		res.Type = "CDEF"

	default:
		return QueryDef{}, fmt.Errorf("Incorrect type '%s' in the query '%v'", strings.ToUpper(items[0]), s)
	}

	it := strings.SplitN(items[1], "=", 2); 
	if len(it) < 2 {
		return QueryDef{}, fmt.Errorf("Incorrect query '%v'", s)
	}

	res.Name = it[0]
	res.Metric = it[1]

	return res, nil
}

func (api API) QueryGetHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	api.CommonHeader(w, r)

	if err = r.ParseForm(); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	req := QueryRequest{}

	if v, ok := r.Form["start"]; ok {
		req.Start, err = TimeFromString(v[0])
		if err != nil {
			BadRequest(w, "%v", err)
			return
		}
	}

	if v, ok := r.Form["end"]; ok {
		req.End, err = TimeFromString(v[0])
		if err != nil {
			BadRequest(w, "%v", err)
			return
		}
	}

	if v, ok := r.Form["step"]; ok {
		req.Step, err = DurationFromString(v[0])
		if err != nil {
			return
		}
	}

	queries, _ := r.Form["query"]
	hiddens, _ := r.Form["hidden"]

	for i, q := range queries {
		q, err = Unquote(q)
		if err != nil {
			BadRequest(w, "Incorrect query '%v': %v\n", q, err)
			return
		}

		qr := QueryRequestQuery{Query: q}
		if i < len(hiddens) {
			qr.Hidden, err = strconv.ParseBool(hiddens[i])
		}

		req.Queries = append(req.Queries, qr)
	}

	if err = req.Check(); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	res, err := api.query(req)
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

func (api API) QueryPostHandler(w http.ResponseWriter, r *http.Request) {
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

	req := QueryRequest{}
	if err = json.Unmarshal(body, &req); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	if err = req.Check(); err != nil {
		BadRequest(w, "%v", err)
		return
	}

	res, err := api.query(req)
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

func (api API) query(req QueryRequest) (QueryResponse, error) {

	rrd := rrd.NewExporter()
	for _, q := range req.Queries {
		def, err := QueryDefFromString(q.Query)
		if err != nil {
			return QueryResponse{}, err
		}

		switch def.Type {
		case "DEF":
			rrd.Def(def.Name, api.FileForMetric(def.Metric), def.DS, def.CF.String(), def.Options)
			if !q.Hidden {
				rrd.XportDef(def.Name, def.Name)
			}

		case "CDEF":
			rrd.CDef(def.Name, def.Metric)
			if !q.Hidden {
				rrd.XportDef(def.Name, def.Name)
			}
		}
	}

	rrdRes, err := rrd.Xport(time.Time(req.Start), time.Time(req.End), time.Duration(req.Step))
	if err != nil {
		return QueryResponse{}, err
	}
	defer rrdRes.FreeValues()

	res := QueryResponse{
		Start:  req.Start,
		End:    req.End,
		Step:   req.Step,
		Result: []QueryRespDataPoints{},
	}

	end := time.Time(req.End)
	for idx, name := range rrdRes.Legends {
		data := QueryRespDataPoints{
			Name:   name,
			Values: make(DataPoints, rrdRes.RowCnt),
		}

		for k, t := 0, rrdRes.Start.Add(rrdRes.Step); t.Before(end) || t.Equal(end); k, t = k+1, t.Add(rrdRes.Step) {
			data.Values[Time(t)] = rrdRes.ValueAt(idx, k)
		}
		res.Result = append(res.Result, data)
	}

	return res, nil
}
