package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
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
}

func (s *Server) getData(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(rw, http.StatusMethodNotAllowed, "Method Not Allowed")
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(rw, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	http.ServeFile(rw, r, s.IndexFile)
}

func (s *Server) Mux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.getIndex)
	mux.HandleFunc("/data.json", s.getData)
	mux.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir(s.AssetsDir))))

	return mux
}
