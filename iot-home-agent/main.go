package main

import (
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gobot.io/x/gobot/drivers/i2c"
	"gobot.io/x/gobot/platforms/raspi"
)

func getenv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		fmt.Fprintf(os.Stderr, "%s is required", name)
		os.Exit(1)
	}
	return v
}

type Agent struct {
	ProjectID  string
	Region     string
	RegistryID string
	DeviceID   string

	PrivateKey *rsa.PrivateKey
	Client     mqtt.Client

	Room         string
	BME280Driver *i2c.BME280Driver
}

func (a *Agent) loadPrivateKey(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(b)
	if err != nil {
		return err
	}
	a.PrivateKey = key
	return nil
}

func (a *Agent) provideCredentials() (string, string) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &jwt.StandardClaims{
		Audience:  a.ProjectID,
		ExpiresAt: now.Add(20 * time.Minute).Unix(),
		IssuedAt:  now.Unix(),
	})
	pass, err := token.SignedString(a.PrivateKey)
	if err != nil {
		panic(err) // TODO(takonomura): Don't PANIC
	}
	return "unused", pass
}

func (a *Agent) clientID() string {
	return fmt.Sprintf("projects/%s/locations/%s/registries/%s/devices/%s",
		a.ProjectID,
		a.Region,
		a.RegistryID,
		a.DeviceID)
}

func (a *Agent) connectMQTT() error {
	opts := mqtt.NewClientOptions().
		AddBroker("ssl://mqtt.googleapis.com:8883").
		SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}).
		SetProtocolVersion(4).
		SetClientID(a.clientID()).
		SetCredentialsProvider(a.provideCredentials)
	a.Client = mqtt.NewClient(opts)

	connectToken := a.Client.Connect()
	if connectToken.Error() != nil {
		return connectToken.Error()
	}
	connectToken.WaitTimeout(10 * time.Second)
	return connectToken.Error()
}

func (a *Agent) initBME280() error {
	a.BME280Driver = i2c.NewBME280Driver(raspi.NewAdaptor(), i2c.WithBus(1), i2c.WithAddress(0x76))
	return a.BME280Driver.Start()
}

func (a *Agent) topic(sub string) string {
	return fmt.Sprintf("/devices/%s/%s", a.DeviceID, sub)
}

func (a *Agent) sendState() error {
	temp, err := a.BME280Driver.Temperature()
	if err != nil {
		return err
	}
	press, err := a.BME280Driver.Pressure()
	if err != nil {
		return err
	}
	humidity, err := a.BME280Driver.Humidity()
	if err != nil {
		return err
	}

	data := struct {
		Room     string  `json:"room"`
		Temp     float32 `json:"temperature"`
		Press    float32 `json:"pressure"`
		Humidity float32 `json:"humidity"`
	}{
		Room:     a.Room,
		Temp:     temp,
		Press:    press,
		Humidity: humidity,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return a.Client.Publish(a.topic("state"), 0, false, b).Error()
}

func main() {
	a := &Agent{
		ProjectID:  getenv("GOOGLE_CLOUD_PROJECT"),
		Region:     getenv("IOT_REGION"),
		RegistryID: getenv("IOT_REGISTRY"),
		DeviceID:   getenv("IOT_DEVICE"),

		Room: getenv("IOT_HOME_ROOM"),
	}

	if err := a.loadPrivateKey(getenv("IOT_KEY_FILE")); err != nil {
		log.Fatalf("loading private key: %s", err)
	}
	if err := a.connectMQTT(); err != nil {
		log.Fatalf("connecting to MQTT: %s", err)
	}
	if err := a.initBME280(); err != nil {
		log.Fatalf("starting BME280: %s", err)
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C
		if err := a.sendState(); err != nil {
			log.Printf("sending state: %s", err)
		}
	}
}
