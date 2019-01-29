package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"

	"github.com/gorilla/pat"
	"github.com/markbates/goth/gothic"
)

var timeRangePattern = regexp.MustCompile(`^[0-9]{1,2}[smhd]$`)

func writeJSON(rw http.ResponseWriter, statusCode int, v interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	if err := json.NewEncoder(rw).Encode(v); err != nil {
		log.Printf("writing json response: %s", err)
	}
}

func writeError(rw http.ResponseWriter, statusCode int, msg string) {
	writeJSON(rw, statusCode, struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
	}{
		Code: statusCode,
		Msg:  msg,
	})
}

func writeData(rw http.ResponseWriter, data Data) {
	writeJSON(rw, http.StatusOK, struct {
		Data Data `json:"data"`
	}{
		Data: data,
	})
}

type Server struct {
	DB *InfluxDB

	AssetsDir string
	IndexFile string

	AllowedEmails []string
}

func (s *Server) authCallback(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Printf("completing user auth: %s", err)
		writeError(w, http.StatusInternalServerError, "Authentication failed")
		return
	}
	gothic.StoreInSession("email", user.Email, r, w)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) isAllowed(r *http.Request) bool {
	v, err := gothic.GetFromSession("email", r)
	if err != nil {
		return false
	}
	for _, s := range s.AllowedEmails {
		if v == s {
			return true
		}
	}
	return false
}

func (s *Server) getData(rw http.ResponseWriter, r *http.Request) {
	if !s.isAllowed(r) {
		writeError(rw, http.StatusForbidden, "forbidden")
		return
	}

	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "30m"
	}
	if !timeRangePattern.MatchString(timeRange) {
		writeError(rw, http.StatusBadRequest, "range is not valid")
		return
	}
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "15s"
	}
	if !timeRangePattern.MatchString(interval) {
		writeError(rw, http.StatusBadRequest, "interval is not valid")
		return
	}

	data, err := s.DB.Query(timeRange, interval)
	if err != nil {
		log.Printf("querying: %s", err)
		writeError(rw, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeData(rw, data)
}

func (s *Server) getIndex(rw http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(rw, r)
		return
	}
	if !s.isAllowed(r) {
		http.Redirect(rw, r, "/auth/google", http.StatusTemporaryRedirect)
		return
	}
	http.ServeFile(rw, r, s.IndexFile)
}

func (s *Server) Mux() *pat.Router {
	p := pat.New()

	p.PathPrefix("/assets").Handler(http.StripPrefix("/assets", http.FileServer(http.Dir(s.AssetsDir))))

	p.Get("/auth/{provider}/callback", s.authCallback)
	p.Get("/auth/{provider}", gothic.BeginAuthHandler)

	p.Get("/data.json", s.getData)
	p.Get("/", s.getIndex)

	return p
}
