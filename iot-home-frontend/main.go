package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/influxdata/influxdb1-client/v2"
)

func envOrDefault(name, def string) string {
	v := os.Getenv(name)
	if v == "" {
		v = def
	}
	return v
}

func main() {
	db, err := NewInfluxDB(envOrDefault("INFLUX_ADDR", "http://localhost:8086"), envOrDefault("IOT_HOME_DB", "iot_home"))
	if err != nil {
		log.Fatalf("creating db: %s", err)
	}

	s := &Server{
		DB: db,

		AssetsDir: envOrDefault("IOT_HOME_ASSETS_DIR", "./assets"),
		IndexFile: envOrDefault("IOT_HOME_INDEX_FILE", "./index.html"),
	}

	http.ListenAndServe(":8080", s.Mux())
}
