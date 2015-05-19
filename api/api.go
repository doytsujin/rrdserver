package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rrdserver/rrdserver/log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type API struct {
	DataDir string
}

func NewAPI(dataDir string) API {
	return API{
		DataDir: dataDir,
	}
}

func (api *API) Serve(router *mux.Router) {
	router.Methods("GET").Path("/query").HandlerFunc(api.QueryGetHandler)
	router.Methods("POST").Path("/query").HandlerFunc(api.QueryPostHandler)

	router.Methods("GET").Path("/suggest/metrics").HandlerFunc(api.suggestMetricsGetHandler)
	router.Methods("POST").Path("/suggest/metrics").HandlerFunc(api.suggestMetricsPostHandler)
}

func (api *API) CommonHeader(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
}

type ErrorResponse struct {
	Code    int    `json:"errorCode"`
	Message string `json:"errorMessage"`
}

func srvError(w http.ResponseWriter, httpStatus int, format string, args ...interface{}) {
	log.Warning(format, args...)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	body, _ := json.Marshal(ErrorResponse{
		Code:    httpStatus,
		Message: fmt.Sprintf(format, args...)})

	w.Write(body)
	w.Write([]byte("\n"))
}

func BadRequest(w http.ResponseWriter, format string, args ...interface{}) {
	srvError(w, http.StatusBadRequest, format, args...)
}

func InternalServerError(w http.ResponseWriter, format string, args ...interface{}) {
	srvError(w, http.StatusInternalServerError, format, args...)
}

func SafeMetric(metric string) string {
	res := strings.Replace(metric, "|", "", -1)
	res = filepath.Clean("/" + res)
	if strings.HasSuffix(metric, "/") {
		res += "/"
	}
	return strings.TrimLeft(res, "/")
}

func isFile(fileName string) bool {
	fileInfo, err := os.Stat(fileName)
	return err == nil && !fileInfo.IsDir()
}

func isDir(fileName string) bool {
	fileInfo, err := os.Stat(fileName)
	return err == nil && fileInfo.IsDir()
}
