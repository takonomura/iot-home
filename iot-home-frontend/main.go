package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	_ "github.com/influxdata/influxdb1-client/v2"
	client "github.com/influxdata/influxdb1-client/v2"
)

var dataRangePattern = regexp.MustCompile(`^[0-9]{1,2}[smhd]$`)

type Point struct {
	X json.Number `json:"x"`
	Y json.Number `json:"y"`
}

type Line struct {
	Room   string  `json:"label"`
	Points []Point `json:"data"`
}

type Data struct {
	Temperature []*Line `json:"temperature"`
	Pressure    []*Line `json:"pressure"`
	Humidity    []*Line `json:"humidity"`
}

func buildQuery(timeRange, interval string) string {
	return fmt.Sprintf(`SELECT mean("temperature"), mean("pressure"), mean("humidity") FROM "room_sensors" WHERE time >= now() - %s GROUP BY "room", time(%s) fill(null)`, timeRange, interval)
}

func parseResponse(resp *client.Response) Data {
	data := Data{
		Temperature: make([]*Line, 0, 1),
		Pressure:    make([]*Line, 0, 1),
		Humidity:    make([]*Line, 0, 1),
	}

	for _, result := range resp.Results {
		for _, row := range result.Series {
			room := row.Tags["room"]

			temp := &Line{Room: room}
			data.Temperature = append(data.Temperature, temp)
			press := &Line{Room: room}
			data.Pressure = append(data.Pressure, press)
			humidity := &Line{Room: room}
			data.Humidity = append(data.Humidity, humidity)

			for _, v := range row.Values {
				x := v[0].(json.Number)
				if y, ok := v[1].(json.Number); ok {
					temp.Points = append(temp.Points, Point{X: x, Y: y})
				}
				if y, ok := v[2].(json.Number); ok {
					press.Points = append(press.Points, Point{X: x, Y: y})
				}
				if y, ok := v[3].(json.Number); ok {
					humidity.Points = append(humidity.Points, Point{X: x, Y: y})
				}
			}
		}
	}
	return data
}

var c client.Client

func initClient() {
	addr := os.Getenv("INFLUXDB_ADDR")
	if addr == "" {
		fmt.Fprintln(os.Stderr, "INFLUXDB_ADDR is required")
		os.Exit(1)
	}

	var err error
	c, err = client.NewHTTPClient(client.HTTPConfig{
		Addr: addr,
	})
	if err != nil {
		log.Fatalf("creating InfluxDB client: %s", err)
	}
}

func getData(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	dataRange := r.URL.Query().Get("range")
	if dataRange == "" {
		dataRange = "30m"
	}
	if !dataRangePattern.MatchString(dataRange) {
		http.Error(rw, "400 Forbidden", http.StatusForbidden)
		return
	}
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "15s"
	}
	if !dataRangePattern.MatchString(interval) {
		http.Error(rw, "400 Forbidden", http.StatusForbidden)
		return
	}

	q := client.NewQuery(buildQuery(dataRange, interval), "iot_home", "ms")
	resp, err := c.Query(q)
	if err != nil {
		log.Printf("querying: %s", err)
		http.Error(rw, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = resp.Error()
	if err != nil {
		log.Printf("querying: %s", err)
		http.Error(rw, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(rw).Encode(parseResponse(resp)); err != nil {
		log.Printf("encoding json: %s", err)
	}
}

func main() {
	initClient()
	defer c.Close()

	http.HandleFunc("/data.json", getData)
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.ListenAndServe(":8080", nil)
}
