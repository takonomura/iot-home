package main

import "encoding/json"

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
