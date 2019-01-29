package main

import (
	"encoding/json"
	"fmt"

	client "github.com/influxdata/influxdb1-client/v2"
)

const queryTemplate = `SELECT mean("temperature"), mean("pressure"), mean("humidity") FROM "room_sensors" WHERE time >= now() - %s GROUP BY "room", time(%s) fill(null)`

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

type InfluxDB struct {
	Client   client.Client
	Database string
}

func NewInfluxDB(addr, db string) (*InfluxDB, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: addr})
	if err != nil {
		return nil, err
	}

	return &InfluxDB{
		Client:   c,
		Database: db,
	}, nil
}

func (c *InfluxDB) Query(timeRange, interval string) (Data, error) {
	q := client.NewQuery(fmt.Sprintf(queryTemplate, timeRange, interval), c.Database, "ms")
	resp, err := c.Client.Query(q)
	if err != nil {
		return Data{}, err
	}
	err = resp.Error()
	if err != nil {
		return Data{}, err
	}

	return parseResponse(resp), nil
}
