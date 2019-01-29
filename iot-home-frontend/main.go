package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
	_ "github.com/influxdata/influxdb1-client/v2"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func getenv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is required", name)
	}
	return v
}

func envList(name string) []string {
	s := getenv(name)
	if strings.Contains(s, ",") {
		return strings.Split(s, ",")
	}
	return strings.Split(s, " ")
}

func envOrDefault(name, def string) string {
	v := os.Getenv(name)
	if v == "" {
		v = def
	}
	return v
}

func envBool(name string, def bool) bool {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	return v == "1" || v == "true"
}

func main() {
	db, err := NewInfluxDB(envOrDefault("INFLUX_ADDR", "http://localhost:8086"), envOrDefault("IOT_HOME_DB", "iot_home"))
	if err != nil {
		log.Fatalf("creating db: %s", err)
	}

	goth.UseProviders(google.New(getenv("GOOGLE_KEY"), getenv("GOOGLE_SECRET"), getenv("BASE_URL")+"/auth/google/callback"))

	cookieStore := sessions.NewCookieStore([]byte(getenv("SESSION_SECRET")))
	cookieStore.Options.HttpOnly = true
	if envBool("SESSION_SECURE", false) {
		cookieStore.Options.Secure = true
	}
	gothic.Store = cookieStore

	s := &Server{
		DB: db,

		AssetsDir: envOrDefault("IOT_HOME_ASSETS_DIR", "./assets"),
		IndexFile: envOrDefault("IOT_HOME_INDEX_FILE", "./index.html"),

		AllowedEmails: envList("ALLOWED_EMAILS"),
	}

	http.ListenAndServe(":8080", s.Mux())
}
