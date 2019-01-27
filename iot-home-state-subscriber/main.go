package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	influxdb "github.com/influxdata/influxdb1-client/v2"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig
		log.Println("Stopping")
		cancel()
	}()

	proj := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if proj == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT is not set")
	}
	client, err := pubsub.NewClient(ctx, proj)
	if err != nil {
		log.Fatalf("creating pubsub client: %s", err)
	}

	sub := os.Getenv("IOT_HOME_STATE_SUBSCRIPTION")
	if sub == "" {
		log.Fatal("IOT_HOME_STATE_SUBSCRIPTION is not set")
	}

	addr := os.Getenv("INFLUXDB_ADDR")
	if addr == "" {
		log.Fatal("INFLUXDB_ADDR is not set")
	}
	db, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr: addr,
	})
	if err != nil {
		log.Fatalf("creating InfluxDB client: %s", err)
	}
	defer db.Close()
	if _, _, err := db.Ping(10 * time.Second); err != nil {
		log.Fatalf("pinging InfluxDB: %s", err)
	}

	err = client.Subscription(sub).Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		fmt.Println(msg.PublishTime)
		fmt.Println(string(msg.Data))

		data := struct {
			Room        string  `json:"room"`
			Temperature float32 `json:"temperature"`
			Pressure    float32 `json:"pressure"`
			Humidity    float32 `json:"humidity"`
		}{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Printf("unmarshaling data: %s", err)
			return
		}

		bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  "iot_home",
			Precision: "s",
		})
		if err != nil {
			log.Printf("creating batch points: %s", err)
		}

		tags := map[string]string{"room": data.Room}
		fields := map[string]interface{}{
			"temperature": data.Temperature,
			"pressure":    data.Pressure,
			"humidity":    data.Humidity,
		}
		p, err := influxdb.NewPoint("room_sensors", tags, fields, msg.PublishTime)
		if err != nil {
			log.Printf("creating point: %s", err)
			return
		}
		bp.AddPoint(p)

		if err := db.Write(bp); err != nil {
			log.Printf("writing to db: %s", err)
			return
		}
		msg.Ack()
	})
	if err != nil {
		log.Fatalf("receiving messages: %s", err)
	}
}
