package rrdserver

import (
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rrdserver/rrdserver/api"
	"github.com/rrdserver/rrdserver/log"
	"net/http"
	"strings"
)

func Serve() {
	config := NewConfig()

	router := mux.NewRouter()

	router.Methods("OPTIONS").HandlerFunc(optionsHandler)
	router.Path("/").HandlerFunc(indexHandler)
	router.Path("/index.html").HandlerFunc(indexHandler)

	// API ............................
	api := api.NewAPI(config.Metrics.DataDir)
	api.Serve(router)
	api.Serve(router.PathPrefix("/api/v1/").Subrouter())
	api.Serve(router.PathPrefix("/api/").Subrouter())

	// Auth ...........................
	var handler http.Handler
	if config.Server.User != "" && config.Server.Password != "" {
		var users = make(map[string]string)
		users[config.Server.User] = config.Server.Password
		handler = NewAuthHandler(users, handler)
	} else {
		handler = router
	}

	log.Info("Starting RRD server")
	log.Info("Listen: %v:%v", config.Server.Bind, config.Server.Port)

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", config.Server.Bind, config.Server.Port), handler)
	if err != nil {
		log.Fatal("Can't start server: %v", err)
	}
}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	html := `<html>
            <head>
            <meta http-equiv='Content-Type' content='text/html; charset=utf-8'>
            <title>RRD server</title>
            <style type='text/css'>
                body { font-family: helvetica, arial, freesans, clean, sans-serif; font-size:15px; 
                    width:840px;margin:20px auto; color:#666;}  
                h1 {font-size:30px;font-weight:300; color:#000}
                a, a:visited {color:#4183c4}
            </style>
            </head>
            <body>
            <h1>RRD server</h1>
            The RRDServer REST API lets you fetch data from .rrd files using simple HTTP methods.<br>
            Detailed information about API, you can found on an <a href='http:rrdserver.io/doc'>documentation</a> page.
            </body>
            </html>`
	w.Write([]byte(html))
}

type AuthHandler struct {
	users   map[string]string
	handler http.Handler
}

func NewAuthHandler(users map[string]string, handler http.Handler) *AuthHandler {
	return &AuthHandler{users: users, handler: handler}
}

func (a *AuthHandler) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		log.Warning("Incorrect authorization header: '%v'", r.Header.Get("Authorization"))
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		log.Warning("Incorrect authorization header: '%v'", r.Header.Get("Authorization"))
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		log.Warning("Incorrect authorization header: '%v'", r.Header.Get("Authorization"))
		return false
	}

	pass, ok := a.users[pair[0]]
	return ok && pass == pair[1]
}

func (a *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" || a.checkAuth(w, r) {
		a.handler.ServeHTTP(w, r)
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="RRD server"`)
	w.WriteHeader(401)
	w.Write([]byte("401 Unauthorized\n"))
}
